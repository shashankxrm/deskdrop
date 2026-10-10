package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const requestIDHeader = "X-Request-Id"

type Server struct {
	log        *slog.Logger
	pool       *pgxpool.Pool
	pepper     string
	reg        *prometheus.Registry
	lat        *prometheus.HistogramVec
	pairings   prometheus.Counter
	claimFails *claimLimiter
	http.Handler
}

func New(log *slog.Logger, pool *pgxpool.Pool, pepper string) *Server {
	if log == nil {
		log = slog.Default()
	}
	reg := prometheus.NewRegistry()
	lat := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "deskdrop_http_request_duration_seconds",
		Help:    "HTTP request latency in seconds. route is the ServeMux pattern, never the raw path.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "code", "route"})
	pairings := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "deskdrop_pairings_total",
		Help: "Successful pairing claims (a device received a token).",
	})
	reg.MustRegister(lat, pairings)

	s := &Server{log: log, pool: pool, pepper: pepper, reg: reg, lat: lat, pairings: pairings, claimFails: newClaimLimiter()}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/readyz", s.readyz)
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	mux.HandleFunc("POST /v1/pairings", s.createPairing)
	mux.HandleFunc("POST /v1/pairings/claim", s.claimPairing)
	mux.HandleFunc("GET /v1/devices", s.requireAuth(s.listDevices))
	mux.HandleFunc("PATCH /v1/devices/{id}", s.requireAuth(s.renameDevice))
	mux.HandleFunc("DELETE /v1/devices/{id}", s.requireAuth(s.revokeDevice))
	mux.HandleFunc("POST /v1/links", s.requireAuth(s.shareLink))
	mux.HandleFunc("GET /v1/links/pending", s.requireAuth(s.pendingLinks))
	s.Handler = s.withRequestID(s.withMetrics(mux))
	return s
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if s.pool == nil {
		http.Error(w, "not ready\n", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.pool.Ping(ctx); err != nil {
		s.log.Error("readyz ping failed", "err", err, "request_id", requestIDFrom(r.Context()))
		http.Error(w, "not ready\n", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

type ctxKey int

const ctxRequestID ctxKey = 1

func requestIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(ctxRequestID).(string)
	return v
}

func (s *Server) withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := sanitizeRequestID(r.Header.Get(requestIDHeader))
		w.Header().Set(requestIDHeader, id)
		ctx := context.WithValue(r.Context(), ctxRequestID, id)
		r = r.WithContext(ctx)
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rec, r)
		attrs := []any{
			"request_id", id,
			"method", r.Method,
			"path", r.URL.Path,
			"route", routeLabel(r),
			"code", rec.code,
			"duration_ms", time.Since(start).Milliseconds(),
		}
		if isProbe(r.URL.Path) {
			s.log.Debug("request", attrs...)
			return
		}
		s.log.Info("request", attrs...)
	})
}

func (s *Server) withMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rec, r)
		s.lat.WithLabelValues(r.Method, strconv.Itoa(rec.code), routeLabel(r)).Observe(time.Since(start).Seconds())
	})
}

func routeLabel(r *http.Request) string {
	if p := r.Pattern; p != "" {
		return p
	}
	return "unmatched"
}

func isProbe(path string) bool {
	return path == "/healthz" || path == "/readyz"
}

func sanitizeRequestID(id string) string {
	id = strings.TrimSpace(id)
	if !validRequestID(id) {
		return newRequestID()
	}
	return id
}

func validRequestID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '-', c == '_':
		default:
			return false
		}
	}
	return true
}

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(b[:])
}

package httpserver

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shashankxrm/deskdrop/internal/testutil"
)

func TestHealthz(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	srv := New(log, nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(requestIDHeader, "test-request-id")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", rec.Code)
	}
	if body := rec.Body.String(); body != "ok\n" {
		t.Fatalf("body=%q", body)
	}
	if got := rec.Header().Get(requestIDHeader); got != "test-request-id" {
		t.Fatalf("X-Request-Id=%q", got)
	}
	if strings.Contains(buf.String(), "request") {
		t.Fatalf("probe must not log at INFO: %s", buf.String())
	}
}

func TestHealthz_debugLog(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	srv := New(log, nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(requestIDHeader, "test-request-id")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if !strings.Contains(buf.String(), `"request_id":"test-request-id"`) {
		t.Fatalf("debug log missing request_id: %s", buf.String())
	}
}

func TestRequestID_sanitize(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		keep  bool
	}{
		{name: "valid", in: "test-request_id-1", keep: true},
		{name: "empty", in: "", keep: false},
		{name: "spaces", in: "has space", keep: false},
		{name: "slash", in: "foo/bar", keep: false},
		{name: "too long", in: strings.Repeat("a", 65), keep: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := New(slog.New(slog.NewJSONHandler(io.Discard, nil)), nil)
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			if tc.in != "" {
				req.Header.Set(requestIDHeader, tc.in)
			}
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)
			got := rec.Header().Get(requestIDHeader)
			if tc.keep {
				if got != tc.in {
					t.Fatalf("got %q want %q", got, tc.in)
				}
				return
			}
			if got == "" || got == tc.in || !validRequestID(got) {
				t.Fatalf("expected generated id, got %q", got)
			}
		})
	}
}

func TestMetrics_histogramRoute(t *testing.T) {
	srv := New(slog.New(slog.NewJSONHandler(io.Discard, nil)), nil)
	hreq := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	hrec := httptest.NewRecorder()
	srv.ServeHTTP(hrec, hreq)

	nreq := httptest.NewRequest(http.MethodGet, "/no-such-route", nil)
	nrec := httptest.NewRecorder()
	srv.ServeHTTP(nrec, nreq)

	mreq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	mrec := httptest.NewRecorder()
	srv.ServeHTTP(mrec, mreq)
	if mrec.Code != http.StatusOK {
		t.Fatalf("metrics status=%d", mrec.Code)
	}
	body := mrec.Body.String()
	if !strings.Contains(body, `deskdrop_http_request_duration_seconds_bucket{code="200",method="GET",route="/healthz"`) {
		t.Fatalf("missing /healthz route label: %s", body)
	}
	if !strings.Contains(body, `route="unmatched"`) {
		t.Fatalf("missing unmatched route label: %s", body)
	}
	if strings.Contains(body, "/no-such-route") {
		t.Fatalf("raw path leaked into metrics: %s", body)
	}
}

func TestReadyz_unavailable(t *testing.T) {
	srv := New(slog.New(slog.NewJSONHandler(io.Discard, nil)), nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want 503", rec.Code)
	}
}

func TestReadyz_ok(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	pg := testutil.StartPostgres(ctx, t)
	pool, err := pgxpool.New(ctx, pg.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	srv := New(slog.New(slog.NewJSONHandler(io.Discard, nil)), pool)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}

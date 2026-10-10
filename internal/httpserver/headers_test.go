package httpserver

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_table(t *testing.T) {
	allow := "https://deskdrop.test"
	srv := New(slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, "test-pepper", allow)

	cases := []struct {
		name     string
		method   string
		path     string
		origin   string
		wantACAO bool
	}{
		{name: "allowed get", method: http.MethodGet, path: "/healthz", origin: allow, wantACAO: true},
		{name: "disallowed get", method: http.MethodGet, path: "/healthz", origin: "https://evil.example", wantACAO: false},
		{name: "no origin", method: http.MethodGet, path: "/healthz", origin: "", wantACAO: false},
		{name: "allowed preflight", method: http.MethodOptions, path: "/v1/links", origin: allow, wantACAO: true},
		{name: "disallowed preflight", method: http.MethodOptions, path: "/v1/links", origin: "https://evil.example", wantACAO: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)
			got := rec.Header().Get("Access-Control-Allow-Origin")
			if tc.wantACAO {
				if got != tc.origin {
					t.Fatalf("ACAO=%q want %q status=%d", got, tc.origin, rec.Code)
				}
				return
			}
			if got != "" {
				t.Fatalf("unexpected ACAO=%q", got)
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	srv := New(slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, "test-pepper")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	h := rec.Header()
	if h.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("nosniff missing: %v", h)
	}
	if h.Get("X-Frame-Options") != "DENY" {
		t.Fatal("frame")
	}
	if h.Get("Referrer-Policy") != "no-referrer" {
		t.Fatal("referrer")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}

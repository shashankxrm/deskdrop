package agent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestClient_claimPendingAck(t *testing.T) {
	var acks []string
	var pending []map[string]any
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/pairings/claim":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["type"] != "laptop" || body["code"] != "ABCD-EFGH" {
				t.Errorf("claim body %+v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"token": "tok", "device_id": "dev-1", "type": "laptop",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/links/pending":
			if got := r.Header.Get("Authorization"); got != "Bearer tok" {
				t.Errorf("auth %q", got)
			}
			mu.Lock()
			defer mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"links": pending})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/ack"):
			id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/links/"), "/ack")
			mu.Lock()
			acks = append(acks, id)
			pending = nil
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	c := &Client{Base: srv.URL, HTTP: srv.Client(), Sleep: noSleep}
	got, err := c.Claim(context.Background(), "abcd efgh", "e2e-mac")
	if err != nil || got.Token != "tok" {
		t.Fatalf("claim %+v %v", got, err)
	}
	c.Token = got.Token
	mu.Lock()
	pending = []map[string]any{{"id": "link-1", "url": "https://example.com/a", "created_at": "2026-01-01T00:00:00Z"}}
	mu.Unlock()
	links, err := c.Pending(context.Background())
	if err != nil || len(links) != 1 || links[0].ID != "link-1" {
		t.Fatalf("pending %+v %v", links, err)
	}
	if err := c.Ack(context.Background(), "link-1"); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(acks) != 1 || acks[0] != "link-1" {
		t.Fatalf("acks %v", acks)
	}
}

func TestRunner_opensOnceThenAckIdempotent(t *testing.T) {
	var (
		mu      sync.Mutex
		acks    int
		pending = []map[string]any{{"id": "L1", "url": "https://example.com/one", "created_at": "2026-01-01T00:00:00Z"}}
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/links/pending":
			mu.Lock()
			defer mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"links": pending})
		case strings.HasSuffix(r.URL.Path, "/ack"):
			mu.Lock()
			acks++
			pending = []map[string]any{}
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	var opened []string
	r := NewRunner(&Client{Base: srv.URL, Token: "t", HTTP: srv.Client(), Sleep: noSleep}, func(_ context.Context, u string) error {
		opened = append(opened, u)
		return nil
	})
	if err := r.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := r.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 1 || opened[0] != "https://example.com/one" {
		t.Fatalf("opened %v", opened)
	}
	mu.Lock()
	defer mu.Unlock()
	if acks != 1 {
		t.Fatalf("acks %d (second cycle must not ack empty pending)", acks)
	}
}

func TestRunner_openOnceIfAckFailsThenRetryAck(t *testing.T) {
	var (
		mu   sync.Mutex
		acks int
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/links/pending":
			_ = json.NewEncoder(w).Encode(map[string]any{"links": []map[string]any{
				{"id": "L1", "url": "https://example.com/x", "created_at": "2026-01-01T00:00:00Z"},
			}})
		case strings.HasSuffix(r.URL.Path, "/ack"):
			mu.Lock()
			acks++
			n := acks
			mu.Unlock()
			if n == 1 {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = io.WriteString(w, `{"error":"internal"}`)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	var opened int
	r := NewRunner(&Client{Base: srv.URL, Token: "t", HTTP: srv.Client(), Sleep: noSleep}, func(_ context.Context, _ string) error {
		opened++
		return nil
	})
	if err := r.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if opened != 1 {
		t.Fatalf("opened %d", opened)
	}
	if err := r.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if opened != 1 {
		t.Fatalf("reopened: %d", opened)
	}
	mu.Lock()
	defer mu.Unlock()
	if acks < 2 {
		t.Fatalf("acks %d want retry", acks)
	}
}

func TestRunner_rejectsNonHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"links": []map[string]any{
			{"id": "L1", "url": "javascript:alert(1)", "created_at": "2026-01-01T00:00:00Z"},
		}})
	}))
	t.Cleanup(srv.Close)
	r := NewRunner(&Client{Base: srv.URL, Token: "t", HTTP: srv.Client()}, func(_ context.Context, _ string) error {
		t.Fatal("must not open")
		return nil
	})
	if err := r.Cycle(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestClient_401NoRetry(t *testing.T) {
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":"unauthorized"}`)
	}))
	t.Cleanup(srv.Close)
	c := &Client{Base: srv.URL, Token: "x", HTTP: srv.Client(), Sleep: noSleep}
	_, err := c.Pending(context.Background())
	var api *APIError
	if !errors.As(err, &api) || api.Status != 401 {
		t.Fatalf("err %v", err)
	}
	if n != 1 {
		t.Fatalf("retries on 401: %d", n)
	}
}

func TestClient_retries429(t *testing.T) {
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n < 3 {
			w.WriteHeader(429)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"links": []any{}})
	}))
	t.Cleanup(srv.Close)
	c := &Client{Base: srv.URL, Token: "x", HTTP: srv.Client(), Sleep: noSleep}
	links, err := c.Pending(context.Background())
	if err != nil || len(links) != 0 {
		t.Fatalf("%v %+v", err, links)
	}
	if n != 3 {
		t.Fatalf("n %d", n)
	}
}

func TestDoubleAckStillOneOpen(t *testing.T) {
	acks := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/ack") {
			acks++
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"links": []map[string]any{
			{"id": "L1", "url": "https://example.com/z", "created_at": "2026-01-01T00:00:00Z"},
		}})
	}))
	t.Cleanup(srv.Close)
	var opened int
	r := NewRunner(&Client{Base: srv.URL, Token: "t", HTTP: srv.Client(), Sleep: noSleep}, func(_ context.Context, _ string) error {
		opened++
		return nil
	})
	_ = r.Cycle(context.Background())
	_ = r.Cycle(context.Background())
	if opened != 1 {
		t.Fatalf("opened %d", opened)
	}
	if acks != 2 {
		t.Fatalf("acks %d (server still returns the link; agent must not re-open)", acks)
	}
}

func noSleep(_ context.Context, _ time.Duration) error { return nil }

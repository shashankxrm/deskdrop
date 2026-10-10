package httpserver

import (
	"net/http"
	"testing"
	"time"
)

func TestWriteLimit_pairingCreatePerIP(t *testing.T) {
	srv, _ := pairingServer(t)
	ip := "198.51.100.50:9"
	var last int
	for i := 0; i < 6; i++ {
		rec := doJSON(t, srv, http.MethodPost, "/v1/pairings", ip, "", "")
		last = rec.Code
		if i < 5 && rec.Code != http.StatusCreated {
			t.Fatalf("create %d status=%d body=%s", i+1, rec.Code, rec.Body.String())
		}
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("6th pairing create status=%d want 429", last)
	}

	other := doJSON(t, srv, http.MethodPost, "/v1/pairings", "198.51.100.51:9", "", "")
	if other.Code != http.StatusCreated {
		t.Fatalf("other IP status=%d body=%s", other.Code, other.Body.String())
	}

	claim := doJSON(t, srv, http.MethodPost, "/v1/pairings/claim", ip, "", `{"code":"ZZZZ-ZZZZ","type":"phone"}`)
	if claim.Code == http.StatusTooManyRequests {
		t.Fatal("claim must not use the pairing-create bucket")
	}
	if claim.Code != http.StatusNotFound {
		t.Fatalf("claim status=%d want 404", claim.Code)
	}
}

func TestTokenBucket_shareBurst(t *testing.T) {
	b := newTokenBucket(shareBurst, shareRate)
	now := time.Now()
	for i := 0; i < shareBurst; i++ {
		if !b.allow("203.0.113.9", now) {
			t.Fatalf("share token %d denied", i+1)
		}
	}
	if b.allow("203.0.113.9", now) {
		t.Fatal("expected share 429 after burst")
	}
	if !b.allow("203.0.113.10", now) {
		t.Fatal("other IP should have its own bucket")
	}
}

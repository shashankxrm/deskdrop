package httpserver

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestHistory_cursorStable(t *testing.T) {
	srv, pool := pairingServer(t)
	phone, _, _ := pairDevice(t, srv, "phone")

	ids := make([]string, 5)
	for i := range ids {
		rec := doShare(t, srv, phone, "h-"+string(rune('a'+i)), `{"url":"https://example.com/`+string(rune('a'+i))+`"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("share %d status=%d body=%s", i, rec.Code, rec.Body.String())
		}
		var body struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		ids[i] = body.ID
		ts := time.Date(2026, 1, 1, 0, 0, i, 0, time.UTC)
		if _, err := pool.Exec(t.Context(), `UPDATE links SET created_at = $1 WHERE id = $2`, ts, body.ID); err != nil {
			t.Fatal(err)
		}
	}

	page1 := doJSON(t, srv, http.MethodGet, "/v1/links?limit=2", "", "Bearer "+phone, "")
	if page1.Code != http.StatusOK {
		t.Fatalf("page1 status=%d body=%s", page1.Code, page1.Body.String())
	}
	var wrap struct {
		Links      []map[string]any `json:"links"`
		NextCursor string           `json:"next_cursor"`
	}
	_ = json.Unmarshal(page1.Body.Bytes(), &wrap)
	if len(wrap.Links) != 2 || wrap.NextCursor == "" {
		t.Fatalf("page1 %#v", wrap)
	}
	if wrap.Links[0]["id"] != ids[4] || wrap.Links[1]["id"] != ids[3] {
		t.Fatalf("page1 order got %v want %s,%s", []any{wrap.Links[0]["id"], wrap.Links[1]["id"]}, ids[4], ids[3])
	}
	cursor := wrap.NextCursor

	newer := doShare(t, srv, phone, "h-new", `{"url":"https://example.com/new"}`)
	if newer.Code != http.StatusCreated {
		t.Fatalf("newer share status=%d", newer.Code)
	}

	page2 := doJSON(t, srv, http.MethodGet, "/v1/links?limit=2&cursor="+url.QueryEscape(cursor), "", "Bearer "+phone, "")
	_ = json.Unmarshal(page2.Body.Bytes(), &wrap)
	if len(wrap.Links) != 2 {
		t.Fatalf("page2 %#v", wrap)
	}
	if wrap.Links[0]["id"] != ids[2] || wrap.Links[1]["id"] != ids[1] {
		t.Fatalf("cursor slipped after insert: %#v", wrap.Links)
	}

	page3 := doJSON(t, srv, http.MethodGet, "/v1/links?limit=2&cursor="+url.QueryEscape(wrap.NextCursor), "", "Bearer "+phone, "")
	var last struct {
		Links      []map[string]any `json:"links"`
		NextCursor string           `json:"next_cursor"`
	}
	_ = json.Unmarshal(page3.Body.Bytes(), &last)
	if len(last.Links) != 1 || last.Links[0]["id"] != ids[0] || last.NextCursor != "" {
		t.Fatalf("page3 %#v", last)
	}

	other, _, _ := pairDevice(t, srv, "phone")
	iso := doJSON(t, srv, http.MethodGet, "/v1/links", "", "Bearer "+other, "")
	var empty struct {
		Links []map[string]any `json:"links"`
	}
	_ = json.Unmarshal(iso.Body.Bytes(), &empty)
	if len(empty.Links) != 0 {
		t.Fatal("history leaked across users")
	}

	if doJSON(t, srv, http.MethodGet, "/v1/links?limit=0", "", "Bearer "+phone, "").Code != http.StatusBadRequest {
		t.Fatal("limit 0")
	}
	if doJSON(t, srv, http.MethodGet, "/v1/links?limit=101", "", "Bearer "+phone, "").Code != http.StatusBadRequest {
		t.Fatal("limit 101")
	}
	if doJSON(t, srv, http.MethodGet, "/v1/links?cursor=not-a-cursor", "", "Bearer "+phone, "").Code != http.StatusBadRequest {
		t.Fatal("bad cursor")
	}
}

func TestAccount_deleteLeavesZeroRows(t *testing.T) {
	srv, pool := pairingServer(t)
	phone, _, _ := pairDevice(t, srv, "phone")
	laptopTok, _ := pairExtra(t, srv, phone, "laptop")
	if doShare(t, srv, phone, "gone", `{"url":"https://example.com/gone"}`).Code != http.StatusCreated {
		t.Fatal("share")
	}
	other, _, _ := pairDevice(t, srv, "laptop")
	if doShare(t, srv, other, "keep", `{"url":"https://example.com/keep"}`).Code != http.StatusCreated {
		t.Fatal("other share")
	}

	var userID string
	if err := pool.QueryRow(t.Context(),
		`SELECT user_id::text FROM links WHERE idempotency_key = 'gone'`).Scan(&userID); err != nil {
		t.Fatal(err)
	}

	del := doJSON(t, srv, http.MethodDelete, "/v1/account", "", "Bearer "+phone, "")
	if del.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", del.Code, del.Body.String())
	}
	if doJSON(t, srv, http.MethodGet, "/v1/devices", "", "Bearer "+phone, "").Code != http.StatusUnauthorized {
		t.Fatal("phone still authed")
	}
	if doJSON(t, srv, http.MethodGet, "/v1/links/pending", "", "Bearer "+laptopTok, "").Code != http.StatusUnauthorized {
		t.Fatal("laptop still authed")
	}

	var n int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM users WHERE id = $1`, userID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("user row remains")
	}
	for _, q := range []string{
		`SELECT count(*) FROM devices WHERE user_id = $1`,
		`SELECT count(*) FROM pairing_codes WHERE user_id = $1`,
		`SELECT count(*) FROM links WHERE user_id = $1`,
		`SELECT count(*) FROM link_deliveries d JOIN links l ON l.id = d.link_id WHERE l.user_id = $1`,
	} {
		if err := pool.QueryRow(t.Context(), q, userID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("leftover rows query %s count=%d", q, n)
		}
	}
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM links WHERE idempotency_key = 'keep'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal("other user's link was deleted")
	}
}

func TestRetention_purgesStampedOldDelivered(t *testing.T) {
	srv, pool := pairingServer(t)
	phone, _, _ := pairDevice(t, srv, "phone")
	laptopTok, _ := pairExtra(t, srv, phone, "laptop")

	old := doShare(t, srv, phone, "old", `{"url":"https://example.com/old"}`)
	fresh := doShare(t, srv, phone, "fresh", `{"url":"https://example.com/fresh"}`)
	pending := doShare(t, srv, phone, "pending", `{"url":"https://example.com/pending"}`)
	var ids [3]struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(old.Body.Bytes(), &ids[0])
	_ = json.Unmarshal(fresh.Body.Bytes(), &ids[1])
	_ = json.Unmarshal(pending.Body.Bytes(), &ids[2])

	ack := func(id string) {
		rec := doJSON(t, srv, http.MethodPost, "/v1/links/"+id+"/ack", "", "Bearer "+laptopTok, "")
		if rec.Code != http.StatusNoContent {
			t.Fatalf("ack %s status=%d", id, rec.Code)
		}
	}
	ack(ids[0].ID)
	ack(ids[1].ID)

	oldTs := time.Now().UTC().Add(-31 * 24 * time.Hour)
	if _, err := pool.Exec(t.Context(),
		`UPDATE links SET created_at = $1 WHERE id = $2 OR id = $3`,
		oldTs, ids[0].ID, ids[2].ID,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(),
		`UPDATE link_deliveries SET acked_at = $1 WHERE link_id = $2`,
		oldTs, ids[0].ID,
	); err != nil {
		t.Fatal(err)
	}

	if err := srv.purgeDelivered(t.Context()); err != nil {
		t.Fatal(err)
	}

	var n int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM links WHERE id = $1`, ids[0].ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("old delivered link should be purged")
	}
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM links WHERE id = $1`, ids[1].ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal("fresh acked link should remain")
	}
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM links WHERE id = $1`, ids[2].ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal("old unacked link should remain")
	}
}

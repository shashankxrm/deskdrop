package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func pairExtra(t *testing.T, srv http.Handler, existingToken, typ string) (token, deviceID string) {
	t.Helper()
	cre := doJSON(t, srv, http.MethodPost, "/v1/pairings", "", "Bearer "+existingToken, "")
	var c struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(cre.Body.Bytes(), &c); err != nil {
		t.Fatal(err)
	}
	rec := doJSON(t, srv, http.MethodPost, "/v1/pairings/claim", "", "", `{"code":"`+c.Code+`","type":"`+typ+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("extra claim status=%d body=%s", rec.Code, rec.Body.String())
	}
	var cl struct {
		Token    string `json:"token"`
		DeviceID string `json:"device_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &cl); err != nil {
		t.Fatal(err)
	}
	return cl.Token, cl.DeviceID
}

func doShare(t *testing.T, srv http.Handler, token, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/links", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.0.2.1:9"
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if key != "" {
		req.Header.Set(idempotencyHeader, key)
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func metricsText(t *testing.T, srv http.Handler) string {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("metrics status=%d", rec.Code)
	}
	return rec.Body.String()
}

func TestShare_tableValidation(t *testing.T) {
	srv, _ := pairingServer(t)
	phone, _, _ := pairDevice(t, srv, "phone")
	good := `{"url":"https://example.com/a","title":"A","userId":"ignored"}`

	cases := []struct {
		name   string
		token  string
		key    string
		body   string
		status int
	}{
		{"unauth", "", "k1", good, http.StatusUnauthorized},
		{"missing key", phone, "", good, http.StatusBadRequest},
		{"long key", phone, strings.Repeat("k", maxIdempotencyKey+1), good, http.StatusBadRequest},
		{"ftp", phone, "k-ftp", `{"url":"ftp://example.com"}`, http.StatusBadRequest},
		{"javascript", phone, "k-js", `{"url":"javascript:alert(1)"}`, http.StatusBadRequest},
		{"no host", phone, "k-host", `{"url":"https://"}`, http.StatusBadRequest},
		{"space in url", phone, "k-sp", `{"url":"https://example.com/a b"}`, http.StatusBadRequest},
		{"empty url", phone, "k-empty", `{"url":""}`, http.StatusBadRequest},
		{"bad json", phone, "k-json", `{`, http.StatusBadRequest},
		{"huge body", phone, "k-huge", `{"url":"https://example.com","title":"` + strings.Repeat("x", linkBodyMax) + `"}`, http.StatusRequestEntityTooLarge},
		{"ok http", phone, "k-ok", `{"url":"http://example.com/x"}`, http.StatusCreated},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := doShare(t, srv, tc.token, tc.key, tc.body)
			if got.Code != tc.status {
				t.Fatalf("status=%d want %d body=%s", got.Code, tc.status, got.Body.String())
			}
		})
	}
}

func TestShare_idempotentReplayAndFanout(t *testing.T) {
	srv, pool := pairingServer(t)
	phone, _, _ := pairDevice(t, srv, "phone")
	laptopTok, laptopID := pairExtra(t, srv, phone, "laptop")
	laptop2Tok, _ := pairExtra(t, srv, phone, "laptop")

	body := `{"url":"https://example.com/story","title":"Story","userId":"attacker"}`
	first := doShare(t, srv, phone, "same-key", body)
	if first.Code != http.StatusCreated {
		t.Fatalf("share status=%d body=%s", first.Code, first.Body.String())
	}
	var created struct {
		ID    string `json:"id"`
		URL   string `json:"url"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.Title != "Story" {
		t.Fatalf("share body %#v", created)
	}

	replay := doShare(t, srv, phone, "same-key", `{"url":"https://evil.example/other","title":"Other"}`)
	if replay.Code != http.StatusOK {
		t.Fatalf("replay status=%d body=%s", replay.Code, replay.Body.String())
	}
	var again struct {
		ID    string `json:"id"`
		URL   string `json:"url"`
		Title string `json:"title"`
	}
	_ = json.Unmarshal(replay.Body.Bytes(), &again)
	if again.ID != created.ID || again.URL != created.URL || again.Title != "Story" {
		t.Fatalf("replay mutated link: %#v vs %#v", again, created)
	}

	var n int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM links`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("links=%d want 1", n)
	}
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM link_deliveries`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("deliveries=%d want 2 laptops", n)
	}

	pending := doJSON(t, srv, http.MethodGet, "/v1/links/pending", "", "Bearer "+laptopTok, "")
	if pending.Code != http.StatusOK {
		t.Fatalf("pending status=%d body=%s", pending.Code, pending.Body.String())
	}
	var wrap struct {
		Links []map[string]any `json:"links"`
	}
	_ = json.Unmarshal(pending.Body.Bytes(), &wrap)
	if len(wrap.Links) != 1 || wrap.Links[0]["id"] != created.ID {
		t.Fatalf("pending %#v", wrap.Links)
	}

	pending2 := doJSON(t, srv, http.MethodGet, "/v1/links/pending", "", "Bearer "+laptop2Tok, "")
	_ = json.Unmarshal(pending2.Body.Bytes(), &wrap)
	if len(wrap.Links) != 1 {
		t.Fatalf("second laptop pending=%d", len(wrap.Links))
	}

	ack := doJSON(t, srv, http.MethodPost, "/v1/links/"+created.ID+"/ack", "", "Bearer "+laptopTok, "")
	if ack.Code != http.StatusNoContent {
		t.Fatalf("ack status=%d", ack.Code)
	}
	ack2 := doJSON(t, srv, http.MethodPost, "/v1/links/"+created.ID+"/ack", "", "Bearer "+laptopTok, "")
	if ack2.Code != http.StatusNoContent {
		t.Fatalf("double ack status=%d", ack2.Code)
	}

	empty := doJSON(t, srv, http.MethodGet, "/v1/links/pending", "", "Bearer "+laptopTok, "")
	_ = json.Unmarshal(empty.Body.Bytes(), &wrap)
	if len(wrap.Links) != 0 {
		t.Fatalf("acked still pending %#v", wrap.Links)
	}
	still := doJSON(t, srv, http.MethodGet, "/v1/links/pending", "", "Bearer "+laptop2Tok, "")
	_ = json.Unmarshal(still.Body.Bytes(), &wrap)
	if len(wrap.Links) != 1 {
		t.Fatal("ack on one laptop must not ack the other")
	}

	if doJSON(t, srv, http.MethodGet, "/v1/links/pending", "", "Bearer "+phone, "").Code != http.StatusForbidden {
		t.Fatal("phone must not pull")
	}
	if doJSON(t, srv, http.MethodPost, "/v1/links/"+created.ID+"/ack", "", "Bearer "+phone, "").Code != http.StatusForbidden {
		t.Fatal("phone must not ack")
	}

	if doJSON(t, srv, http.MethodPost, "/v1/links/not-a-uuid/ack", "", "Bearer "+laptopTok, "").Code != http.StatusBadRequest {
		t.Fatal("bad ack id")
	}
	if doJSON(t, srv, http.MethodPost, "/v1/links/00000000-0000-0000-0000-000000000000/ack", "", "Bearer "+laptopTok, "").Code != http.StatusNotFound {
		t.Fatal("missing delivery should 404")
	}

	_ = laptopID
	m := metricsText(t, srv)
	for _, name := range []string{"deskdrop_shares_total", "deskdrop_pulls_total", "deskdrop_acks_total"} {
		if !strings.Contains(m, name) {
			t.Fatalf("metrics missing %s:\n%s", name, m)
		}
	}
	if !strings.Contains(m, "deskdrop_shares_total 1") {
		t.Fatalf("replay counted as a share:\n%s", m)
	}
}

func TestShare_lateLaptopMissesOldLink(t *testing.T) {
	srv, _ := pairingServer(t)
	phone, _, _ := pairDevice(t, srv, "phone")
	if doShare(t, srv, phone, "early", `{"url":"https://example.com/early"}`).Code != http.StatusCreated {
		t.Fatal("share")
	}
	laptopTok, _ := pairExtra(t, srv, phone, "laptop")
	pending := doJSON(t, srv, http.MethodGet, "/v1/links/pending", "", "Bearer "+laptopTok, "")
	var wrap struct {
		Links []map[string]any `json:"links"`
	}
	_ = json.Unmarshal(pending.Body.Bytes(), &wrap)
	if len(wrap.Links) != 0 {
		t.Fatalf("late laptop should miss pre-pair shares, got %#v", wrap.Links)
	}
}

func TestShare_webCannotPull(t *testing.T) {
	srv, _ := pairingServer(t)
	_, _, rec := pairDevice(t, srv, "web")
	var ck *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == cookieName {
			ck = c
			break
		}
	}
	if ck == nil {
		t.Fatal("web cookie")
	}
	if doJSON(t, srv, http.MethodGet, "/v1/links/pending", "", "", "", ck).Code != http.StatusForbidden {
		t.Fatal("web pull")
	}
}

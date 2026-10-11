package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shashankxrm/deskdrop/internal/testutil"
	"github.com/shashankxrm/deskdrop/internal/tokhash"
)

const testPepper = "test-pepper"

func pairingServer(t *testing.T) (*Server, *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	pg := testutil.StartPostgres(ctx, t)
	pool, err := pgxpool.New(ctx, pg.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return New(slog.New(slog.NewJSONHandler(io.Discard, nil)), pool, testPepper), pool
}

func doJSON(t *testing.T, srv http.Handler, method, path, remote, auth, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if remote != "" {
		req.RemoteAddr = remote
	} else {
		req.RemoteAddr = "192.0.2.1:9"
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func pairDevice(t *testing.T, srv http.Handler, typ string) (token, deviceID string, rec *httptest.ResponseRecorder) {
	t.Helper()
	cre := doJSON(t, srv, http.MethodPost, "/v1/pairings", "", "", "")
	var c struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(cre.Body.Bytes(), &c); err != nil {
		t.Fatal(err)
	}
	rec = doJSON(t, srv, http.MethodPost, "/v1/pairings/claim", "", "", `{"code":"`+c.Code+`","type":"`+typ+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("claim status=%d body=%s", rec.Code, rec.Body.String())
	}
	var cl struct {
		Token    string `json:"token"`
		DeviceID string `json:"device_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &cl); err != nil {
		t.Fatal(err)
	}
	return cl.Token, cl.DeviceID, rec
}

func TestPairing_bootstrapClaimAndIdempotent(t *testing.T) {
	srv, pool := pairingServer(t)

	created := doJSON(t, srv, http.MethodPost, "/v1/pairings", "", "", `{"userId":"should-be-ignored"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var cre struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &cre); err != nil {
		t.Fatal(err)
	}

	claimBody := `{"code":"` + cre.Code + `","type":"laptop","name":"mac","userId":"nope"}`
	claimed := doJSON(t, srv, http.MethodPost, "/v1/pairings/claim", "", "", claimBody)
	if claimed.Code != http.StatusCreated {
		t.Fatalf("claim status=%d body=%s", claimed.Code, claimed.Body.String())
	}
	var cl struct {
		Token    string `json:"token"`
		DeviceID string `json:"device_id"`
	}
	if err := json.Unmarshal(claimed.Body.Bytes(), &cl); err != nil {
		t.Fatal(err)
	}
	if cl.Token == "" || cl.DeviceID == "" {
		t.Fatalf("missing token or device: %+v", cl)
	}

	var hash []byte
	if err := pool.QueryRow(context.Background(), `SELECT token_hash FROM devices WHERE id = $1`, cl.DeviceID).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(hash, tokhash.Sum(testPepper, cl.Token)) {
		t.Fatal("token_hash is not HMAC-SHA256(pepper, token)")
	}
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM devices WHERE convert_from(token_hash, 'UTF8') = $1`, cl.Token).Scan(&n); err != nil {
		// convert_from may fail on binary HMAC; that's fine — plaintext must not match as text either
		n = 0
	}
	if n != 0 {
		t.Fatal("plaintext token stored")
	}

	phone := doJSON(t, srv, http.MethodPost, "/v1/pairings/claim", "", "", `{"code":"`+cre.Code+`","type":"phone","name":"pixel"}`)
	if phone.Code != http.StatusCreated {
		t.Fatalf("second claim (phone, same QR) status=%d want 201 body=%s", phone.Code, phone.Body.String())
	}

	metrics := httptest.NewRecorder()
	srv.ServeHTTP(metrics, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(metrics.Body.String(), "deskdrop_pairings_total") {
		t.Fatalf("missing pairings counter: %s", metrics.Body.String())
	}
	if !strings.Contains(metrics.Body.String(), "deskdrop_pairings_total 2") {
		t.Fatalf("pairings counter not 2: %s", metrics.Body.String())
	}
}

func TestPairing_twoBootstrapsDifferentUsers(t *testing.T) {
	srv, pool := pairingServer(t)
	u := map[string]string{}
	for i := 0; i < 2; i++ {
		cre := doJSON(t, srv, http.MethodPost, "/v1/pairings", "", "", "")
		var c struct {
			Code string `json:"code"`
		}
		_ = json.Unmarshal(cre.Body.Bytes(), &c)
		cl := doJSON(t, srv, http.MethodPost, "/v1/pairings/claim", "", "", `{"code":"`+c.Code+`","type":"phone"}`)
		var res struct {
			DeviceID string `json:"device_id"`
		}
		_ = json.Unmarshal(cl.Body.Bytes(), &res)
		var uid string
		if err := pool.QueryRow(context.Background(), `SELECT user_id::text FROM devices WHERE id = $1`, res.DeviceID).Scan(&uid); err != nil {
			t.Fatal(err)
		}
		u[uid] = res.DeviceID
	}
	if len(u) != 2 {
		t.Fatalf("want 2 users, got %v", u)
	}
}

func TestPairing_expired(t *testing.T) {
	srv, pool := pairingServer(t)
	cre := doJSON(t, srv, http.MethodPost, "/v1/pairings", "", "", "")
	var c struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(cre.Body.Bytes(), &c)
	_, err := pool.Exec(context.Background(),
		`UPDATE pairing_codes SET expires_at = now() - interval '1 minute' WHERE code_hash = $1`,
		tokhash.Sum(testPepper, normalizeCode(c.Code)),
	)
	if err != nil {
		t.Fatal(err)
	}
	cl := doJSON(t, srv, http.MethodPost, "/v1/pairings/claim", "", "", `{"code":"`+c.Code+`","type":"web"}`)
	if cl.Code != http.StatusGone {
		t.Fatalf("status=%d want 410 body=%s", cl.Code, cl.Body.String())
	}
}

func TestPairing_authenticatedSecondDevice(t *testing.T) {
	srv, pool := pairingServer(t)
	cre := doJSON(t, srv, http.MethodPost, "/v1/pairings", "", "", "")
	var c struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(cre.Body.Bytes(), &c)
	first := doJSON(t, srv, http.MethodPost, "/v1/pairings/claim", "", "", `{"code":"`+c.Code+`","type":"phone"}`)
	var p struct {
		Token    string `json:"token"`
		DeviceID string `json:"device_id"`
	}
	_ = json.Unmarshal(first.Body.Bytes(), &p)

	secondCode := doJSON(t, srv, http.MethodPost, "/v1/pairings", "", "Bearer "+p.Token, "")
	if secondCode.Code != http.StatusCreated {
		t.Fatalf("auth create status=%d body=%s", secondCode.Code, secondCode.Body.String())
	}
	var c2 struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(secondCode.Body.Bytes(), &c2)
	second := doJSON(t, srv, http.MethodPost, "/v1/pairings/claim", "", "", `{"code":"`+c2.Code+`","type":"laptop"}`)
	if second.Code != http.StatusCreated {
		t.Fatalf("second claim status=%d body=%s", second.Code, second.Body.String())
	}
	var d2 struct {
		DeviceID string `json:"device_id"`
	}
	_ = json.Unmarshal(second.Body.Bytes(), &d2)

	var u1, u2 string
	_ = pool.QueryRow(context.Background(), `SELECT user_id::text FROM devices WHERE id = $1`, p.DeviceID).Scan(&u1)
	_ = pool.QueryRow(context.Background(), `SELECT user_id::text FROM devices WHERE id = $1`, d2.DeviceID).Scan(&u2)
	if u1 == "" || u1 != u2 {
		t.Fatalf("devices not same user: %s %s", u1, u2)
	}

	bad := doJSON(t, srv, http.MethodPost, "/v1/pairings", "", "Bearer not-a-real-token", "")
	if bad.Code != http.StatusUnauthorized {
		t.Fatalf("bad bearer status=%d", bad.Code)
	}
}

func TestPairing_webOmitsTokenSetsCookie(t *testing.T) {
	srv, _ := pairingServer(t)
	cre := doJSON(t, srv, http.MethodPost, "/v1/pairings", "", "", "")
	var c struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(cre.Body.Bytes(), &c)
	rec := doJSON(t, srv, http.MethodPost, "/v1/pairings/claim", "", "", `{"code":"`+c.Code+`","type":"web","name":"dash"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var cl struct {
		Token    string `json:"token"`
		DeviceID string `json:"device_id"`
		Type     string `json:"type"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &cl)
	if cl.Token != "" {
		t.Fatal("web claim must omit token from JSON")
	}
	if cl.Type != "web" || cl.DeviceID == "" {
		t.Fatalf("%+v", cl)
	}
	found := false
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == cookieName && ck.Value != "" && ck.HttpOnly {
			found = true
		}
	}
	if !found {
		t.Fatal("missing httpOnly session cookie")
	}
}

func TestPairing_claimRateLimit(t *testing.T) {
	srv, _ := pairingServer(t)
	ip := "198.51.100.9:4444"
	var last int
	for i := 0; i < 6; i++ {
		rec := doJSON(t, srv, http.MethodPost, "/v1/pairings/claim", ip, "", `{"code":"XXXX-XXXX","type":"phone"}`)
		last = rec.Code
		if i < 5 && rec.Code != http.StatusNotFound {
			t.Fatalf("attempt %d status=%d want 404 body=%s", i+1, rec.Code, rec.Body.String())
		}
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("6th wrong claim status=%d want 429", last)
	}
}

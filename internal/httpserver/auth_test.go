package httpserver

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAuth_bearerListRenameRevoke(t *testing.T) {
	srv, _ := pairingServer(t)
	token, id, _ := pairDevice(t, srv, "laptop")

	listed := doJSON(t, srv, http.MethodGet, "/v1/devices", "", "Bearer "+token, "")
	if listed.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listed.Code, listed.Body.String())
	}
	var wrap struct {
		Devices []map[string]any `json:"devices"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &wrap); err != nil {
		t.Fatal(err)
	}
	if len(wrap.Devices) != 1 {
		t.Fatalf("devices=%d", len(wrap.Devices))
	}

	ren := doJSON(t, srv, http.MethodPatch, "/v1/devices/"+id, "", "Bearer "+token,
		`{"name":"kitchen mac","userId":"attacker-user"}`)
	if ren.Code != http.StatusNoContent {
		t.Fatalf("rename status=%d body=%s", ren.Code, ren.Body.String())
	}
	listed = doJSON(t, srv, http.MethodGet, "/v1/devices", "", "Bearer "+token, "")
	_ = json.Unmarshal(listed.Body.Bytes(), &wrap)
	if wrap.Devices[0]["name"] != "kitchen mac" {
		t.Fatalf("name not updated: %#v", wrap.Devices[0])
	}

	rev := doJSON(t, srv, http.MethodDelete, "/v1/devices/"+id, "", "Bearer "+token, "")
	if rev.Code != http.StatusNoContent {
		t.Fatalf("revoke status=%d", rev.Code)
	}

	if doJSON(t, srv, http.MethodGet, "/v1/devices", "", "Bearer "+token, "").Code != http.StatusUnauthorized {
		t.Fatal("revoked token still listed devices")
	}
	if doJSON(t, srv, http.MethodGet, "/v1/links/pending", "", "Bearer "+token, "").Code != http.StatusUnauthorized {
		t.Fatal("revoked token can pull")
	}
	if doJSON(t, srv, http.MethodPost, "/v1/links", "", "Bearer "+token, `{"url":"https://example.com","userId":"x"}`).Code != http.StatusUnauthorized {
		t.Fatal("revoked token can share")
	}
}

func TestAuth_cookieVsBearer(t *testing.T) {
	srv, _ := pairingServer(t)
	_, _, claim := pairDevice(t, srv, "web")
	cookies := claim.Result().Cookies()
	var session *http.Cookie
	for _, c := range cookies {
		if c.Name == cookieName {
			session = c
			break
		}
	}
	if session == nil || !session.HttpOnly {
		t.Fatalf("missing httpOnly session cookie: %+v", cookies)
	}

	listed := doJSON(t, srv, http.MethodGet, "/v1/devices", "", "", "", session)
	if listed.Code != http.StatusOK {
		t.Fatalf("cookie list status=%d body=%s", listed.Code, listed.Body.String())
	}

	unlisted := doJSON(t, srv, http.MethodGet, "/v1/devices", "", "", "")
	if unlisted.Code != http.StatusUnauthorized {
		t.Fatalf("no creds status=%d", unlisted.Code)
	}
}

func TestAuth_userIdIgnoredOnRename(t *testing.T) {
	srv, pool := pairingServer(t)
	tokenA, idA, _ := pairDevice(t, srv, "phone")
	tokenB, _, _ := pairDevice(t, srv, "laptop")

	// Body userId must not let A rename B's device (different users).
	ren := doJSON(t, srv, http.MethodPatch, "/v1/devices/"+idA, "", "Bearer "+tokenB,
		`{"name":"stolen","userId":"`+idA+`"}`)
	if ren.Code != http.StatusNotFound {
		t.Fatalf("cross-user rename status=%d want 404", ren.Code)
	}
	var n int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM devices WHERE id = $1 AND name = ''`, idA).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal("A's device name should be unchanged")
	}
	_ = tokenA
}

func TestAuth_secondDeviceSurvivesRevokeOfFirst(t *testing.T) {
	srv, _ := pairingServer(t)
	token1, id1, _ := pairDevice(t, srv, "phone")
	code := doJSON(t, srv, http.MethodPost, "/v1/pairings", "", "Bearer "+token1, "")
	var c struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(code.Body.Bytes(), &c)
	cl := doJSON(t, srv, http.MethodPost, "/v1/pairings/claim", "", "", `{"code":"`+c.Code+`","type":"laptop"}`)
	var d2 struct {
		Token    string `json:"token"`
		DeviceID string `json:"device_id"`
	}
	_ = json.Unmarshal(cl.Body.Bytes(), &d2)

	if doJSON(t, srv, http.MethodDelete, "/v1/devices/"+id1, "", "Bearer "+d2.Token, "").Code != http.StatusNoContent {
		t.Fatal("revoke first from second")
	}
	if doJSON(t, srv, http.MethodGet, "/v1/devices", "", "Bearer "+token1, "").Code != http.StatusUnauthorized {
		t.Fatal("revoked phone still authed")
	}
	if doJSON(t, srv, http.MethodGet, "/v1/devices", "", "Bearer "+d2.Token, "").Code != http.StatusOK {
		t.Fatal("laptop should still auth")
	}
	if doJSON(t, srv, http.MethodGet, "/v1/links/pending", "", "Bearer "+d2.Token, "").Code != http.StatusOK {
		t.Fatal("authed laptop pull")
	}
}

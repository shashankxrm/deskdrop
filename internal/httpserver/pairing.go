package httpserver

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shashankxrm/deskdrop/internal/tokhash"
)

const (
	pairingTTL     = 3 * time.Minute
	pairingBodyMax = 4096
	codeAlphabet   = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	codeLen        = 8
)

type createPairingBody struct {
	UserID json.RawMessage `json:"userId"`
}

type claimPairingBody struct {
	Code   string          `json:"code"`
	Type   string          `json:"type"`
	Name   string          `json:"name"`
	UserID json.RawMessage `json:"userId"`
}

func (s *Server) createPairing(w http.ResponseWriter, r *http.Request) {
	var body createPairingBody
	if r.ContentLength > 0 {
		_ = decodeJSON(w, r, pairingBodyMax, &body)
	}
	_ = body.UserID // identity never comes from JSON

	var userID uuid.UUID
	auth := r.Header.Get("Authorization")
	if auth == "" {
		if err := s.pool.QueryRow(r.Context(), `INSERT INTO users DEFAULT VALUES RETURNING id`).Scan(&userID); err != nil {
			s.log.Error("bootstrap user", "err", err, "request_id", requestIDFrom(r.Context()))
			writeErr(w, http.StatusInternalServerError, "internal")
			return
		}
	} else {
		token, ok := bearerToken(auth)
		if !ok {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		err := s.pool.QueryRow(r.Context(),
			`SELECT user_id FROM devices WHERE token_hash = $1 AND revoked_at IS NULL`,
			tokhash.Sum(s.pepper, token),
		).Scan(&userID)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
	}

	plain, err := randomCode()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	expires := time.Now().UTC().Add(pairingTTL)
	_, err = s.pool.Exec(r.Context(),
		`INSERT INTO pairing_codes (user_id, code_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokhash.Sum(s.pepper, plain), expires,
	)
	if err != nil {
		s.log.Error("insert pairing code", "err", err, "request_id", requestIDFrom(r.Context()))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"code":       displayCode(plain),
		"expires_at": expires.Format(time.RFC3339),
	})
}

func (s *Server) claimPairing(w http.ResponseWriter, r *http.Request) {
	var body claimPairingBody
	if err := decodeJSON(w, r, pairingBodyMax, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	_ = body.UserID
	code := normalizeCode(body.Code)
	if code == "" || !validDeviceType(body.Type) {
		s.rejectClaim(w, r, http.StatusBadRequest, "invalid claim")
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	defer tx.Rollback(r.Context())

	var (
		codeID    uuid.UUID
		userID    uuid.UUID
		expiresAt time.Time
		usedAt    *time.Time
	)
	err = tx.QueryRow(r.Context(),
		`SELECT id, user_id, expires_at, used_at FROM pairing_codes WHERE code_hash = $1 FOR UPDATE`,
		tokhash.Sum(s.pepper, code),
	).Scan(&codeID, &userID, &expiresAt, &usedAt)
	if err != nil {
		s.rejectClaim(w, r, http.StatusNotFound, "not found")
		return
	}
	if usedAt != nil {
		s.rejectClaim(w, r, http.StatusConflict, "already used")
		return
	}
	if time.Now().UTC().After(expiresAt) {
		s.rejectClaim(w, r, http.StatusGone, "expired")
		return
	}

	token, err := randomToken()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	var deviceID uuid.UUID
	err = tx.QueryRow(r.Context(),
		`INSERT INTO devices (user_id, name, type, token_hash) VALUES ($1, $2, $3, $4) RETURNING id`,
		userID, strings.TrimSpace(body.Name), body.Type, tokhash.Sum(s.pepper, token),
	).Scan(&deviceID)
	if err != nil {
		s.log.Error("insert device", "err", err, "request_id", requestIDFrom(r.Context()))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	if _, err := tx.Exec(r.Context(), `UPDATE pairing_codes SET used_at = now() WHERE id = $1`, codeID); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}

	s.pairings.Inc()
	writeJSON(w, http.StatusCreated, map[string]any{
		"token":     token,
		"device_id": deviceID.String(),
		"type":      body.Type,
	})
}

func (s *Server) rejectClaim(w http.ResponseWriter, r *http.Request, status int, msg string) {
	n := s.claimFails.recordFailure(clientIP(r), time.Now())
	if n >= claimFailLimit+1 {
		writeErr(w, http.StatusTooManyRequests, "rate limited")
		return
	}
	writeErr(w, status, msg)
}

func bearerToken(h string) (string, bool) {
	const p = "Bearer "
	if !strings.HasPrefix(h, p) {
		return "", false
	}
	t := strings.TrimSpace(strings.TrimPrefix(h, p))
	return t, t != ""
}

func validDeviceType(t string) bool {
	return t == "phone" || t == "laptop" || t == "web"
}

func normalizeCode(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if r == '-' || r == ' ' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func displayCode(plain string) string {
	if len(plain) == codeLen {
		return plain[:4] + "-" + plain[4:]
	}
	return plain
}

func randomCode() (string, error) {
	var raw [codeLen]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	out := make([]byte, codeLen)
	for i := 0; i < codeLen; i++ {
		out[i] = codeAlphabet[int(raw[i])%len(codeAlphabet)]
	}
	return string(out), nil
}

func randomToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

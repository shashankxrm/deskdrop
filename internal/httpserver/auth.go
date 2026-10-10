package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/shashankxrm/deskdrop/internal/tokhash"
)

const (
	cookieName   = "deskdrop_device"
	cookieMaxAge = 365 * 24 * 3600
)

const ctxDevice ctxKey = 2

// Device is the authenticated caller. User id always comes from here, never JSON.
type Device struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Type   string
	Name   string
}

func deviceFrom(ctx context.Context) (Device, bool) {
	d, ok := ctx.Value(ctxDevice).(Device)
	return d, ok
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		d, err := s.authenticate(r)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxDevice, d)))
	}
}

func hasCredentials(r *http.Request) bool {
	if r.Header.Get("Authorization") != "" {
		return true
	}
	_, err := r.Cookie(cookieName)
	return err == nil
}

func (s *Server) authenticate(r *http.Request) (Device, error) {
	token, ok := tokenFromRequest(r)
	if !ok {
		return Device{}, errUnauthorized
	}
	var d Device
	err := s.pool.QueryRow(r.Context(),
		`SELECT id, user_id, type, name FROM devices WHERE token_hash = $1 AND revoked_at IS NULL`,
		tokhash.Sum(s.pepper, token),
	).Scan(&d.ID, &d.UserID, &d.Type, &d.Name)
	if err != nil {
		return Device{}, errUnauthorized
	}
	_, _ = s.pool.Exec(r.Context(), `UPDATE devices SET last_seen_at = now() WHERE id = $1`, d.ID)
	return d, nil
}

func tokenFromRequest(r *http.Request) (string, bool) {
	if t, ok := bearerToken(r.Header.Get("Authorization")); ok {
		return t, true
	}
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" {
		return "", false
	}
	return c.Value, true
}

func sessionCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // D0 sets HTTPS; local Compose is HTTP
	}
}

var errUnauthorized = errors.New("unauthorized")

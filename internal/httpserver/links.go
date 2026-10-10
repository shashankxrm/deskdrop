package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	linkBodyMax       = 16 << 10
	maxURLLen         = 2048
	maxTitleLen       = 512
	maxIdempotencyKey = 128
	idempotencyHeader = "Idempotency-Key"
)

type shareBody struct {
	URL    string          `json:"url"`
	Title  string          `json:"title"`
	UserID json.RawMessage `json:"userId"`
}

func (s *Server) shareLink(w http.ResponseWriter, r *http.Request) {
	d, ok := deviceFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	key := strings.TrimSpace(r.Header.Get(idempotencyHeader))
	if key == "" || utf8.RuneCountInString(key) > maxIdempotencyKey {
		writeErr(w, http.StatusBadRequest, "idempotency key required")
		return
	}
	var body shareBody
	if err := decodeJSON(w, r, linkBodyMax, &body); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeErr(w, http.StatusRequestEntityTooLarge, "payload too large")
			return
		}
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	_ = body.UserID
	u, err := validateShareURL(body.URL)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid url")
		return
	}
	title := strings.TrimSpace(body.Title)
	if utf8.RuneCountInString(title) > maxTitleLen {
		writeErr(w, http.StatusBadRequest, "title too long")
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	defer tx.Rollback(r.Context())

	var (
		linkID      uuid.UUID
		storedURL   string
		storedTitle *string
	)
	err = tx.QueryRow(r.Context(),
		`INSERT INTO links (user_id, created_by_device_id, url, title, idempotency_key)
		 VALUES ($1, $2, $3, NULLIF($4, ''), $5)
		 ON CONFLICT (user_id, idempotency_key) DO NOTHING
		 RETURNING id, url, title`,
		d.UserID, d.ID, u, title, key,
	).Scan(&linkID, &storedURL, &storedTitle)
	created := err == nil
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(r.Context(),
			`SELECT id, url, title FROM links WHERE user_id = $1 AND idempotency_key = $2`,
			d.UserID, key,
		).Scan(&linkID, &storedURL, &storedTitle)
	}
	if err != nil {
		s.log.Error("share insert", "err", err, "request_id", requestIDFrom(r.Context()))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}

	if created {
		if _, err := tx.Exec(r.Context(),
			`INSERT INTO link_deliveries (link_id, device_id)
			 SELECT $1, id FROM devices
			 WHERE user_id = $2 AND type = 'laptop' AND revoked_at IS NULL
			 ON CONFLICT DO NOTHING`,
			linkID, d.UserID,
		); err != nil {
			writeErr(w, http.StatusInternalServerError, "internal")
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}

	status := http.StatusOK
	if created {
		s.shares.Inc()
		status = http.StatusCreated
	}
	resp := map[string]any{"id": linkID.String(), "url": storedURL}
	if storedTitle != nil && *storedTitle != "" {
		resp["title"] = *storedTitle
	}
	writeJSON(w, status, resp)
}

func (s *Server) pendingLinks(w http.ResponseWriter, r *http.Request) {
	d, ok := deviceFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if d.Type != "laptop" {
		writeErr(w, http.StatusForbidden, "laptop only")
		return
	}
	rows, err := s.pool.Query(r.Context(),
		`SELECT l.id, l.url, l.title, l.created_at
		 FROM link_deliveries d
		 JOIN links l ON l.id = d.link_id
		 WHERE d.device_id = $1 AND d.acked_at IS NULL
		 ORDER BY l.created_at, l.id`,
		d.ID,
	)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	defer rows.Close()

	out := make([]map[string]any, 0)
	for rows.Next() {
		var (
			id      uuid.UUID
			u       string
			title   *string
			created time.Time
		)
		if err := rows.Scan(&id, &u, &title, &created); err != nil {
			writeErr(w, http.StatusInternalServerError, "internal")
			return
		}
		item := map[string]any{
			"id":         id.String(),
			"url":        u,
			"created_at": created.UTC().Format(time.RFC3339),
		}
		if title != nil && *title != "" {
			item["title"] = *title
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	s.pulls.Inc()
	writeJSON(w, http.StatusOK, map[string]any{"links": out})
}

func (s *Server) ackLink(w http.ResponseWriter, r *http.Request) {
	d, ok := deviceFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if d.Type != "laptop" {
		writeErr(w, http.StatusForbidden, "laptop only")
		return
	}
	linkID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var n int
	if err := s.pool.QueryRow(r.Context(),
		`SELECT count(*) FROM link_deliveries WHERE link_id = $1 AND device_id = $2`,
		linkID, d.ID,
	).Scan(&n); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	if n == 0 {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	_, _ = s.pool.Exec(r.Context(),
		`UPDATE link_deliveries SET acked_at = now()
		 WHERE link_id = $1 AND device_id = $2 AND acked_at IS NULL`,
		linkID, d.ID,
	)
	s.acks.Inc()
	w.WriteHeader(http.StatusNoContent)
}

func validateShareURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || utf8.RuneCountInString(raw) > maxURLLen {
		return "", errInvalidURL
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", errInvalidURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errInvalidURL
	}
	if strings.ContainsAny(raw, " \t\r\n") {
		return "", errInvalidURL
	}
	return u.String(), nil
}

var errInvalidURL = errors.New("invalid url")

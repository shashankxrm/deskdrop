package httpserver

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	historyDefaultLimit = 20
	historyMaxLimit     = 100
)

func (s *Server) listHistory(w http.ResponseWriter, r *http.Request) {
	d, ok := deviceFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	limit := historyDefaultLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > historyMaxLimit {
			writeErr(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = n
	}

	var (
		rows pgx.Rows
		err  error
	)
	if raw := strings.TrimSpace(r.URL.Query().Get("cursor")); raw != "" {
		t, id, cerr := decodeHistoryCursor(raw)
		if cerr != nil {
			writeErr(w, http.StatusBadRequest, "invalid cursor")
			return
		}
		rows, err = s.pool.Query(r.Context(),
			`SELECT id, url, title, created_at FROM links
			 WHERE user_id = $1 AND (created_at, id) < ($2, $3)
			 ORDER BY created_at DESC, id DESC
			 LIMIT $4`,
			d.UserID, t, id, limit+1,
		)
	} else {
		rows, err = s.pool.Query(r.Context(),
			`SELECT id, url, title, created_at FROM links
			 WHERE user_id = $1
			 ORDER BY created_at DESC, id DESC
			 LIMIT $2`,
			d.UserID, limit+1,
		)
	}
	if err != nil {
		s.log.Error("history", "err", err, "request_id", requestIDFrom(r.Context()))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	defer rows.Close()

	type row struct {
		id      uuid.UUID
		url     string
		title   *string
		created time.Time
	}
	got := make([]row, 0, limit+1)
	for rows.Next() {
		var item row
		if err := rows.Scan(&item.id, &item.url, &item.title, &item.created); err != nil {
			writeErr(w, http.StatusInternalServerError, "internal")
			return
		}
		got = append(got, item)
	}
	if err := rows.Err(); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}

	hasMore := len(got) > limit
	if hasMore {
		got = got[:limit]
	}
	out := make([]map[string]any, 0, len(got))
	for _, item := range got {
		m := map[string]any{
			"id":         item.id.String(),
			"url":        item.url,
			"created_at": item.created.UTC().Format(time.RFC3339),
		}
		if item.title != nil && *item.title != "" {
			m["title"] = *item.title
		}
		out = append(out, m)
	}
	resp := map[string]any{"links": out}
	if hasMore && len(got) > 0 {
		last := got[len(got)-1]
		resp["next_cursor"] = encodeHistoryCursor(last.created, last.id)
	}
	writeJSON(w, http.StatusOK, resp)
}

func encodeHistoryCursor(created time.Time, id uuid.UUID) string {
	plain := created.UTC().Format(time.RFC3339Nano) + " " + id.String()
	return base64.RawURLEncoding.EncodeToString([]byte(plain))
}

func decodeHistoryCursor(raw string) (time.Time, uuid.UUID, error) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	ts, idStr, ok := strings.Cut(string(b), " ")
	if !ok {
		return time.Time{}, uuid.Nil, errInvalidCursor
	}
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	return t, id, nil
}

var errInvalidCursor = errors.New("invalid cursor")

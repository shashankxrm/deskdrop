package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type renameDeviceBody struct {
	Name   string          `json:"name"`
	UserID json.RawMessage `json:"userId"`
}

func (s *Server) listDevices(w http.ResponseWriter, r *http.Request) {
	d, ok := deviceFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, err := s.pool.Query(r.Context(),
		`SELECT id, name, type, created_at, last_seen_at
		 FROM devices WHERE user_id = $1 AND revoked_at IS NULL
		 ORDER BY created_at`,
		d.UserID,
	)
	if err != nil {
		s.log.Error("list devices", "err", err, "request_id", requestIDFrom(r.Context()))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	defer rows.Close()

	out := make([]map[string]any, 0)
	for rows.Next() {
		var (
			id        uuid.UUID
			name      string
			typ       string
			created   time.Time
			lastSeen  *time.Time
		)
		if err := rows.Scan(&id, &name, &typ, &created, &lastSeen); err != nil {
			writeErr(w, http.StatusInternalServerError, "internal")
			return
		}
		item := map[string]any{
			"id":         id.String(),
			"name":       name,
			"type":       typ,
			"created_at": created.UTC().Format(time.RFC3339),
		}
		if lastSeen != nil {
			item["last_seen_at"] = lastSeen.UTC().Format(time.RFC3339)
		}
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": out})
}

func (s *Server) renameDevice(w http.ResponseWriter, r *http.Request) {
	d, ok := deviceFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body renameDeviceBody
	if err := decodeJSON(w, r, pairingBodyMax, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	_ = body.UserID
	name := strings.TrimSpace(body.Name)
	if name == "" || len(name) > 64 {
		writeErr(w, http.StatusBadRequest, "invalid name")
		return
	}
	tag, err := s.pool.Exec(r.Context(),
		`UPDATE devices SET name = $1 WHERE id = $2 AND user_id = $3 AND revoked_at IS NULL`,
		name, id, d.UserID,
	)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	if tag.RowsAffected() == 0 {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) revokeDevice(w http.ResponseWriter, r *http.Request) {
	d, ok := deviceFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	tag, err := s.pool.Exec(r.Context(),
		`UPDATE devices SET revoked_at = now() WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`,
		id, d.UserID,
	)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	if tag.RowsAffected() == 0 {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// shareLink and pendingLinks are auth-gated stubs until B4.
func (s *Server) shareLink(w http.ResponseWriter, r *http.Request) {
	_, ok := deviceFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeErr(w, http.StatusNotImplemented, "not implemented")
}

func (s *Server) pendingLinks(w http.ResponseWriter, r *http.Request) {
	_, ok := deviceFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeErr(w, http.StatusNotImplemented, "not implemented")
}

package httpserver

import "net/http"

func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request) {
	d, ok := deviceFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM users WHERE id = $1`, d.UserID); err != nil {
		s.log.Error("delete account", "err", err, "request_id", requestIDFrom(r.Context()))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

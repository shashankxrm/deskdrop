// A2 placeholder: /healthz only. B1 replaces this with slog, timeouts, readyz, and /metrics.
package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	log.Printf("deskdrop api placeholder listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// Package handlers holds the API's HTTP handlers.
package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

const healthTimeout = 2 * time.Second

type Pinger interface {
	Ping(ctx context.Context) error
}

// Health reports whether the database answers. It is for container
// health checks and is not exposed through the reverse proxy.
func Health(db Pinger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), healthTimeout)
		defer cancel()

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if err := db.Ping(ctx); err != nil {
			slog.WarnContext(r.Context(), "health check failed", "error", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("database unavailable\n"))
			return
		}
		_, _ = w.Write([]byte("ok\n"))
	})
}

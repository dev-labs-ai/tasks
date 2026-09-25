// Package middleware wraps the API's routes with cross-cutting behavior.
package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/dev-labs-ai/tasks/apps/api/internal/handlers"
)

// Log records one line per request. It logs the path without the query
// string, and never headers, so cookies and tokens stay out of the logs.
func Log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		slog.InfoContext(r.Context(), "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", time.Since(start),
		)
	})
}

// Recover turns a panic into 500 INTERNAL_ERROR. The panic value and stack
// go to the logs only.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if v == http.ErrAbortHandler {
				panic(v)
			}
			slog.ErrorContext(r.Context(), "panic serving request",
				"method", r.Method,
				"path", r.URL.Path,
				"panic", v,
				"stack", string(debug.Stack()),
			)
			handlers.WriteError(w, handlers.InternalError())
		}()
		next.ServeHTTP(w, r)
	})
}

// CrossOrigin rejects non-safe cross-origin browser requests with
// 403 FORBIDDEN_ORIGIN (tech spec §5.5).
func CrossOrigin(next http.Handler) http.Handler {
	protection := http.NewCrossOriginProtection()
	protection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlers.WriteError(w, handlers.ForbiddenOrigin())
	}))
	return protection.Handler(next)
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(status int) {
	if !r.wroteHeader {
		r.status = status
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	r.wroteHeader = true
	return r.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

package handlers

import (
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/dev-labs-ai/tasks/apps/api/internal/services"
)

// writeServiceError maps a service error to its response (tech spec
// §6.4). Anything unexpected is logged and answered with a bare 500.
func writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	if ve, ok := errors.AsType[*services.ValidationError](err); ok {
		WriteError(w, ValidationFailed(ve.Fields))
		return
	}
	if tooMany, ok := errors.AsType[*services.TooManyAttemptsError](err); ok {
		w.Header().Set("Retry-After", retryAfter(tooMany.RetryAt))
		WriteError(w, &Error{Status: http.StatusTooManyRequests, Code: CodeTooManyAttempts, Message: "Too many login attempts; try again later"})
		return
	}

	switch {
	case errors.Is(err, services.ErrEmailTaken):
		WriteError(w, &Error{Status: http.StatusConflict, Code: CodeEmailTaken, Message: "This email is already registered"})
	case errors.Is(err, services.ErrInvalidCredentials):
		WriteError(w, &Error{Status: http.StatusUnauthorized, Code: CodeInvalidCredentials, Message: "Invalid email or password"})
	case errors.Is(err, services.ErrUnauthenticated):
		WriteError(w, &Error{Status: http.StatusUnauthorized, Code: CodeUnauthenticated, Message: "Log in to continue"})
	case errors.Is(err, services.ErrTaskNotFound):
		WriteError(w, &Error{Status: http.StatusNotFound, Code: CodeTaskNotFound, Message: "Task not found"})
	case errors.Is(err, services.ErrEmptyUpdate):
		WriteError(w, MalformedRequest())
	default:
		slog.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		WriteError(w, InternalError())
	}
}

// retryAfter returns whole seconds until t, rounded up and at least 1.
func retryAfter(t time.Time) string {
	seconds := math.Ceil(time.Until(t).Seconds())
	return strconv.Itoa(max(1, int(seconds)))
}

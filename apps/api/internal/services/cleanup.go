package services

import (
	"context"
	"log/slog"
	"time"
)

// CleanupInterval is how often expired sessions and ended login windows
// are deleted.
const CleanupInterval = time.Hour

// CleanupQueries are the deletes RunCleanup runs.
type CleanupQueries interface {
	DeleteExpiredSessions(ctx context.Context) (int64, error)
	DeleteStaleLoginAttempts(ctx context.Context) (int64, error)
}

// RunCleanup deletes expired sessions and stale login attempts once at
// start and then every interval, until ctx is cancelled. Expired rows are
// already ignored by the queries that read them; this only keeps the
// tables small.
func RunCleanup(ctx context.Context, q CleanupQueries, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		cleanup(ctx, q)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func cleanup(ctx context.Context, q CleanupQueries) {
	if n, err := q.DeleteExpiredSessions(ctx); err != nil {
		if ctx.Err() == nil {
			slog.ErrorContext(ctx, "delete expired sessions", "error", err)
		}
	} else if n > 0 {
		slog.InfoContext(ctx, "deleted expired sessions", "count", n)
	}

	if n, err := q.DeleteStaleLoginAttempts(ctx); err != nil {
		if ctx.Err() == nil {
			slog.ErrorContext(ctx, "delete stale login attempts", "error", err)
		}
	} else if n > 0 {
		slog.InfoContext(ctx, "deleted stale login attempts", "count", n)
	}
}

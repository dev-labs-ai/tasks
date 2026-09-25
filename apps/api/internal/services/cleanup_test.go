package services

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

type countingCleanup struct {
	sessions, attempts atomic.Int32
}

func (c *countingCleanup) DeleteExpiredSessions(context.Context) (int64, error) {
	c.sessions.Add(1)
	return 0, nil
}

func (c *countingCleanup) DeleteStaleLoginAttempts(context.Context) (int64, error) {
	c.attempts.Add(1)
	return 0, nil
}

func TestRunCleanupRunsOnScheduleAndStops(t *testing.T) {
	// synctest's fake clock lets three hours pass instantly.
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		q := &countingCleanup{}
		done := make(chan struct{})

		go func() {
			RunCleanup(ctx, q, time.Hour)
			close(done)
		}()

		synctest.Wait()
		if s, a := q.sessions.Load(), q.attempts.Load(); s != 1 || a != 1 {
			t.Fatalf("at start: %d session and %d attempt cleanups, want 1 each", s, a)
		}

		time.Sleep(3*time.Hour + time.Second)
		synctest.Wait()
		if s, a := q.sessions.Load(), q.attempts.Load(); s != 4 || a != 4 {
			t.Errorf("after 3 hours: %d session and %d attempt cleanups, want 4 each", s, a)
		}

		cancel()
		synctest.Wait()
		select {
		case <-done:
		default:
			t.Fatal("RunCleanup still running after its context was cancelled")
		}
	})
}

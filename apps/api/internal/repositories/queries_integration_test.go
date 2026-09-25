//go:build integration

package repositories_test

import (
	"errors"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-labs-ai/tasks/apps/api/internal/repositories"
	"github.com/dev-labs-ai/tasks/apps/api/internal/testdb"
)

func TestMain(m *testing.M) {
	os.Exit(testdb.Run(m))
}

func TestListTasksOrdersNewestFirstAndFilters(t *testing.T) {
	ctx := t.Context()
	pool := testdb.NewMigratedDatabase(t)
	q := repositories.New(pool)
	ana := createUser(t, q, "ana@example.com")
	bob := createUser(t, q, "bob@example.com")

	oldest := createTask(t, q, ana, "oldest")
	tied := createTask(t, q, ana, "tied with oldest, higher id")
	newest := createTask(t, q, ana, "newest")
	createTask(t, q, bob, "someone else's")

	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	setCreatedAt(t, pool, oldest, base)
	setCreatedAt(t, pool, tied, base)
	setCreatedAt(t, pool, newest, base.Add(time.Minute))

	if _, err := q.UpdateTask(ctx, repositories.UpdateTaskParams{
		ID: tied, UserID: ana, Completed: pgtype.Bool{Bool: true, Valid: true},
	}); err != nil {
		t.Fatalf("complete task: %v", err)
	}

	tests := []struct {
		name      string
		completed pgtype.Bool
		want      []int64
	}{
		{"all", pgtype.Bool{}, []int64{newest, tied, oldest}},
		{"completed", pgtype.Bool{Bool: true, Valid: true}, []int64{tied}},
		{"pending", pgtype.Bool{Bool: false, Valid: true}, []int64{newest, oldest}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks, err := q.ListTasks(t.Context(), repositories.ListTasksParams{UserID: ana, Completed: tt.completed})
			if err != nil {
				t.Fatalf("ListTasks: %v", err)
			}
			if got := ids(tasks); !slices.Equal(got, tt.want) {
				t.Errorf("ids = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskQueriesIgnoreOtherUsersTasks(t *testing.T) {
	ctx := t.Context()
	pool := testdb.NewMigratedDatabase(t)
	q := repositories.New(pool)
	ana := createUser(t, q, "ana@example.com")
	bob := createUser(t, q, "bob@example.com")
	task := createTask(t, q, ana, "Ana's task")

	_, err := q.UpdateTask(ctx, repositories.UpdateTaskParams{
		ID: task, UserID: bob, Title: pgtype.Text{String: "taken over", Valid: true},
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("UpdateTask as another user: err = %v, want pgx.ErrNoRows", err)
	}

	n, err := q.DeleteTask(ctx, repositories.DeleteTaskParams{ID: task, UserID: bob})
	if err != nil || n != 0 {
		t.Errorf("DeleteTask as another user: rows = %d, err = %v; want 0 rows", n, err)
	}

	tasks, err := q.ListTasks(ctx, repositories.ListTasksParams{UserID: ana})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Title != "Ana's task" {
		t.Fatalf("Ana's tasks = %+v, want her task unchanged", tasks)
	}

	n, err = q.DeleteTask(ctx, repositories.DeleteTaskParams{ID: task, UserID: ana})
	if err != nil || n != 1 {
		t.Errorf("DeleteTask as the owner: rows = %d, err = %v; want 1 row", n, err)
	}
}

func TestUpdateTaskChangesOnlyGivenFields(t *testing.T) {
	ctx := t.Context()
	pool := testdb.NewMigratedDatabase(t)
	q := repositories.New(pool)
	ana := createUser(t, q, "ana@example.com")
	task := createTask(t, q, ana, "Buy milk")

	past := time.Now().Add(-time.Hour).UTC()
	if _, err := pool.Exec(ctx, "UPDATE tasks SET updated_at = $1 WHERE id = $2", past, task); err != nil {
		t.Fatalf("backdate updated_at: %v", err)
	}

	renamed, err := q.UpdateTask(ctx, repositories.UpdateTaskParams{
		ID: task, UserID: ana, Title: pgtype.Text{String: "Buy oat milk", Valid: true},
	})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if renamed.Title != "Buy oat milk" || renamed.Completed {
		t.Errorf("after rename: title %q, completed %v; want new title, still pending", renamed.Title, renamed.Completed)
	}
	if !renamed.UpdatedAt.After(past) {
		t.Errorf("updated_at = %v, want later than %v", renamed.UpdatedAt, past)
	}

	completed, err := q.UpdateTask(ctx, repositories.UpdateTaskParams{
		ID: task, UserID: ana, Completed: pgtype.Bool{Bool: true, Valid: true},
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if completed.Title != "Buy oat milk" || !completed.Completed {
		t.Errorf("after complete: title %q, completed %v; want title kept, completed", completed.Title, completed.Completed)
	}
}

func TestGetSessionUserIDIgnoresExpiredSessions(t *testing.T) {
	ctx := t.Context()
	pool := testdb.NewMigratedDatabase(t)
	q := repositories.New(pool)
	ana := createUser(t, q, "ana@example.com")

	createSession(t, q, ana, []byte("valid"), time.Now().Add(time.Hour))
	createSession(t, q, ana, []byte("expired"), time.Now().Add(-time.Second))

	if got, err := q.GetSessionUserID(ctx, []byte("valid")); err != nil || got != ana {
		t.Errorf("valid session: user %d, err %v; want user %d", got, err, ana)
	}
	if _, err := q.GetSessionUserID(ctx, []byte("expired")); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("expired session: err = %v, want pgx.ErrNoRows", err)
	}
}

func TestRegisterLoginAttemptAllowsFivePerWindowUnderConcurrency(t *testing.T) {
	ctx := t.Context()
	pool := testdb.NewMigratedDatabase(t)
	q := repositories.New(pool)
	const email = "ana@example.com"
	const requests = 20

	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		allowed  int
		attempts = map[int32]bool{}
	)
	for range requests {
		wg.Go(func() {
			row, err := q.RegisterLoginAttempt(ctx, email)
			if err != nil {
				t.Errorf("RegisterLoginAttempt: %v", err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			attempts[row.Attempts] = true
			if row.Attempts <= 5 {
				allowed++
			}
		})
	}
	wg.Wait()

	if allowed != 5 {
		t.Errorf("%d of %d concurrent attempts saw a count of 5 or less, want exactly 5", allowed, requests)
	}
	if len(attempts) != requests {
		t.Errorf("got %d distinct counts, want %d: each attempt must see its own count", len(attempts), requests)
	}

	// Refused attempts do not extend the window: once it has ended, the
	// count starts again at 1.
	if _, err := pool.Exec(ctx, "UPDATE login_attempts SET window_start = now() - interval '16 minutes'"); err != nil {
		t.Fatalf("end the window: %v", err)
	}
	row, err := q.RegisterLoginAttempt(ctx, email)
	if err != nil {
		t.Fatalf("RegisterLoginAttempt after the window: %v", err)
	}
	if row.Attempts != 1 {
		t.Errorf("attempts after the window ended = %d, want 1", row.Attempts)
	}
	if until := time.Until(row.WindowEnd); until < 14*time.Minute || until > 15*time.Minute {
		t.Errorf("new window ends in %v, want about 15 minutes", until)
	}

	other, err := q.RegisterLoginAttempt(ctx, "bob@example.com")
	if err != nil || other.Attempts != 1 {
		t.Errorf("another email: attempts %d, err %v; want its own count starting at 1", other.Attempts, err)
	}
}

func TestCleanupQueriesDeleteOnlyStaleRows(t *testing.T) {
	ctx := t.Context()
	pool := testdb.NewMigratedDatabase(t)
	q := repositories.New(pool)
	ana := createUser(t, q, "ana@example.com")

	createSession(t, q, ana, []byte("valid"), time.Now().Add(time.Hour))
	createSession(t, q, ana, []byte("expired"), time.Now().Add(-time.Second))

	if n, err := q.DeleteExpiredSessions(ctx); err != nil || n != 1 {
		t.Errorf("DeleteExpiredSessions: rows %d, err %v; want 1", n, err)
	}
	if _, err := q.GetSessionUserID(ctx, []byte("valid")); err != nil {
		t.Errorf("valid session after cleanup: %v", err)
	}

	for _, email := range []string{"fresh@example.com", "stale@example.com"} {
		if _, err := q.RegisterLoginAttempt(ctx, email); err != nil {
			t.Fatalf("RegisterLoginAttempt(%s): %v", email, err)
		}
	}
	if _, err := pool.Exec(ctx,
		"UPDATE login_attempts SET window_start = now() - interval '16 minutes' WHERE email = 'stale@example.com'"); err != nil {
		t.Fatalf("age attempt: %v", err)
	}

	if n, err := q.DeleteStaleLoginAttempts(ctx); err != nil || n != 1 {
		t.Errorf("DeleteStaleLoginAttempts: rows %d, err %v; want 1", n, err)
	}
	rows, err := pool.Query(ctx, "SELECT email FROM login_attempts")
	if err != nil {
		t.Fatalf("list attempts: %v", err)
	}
	remaining, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatalf("list attempts: %v", err)
	}
	if !slices.Equal(remaining, []string{"fresh@example.com"}) {
		t.Errorf("remaining attempts = %v, want only the fresh one", remaining)
	}
}

func createUser(t *testing.T, q *repositories.Queries, email string) int64 {
	t.Helper()
	u, err := q.CreateUser(t.Context(), repositories.CreateUserParams{Email: email, PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("CreateUser(%s): %v", email, err)
	}
	return u.ID
}

func createTask(t *testing.T, q *repositories.Queries, userID int64, title string) int64 {
	t.Helper()
	task, err := q.CreateTask(t.Context(), repositories.CreateTaskParams{UserID: userID, Title: title})
	if err != nil {
		t.Fatalf("CreateTask(%q): %v", title, err)
	}
	return task.ID
}

func createSession(t *testing.T, q *repositories.Queries, userID int64, tokenHash []byte, expiresAt time.Time) {
	t.Helper()
	err := q.CreateSession(t.Context(), repositories.CreateSessionParams{TokenHash: tokenHash, UserID: userID, ExpiresAt: expiresAt})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
}

func setCreatedAt(t *testing.T, pool *pgxpool.Pool, taskID int64, at time.Time) {
	t.Helper()
	if _, err := pool.Exec(t.Context(), "UPDATE tasks SET created_at = $1 WHERE id = $2", at, taskID); err != nil {
		t.Fatalf("set created_at: %v", err)
	}
}

func ids(tasks []repositories.Task) []int64 {
	out := make([]int64, len(tasks))
	for i, task := range tasks {
		out[i] = task.ID
	}
	return out
}

//go:build integration

package migrations_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-labs-ai/tasks/apps/api/internal/database"
	"github.com/dev-labs-ai/tasks/apps/api/internal/testdb"
)

const (
	checkViolation  = "23514"
	uniqueViolation = "23505"
)

func TestMain(m *testing.M) {
	os.Exit(testdb.Run(m))
}

func TestMigrationsApplyUpAndDown(t *testing.T) {
	ctx := t.Context()

	db, err := sql.Open("pgx", testdb.NewDatabase(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	migrator, err := database.NewMigrator(db)
	if err != nil {
		t.Fatalf("NewMigrator: %v", err)
	}
	t.Cleanup(func() { _ = migrator.Close() })

	wantTables := []string{"login_attempts", "sessions", "tasks", "users"}

	if _, err := migrator.Up(ctx); err != nil {
		t.Fatalf("Up: %v", err)
	}
	if got := tables(t, db); strings.Join(got, ",") != strings.Join(wantTables, ",") {
		t.Fatalf("tables after Up = %v, want %v", got, wantTables)
	}

	if _, err := migrator.DownTo(ctx, 0); err != nil {
		t.Fatalf("DownTo(0): %v", err)
	}
	if got := tables(t, db); len(got) != 0 {
		t.Fatalf("tables after DownTo(0) = %v, want none", got)
	}

	// A second Up proves Down removed everything Up creates.
	if _, err := migrator.Up(ctx); err != nil {
		t.Fatalf("Up after DownTo(0): %v", err)
	}
}

func TestUserEmailConstraints(t *testing.T) {
	pool := testdb.NewMigratedDatabase(t)

	insertUser(t, pool, "ana@example.com")

	tests := []struct {
		name     string
		email    string
		wantCode string
	}{
		{"uppercase", "Bob@example.com", checkViolation},
		{"duplicate", "ana@example.com", uniqueViolation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pool.Exec(t.Context(),
				"INSERT INTO users (email, password_hash) VALUES ($1, 'hash')", tt.email)
			if code := pgCode(err); code != tt.wantCode {
				t.Errorf("insert %q: SQLSTATE %q, want %q (err: %v)", tt.email, code, tt.wantCode, err)
			}
		})
	}
}

func TestTaskTitleLengthCountsCodePoints(t *testing.T) {
	pool := testdb.NewMigratedDatabase(t)
	userID := insertUser(t, pool, "ana@example.com")

	tests := []struct {
		name     string
		title    string
		wantCode string
	}{
		{"empty", "", checkViolation},
		{"201 ASCII characters", strings.Repeat("a", 201), checkViolation},
		{"200 emoji", strings.Repeat("😀", 200), ""},
		{"201 emoji", strings.Repeat("😀", 201), checkViolation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pool.Exec(t.Context(),
				"INSERT INTO tasks (user_id, title) VALUES ($1, $2)", userID, tt.title)
			if code := pgCode(err); code != tt.wantCode {
				t.Errorf("SQLSTATE %q, want %q (err: %v)", code, tt.wantCode, err)
			}
		})
	}
}

func TestDeletingUserCascades(t *testing.T) {
	ctx := t.Context()
	pool := testdb.NewMigratedDatabase(t)
	userID := insertUser(t, pool, "ana@example.com")

	mustExec(t, pool, "INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ('\\x01', $1, now() + interval '1 day')", userID)
	mustExec(t, pool, "INSERT INTO tasks (user_id, title) VALUES ($1, 'Buy milk')", userID)
	mustExec(t, pool, "DELETE FROM users WHERE id = $1", userID)

	for _, table := range []string{"sessions", "tasks"} {
		var n int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if n != 0 {
			t.Errorf("%s has %d rows after deleting the user, want 0", table, n)
		}
	}
}

func insertUser(t *testing.T, pool *pgxpool.Pool, email string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(t.Context(),
		"INSERT INTO users (email, password_hash) VALUES ($1, 'hash') RETURNING id", email).Scan(&id)
	if err != nil {
		t.Fatalf("insert user %q: %v", email, err)
	}
	return id
}

func mustExec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(t.Context(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func tables(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), `
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name <> 'goose_db_version'
		ORDER BY table_name`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table name: %v", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("list tables: %v", err)
	}
	return names
}

// pgCode returns the SQLSTATE of a PostgreSQL error, or "" for nil.
func pgCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	if err != nil {
		return "not a PostgreSQL error: " + err.Error()
	}
	return ""
}

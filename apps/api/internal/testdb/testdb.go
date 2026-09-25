//go:build integration

// Package testdb gives integration tests a real PostgreSQL: one container
// per test binary, and a fresh database for each test so tests never
// share state.
package testdb

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/dev-labs-ai/tasks/apps/api/internal/database"
)

// Image matches the PostgreSQL version the application runs on.
const Image = "postgres:18"

var (
	adminURL string
	dbCount  atomic.Int64
)

// Run starts PostgreSQL, runs the package's tests and removes the
// container. Call it from TestMain: os.Exit(testdb.Run(m)).
func Run(m *testing.M) int {
	ctx := context.Background()

	container, err := postgres.Run(ctx, Image,
		postgres.WithDatabase("postgres"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.BasicWaitStrategies(),
	)
	defer func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			fmt.Fprintf(os.Stderr, "testdb: remove container: %v\n", err)
		}
	}()
	if err != nil {
		fmt.Fprintf(os.Stderr, "testdb: start %s (is Docker running?): %v\n", Image, err)
		return 1
	}

	adminURL, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "testdb: connection string: %v\n", err)
		return 1
	}

	return m.Run()
}

// NewDatabase creates an empty database for the calling test and returns
// its URL. The database is dropped when the test ends.
func NewDatabase(t *testing.T) string {
	t.Helper()

	name := fmt.Sprintf("test_%d", dbCount.Add(1))
	adminExec(t, "CREATE DATABASE "+name)
	t.Cleanup(func() { adminExec(t, "DROP DATABASE "+name+" WITH (FORCE)") })

	u, err := url.Parse(adminURL)
	if err != nil {
		t.Fatalf("testdb: parse admin URL: %v", err)
	}
	u.Path = "/" + name
	return u.String()
}

// NewMigratedDatabase creates a database with every migration applied and
// returns a pool to it. The pool is closed and the database dropped when
// the test ends.
func NewMigratedDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dbURL := NewDatabase(t)
	if err := database.Migrate(t.Context(), dbURL); err != nil {
		t.Fatalf("testdb: migrate: %v", err)
	}

	pool, err := pgxpool.New(t.Context(), dbURL)
	if err != nil {
		t.Fatalf("testdb: open pool: %v", err)
	}
	// Cleanups run last-in first-out, so the pool closes before the
	// database is dropped.
	t.Cleanup(pool.Close)
	return pool
}

// adminExec runs a statement on the container's maintenance database. It
// uses its own context because it also runs from cleanups, after the
// test's context is cancelled.
func adminExec(t *testing.T, sql string) {
	t.Helper()
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatalf("testdb: connect: %v", err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, sql); err != nil {
		t.Fatalf("testdb: %s: %v", sql, err)
	}
}

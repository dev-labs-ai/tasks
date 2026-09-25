// Package database opens the PostgreSQL connection pool.
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Open creates the pool. It does not wait for PostgreSQL to accept
// connections, so the server can start, and report itself unhealthy,
// while the database is down.
func Open(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	return pool, nil
}

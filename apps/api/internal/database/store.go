package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-labs-ai/tasks/apps/api/internal/repositories"
)

// Store runs the generated queries on the pool, and inside transactions
// when a service needs several writes to succeed or fail together.
type Store struct {
	*repositories.Queries
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{Queries: repositories.New(pool), pool: pool}
}

// InTx runs fn in a transaction. It commits when fn returns nil and rolls
// back otherwise.
func (s *Store) InTx(ctx context.Context, fn func(repositories.Querier) error) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(s.WithTx(tx))
	})
}

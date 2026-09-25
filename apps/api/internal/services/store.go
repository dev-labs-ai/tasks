package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/dev-labs-ai/tasks/apps/api/internal/repositories"
)

// Store is what the services need from the database: the generated
// queries, plus transactions. database.Store implements it.
type Store interface {
	repositories.Querier
	InTx(ctx context.Context, fn func(repositories.Querier) error) error
}

// isUniqueViolation reports whether err is PostgreSQL's unique_violation.
func isUniqueViolation(err error) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pgErr.Code == "23505"
}

package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	// Registers the "pgx" database/sql driver that goose runs on.
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/dev-labs-ai/tasks/apps/api/migrations"
)

// Migrate applies every pending migration. goose works on database/sql,
// so it uses its own short-lived connection rather than the pgx pool.
func Migrate(ctx context.Context, databaseURL string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open migration connection: %w", err)
	}

	migrator, err := NewMigrator(db)
	if err != nil {
		_ = db.Close()
		return err
	}
	defer migrator.Close()

	results, err := migrator.Up(ctx)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	for _, r := range results {
		slog.InfoContext(ctx, "migration applied", "version", r.Source.Version, "file", r.Source.Path, "duration", r.Duration)
	}
	if len(results) == 0 {
		slog.InfoContext(ctx, "database schema up to date")
	}
	return nil
}

// NewMigrator returns a goose provider over the embedded migrations.
// Closing the provider closes db.
func NewMigrator(db *sql.DB) (*goose.Provider, error) {
	migrator, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		return nil, fmt.Errorf("load migrations: %w", err)
	}
	return migrator, nil
}

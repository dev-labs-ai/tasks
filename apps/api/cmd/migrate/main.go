// Command migrate applies pending database migrations and exits. The
// server applies them at startup too; this runs them on their own.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/dev-labs-ai/tasks/apps/api/internal/config"
	"github.com/dev-labs-ai/tasks/apps/api/internal/database"
)

func main() {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		slog.Error("migrate failed", "error", err)
		os.Exit(1)
	}
	if err := database.Migrate(context.Background(), cfg.DatabaseURL); err != nil {
		slog.Error("migrate failed", "error", err)
		os.Exit(1)
	}
}

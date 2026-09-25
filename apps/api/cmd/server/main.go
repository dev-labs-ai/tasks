// Command server runs the Todo List API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/dev-labs-ai/tasks/apps/api/internal/auth"
	"github.com/dev-labs-ai/tasks/apps/api/internal/config"
	"github.com/dev-labs-ai/tasks/apps/api/internal/database"
	"github.com/dev-labs-ai/tasks/apps/api/internal/handlers"
	"github.com/dev-labs-ai/tasks/apps/api/internal/middleware"
	"github.com/dev-labs-ai/tasks/apps/api/internal/services"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}

	logger := newLogger(cfg)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := database.Migrate(ctx, cfg.DatabaseURL); err != nil {
		return err
	}

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	store := database.NewStore(pool)

	// The cleanup job stops, and is waited for, before the deferred
	// pool.Close above runs.
	jobsCtx, stopJobs := context.WithCancel(ctx)
	cleanupDone := make(chan struct{})
	go func() {
		defer close(cleanupDone)
		services.RunCleanup(jobsCtx, store, services.CleanupInterval)
	}()
	defer func() {
		stopJobs()
		<-cleanupDone
	}()

	router := handlers.NewRouter(handlers.Deps{
		DB:            pool,
		Auth:          services.NewAuthService(store, auth.Passwords{}),
		SecureCookies: cfg.IsProduction(),
	})
	// Log wraps Recover so a request that panics is still logged, with the
	// 500 that Recover writes.
	handler := middleware.Log(middleware.Recover(middleware.CrossOrigin(router)))

	srv := &http.Server{
		Addr:              net.JoinHostPort("", strconv.Itoa(cfg.Port)),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", srv.Addr, "env", cfg.Env)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	// Restore default signal handling, so a second signal kills the
	// process if shutdown hangs.
	stop()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shut down: %w", err)
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	slog.Info("server stopped")
	return nil
}

func newLogger(cfg config.Config) *slog.Logger {
	if cfg.IsProduction() {
		return slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nightiz/pastebin-clone/backend/internal/config"
	"github.com/nightiz/pastebin-clone/backend/internal/httpserver"
	"github.com/nightiz/pastebin-clone/backend/internal/paste"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	store, err := paste.NewSQLiteStore(cfg.DatabasePath)
	if err != nil {
		slog.Error("failed to initialize sqlite store", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Background sweeper for expired pastes
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				n, purgeErr := store.PurgeExpired(context.Background())
				if purgeErr != nil {
					slog.Warn("background purge failed", "error", purgeErr)
				} else if n > 0 {
					slog.Info("purged expired pastes", "count", n)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	svc := paste.NewService(store, cfg.MaxPasteBytes, cfg.DefaultExpirySeconds)
	handler := paste.NewHandler(svc)
	serverHandler := httpserver.New(cfg, handler)

	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      serverHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("starting backend server", "addr", cfg.Addr(), "db", cfg.DatabasePath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed to listen and serve", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down server gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server exited cleanly")
}

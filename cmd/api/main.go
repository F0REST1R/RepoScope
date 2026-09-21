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

	"github.com/example/reposcope/internal/config"
	gh "github.com/example/reposcope/internal/github"
	"github.com/example/reposcope/internal/httpapi"
	"github.com/example/reposcope/internal/repository"
	"github.com/example/reposcope/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		log.Error("configuration error", "error", err)
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database configuration error", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	connectCtx, connectCancel := context.WithTimeout(ctx, 10*time.Second)
	defer connectCancel()
	if err = db.Ping(connectCtx); err != nil {
		log.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	if err = repository.Migrate(connectCtx, db); err != nil {
		log.Error("migration failed", "error", err)
		os.Exit(1)
	}
	favorites := repository.NewFavorites(db)
	githubClient := gh.NewClient(cfg.GitHubBaseURL, cfg.GitHubToken, cfg.GitHubAPIVersion, cfg.HTTPClientTimeout, cfg.CacheTTL)
	svc := &service.Service{GitHub: githubClient, Favorites: favorites}
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: httpapi.New(svc, log, cfg.AllowedOrigins), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("server started", "address", cfg.HTTPAddr, "environment", cfg.Environment, "github_api_version", cfg.GitHubAPIVersion, "github_authenticated", cfg.GitHubToken != "")
		serverErrors <- server.ListenAndServe()
	}()
	select {
	case err = <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		log.Info("shutdown requested")
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()
	if err = server.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
		_ = server.Close()
		os.Exit(1)
	}
	log.Info("server stopped")
}

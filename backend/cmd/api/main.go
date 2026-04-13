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

	"github.com/gin-gonic/gin"

	"github.com/taskflow/backend/internal/config"
	"github.com/taskflow/backend/internal/repository"
	"github.com/taskflow/backend/internal/router"
	"github.com/taskflow/backend/internal/service"
	"github.com/taskflow/backend/pkg/database"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	gin.SetMode(cfg.GinMode)

	db, err := database.NewPostgresDB(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}

	repos := repository.NewRepositories(db)
	svcs := service.NewServices(db, repos, cfg.JWTSecret, cfg.JWTExpiry, cfg.BcryptCost)

	engine := router.SetupRouter(svcs, cfg.JWTSecret)

	srv := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		slog.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutdown signal received, draining connections")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown", "error", err)
	}

	if err := db.Close(); err != nil {
		slog.Error("database close", "error", err)
	}

	slog.Info("server stopped")
}

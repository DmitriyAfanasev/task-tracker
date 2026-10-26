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

	"secunda/internal/adapter/mysql"
	httpapi "secunda/internal/delivery/http"
	"secunda/internal/migrations"
	"secunda/internal/platform/auth"
	"secunda/internal/platform/config"
	"secunda/internal/usecase"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel})).
		With("service", "secunda", "env", cfg.Env)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := mysql.Open(ctx, cfg.DBDSN, cfg.DBMaxOpenConns, cfg.DBMaxIdleConns, cfg.DBConnMaxLifetime)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	logger.Info("connected to mysql")

	if cfg.AutoMigrate {
		if err := migrations.Up(db.DB, logger); err != nil {
			return err
		}
	}

	tokens := auth.NewJWTIssuer(cfg.JWTSecret, cfg.JWTTTL)
	hasher := auth.NewBcryptHasher(cfg.BcryptCost)

	users := mysql.NewUserRepository(db)
	teams := mysql.NewTeamRepository(db)

	authUseCase := usecase.NewAuthUseCase(users, tokens, hasher)
	teamUseCase := usecase.NewTeamUseCase(teams, users)

	handler := httpapi.NewHandler(authUseCase, teamUseCase)
	router := httpapi.NewRouter(handler, tokens, db, logger)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("http server started", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	logger.Info("http server stopped")
	return nil
}

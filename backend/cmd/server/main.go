package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gorm.io/gorm"

	"label3130/backend/internal/auth"
	"label3130/backend/internal/config"
	"label3130/backend/internal/database"
	"label3130/backend/internal/handler"
	"label3130/backend/internal/logger"
	"label3130/backend/internal/seed"
	"label3130/backend/internal/service"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)

	if err := cfg.Validate(); err != nil {
		log.Error("configuration validation failed", "error", err.Error())
		os.Exit(1)
	}

	poolCfg := database.PoolConfig{
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
	}
	db, err := connectWithRetry(cfg.DSN(), poolCfg, 15, 2*time.Second)
	if err != nil {
		log.Error("database connect failed", "error", err.Error())
		os.Exit(1)
	}
	defer func() {
		if err := database.Close(db); err != nil {
			log.Error("database close failed", "error", err.Error())
		}
		log.Info("database connection closed")
	}()

	if err := seed.Run(db, log); err != nil {
		log.Error("database seed failed", "error", err.Error())
		os.Exit(1)
	}

	tokens := auth.NewTokenManager(cfg.JWTSecret)
	authSvc := service.NewAuthService(db, tokens, log)
	questionSvc := service.NewQuestionService(db, log)
	attemptSvc := service.NewAttemptService(db, log)

	h := handler.New(authSvc, questionSvc, attemptSvc, tokens, log)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           h.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("server started", "port", cfg.Port, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server stopped unexpectedly", "error", err.Error())
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutdown signal received, shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server forced to shutdown", "error", err.Error())
	}

	log.Info("server exited gracefully")
}

func connectWithRetry(dsn string, poolCfg database.PoolConfig, retries int, interval time.Duration) (*gorm.DB, error) {
	var lastErr error
	for i := 0; i < retries; i++ {
		db, err := database.Connect(dsn, poolCfg)
		if err == nil {
			return db, nil
		}
		lastErr = err
		time.Sleep(interval)
	}
	return nil, fmt.Errorf("connect with retry failed: %w", lastErr)
}

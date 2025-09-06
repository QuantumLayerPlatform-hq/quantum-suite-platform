package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/quantum-suite/platform/internal/services/cache"
	"github.com/quantum-suite/platform/pkg/shared/env"
	"github.com/quantum-suite/platform/pkg/shared/logger"
)

func main() {
	cfg, err := env.LoadConfig()
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	log := logger.NewFromEnv().
		WithService("qlens-cache").
		WithField("version", cfg.Version)

	log.Info("Starting QLens Cache", logger.F("port", cfg.Port))

	cacheService, err := cache.NewService(cfg, log)
	if err != nil {
		log.Fatal("Failed to create cache service", logger.F("error", err))
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: cacheService.Handler(),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    30 * time.Second,
	}

	// Start server in background
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start", logger.F("error", err))
		}
	}()

	log.Info("QLens Cache started successfully")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down QLens Cache...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", logger.F("error", err))
	}

	if err := cacheService.Close(); err != nil {
		log.Error("Error closing cache service", logger.F("error", err))
	}

	log.Info("QLens Cache stopped")
}
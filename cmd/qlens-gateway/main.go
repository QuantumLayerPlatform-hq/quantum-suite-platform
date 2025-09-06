package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/quantum-suite/platform/internal/services/gateway"
	"github.com/quantum-suite/platform/pkg/shared/env"
	"github.com/quantum-suite/platform/pkg/shared/logger"
)

func main() {
	cfg := env.DetectEnvironment()

	log := logger.NewFromEnv().
		WithField("service", "qlens-gateway").
		WithField("version", cfg.Version)

	log.Info("Starting QLens Gateway", logger.F("port", cfg.Port))

	gatewayService, err := gateway.NewService(cfg, log)
	if err != nil {
		log.Fatal("Failed to create gateway service", logger.F("error", err))
	}

	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.Port),
		Handler: gatewayService.Handler(),
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	// Start server in background
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start", logger.F("error", err))
		}
	}()

	log.Info("QLens Gateway started successfully")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down QLens Gateway...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", logger.F("error", err))
	}

	if err := gatewayService.Close(); err != nil {
		log.Error("Error closing gateway service", logger.F("error", err))
	}

	log.Info("QLens Gateway stopped")
}
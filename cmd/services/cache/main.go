package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/quantum-suite/platform/internal/services/cache"
	"github.com/quantum-suite/platform/pkg/shared/env"
	"github.com/quantum-suite/platform/pkg/shared/logger"
)

const (
	serviceName = "qlens-cache"
	defaultPort = "8082"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		healthCheck()
		return
	}

	// Initialize environment configuration
	config := env.DetectEnvironment()

	// Initialize logger
	log := logger.NewFromEnv()

	log.Info("Starting QLens Cache Service",
		logger.F("version", config.Version),
		logger.F("environment", config.Environment),
		logger.F("port", getPort()))

	// Initialize service
	service, err := cache.NewService(config, log)
	if err != nil {
		log.Fatal("Failed to initialize cache service", logger.F("error", err))
	}

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + getPort(),
		Handler:      service.Handler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info("Server starting", logger.F("address", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server", logger.F("error", err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server
	if err := server.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", logger.F("error", err))
	}

	// Close service resources
	if err := service.Close(); err != nil {
		log.Error("Error closing service resources", logger.F("error", err))
	}

	log.Info("Server exited")
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	return port
}

func healthCheck() {
	port := getPort()
	url := fmt.Sprintf("http://localhost:%s/health", port)
	
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Health check failed with status: %d\n", resp.StatusCode)
		os.Exit(1)
	}

	fmt.Println("Health check passed")
}
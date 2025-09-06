package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quantum-suite/platform/docs"
	"github.com/quantum-suite/platform/internal/services/gateway"
	"github.com/quantum-suite/platform/pkg/shared/env"
	"github.com/quantum-suite/platform/pkg/shared/logger"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title QLens Gateway API
// @version 1.0.0
// @description QLens LLM Gateway Service - Unified API for multiple LLM providers
// @termsOfService https://quantumlayer.ai/terms
// @contact.name QLens Support
// @contact.url https://quantumlayer.ai/support
// @contact.email support@quantumlayer.ai
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
// @host localhost:8080
// @BasePath /v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
// @securityDefinitions.apikey TenantID
// @in header
// @name X-Tenant-ID
// @description Tenant identifier for multi-tenancy
func main() {
	// Load configuration
	config, err := env.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log := logger.New(logger.Config{
		Level:  config.LogLevel,
		Format: "json",
		Output: "stdout",
	})

	// Set Gin mode based on environment
	if config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create gateway service
	gatewayService, err := gateway.NewService(config, log)
	if err != nil {
		log.Fatal("Failed to create gateway service", "error", err)
	}

	// Setup routes with Swagger
	router := setupRouter(gatewayService, config, log)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info("Starting QLens Gateway", 
			"port", config.Port,
			"environment", config.Environment,
			"version", getVersion())

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server", "error", err)
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

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown", "error", err)
	}

	log.Info("Server exiting")
}

// setupRouter configures the Gin router with all routes and middleware
func setupRouter(service *gateway.Service, config *env.Config, log logger.Logger) *gin.Engine {
	router := gin.New()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Tenant-ID")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// Health check endpoint (no auth required)
	router.GET("/health", service.HealthCheck)
	router.GET("/ready", service.ReadinessCheck)

	// Swagger documentation
	if config.Environment != "production" {
		docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%d", config.Port)
		docs.SwaggerInfo.BasePath = "/v1"
		docs.SwaggerInfo.Version = getVersion()
		
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		router.GET("/docs", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
		})
	}

	// API v1 routes
	v1 := router.Group("/v1")
	{
		// Models endpoint
		v1.GET("/models", service.ListModels)
		
		// Chat completions
		v1.POST("/chat/completions", service.CreateChatCompletion)
		v1.POST("/completions", service.CreateCompletion) // Legacy compatibility
		
		// Embeddings
		v1.POST("/embeddings", service.CreateEmbedding)
		
		// Internal endpoints (for monitoring)
		internal := v1.Group("/internal")
		{
			internal.GET("/metrics", service.GetMetrics)
			internal.GET("/health", service.HealthCheck)
		}
	}

	return router
}

// getVersion returns the application version
func getVersion() string {
	if version := os.Getenv("APP_VERSION"); version != "" {
		return version
	}
	return "1.0.0"
}
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/pkg/qlens"
)

// Server represents the QLens HTTP server
type Server struct {
	client *qlens.QLens
	router *gin.Engine
	port   string
}

// NewServer creates a new QLens HTTP server
func NewServer(client *qlens.QLens, port string) *Server {
	if port == "" {
		port = "8105"
	}

	server := &Server{
		client: client,
		port:   port,
	}

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	server.router = gin.New()
	server.router.Use(gin.Logger(), gin.Recovery())
	server.setupRoutes()

	return server
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting QLens server on port %s", s.port)
	return s.router.Run(":" + s.port)
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Health endpoint
	s.router.GET("/health", s.handleHealth)

	// Models endpoints
	s.router.GET("/models", s.handleListModels)

	// Completions endpoints
	s.router.POST("/completions", s.handleCreateCompletion)

	// Embeddings endpoints
	s.router.POST("/embeddings", s.handleCreateEmbeddings)

	// Metrics endpoint
	s.router.GET("/metrics", s.handleMetrics)

	// Template endpoints (placeholder for future implementation)
	s.router.GET("/templates", s.handleListTemplates)
	s.router.POST("/templates", s.handleCreateTemplate)
	s.router.POST("/templates/:id/render", s.handleRenderTemplate)

	// Usage endpoints
	s.router.GET("/usage", s.handleGetUsage)
}

// Health check handler
func (s *Server) handleHealth(c *gin.Context) {
	ctx := c.Request.Context()
	
	health, err := s.client.HealthCheck(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"type":    "internal_error",
				"message": "Health check failed",
				"details": gin.H{"error": err.Error()},
			},
		})
		return
	}
	
	status := http.StatusOK
	if health.Status != "healthy" {
		status = http.StatusServiceUnavailable
	}
	
	c.JSON(status, health)
}

// List models handler
func (s *Server) handleListModels(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Parse query parameters
	opts := &qlens.ListModelsOptions{}
	
	if provider := c.Query("provider"); provider != "" {
		opts.Provider = domain.Provider(provider)
	}
	
	if capability := c.Query("capability"); capability != "" {
		opts.Capability = domain.Capability(capability)
	}
	
	models, err := s.client.ListModels(ctx, opts)
	if err != nil {
		s.handleError(c, err)
		return
	}
	
	c.JSON(http.StatusOK, models)
}

// Create completion handler
func (s *Server) handleCreateCompletion(c *gin.Context) {
	ctx := c.Request.Context()
	
	var req qlens.CompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"type":    "invalid_request",
				"message": "Invalid request format",
				"details": gin.H{"error": err.Error()},
			},
		})
		return
	}
	
	// Extract tenant and user from headers or context
	s.enrichRequestContext(&req, c)
	
	// Handle streaming vs non-streaming
	if req.Stream {
		s.handleStreamingCompletion(ctx, &req, c)
		return
	}
	
	response, err := s.client.CreateCompletion(ctx, &req)
	if err != nil {
		s.handleError(c, err)
		return
	}
	
	c.JSON(http.StatusOK, response)
}

// Handle streaming completions
func (s *Server) handleStreamingCompletion(ctx context.Context, req *qlens.CompletionRequest, c *gin.Context) {
	// Set headers for Server-Sent Events
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	
	streamChan, err := s.client.CreateCompletionStream(ctx, req)
	if err != nil {
		s.handleError(c, err)
		return
	}
	
	// Stream responses
	for {
		select {
		case response, ok := <-streamChan:
			if !ok {
				return
			}
			
			if response.Error != nil {
				errorData := map[string]interface{}{
					"error": response.Error,
				}
				data, _ := json.Marshal(errorData)
				c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", data)))
				c.Writer.Flush()
				return
			}
			
			if response.Done {
				c.Writer.Write([]byte("data: [DONE]\n\n"))
				c.Writer.Flush()
				return
			}
			
			data, _ := json.Marshal(response)
			c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", data)))
			c.Writer.Flush()
			
		case <-ctx.Done():
			return
		}
	}
}

// Create embeddings handler
func (s *Server) handleCreateEmbeddings(c *gin.Context) {
	ctx := c.Request.Context()
	
	var req qlens.EmbeddingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"type":    "invalid_request",
				"message": "Invalid request format",
				"details": gin.H{"error": err.Error()},
			},
		})
		return
	}
	
	// Extract tenant and user from headers or context
	s.enrichEmbeddingRequestContext(&req, c)
	
	response, err := s.client.CreateEmbeddings(ctx, &req)
	if err != nil {
		s.handleError(c, err)
		return
	}
	
	c.JSON(http.StatusOK, response)
}

// Metrics handler
func (s *Server) handleMetrics(c *gin.Context) {
	// For now, return a simple health status
	// In a full implementation, this would return Prometheus metrics
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, "# QLens Metrics\n# Implementation pending\n")
}

// Template handlers (placeholders)
func (s *Server) handleListTemplates(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   []interface{}{},
	})
}

func (s *Server) handleCreateTemplate(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"type":    "not_implemented",
			"message": "Template creation not yet implemented",
		},
	})
}

func (s *Server) handleRenderTemplate(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"type":    "not_implemented",
			"message": "Template rendering not yet implemented",
		},
	})
}

// Usage handler
func (s *Server) handleGetUsage(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"type":    "not_implemented",
			"message": "Usage statistics not yet implemented",
		},
	})
}

// Helper methods

func (s *Server) enrichRequestContext(req *qlens.CompletionRequest, c *gin.Context) {
	// Extract tenant ID from header
	if tenantID := c.GetHeader("X-Tenant-ID"); tenantID != "" {
		req.TenantID = domain.TenantID(tenantID)
	}
	
	// Extract user ID from header
	if userID := c.GetHeader("X-User-ID"); userID != "" {
		req.UserID = domain.UserID(userID)
	}
	
	// Set priority from header
	if priority := c.GetHeader("X-Priority"); priority != "" {
		req.Priority = domain.Priority(strings.ToLower(priority))
	}
	
	// Set request ID from header or generate one
	if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
		req.RequestID = requestID
	}
	
	// Set cache options from headers
	if cacheEnabled := c.GetHeader("X-Cache-Enabled"); cacheEnabled != "" {
		if enabled, err := strconv.ParseBool(cacheEnabled); err == nil {
			req.CacheEnabled = enabled
		}
	}
	
	if cacheTTL := c.GetHeader("X-Cache-TTL"); cacheTTL != "" {
		if ttl, err := time.ParseDuration(cacheTTL); err == nil {
			req.CacheTTL = ttl
		}
	}
}

func (s *Server) enrichEmbeddingRequestContext(req *qlens.EmbeddingRequest, c *gin.Context) {
	// Extract tenant ID from header
	if tenantID := c.GetHeader("X-Tenant-ID"); tenantID != "" {
		req.TenantID = domain.TenantID(tenantID)
	}
	
	// Extract user ID from header
	if userID := c.GetHeader("X-User-ID"); userID != "" {
		req.UserID = domain.UserID(userID)
	}
	
	// Set priority from header
	if priority := c.GetHeader("X-Priority"); priority != "" {
		req.Priority = domain.Priority(strings.ToLower(priority))
	}
	
	// Set request ID from header or generate one
	if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
		req.RequestID = requestID
	}
}

func (s *Server) handleError(c *gin.Context, err error) {
	// Convert QLens errors to HTTP responses
	if qlensErr, ok := err.(*qlens.QLensError); ok {
		status := s.getHTTPStatusFromError(qlensErr.Type)
		c.JSON(status, gin.H{
			"error": gin.H{
				"type":       qlensErr.Type,
				"message":    qlensErr.Message,
				"code":       qlensErr.Code,
				"details":    qlensErr.Details,
				"provider":   qlensErr.Provider,
				"request_id": qlensErr.RequestID,
			},
		})
		return
	}
	
	// Generic error response
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": gin.H{
			"type":    "internal_error",
			"message": "An unexpected error occurred",
			"details": gin.H{"error": err.Error()},
		},
	})
}

func (s *Server) getHTTPStatusFromError(errorType string) int {
	switch errorType {
	case qlens.ErrorTypeInvalidRequest:
		return http.StatusBadRequest
	case qlens.ErrorTypeAuthenticationError:
		return http.StatusUnauthorized
	case qlens.ErrorTypeAuthorizationError:
		return http.StatusForbidden
	case qlens.ErrorTypeRateLimitExceeded:
		return http.StatusTooManyRequests
	case qlens.ErrorTypeProviderError:
		return http.StatusBadGateway
	case qlens.ErrorTypeTimeout:
		return http.StatusGatewayTimeout
	case qlens.ErrorTypeProviderUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func main() {
	// Get configuration from environment variables
	port := os.Getenv("QLENS_PORT")
	if port == "" {
		port = "8105"
	}
	
	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}
	
	// Create QLens client
	client, err := qlens.NewWithOpenAI(
		openAIKey,
		qlens.WithCaching(true, 15*time.Minute),
		qlens.WithTimeout(30*time.Second),
		qlens.WithRetries(3, time.Second),
		qlens.WithObservability(true, false), // metrics enabled, tracing disabled
	)
	if err != nil {
		log.Fatalf("Failed to create QLens client: %v", err)
	}
	
	// Create server
	server := NewServer(client, port)
	
	// Setup graceful shutdown
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()
	
	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("Shutting down QLens server...")
	
	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Close QLens client
	if err := client.Close(); err != nil {
		log.Printf("Error closing QLens client: %v", err)
	}
	
	log.Println("QLens server stopped")
}
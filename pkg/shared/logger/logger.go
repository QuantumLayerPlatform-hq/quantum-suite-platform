package logger

import (
	"context"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger interface for structured logging with context
type Logger interface {
	// Context methods
	WithCorrelationID(id string) Logger
	WithTenant(tenantID string) Logger
	WithUser(userID string) Logger
	WithProvider(provider string) Logger
	WithError(err error) Logger
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
	
	// Logging methods
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	
	// Request lifecycle
	StartRequest(method, path string) Logger
	EndRequest(statusCode int, duration time.Duration)
	
	// Provider operations
	LogProviderRequest(provider, model string, tokens int)
	LogProviderResponse(provider, model string, tokens int, cost float64, cached bool)
	LogProviderError(provider, model string, errorType string, err error)
}

// Field represents a logging field
type Field struct {
	Key   string
	Value interface{}
}

// zapLogger implements Logger interface using zap
type zapLogger struct {
	zap    *zap.Logger
	fields []zap.Field
}

// LogLevel represents logging levels
type LogLevel string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	FatalLevel LogLevel = "fatal"
)

// Config for logger initialization
type Config struct {
	Level       LogLevel
	Environment string
	Service     string
	Version     string
	Structured  bool
	AddCaller   bool
}

// NewLogger creates a new logger instance
func NewLogger(cfg Config) Logger {
	var zapConfig zap.Config
	
	// Choose config based on environment
	if cfg.Environment == "production" {
		zapConfig = zap.NewProductionConfig()
		zapConfig.DisableCaller = !cfg.AddCaller
		zapConfig.DisableStacktrace = true
	} else {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.DisableCaller = !cfg.AddCaller
		if !cfg.Structured {
			zapConfig.Encoding = "console"
		}
	}
	
	// Set log level
	switch cfg.Level {
	case DebugLevel:
		zapConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case InfoLevel:
		zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case WarnLevel:
		zapConfig.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case ErrorLevel:
		zapConfig.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	case FatalLevel:
		zapConfig.Level = zap.NewAtomicLevelAt(zap.FatalLevel)
	}
	
	// Custom time encoder for better readability
	if cfg.Environment != "production" {
		zapConfig.EncoderConfig.TimeKey = "time"
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}
	
	baseLogger, err := zapConfig.Build()
	if err != nil {
		// Fallback to basic logger
		baseLogger = zap.NewNop()
	}
	
	// Add base fields
	baseLogger = baseLogger.With(
		zap.String("service", cfg.Service),
		zap.String("version", cfg.Version),
		zap.String("environment", cfg.Environment),
	)
	
	return &zapLogger{
		zap:    baseLogger,
		fields: make([]zap.Field, 0),
	}
}

// NewFromEnv creates logger from environment variables
func NewFromEnv() Logger {
	cfg := Config{
		Level:       LogLevel(getEnvOrDefault("LOG_LEVEL", "info")),
		Environment: getEnvOrDefault("ENVIRONMENT", "development"),
		Service:     getEnvOrDefault("SERVICE_NAME", "qlens"),
		Version:     getEnvOrDefault("VERSION", "dev"),
		Structured:  getEnvOrDefault("LOG_FORMAT", "json") == "json",
		AddCaller:   getEnvOrDefault("LOG_CALLER", "true") == "true",
	}
	
	return NewLogger(cfg)
}

// Context methods

func (l *zapLogger) WithCorrelationID(id string) Logger {
	return l.withField("correlation_id", id)
}

func (l *zapLogger) WithTenant(tenantID string) Logger {
	return l.withField("tenant_id", tenantID)
}

func (l *zapLogger) WithUser(userID string) Logger {
	return l.withField("user_id", userID)
}

func (l *zapLogger) WithProvider(provider string) Logger {
	return l.withField("provider", provider)
}

func (l *zapLogger) WithError(err error) Logger {
	if err == nil {
		return l
	}
	return l.withField("error", err.Error())
}

func (l *zapLogger) WithField(key string, value interface{}) Logger {
	return l.withField(key, value)
}

func (l *zapLogger) WithFields(fields map[string]interface{}) Logger {
	newLogger := &zapLogger{
		zap:    l.zap,
		fields: make([]zap.Field, len(l.fields)+len(fields)),
	}
	
	copy(newLogger.fields, l.fields)
	
	idx := len(l.fields)
	for key, value := range fields {
		newLogger.fields[idx] = zap.Any(key, value)
		idx++
	}
	
	return newLogger
}

func (l *zapLogger) withField(key string, value interface{}) Logger {
	newFields := make([]zap.Field, len(l.fields)+1)
	copy(newFields, l.fields)
	newFields[len(l.fields)] = zap.Any(key, value)
	
	return &zapLogger{
		zap:    l.zap,
		fields: newFields,
	}
}

// Logging methods

func (l *zapLogger) Debug(msg string, fields ...Field) {
	l.zap.Debug(msg, l.combineFields(fields)...)
}

func (l *zapLogger) Info(msg string, fields ...Field) {
	l.zap.Info(msg, l.combineFields(fields)...)
}

func (l *zapLogger) Warn(msg string, fields ...Field) {
	l.zap.Warn(msg, l.combineFields(fields)...)
}

func (l *zapLogger) Error(msg string, fields ...Field) {
	l.zap.Error(msg, l.combineFields(fields)...)
}

func (l *zapLogger) Fatal(msg string, fields ...Field) {
	l.zap.Fatal(msg, l.combineFields(fields)...)
}

// Request lifecycle methods

func (l *zapLogger) StartRequest(method, path string) Logger {
	return l.WithFields(map[string]interface{}{
		"http_method": method,
		"http_path":   path,
		"request_start": time.Now().UTC(),
	}).WithField("event", "request_start")
}

func (l *zapLogger) EndRequest(statusCode int, duration time.Duration) {
	l.WithFields(map[string]interface{}{
		"http_status_code": statusCode,
		"duration_ms":      duration.Milliseconds(),
		"event":           "request_end",
	}).Info("Request completed")
}

// Provider operation methods

func (l *zapLogger) LogProviderRequest(provider, model string, tokens int) {
	l.WithFields(map[string]interface{}{
		"provider":      provider,
		"model":         model,
		"tokens":        tokens,
		"event":         "provider_request",
		"request_time":  time.Now().UTC(),
	}).Info("Provider request initiated")
}

func (l *zapLogger) LogProviderResponse(provider, model string, tokens int, cost float64, cached bool) {
	l.WithFields(map[string]interface{}{
		"provider":      provider,
		"model":         model,
		"tokens":        tokens,
		"cost_usd":      cost,
		"cached":        cached,
		"event":         "provider_response",
		"response_time": time.Now().UTC(),
	}).Info("Provider request completed")
}

func (l *zapLogger) LogProviderError(provider, model string, errorType string, err error) {
	l.WithFields(map[string]interface{}{
		"provider":   provider,
		"model":      model,
		"error_type": errorType,
		"error":      err.Error(),
		"event":      "provider_error",
		"error_time": time.Now().UTC(),
	}).Error("Provider request failed")
}

// Helper methods

func (l *zapLogger) combineFields(fields []Field) []zap.Field {
	combined := make([]zap.Field, len(l.fields)+len(fields))
	copy(combined, l.fields)
	
	for i, field := range fields {
		combined[len(l.fields)+i] = zap.Any(field.Key, field.Value)
	}
	
	return combined
}

// Context integration

type contextKey string

const loggerContextKey contextKey = "logger"

// FromContext extracts logger from context
func FromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(loggerContextKey).(Logger); ok {
		return logger
	}
	// Return default logger if none in context
	return NewFromEnv()
}

// ToContext adds logger to context
func ToContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, logger)
}

// WithCorrelationIDFromContext extracts correlation ID from context headers
func WithCorrelationIDFromContext(ctx context.Context, logger Logger) Logger {
	// Try common correlation ID headers
	correlationHeaders := []string{
		"x-correlation-id",
		"x-request-id",
		"x-trace-id",
		"correlation-id",
		"request-id",
	}
	
	for _, header := range correlationHeaders {
		if value := ctx.Value(header); value != nil {
			if strValue, ok := value.(string); ok && strValue != "" {
				return logger.WithCorrelationID(strValue)
			}
		}
	}
	
	// Generate new correlation ID if none found
	return logger.WithCorrelationID(generateCorrelationID())
}

// Utility functions

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func generateCorrelationID() string {
	// Simple correlation ID generation
	// In production, use a proper UUID library
	return strings.ReplaceAll(time.Now().Format("20060102-150405.000000"), ".", "")
}

// Convenience functions for common logging patterns

// F creates a field for logging
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Event logs an event with structured data
func Event(logger Logger, event string, fields ...Field) {
	allFields := append(fields, F("event", event))
	logger.Info("Event occurred", allFields...)
}

// Performance logs performance metrics
func Performance(logger Logger, operation string, duration time.Duration, fields ...Field) {
	allFields := append(fields, 
		F("operation", operation),
		F("duration_ms", duration.Milliseconds()),
		F("event", "performance"),
	)
	logger.Info("Performance metric", allFields...)
}

// Business logs business events
func Business(logger Logger, action string, entity string, fields ...Field) {
	allFields := append(fields,
		F("business_action", action),
		F("business_entity", entity),
		F("event", "business"),
	)
	logger.Info("Business event", allFields...)
}
module github.com/quantum-suite/platform

go 1.21

require (
	// Core dependencies
	github.com/gin-gonic/gin v1.9.1
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.18.1
	google.golang.org/grpc v1.59.0
	google.golang.org/protobuf v1.31.0
	
	// Database
	github.com/jackc/pgx/v5 v5.5.0
	github.com/pressly/goose/v3 v3.16.0
	github.com/redis/go-redis/v9 v9.3.0
	
	// Vector Database
	github.com/qdrant/go-client v1.6.0
	github.com/weaviate/weaviate-go-client/v4 v4.13.1
	
	// Event Store
	github.com/EventStore/EventStore-Client-Go v1.0.2
	
	// Configuration
	github.com/spf13/viper v1.17.0
	github.com/spf13/cobra v1.7.0
	
	// Observability
	github.com/prometheus/client_golang v1.17.0
	go.opentelemetry.io/otel v1.21.0
	go.opentelemetry.io/otel/trace v1.21.0
	go.opentelemetry.io/otel/metric v1.21.0
	
	// Utilities
	github.com/google/uuid v1.4.0
	github.com/go-playground/validator/v10 v10.16.0
	go.uber.org/zap v1.26.0
	golang.org/x/sync v0.5.0
	
	// Security
	github.com/golang-jwt/jwt/v5 v5.0.0
	golang.org/x/crypto v0.15.0
	
	// Testing
	github.com/stretchr/testify v1.8.4
	github.com/testcontainers/testcontainers-go v0.25.0
)
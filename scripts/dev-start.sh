#!/bin/bash

# QLens Development Startup Script
set -e

echo "🚀 Starting QLens Development Environment"

# Check if required environment variables are set
if [ -z "$OPENAI_API_KEY" ]; then
    echo "⚠️  Warning: OPENAI_API_KEY not set. Some features may not work."
fi

# Create logs directory
mkdir -p logs

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker first."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo "❌ docker-compose not found. Please install docker-compose."
    exit 1
fi

echo "🔧 Building and starting services..."

# Build and start services
docker-compose -f docker-compose.dev.yml up --build -d

echo "⏳ Waiting for services to be healthy..."

# Wait for services to be ready
sleep 10

# Check service health
check_service() {
    local service_name=$1
    local port=$2
    
    echo -n "Checking $service_name... "
    
    for i in {1..30}; do
        if curl -s http://localhost:$port/health > /dev/null 2>&1; then
            echo "✅ Ready"
            return 0
        fi
        sleep 2
    done
    
    echo "❌ Failed to start"
    return 1
}

# Check all services
check_service "qlens-gateway" 8105
check_service "qlens-router" 8106
check_service "qlens-cache" 8107

echo ""
echo "🎉 QLens Development Environment is ready!"
echo ""
echo "📋 Service URLs:"
echo "  • Gateway:  http://localhost:8105"
echo "  • Router:   http://localhost:8106"
echo "  • Cache:    http://localhost:8107"
echo ""
echo "📖 API Documentation: http://localhost:8105/swagger (coming soon)"
echo ""
echo "🔍 To view logs:"
echo "  docker-compose -f docker-compose.dev.yml logs -f [service_name]"
echo ""
echo "🛑 To stop:"
echo "  ./scripts/dev-stop.sh"
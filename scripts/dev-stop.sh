#!/bin/bash

# QLens Development Stop Script
set -e

echo "🛑 Stopping QLens Development Environment"

# Stop and remove containers
docker-compose -f docker-compose.dev.yml down

echo "🧹 Cleaning up..."

# Remove unused networks (optional)
docker network prune -f > /dev/null 2>&1 || true

echo "✅ QLens Development Environment stopped"
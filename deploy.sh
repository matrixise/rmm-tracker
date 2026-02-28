#!/bin/bash
set -e

echo "RMM Tracker - Quick deploy"
echo

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "Docker is not installed. Install Docker first:"
    echo "   https://docs.docker.com/get-docker/"
    exit 1
fi

if ! command -v docker compose &> /dev/null; then
    echo "Docker Compose is not installed."
    exit 1
fi

# Check if .env exists
if [ ! -f .env ]; then
    echo "Creating .env file..."
    cp .env.example .env
    echo
    echo "IMPORTANT: Edit .env with your settings:"
    echo "   - DATABASE_URL (required)"
    echo "   - RMM_TRACKER_RPC_URLS (recommended)"
    echo "   - RMM_TRACKER_WALLETS"
    echo
    echo "Then run again: ./deploy.sh"
    exit 0
fi

# Check if config.toml exists
if [ ! -f config.toml ]; then
    echo "config.toml not found. Creating from config.toml.example..."
    if [ -f config.toml.example ]; then
        cp config.toml.example config.toml
    else
        echo "config.toml.example not found"
        exit 1
    fi
fi

echo "Building Docker image..."
docker compose build

echo
echo "Validating configuration..."
if ! docker compose run --rm app validate-config; then
    echo "Invalid configuration. Fix the errors above."
    exit 1
fi

echo
echo "Starting services..."
docker compose up -d

echo
echo "Waiting for startup (5 seconds)..."
sleep 5

echo
echo "Container status:"
docker compose ps

echo
echo "Deploy complete!"
echo
echo "Useful commands:"
echo "   docker compose logs -f app        # Follow logs"
echo "   docker compose ps                 # Service status"
echo "   docker compose down               # Stop"
echo "   docker compose restart app        # Restart app"
echo "   curl http://localhost:8080/health # Health check"
echo

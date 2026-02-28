.PHONY: help build test validate deploy start stop restart logs status health clean install

# Variables
APP_NAME=rmm-tracker
DOCKER_IMAGE=$(APP_NAME):latest
GO_VERSION=1.26

help: ## Show this help
	@echo "Available commands:"
	@echo
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo

# Local development
build: ## Build the Go application
	@echo "Building..."
	@go build -o $(APP_NAME) .
	@echo "Done: ./$(APP_NAME)"

test: ## Run unit tests
	@echo "Running unit tests..."
	@go test ./... -v

test-short: ## Run fast tests
	@go test ./... -short

validate: ## Validate configuration
	@echo "Validating configuration..."
	@DATABASE_URL="$${DATABASE_URL}" ./$(APP_NAME) validate-config

run-once: build ## Build and run once
	@echo "Running once..."
	@DATABASE_URL="$${DATABASE_URL}" ./$(APP_NAME) run

run-daemon: build ## Build and run in daemon mode
	@echo "Starting daemon mode..."
	@DATABASE_URL="$${DATABASE_URL}" ./$(APP_NAME) run --interval 5m

# Docker
docker-build: ## Build the Docker image
	@echo "Building Docker image..."
	@docker compose build

docker-validate: docker-build ## Validate config with Docker
	@echo "Validating (Docker)..."
	@docker compose run --rm app validate-config

# Deployment
deploy: ## Deploy with Docker Compose (full script)
	@./deploy.sh

start: docker-build ## Start services
	@echo "Starting services..."
	@docker compose up -d
	@sleep 3
	@make status

stop: ## Stop services
	@echo "Stopping services..."
	@docker compose down

restart: ## Restart services
	@echo "Restarting..."
	@docker compose restart app
	@sleep 3
	@make status

# Monitoring
logs: ## Follow application logs
	@docker compose logs -f app

logs-all: ## Follow all service logs
	@docker compose logs -f

status: ## Show service status
	@echo "Service status:"
	@docker compose ps

health: ## Check application health
	@echo "Health check:"
	@curl -f http://localhost:8080/health | jq . || echo "Service unavailable"

# Cleanup
clean: ## Remove compiled binaries
	@echo "Cleaning..."
	@rm -f $(APP_NAME)
	@go clean

clean-docker: stop ## Remove Docker resources (including volumes)
	@echo "Cleaning Docker..."
	@docker compose down -v
	@docker rmi $(APP_NAME):latest 2>/dev/null || true

# Installation
install: ## Install dependencies
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Release
version: ## Show application version
	@./$(APP_NAME) version 2>/dev/null || echo "Build first: make build"

# Configuration
setup: ## Initial setup
	@echo "Initial setup..."
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo ".env created from .env.example"; \
		echo "Edit .env with your settings"; \
	else \
		echo ".env already exists"; \
	fi
	@if [ ! -f config.toml ]; then \
		if [ -f config.toml.example ]; then \
			cp config.toml.example config.toml; \
			echo "config.toml created from config.toml.example"; \
		fi; \
	else \
		echo "config.toml already exists"; \
	fi

# Quick start
quickstart: setup docker-build start ## Setup + build + start
	@echo
	@echo "Quick start complete!"
	@echo
	@make health

.PHONY: help build test validate deploy start stop restart logs status health clean install

# Variables
APP_NAME=realt-rmm
DOCKER_IMAGE=$(APP_NAME):latest
GO_VERSION=1.25

help: ## Affiche cette aide
	@echo "📖 Commandes disponibles:"
	@echo
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo

# Développement local
build: ## Compile l'application Go
	@echo "🔨 Compilation..."
	@go build -o $(APP_NAME) .
	@echo "✅ Compilé: ./$(APP_NAME)"

test: ## Lance les tests
	@echo "🧪 Tests unitaires..."
	@go test ./... -v

test-short: ## Lance les tests rapides
	@go test ./... -short

validate: ## Valide la configuration
	@echo "✅ Validation de la configuration..."
	@DATABASE_URL="$${DATABASE_URL}" ./$(APP_NAME) validate-config

run-once: build ## Exécute une fois
	@echo "▶️  Exécution unique..."
	@DATABASE_URL="$${DATABASE_URL}" ./$(APP_NAME) run

run-daemon: build ## Exécute en mode daemon
	@echo "▶️  Mode daemon..."
	@DATABASE_URL="$${DATABASE_URL}" ./$(APP_NAME) run --interval 5m

# Docker
docker-build: ## Construit l'image Docker
	@echo "🐳 Construction de l'image Docker..."
	@docker compose build

docker-validate: docker-build ## Valide la config avec Docker
	@echo "✅ Validation (Docker)..."
	@docker compose run --rm app validate-config

# Déploiement
deploy: ## Déploie avec Docker Compose (script complet)
	@./deploy.sh

start: docker-build ## Démarre les services
	@echo "🚀 Démarrage des services..."
	@docker compose up -d
	@sleep 3
	@make status

stop: ## Arrête les services
	@echo "🛑 Arrêt des services..."
	@docker compose down

restart: ## Redémarre les services
	@echo "🔄 Redémarrage..."
	@docker compose restart app
	@sleep 3
	@make status

# Monitoring
logs: ## Affiche les logs
	@docker compose logs -f app

logs-all: ## Affiche tous les logs
	@docker compose logs -f

status: ## Affiche l'état des services
	@echo "📊 État des services:"
	@docker compose ps

health: ## Vérifie la santé de l'application
	@echo "🏥 Health check:"
	@curl -f http://localhost:8080/health | jq . || echo "❌ Service non disponible"

# Nettoyage
clean: ## Nettoie les fichiers compilés
	@echo "🧹 Nettoyage..."
	@rm -f $(APP_NAME)
	@go clean

clean-docker: stop ## Nettoie Docker (volumes inclus)
	@echo "🧹 Nettoyage Docker..."
	@docker compose down -v
	@docker rmi $(APP_NAME):latest 2>/dev/null || true

# Installation
install: ## Installe les dépendances
	@echo "📦 Installation des dépendances..."
	@go mod download
	@go mod tidy

# Release
version: ## Affiche la version
	@./$(APP_NAME) version 2>/dev/null || echo "Compilez d'abord: make build"

# Configuration
setup: ## Configuration initiale
	@echo "🔧 Configuration initiale..."
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "✅ .env créé depuis .env.example"; \
		echo "⚠️  Éditez .env avec vos paramètres"; \
	else \
		echo "✅ .env existe déjà"; \
	fi
	@if [ ! -f config.toml ]; then \
		if [ -f config.toml.example ]; then \
			cp config.toml.example config.toml; \
			echo "✅ config.toml créé depuis config.toml.example"; \
		fi; \
	else \
		echo "✅ config.toml existe déjà"; \
	fi

# Quick start
quickstart: setup docker-build start ## Configuration + build + démarrage
	@echo
	@echo "✅ Démarrage rapide terminé !"
	@echo
	@make health

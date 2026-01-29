#!/bin/bash
set -e

echo "🚀 RMM Tracker - Déploiement rapide"
echo

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker n'est pas installé. Installez Docker d'abord:"
    echo "   https://docs.docker.com/get-docker/"
    exit 1
fi

if ! command -v docker compose &> /dev/null; then
    echo "❌ Docker Compose n'est pas installé."
    exit 1
fi

# Check if .env exists
if [ ! -f .env ]; then
    echo "📝 Création du fichier .env..."
    cp .env.example .env
    echo
    echo "⚠️  IMPORTANT: Éditez le fichier .env avec vos paramètres:"
    echo "   - DATABASE_URL (obligatoire)"
    echo "   - RMM_TRACKER_RPC_URLS (recommandé)"
    echo "   - RMM_TRACKER_WALLETS"
    echo
    echo "Puis relancez: ./deploy.sh"
    exit 0
fi

# Check if config.toml exists
if [ ! -f config.toml ]; then
    echo "⚠️  config.toml n'existe pas. Création depuis config.toml.example..."
    if [ -f config.toml.example ]; then
        cp config.toml.example config.toml
    else
        echo "❌ config.toml.example introuvable"
        exit 1
    fi
fi

echo "🔨 Construction de l'image Docker..."
docker compose build

echo
echo "✅ Validation de la configuration..."
if ! docker compose run --rm app validate-config; then
    echo "❌ Configuration invalide. Corrigez les erreurs ci-dessus."
    exit 1
fi

echo
echo "🎯 Démarrage des services..."
docker compose up -d

echo
echo "⏳ Attente du démarrage (5 secondes)..."
sleep 5

echo
echo "📊 État des conteneurs:"
docker compose ps

echo
echo "✅ Déploiement terminé !"
echo
echo "📖 Commandes utiles:"
echo "   docker compose logs -f app        # Voir les logs"
echo "   docker compose ps                 # État des services"
echo "   docker compose down               # Arrêter"
echo "   docker compose restart app        # Redémarrer l'app"
echo "   curl http://localhost:8080/health # Vérifier la santé"
echo

# Configuration Docker Hub pour CI/CD

## Secrets GitHub requis

Pour que le workflow GitHub Actions puisse publier sur Docker Hub, vous devez configurer deux secrets :

### 1. Créer un Access Token Docker Hub

1. Allez sur https://hub.docker.com/settings/security
2. Cliquez sur "New Access Token"
3. Nom : `GitHub Actions rmm-tracker`
4. Permissions : `Read & Write`
5. Copiez le token généré (vous ne le verrez qu'une fois !)

### 2. Ajouter les secrets dans GitHub

1. Allez sur https://github.com/matrixise/realt-rmm/settings/secrets/actions
2. Cliquez sur "New repository secret"

**Secret 1 : DOCKERHUB_USERNAME**
- Name: `DOCKERHUB_USERNAME`
- Value: `matrixise`

**Secret 2 : DOCKERHUB_TOKEN**
- Name: `DOCKERHUB_TOKEN`
- Value: (collez le token créé à l'étape 1)

## Vérification

Une fois les secrets configurés, le prochain push sur `main` ou un tag `v*` déclenchera automatiquement :
- Construction multi-arch (AMD64 + ARM64)
- Publication sur ghcr.io/matrixise/realt-rmm
- Publication sur docker.io/matrixise/rmm-tracker

## Images disponibles

Après publication, vous pourrez utiliser :

```bash
# Docker Hub (recommandé pour usage public)
docker pull matrixise/rmm-tracker:latest
docker pull matrixise/rmm-tracker:v1.0.0

# GitHub Container Registry (requiert authentification GitHub)
docker pull ghcr.io/matrixise/realt-rmm:latest
```

Les images fonctionneront automatiquement sur :
- ✅ Serveurs Linux AMD64 (x86_64)
- ✅ MacBooks M1/M2/M3 (ARM64)
- ✅ Raspberry Pi 4/5 (ARM64)

## Test local

Pour tester la construction multi-arch localement :

```bash
# Configuration initiale (une seule fois)
task docker:buildx:setup

# Construire et pousser vers Docker Hub
task docker:buildx:push
```

#!/bin/bash
# Script de test pour les variables d'environnement

echo "=== Test 1: Override RPC_URL ==="
RPC_URL="https://rpc.gnosischain.com" \
WALLETS="0x1234567890123456789012345678901234567890" \
DATABASE_URL="postgres://realt:realt@localhost:5432/realt_rmm?sslmode=disable" \
./rmm-tracker

echo ""
echo "=== Test 2: Override WALLETS ==="
WALLETS="0x1234567890123456789012345678901234567890,0x2345678901234567890123456789012345678901" \
DATABASE_URL="postgres://realt:realt@localhost:5432/realt_rmm?sslmode=disable" \
./rmm-tracker

echo ""
echo "=== Test 3: LOG_LEVEL=debug ==="
LOG_LEVEL=debug \
DATABASE_URL="postgres://realt:realt@localhost:5432/realt_rmm?sslmode=disable" \
./rmm-tracker

echo ""
echo "=== Test 4: Adresse invalide (doit échouer) ==="
WALLETS="0xinvalid" \
DATABASE_URL="postgres://realt:realt@localhost:5432/realt_rmm?sslmode=disable" \
./rmm-tracker

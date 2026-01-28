#!/bin/bash
#
# Test script for health endpoint
#
# Usage: ./test-health.sh [port]
#

PORT=${1:-8080}
ENDPOINT="http://localhost:$PORT/health"

echo "Testing health endpoint at $ENDPOINT"
echo ""

# Wait for server to be ready
echo "Waiting for server..."
for i in {1..10}; do
    if curl -s -o /dev/null -w "%{http_code}" "$ENDPOINT" > /dev/null 2>&1; then
        echo "✓ Server is responding"
        break
    fi
    sleep 1
done

echo ""
echo "=== Health Check Response ==="
response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" "$ENDPOINT")
http_status=$(echo "$response" | grep "HTTP_STATUS" | cut -d: -f2)
body=$(echo "$response" | sed '/HTTP_STATUS/d')

echo "$body" | jq . 2>/dev/null || echo "$body"
echo ""
echo "HTTP Status: $http_status"
echo ""

# Check status
if [ "$http_status" = "200" ]; then
    echo "✓ Health check passed (200 OK)"
    exit 0
elif [ "$http_status" = "503" ]; then
    echo "⚠ Service unhealthy (503 Service Unavailable)"
    exit 1
else
    echo "✗ Unexpected status code: $http_status"
    exit 1
fi

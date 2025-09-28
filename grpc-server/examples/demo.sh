#!/bin/bash

# Dispense gRPC + REST Gateway Demo Script
# This script demonstrates how to use both gRPC and REST APIs

set -e

# Configuration
GRPC_ADDR="localhost:8080"
HTTP_ADDR="localhost:8081"
API_KEY="demo-api-key"

echo "🚀 Dispense gRPC + REST Gateway Demo"
echo "======================================"
echo ""

# Check if servers are running
echo "🔍 Checking server health..."

# Check HTTP gateway health
HTTP_HEALTH=$(curl -s http://$HTTP_ADDR/health 2>/dev/null || echo "DOWN")
if [[ $HTTP_HEALTH == *"healthy"* ]]; then
    echo "✅ HTTP Gateway is healthy"
else
    echo "❌ HTTP Gateway is not responding"
    echo "💡 Start the combined server with: go run cmd/combined/main.go"
    exit 1
fi

# Check gRPC server (if grpcurl is available)
if command -v grpcurl >/dev/null 2>&1; then
    if grpcurl -plaintext $GRPC_ADDR list >/dev/null 2>&1; then
        echo "✅ gRPC Server is healthy"
    else
        echo "❌ gRPC Server is not responding"
        exit 1
    fi
else
    echo "ℹ️  grpcurl not installed, skipping gRPC health check"
fi

echo ""

# Demo 1: REST API
echo "🌐 REST API Demo"
echo "=================="

echo "📝 1. Creating sandbox via REST..."
SANDBOX_RESPONSE=$(curl -s -X POST "http://$HTTP_ADDR/v1/sandboxes" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{
    "name": "demo-rest-sandbox",
    "is_remote": false,
    "force": true
  }' | jq -r '.')

if [[ $SANDBOX_RESPONSE == *"error"* ]]; then
    echo "❌ Failed to create sandbox: $SANDBOX_RESPONSE"
    exit 1
fi

echo "✅ Sandbox created via REST"
echo "$SANDBOX_RESPONSE" | jq '.'

echo ""
echo "📋 2. Listing sandboxes via REST..."
curl -s -X GET "http://$HTTP_ADDR/v1/sandboxes" \
  -H "X-API-Key: $API_KEY" | jq '.sandboxes[] | {id, name, type, state}'

echo ""
echo "🔍 3. Getting sandbox details via REST..."
curl -s -X GET "http://$HTTP_ADDR/v1/sandboxes/demo-rest-sandbox" \
  -H "X-API-Key: $API_KEY" | jq '.sandbox | {id, name, type, state}'

echo ""
echo "⚡ 4. Getting Claude status via REST..."
curl -s -X GET "http://$HTTP_ADDR/v1/claude/demo-rest-sandbox/status" \
  -H "X-API-Key: $API_KEY" | jq '.'

# Demo 2: gRPC API (if grpcurl is available)
if command -v grpcurl >/dev/null 2>&1; then
    echo ""
    echo "🔧 gRPC API Demo"
    echo "================="

    echo "📝 1. Creating sandbox via gRPC..."
    grpcurl -plaintext -d '{
      "name": "demo-grpc-sandbox",
      "is_remote": false,
      "force": true
    }' $GRPC_ADDR dispense.DispenseService/CreateSandbox | jq '.'

    echo ""
    echo "📋 2. Listing sandboxes via gRPC..."
    grpcurl -plaintext -d '{
      "show_local": true,
      "show_remote": true
    }' $GRPC_ADDR dispense.DispenseService/ListSandboxes | jq '.sandboxes[] | {id, name, type, state}'
fi

# Demo 3: Configuration endpoints (no auth required)
echo ""
echo "⚙️  Configuration Demo"
echo "======================"

echo "🔑 1. Validating API key via REST..."
curl -s -X POST "http://$HTTP_ADDR/v1/config/api-key/validate" \
  -H "Content-Type: application/json" \
  -d "{\"api_key\": \"$API_KEY\"}" | jq '.'

# Demo 4: Health and monitoring
echo ""
echo "🏥 Health & Monitoring Demo"
echo "==========================="

echo "💗 1. Health check..."
curl -s "http://$HTTP_ADDR/health" | jq '.'

echo ""
echo "🎯 2. Readiness check..."
curl -s "http://$HTTP_ADDR/ready" | jq '.'

# Cleanup
echo ""
echo "🧹 Cleanup"
echo "=========="

echo "🗑️  Deleting REST sandbox..."
curl -s -X DELETE "http://$HTTP_ADDR/v1/sandboxes/demo-rest-sandbox" \
  -H "X-API-Key: $API_KEY" | jq '.'

if command -v grpcurl >/dev/null 2>&1; then
    echo "🗑️  Deleting gRPC sandbox..."
    grpcurl -plaintext -d '{
      "identifier": "demo-grpc-sandbox"
    }' $GRPC_ADDR dispense.DispenseService/DeleteSandbox | jq '.'
fi

echo ""
echo "✨ Demo completed successfully!"
echo ""
echo "📖 For more examples, see:"
echo "   - REST API: REST_API.md"
echo "   - gRPC: README.md"
echo ""
echo "🚀 Try the streaming API:"
echo "   curl -X POST http://$HTTP_ADDR/v1/claude/tasks \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -H 'X-API-Key: $API_KEY' \\"
echo "     -H 'Accept: text/event-stream' \\"
echo "     -d '{\"sandbox_identifier\": \"your-sandbox\", \"task_description\": \"List files\"}'"
#!/bin/bash

# Build and embed dashboard script
set -e

echo "🎯 Building Dispense Dashboard..."

# Build the dashboard with Nx
echo "📦 Building React dashboard..."
yarn nx build dashboard

# Create the static directory for embedding
echo "📁 Preparing files for Go embedding..."
mkdir -p dispense/internal/dashboard/static

# Copy built files to the embed location
echo "📋 Copying dashboard files..."
cp -r dist/dashboard/* dispense/internal/dashboard/static/

echo "✅ Dashboard build complete!"
echo "📊 Dashboard will be available at http://localhost:8081 when server is running"
echo "🔌 API will be available at http://localhost:8081/api when server is running"
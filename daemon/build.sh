#!/bin/bash
set -e

# Create directories
mkdir -p ../dist/daemon
mkdir -p ../dispense/pkg/daemon

# Build the daemon
go build -o ../dist/daemon/daemon ./cmd/main.go

# Copy to Dispense package
cp ../dist/daemon/daemon ../dispense/pkg/daemon/daemon-linux-amd64

echo "Build completed and binary copied to Dispense package"
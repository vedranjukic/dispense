#!/bin/bash
set -e

# Create directories
mkdir -p ../dist/daemon
mkdir -p ../dispense/pkg/daemon

# Build the daemon for Linux AMD64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ../dist/daemon/daemon-linux-amd64 ./cmd/main.go

# Copy to Dispense package
cp ../dist/daemon/daemon-linux-amd64 ../dispense/pkg/daemon/daemon-linux-amd64

echo "Linux AMD64 build completed and binary copied to Dispense package"
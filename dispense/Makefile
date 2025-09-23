# Dispense Build Makefile
.PHONY: build build-linux build-darwin build-all clean version help

# Version information
VERSION := $(shell cat ../VERSION 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Linker flags for version embedding
LDFLAGS := -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildTime=$(BUILD_TIME)

# Default target
all: build-all

# Build for current platform
build: build-daemon
	@echo "Building for current platform..."
	@mkdir -p ../dist/dispense
	@go build -ldflags "$(LDFLAGS)" -o ../dist/dispense/dispense ./cmd/main.go ./cmd/list.go ./cmd/shell.go ./cmd/ssh.go ./cmd/new.go ./cmd/daemon.go ./cmd/db.go ./cmd/delete.go ./cmd/claude.go

# Build for Linux AMD64
build-linux: build-daemon
	@echo "Building for Linux AMD64..."
	@mkdir -p ../dist/dispense
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o ../dist/dispense/dispense-linux-amd64 ./cmd/main.go ./cmd/list.go ./cmd/shell.go ./cmd/ssh.go ./cmd/new.go ./cmd/daemon.go ./cmd/db.go ./cmd/delete.go ./cmd/claude.go

# Build for macOS Apple Silicon
build-darwin: build-daemon
	@echo "Building for macOS Apple Silicon..."
	@mkdir -p ../dist/dispense
	@GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o ../dist/dispense/dispense-darwin-arm64 ./cmd/main.go ./cmd/list.go ./cmd/shell.go ./cmd/ssh.go ./cmd/new.go ./cmd/daemon.go ./cmd/db.go ./cmd/delete.go ./cmd/claude.go

# Build all platforms
build-all: build-linux build-darwin
	@echo "All builds completed!"
	@ls -la ../dist/dispense/dispense-*

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf ../dist/dispense/
	@rm -f pkg/daemon/daemon-linux-amd64

# Show version information
version:
	@go run ./cmd/main.go version

# Build daemon binary
build-daemon:
	@echo "Building daemon binary (Linux AMD64)..."
	@mkdir -p pkg/daemon
	@cd ../daemon && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w $(LDFLAGS)" -o ../dispense/pkg/daemon/daemon-linux-amd64 ./cmd/main.go

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build for current platform"
	@echo "  build-linux  - Build for Linux AMD64"
	@echo "  build-darwin - Build for macOS Apple Silicon"
	@echo "  build-all    - Build for all platforms"
	@echo "  build-daemon - Build daemon binary for embedding"
	@echo "  clean        - Clean build artifacts"
	@echo "  version      - Show version information"
	@echo "  help         - Show this help message"

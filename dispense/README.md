# Dispense Cross-Compilation Build

This directory contains the Go Dispense application with cross-compilation support for Linux AMD64 and macOS Apple Silicon.

## Build Targets

### Nx Commands

```bash
# Build for current platform
npx nx run dispense:build

# Build for Linux AMD64
npx nx run dispense:build-linux-amd64

# Build for macOS Apple Silicon
npx nx run dispense:build-darwin-arm64

# Build for all platforms
npx nx run dispense:build-all
```

### Make Commands

```bash
# Build for current platform
make build

# Build for Linux AMD64
make build-linux

# Build for macOS Apple Silicon
make build-darwin

# Build for all platforms
make build-all

# Clean build artifacts
make clean

# Show version information
make version

# Show help
make help
```

### Go Build Script

```bash
# Run the comprehensive build script
go run build.go
```

## Output

All binaries are built in the project root `dist/dispense/` directory:

- `dist/dispense/dispense` - Current platform binary
- `dist/dispense/dispense-linux-amd64` - Linux AMD64 binary
- `dist/dispense/dispense-darwin-arm64` - macOS Apple Silicon binary

## Features

- **Cross-compilation**: Build for multiple platforms from a single machine
- **Static binaries**: CGO disabled for maximum compatibility
- **Version metadata**: Git commit, build time, and version information embedded
- **Build verification**: File size reporting and build success confirmation

## Version Information

The Dispense binary includes a `version` command that displays:

- Version (from git tag or "dev")
- Git commit hash
- Build timestamp
- Go version used for compilation

```bash
./dist/dispense/dispense-linux-amd64 version
```

## Build Requirements

- Go 1.24.0 or later
- Git repository (for version metadata)
- Nx (for Nx commands)
- Make (for Make commands)

## Notes

- All cross-compiled binaries are statically linked (CGO_ENABLED=0)
- The build script automatically detects git information for versioning
- Binaries are named with platform and architecture suffixes for easy identification
- The build process includes dependency tracking in Nx for efficient builds

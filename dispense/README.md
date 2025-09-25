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

# Run the application (development)
npx nx run dispense:serve [args]

# Start MCP server mode (development)
npx nx run dispense:server

# Start MCP server with debug logging
npx nx run dispense:server --configuration=debug

# Start MCP server using built binary
npx nx run dispense:server --configuration=production
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
- **MCP Integration**: Built-in Model Context Protocol server for AI assistant integration

## Version Information

The Dispense binary includes a `version` command that displays:

- Version (from git tag or "dev")
- Git commit hash
- Build timestamp
- Go version used for compilation

```bash
./dist/dispense/dispense-linux-amd64 version
```

## MCP (Model Context Protocol) Support

The dispense binary includes built-in MCP server functionality, allowing AI assistants to interact with dispense commands through structured function calls.

### Available Commands

```bash
# Start MCP server (for MCP client integration)
./dist/dispense/dispense mcp

# With debug logging
./dist/dispense/dispense mcp --debug --log-level debug

# Show MCP command help
./dist/dispense/dispense mcp --help
```

### Development with Nx

```bash
# Start MCP server in development mode
yarn nx server dispense

# With extra debug logging
yarn nx server dispense --configuration=debug

# Using the built binary (faster startup)
yarn nx server dispense --configuration=production
```

### MCP Client Integration

Use this configuration in MCP clients like Claude Code:

```json
{
  "mcpServers": {
    "dispense": {
      "command": "/path/to/dist/dispense/dispense",
      "args": ["mcp"],
      "env": {
        "DISPENSE_LOG_LEVEL": "info"
      }
    }
  }
}
```

### Available MCP Tools

- **`dispense_create_sandbox`** - Create new sandboxes for GitHub issues
- **More tools coming soon** - Additional sandbox management capabilities

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

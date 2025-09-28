# Dispense gRPC Service & REST Gateway

This is a comprehensive gRPC service implementation with REST API gateway that exposes the existing service layer (`claude_service.go`, `config_service.go`, `sandbox_service.go`) to external clients. It provides both native gRPC and REST/HTTP access to all functionality.

## Architecture

```
grpc-server/
├── proto/                    # Protocol buffer definitions
│   ├── common.proto         # Shared message types
│   ├── dispense.proto       # Main service definitions (with REST annotations)
│   ├── *.pb.go              # Generated Go code
│   ├── *_grpc.pb.go         # Generated gRPC code
│   └── *.pb.gw.go           # Generated gRPC-Gateway code
├── server/                  # gRPC server implementation
│   ├── server.go            # Main gRPC server with all handlers
│   └── middleware/          # gRPC middleware components
│       ├── auth.go          # Authentication middleware
│       ├── logging.go       # Request/response logging
│       └── validation.go    # Input validation
├── gateway/                 # HTTP/REST gateway
│   ├── gateway.go           # gRPC-Gateway server
│   └── middleware.go        # HTTP middleware (CORS, logging, etc.)
├── client/                  # Client wrapper
│   └── client.go            # Go gRPC client wrapper
└── cmd/                     # Server entry points
    ├── server/              # Pure gRPC server
    │   └── main.go
    ├── gateway/             # Standalone HTTP gateway
    │   └── main.go
    └── combined/            # Combined gRPC + HTTP server
        └── main.go
```

## Features

### Sandbox Management
- Create sandboxes (local or remote)
- List sandboxes with filtering
- Delete sandboxes
- Get sandbox information
- Wait for sandbox readiness

### Claude Operations
- Run Claude tasks with streaming response
- Get Claude daemon status
- Retrieve Claude logs

### Configuration Management
- Get/Set API keys
- Validate API keys

## Usage

### Server Options

You can run the service in three different modes:

#### 1. Pure gRPC Server
```bash
cd grpc-server
go run cmd/server/main.go
```

#### 2. Combined gRPC + REST Server (Recommended)
```bash
cd grpc-server
go run cmd/combined/main.go
```

#### 3. Standalone REST Gateway (connects to existing gRPC server)
```bash
cd grpc-server
go run cmd/gateway/main.go
```

### Environment Variables

#### gRPC Server
- `DISPENSE_GRPC_PORT`: gRPC server port (default: ":8080")
- `DISPENSE_API_KEY`: API key for authentication
- `DISPENSE_GRPC_AUTH_ENABLED`: Enable authentication (default: "true")
- `DISPENSE_GRPC_REFLECTION`: Enable gRPC reflection (default: "false")

#### HTTP Gateway
- `DISPENSE_HTTP_PORT`: HTTP gateway port (default: ":8081")
- `DISPENSE_GRPC_ENDPOINT`: gRPC server endpoint (default: "localhost:8080")

#### Combined Server
- `DISPENSE_GRPC_PORT`: gRPC server port (default: ":8080")
- `DISPENSE_HTTP_PORT`: HTTP gateway port (default: ":8081")

### Using the Client

```go
package main

import (
    "log"

    "dispense/grpc-server/client"
    pb "dispense/grpc-server/proto"
)

func main() {
    // Create client
    client, err := client.NewDispenseClient(&client.ClientConfig{
        Address: "localhost:8080",
        APIKey:  "your-api-key",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Create a sandbox
    resp, err := client.CreateLocalSandbox("my-sandbox")
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Created sandbox: %s", resp.Sandbox.Id)

    // Run a Claude task
    err = client.RunTaskInSandbox(resp.Sandbox.Id, "List files", func(taskResp *pb.RunClaudeTaskResponse) error {
        log.Printf("Task output: %s", taskResp.Content)
        return nil
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

### REST API Usage

The HTTP gateway provides a full REST API:

```bash
# Create a sandbox
curl -X POST http://localhost:8081/v1/sandboxes \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{"name": "test", "is_remote": false}'

# List sandboxes
curl -X GET http://localhost:8081/v1/sandboxes \
  -H "X-API-Key: your-api-key"

# Run Claude task (streaming)
curl -X POST http://localhost:8081/v1/claude/tasks \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -H "Accept: text/event-stream" \
  -d '{
    "sandbox_identifier": "test",
    "task_description": "List files"
  }'

# Health check (no auth required)
curl -X GET http://localhost:8081/health
```

**📖 See [REST_API.md](REST_API.md) for complete API documentation**

### gRPC CLI Usage

You can also use standard gRPC tools like `grpcurl`:

```bash
# Enable reflection in development
export DISPENSE_GRPC_REFLECTION=true

# List services
grpcurl -plaintext localhost:8080 list

# Create a sandbox
grpcurl -plaintext -d '{"name": "test", "is_remote": false}' localhost:8080 dispense.DispenseService/CreateSandbox

# List sandboxes
grpcurl -plaintext -d '{"show_local": true, "show_remote": true}' localhost:8080 dispense.DispenseService/ListSandboxes
```

## Security

### Authentication

The server supports API key authentication via gRPC metadata:

```go
md := metadata.New(map[string]string{
    "api-key": "your-api-key",
})
ctx = metadata.NewOutgoingContext(ctx, md)
```

### Middleware

- **Authentication**: API key validation (can be disabled for development)
- **Logging**: Request/response logging with timing
- **Validation**: Input validation and sanitization

## Development

### Building

```bash
go build -o grpc-server cmd/server/main.go
```

### Running Tests

```bash
go test ./...
```

### Regenerating Proto Code

```bash
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/common.proto proto/dispense.proto
```

## API Reference

See the proto files for detailed API documentation:
- [dispense.proto](proto/dispense.proto) - Main service definitions
- [common.proto](proto/common.proto) - Shared message types

## Error Handling

The service provides structured error responses with error codes and messages. Errors are mapped from internal error types to gRPC status codes.

## Performance Considerations

- Streaming is used for long-running operations like Claude tasks
- Connection pooling is supported through the client wrapper
- Request timeouts are configurable
- Graceful shutdown is implemented

## Gateway Features

### HTTP/REST Gateway
- **Full REST API**: All gRPC endpoints mapped to REST
- **OpenAPI Compatible**: Standard REST patterns
- **Streaming Support**: Server-Sent Events for streaming operations
- **CORS Support**: Cross-origin requests enabled
- **Content Negotiation**: JSON request/response

### Gateway Middleware
- **Authentication**: API key validation (header, bearer, query param)
- **Logging**: HTTP request/response logging
- **Rate Limiting**: Per-IP rate limiting (100 req/min)
- **Health Checks**: `/health`, `/ready` endpoints
- **Security Headers**: Standard security headers
- **CORS**: Cross-origin resource sharing

### Deployment Options
- **Combined Mode**: Single process serving both gRPC and HTTP
- **Separate Mode**: Independent gRPC and HTTP gateway processes
- **Load Balancer Friendly**: Health checks and graceful shutdown

## Monitoring

The service includes:
- Structured logging for all operations (gRPC and HTTP)
- Request/response timing
- Error tracking
- Health checks (`/health`, `/ready`, gRPC reflection)
- Rate limiting metrics
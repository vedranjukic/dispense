# gRPC Service Implementation Specification

## Overview

This specification outlines the implementation of a gRPC service that exposes the existing service layer (`claude_service.go`, `config_service.go`, `sandbox_service.go`) to external clients. The service will provide a unified API for managing sandboxes, executing Claude tasks, and managing configuration.

## Architecture

### Service Structure
```
grpc-server/
├── proto/
│   ├── dispense.proto          # Main service definitions
│   └── common.proto            # Shared message types
├── server/
│   ├── server.go               # gRPC server implementation
│   ├── handlers/
│   │   ├── sandbox_handler.go  # Sandbox service handlers
│   │   ├── claude_handler.go   # Claude service handlers
│   │   └── config_handler.go   # Config service handlers
│   └── middleware/
│       ├── auth.go             # Authentication middleware
│       ├── logging.go          # Request/response logging
│       └── validation.go       # Input validation
├── client/
│   └── client.go               # gRPC client wrapper
└── cmd/
    └── server/
        └── main.go             # Server entry point
```

## Proto Definitions

### 1. Main Service Definition (`dispense.proto`)

```protobuf
syntax = "proto3";

package dispense;

option go_package = "dispense/grpc-server/proto";

import "common.proto";

// Main Dispense service that aggregates all functionality
service DispenseService {
  // Sandbox management
  rpc CreateSandbox(CreateSandboxRequest) returns (CreateSandboxResponse);
  rpc ListSandboxes(ListSandboxesRequest) returns (ListSandboxesResponse);
  rpc DeleteSandbox(DeleteSandboxRequest) returns (DeleteSandboxResponse);
  rpc GetSandbox(GetSandboxRequest) returns (GetSandboxResponse);
  rpc WaitForSandbox(WaitForSandboxRequest) returns (WaitForSandboxResponse);
  
  // Claude operations
  rpc RunClaudeTask(RunClaudeTaskRequest) returns (stream RunClaudeTaskResponse);
  rpc GetClaudeStatus(GetClaudeStatusRequest) returns (GetClaudeStatusResponse);
  rpc GetClaudeLogs(GetClaudeLogsRequest) returns (GetClaudeLogsResponse);
  
  // Configuration management
  rpc GetAPIKey(GetAPIKeyRequest) returns (GetAPIKeyResponse);
  rpc SetAPIKey(SetAPIKeyRequest) returns (SetAPIKeyResponse);
  rpc ValidateAPIKey(ValidateAPIKeyRequest) returns (ValidateAPIKeyResponse);
}
```

### 2. Common Message Types (`common.proto`)

```protobuf
syntax = "proto3";

package dispense;

option go_package = "dispense/grpc-server/proto";

import "google/protobuf/timestamp.proto";

// Sandbox types
enum SandboxType {
  SANDBOX_TYPE_UNSPECIFIED = 0;
  SANDBOX_TYPE_LOCAL = 1;
  SANDBOX_TYPE_REMOTE = 2;
}

// GitHub integration types
message GitHubIssue {
  string url = 1;
  int32 number = 2;
  string owner = 3;
  string repo = 4;
  string title = 5;
  string body = 6;
}

message GitHubPR {
  string url = 1;
  int32 number = 2;
  string owner = 3;
  string repo = 4;
  string title = 5;
  string body = 6;
}

message TaskData {
  string description = 1;
  GitHubIssue github_issue = 2;
  GitHubPR github_pr = 3;
}

// Sandbox information
message SandboxInfo {
  string id = 1;
  string name = 2;
  SandboxType type = 3;
  string state = 4;
  string shell_command = 5;
  google.protobuf.Timestamp created_at = 6;
  string group = 7;
  map<string, string> metadata = 8;
}

// Resource allocation for remote sandboxes
message ResourceAllocation {
  string snapshot = 1;
  string target = 2;
  int32 cpu = 3;
  int32 memory = 4;
  int32 disk = 5;
  int32 auto_stop = 6;
}

// Error response
message ErrorResponse {
  string code = 1;
  string message = 2;
  map<string, string> details = 3;
}
```

## Service Implementations

### 1. Sandbox Service Handlers

#### CreateSandbox
- **Request**: `CreateSandboxRequest`
  - `string name`
  - `string branch_name`
  - `bool is_remote`
  - `bool force`
  - `bool skip_copy`
  - `bool skip_daemon`
  - `string group`
  - `string model`
  - `string task`
  - `ResourceAllocation resources`
  - `string source_directory`
  - `TaskData task_data`

- **Response**: `CreateSandboxResponse`
  - `SandboxInfo sandbox`
  - `ErrorResponse error`

#### ListSandboxes
- **Request**: `ListSandboxesRequest`
  - `bool show_local`
  - `bool show_remote`
  - `bool verbose`
  - `string group`

- **Response**: `ListSandboxesResponse`
  - `repeated SandboxInfo sandboxes`
  - `ErrorResponse error`

#### DeleteSandbox
- **Request**: `DeleteSandboxRequest`
  - `string identifier`
  - `bool delete_all`
  - `bool force`

- **Response**: `DeleteSandboxResponse`
  - `bool success`
  - `string message`
  - `ErrorResponse error`

#### GetSandbox
- **Request**: `GetSandboxRequest`
  - `string identifier`

- **Response**: `GetSandboxResponse`
  - `SandboxInfo sandbox`
  - `ErrorResponse error`

#### WaitForSandbox
- **Request**: `WaitForSandboxRequest`
  - `string identifier`
  - `int32 timeout_seconds`
  - `string group`

- **Response**: `WaitForSandboxResponse`
  - `bool success`
  - `string message`
  - `ErrorResponse error`

### 2. Claude Service Handlers

#### RunClaudeTask (Streaming)
- **Request**: `RunClaudeTaskRequest`
  - `string sandbox_identifier`
  - `string task_description`
  - `string model`

- **Response**: `RunClaudeTaskResponse` (stream)
  - `enum ResponseType { STDOUT = 0; STDERR = 1; STATUS = 2; ERROR = 3; }`
  - `ResponseType type`
  - `string content`
  - `int64 timestamp`
  - `int32 exit_code`
  - `bool is_finished`

#### GetClaudeStatus
- **Request**: `GetClaudeStatusRequest`
  - `string sandbox_identifier`

- **Response**: `GetClaudeStatusResponse`
  - `bool connected`
  - `string daemon_info`
  - `string work_dir`
  - `ErrorResponse error`

#### GetClaudeLogs
- **Request**: `GetClaudeLogsRequest`
  - `string sandbox_identifier`
  - `string task_id` (optional)

- **Response**: `GetClaudeLogsResponse`
  - `bool success`
  - `repeated string logs`
  - `ErrorResponse error`

### 3. Config Service Handlers

#### GetAPIKey
- **Request**: `GetAPIKeyRequest`
  - `bool interactive` (whether to prompt if not found)

- **Response**: `GetAPIKeyResponse`
  - `string api_key`
  - `ErrorResponse error`

#### SetAPIKey
- **Request**: `SetAPIKeyRequest`
  - `string api_key`

- **Response**: `SetAPIKeyResponse`
  - `bool success`
  - `string message`
  - `ErrorResponse error`

#### ValidateAPIKey
- **Request**: `ValidateAPIKeyRequest`
  - `string api_key`

- **Response**: `ValidateAPIKeyResponse`
  - `bool valid`
  - `string message`
  - `ErrorResponse error`

## Implementation Details

### 1. Server Structure

```go
// server/server.go
type DispenseServer struct {
    pb.UnimplementedDispenseServiceServer
    serviceContainer *services.ServiceContainer
    logger           *log.Logger
}

func NewDispenseServer(serviceContainer *services.ServiceContainer) *DispenseServer {
    return &DispenseServer{
        serviceContainer: serviceContainer,
        logger:           log.New(os.Stdout, "[grpc-server] ", log.LstdFlags),
    }
}
```

### 2. Handler Implementation Pattern

Each handler should:
1. **Validate input** using middleware
2. **Convert proto messages** to internal models
3. **Call the appropriate service method**
4. **Convert response** back to proto format
5. **Handle errors** consistently
6. **Log operations** for debugging

Example handler structure:
```go
func (s *DispenseServer) CreateSandbox(ctx context.Context, req *pb.CreateSandboxRequest) (*pb.CreateSandboxResponse, error) {
    // 1. Validate request
    if err := s.validateCreateSandboxRequest(req); err != nil {
        return &pb.CreateSandboxResponse{
            Error: s.convertError(err),
        }, nil
    }
    
    // 2. Convert to internal model
    createReq := s.convertCreateSandboxRequest(req)
    
    // 3. Call service
    sandboxInfo, err := s.serviceContainer.SandboxService.Create(createReq)
    if err != nil {
        return &pb.CreateSandboxResponse{
            Error: s.convertError(err),
        }, nil
    }
    
    // 4. Convert response
    return &pb.CreateSandboxResponse{
        Sandbox: s.convertSandboxInfo(sandboxInfo),
    }, nil
}
```

### 3. Error Handling

Implement consistent error handling:
- Convert internal errors to gRPC status codes
- Provide meaningful error messages
- Include error codes for programmatic handling
- Log errors for debugging

```go
func (s *DispenseServer) convertError(err error) *pb.ErrorResponse {
    // Map internal error codes to gRPC status codes
    // Return structured error response
}
```

### 4. Middleware

Implement middleware for:
- **Authentication**: API key validation
- **Logging**: Request/response logging
- **Validation**: Input validation
- **Rate limiting**: Prevent abuse
- **Metrics**: Performance monitoring

### 5. Client Wrapper

Provide a Go client wrapper for easy integration:

```go
type DispenseClient struct {
    conn   *grpc.ClientConn
    client pb.DispenseServiceClient
}

func NewDispenseClient(address string) (*DispenseClient, error) {
    // Create gRPC connection
    // Return client wrapper
}

func (c *DispenseClient) CreateSandbox(req *CreateSandboxRequest) (*CreateSandboxResponse, error) {
    // Convert to proto and call gRPC service
}
```

## Configuration

### Server Configuration
```yaml
server:
  address: ":8080"
  tls:
    enabled: false
    cert_file: ""
    key_file: ""
  auth:
    enabled: true
    api_key_required: true
  logging:
    level: "info"
    format: "json"
  rate_limiting:
    enabled: true
    requests_per_minute: 100
```

### Environment Variables
- `DISPENSE_GRPC_ADDRESS`: Server address (default: ":8080")
- `DISPENSE_GRPC_TLS_ENABLED`: Enable TLS (default: "false")
- `DISPENSE_GRPC_CERT_FILE`: TLS certificate file
- `DISPENSE_GRPC_KEY_FILE`: TLS private key file
- `DISPENSE_GRPC_AUTH_ENABLED`: Enable authentication (default: "true")

## Testing Strategy

### 1. Unit Tests
- Test each handler individually
- Mock service dependencies
- Test error scenarios
- Test input validation

### 2. Integration Tests
- Test with real service implementations
- Test streaming operations
- Test error propagation
- Test authentication

### 3. Load Tests
- Test concurrent requests
- Test streaming performance
- Test memory usage
- Test rate limiting

## Security Considerations

### 1. Authentication
- Require API key for all operations
- Validate API key format
- Implement key rotation

### 2. Authorization
- Role-based access control
- Sandbox isolation
- Resource limits

### 3. Input Validation
- Validate all input parameters
- Sanitize user input
- Prevent injection attacks

### 4. Network Security
- Use TLS in production
- Implement proper CORS
- Rate limiting

## Monitoring and Observability

### 1. Metrics
- Request count and duration
- Error rates
- Resource usage
- Streaming metrics

### 2. Logging
- Structured logging
- Request/response logging
- Error logging
- Performance logging

### 3. Tracing
- Distributed tracing
- Request correlation
- Performance analysis

## Deployment

### 1. Docker
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o grpc-server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/grpc-server .
CMD ["./grpc-server"]
```

### 2. Kubernetes
- Deployment manifest
- Service manifest
- ConfigMap for configuration
- Secret for API keys

### 3. Health Checks
- gRPC health check protocol
- Liveness probe
- Readiness probe

## Migration Strategy

### Phase 1: Basic Implementation
1. Create proto definitions
2. Implement basic server structure
3. Implement sandbox handlers
4. Add basic error handling

### Phase 2: Full Feature Set
1. Implement Claude handlers
2. Implement config handlers
3. Add streaming support
4. Add authentication

### Phase 3: Production Ready
1. Add comprehensive testing
2. Add monitoring
3. Add security features
4. Performance optimization

## Future Enhancements

### 1. Additional Services
- File management service
- Environment management
- Resource monitoring

### 2. Advanced Features
- WebSocket support
- Real-time notifications
- Batch operations
- Caching layer

### 3. Integration
- REST API gateway
- GraphQL support
- Webhook support
- Event streaming

## Conclusion

This specification provides a comprehensive plan for implementing a gRPC service that exposes the existing service layer. The implementation should follow the outlined structure, maintain consistency with existing code patterns, and provide a robust, scalable API for external clients.

The key success factors are:
1. **Consistent error handling** across all endpoints
2. **Proper input validation** and sanitization
3. **Comprehensive testing** at all levels
4. **Security best practices** implementation
5. **Monitoring and observability** from day one
6. **Clear documentation** and examples

This implementation will provide a solid foundation for external integrations while maintaining the existing service architecture and patterns.

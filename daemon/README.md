# Daemon

A Go daemon application with gRPC server built with Nx.

## Features

- **gRPC Server** running on port 28080
- **ProjectService** with Init and Logs (streaming) methods
- **AgentService** with Init and CreateTask methods
- **Graceful shutdown** handling
- **Cross-platform builds** for Linux and macOS

## gRPC Services

### ProjectService
- `Init(InitRequest) -> InitResponse` - Initialize the project
- `Logs(LogsRequest) -> stream LogsResponse` - Stream project logs

### AgentService
- `Init(InitRequest) -> InitResponse` - Initialize the agent
- `CreateTask(CreateTaskRequest) -> CreateTaskResponse` - Create a new task

### InitRequest
- `project_type` (string) - The type of project to initialize

### CreateTaskRequest
- `prompt` (string) - The task prompt

## Development

- `yarn nx run daemon:serve` - Run the daemon with gRPC server
- `yarn nx run daemon:build` - Build the daemon binary
- `yarn nx run daemon:build-client` - Build the gRPC client example
- `yarn nx run daemon:run-client` - Run the gRPC client example
- `yarn nx run daemon:generate` - Regenerate gRPC code from protobuf
- `yarn nx run daemon:test` - Run tests
- `yarn nx run daemon:tidy` - Run `go mod tidy`
- `yarn nx run daemon:format` - Format Go code

## Testing the gRPC Server

1. Start the daemon: `yarn nx run daemon:serve`
2. In another terminal, run the client: `yarn nx run daemon:run-client`

The client will test all gRPC methods and demonstrate the streaming logs functionality.

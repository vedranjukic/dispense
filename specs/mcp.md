# MCP Integration for Dispense Binary

## Project Overview
✅ **COMPLETED**: Dispense now includes built-in MCP (Model Context Protocol) support, allowing AI assistants to interact with dispense commands through structured function calls.

The MCP server is bundled directly into the dispense binary and can be activated using the `dispense mcp` command.

## Current Implementation

### How to Use
```bash
# Start MCP server (communicates via stdin/stdout)
dispense mcp

# Use in MCP client configuration
{
  "mcpServers": {
    "dispense": {
      "command": "/path/to/dispense",
      "args": ["mcp"],
      "env": {
        "DISPENSE_LOG_LEVEL": "info"
      }
    }
  }
}
```

### Available Tools
- ✅ **`dispense_create_sandbox`** - Create new sandboxes for GitHub issues
- 🚧 **Additional tools** - Status, logs, wait, delete, list, shell (coming soon)

### Implementation Details
- **Location**: `dispense/pkg/mcp/` package
- **Architecture**: Uses `github.com/modelcontextprotocol/go-sdk`
- **Command**: `dispense mcp` subcommand
- **Transport**: stdio (standard input/output)
- **Validation**: Parameter validation with `go-playground/validator`

## Original Requirements (For Reference)

### Binary Application Details
- **Binary name**: `dispense`
- **Available commands**: 
  - Main command: `dispense --name <sandbox-name> --task <github-issue-url>`
  - Sub-commands: `dispense claude <sandbox-name> status`, `dispense claude <sandbox-name> logs`
  - Management: `dispense wait <sandbox-name>`, `dispense delete <sandbox-name> [--force]`
  - Info: `dispense list`, `dispense shell <sandbox-name>`
- **Common flags/parameters**: 
  - `--name`: Sandbox name (required for create)
  - `--task`: GitHub issue URL (required for create)
  - `--remote`: Create remote Daytona sandbox instead of local
  - `--force`: Force deletion without confirmation (for delete command)
- **Output format**: Mixed text output with emojis, status indicators, and structured information
- **Error handling**: Exit codes and descriptive error messages in stdout/stderr

### MCP Server Specifications
1. **Language**: Go
2. **MCP SDK**: Use https://github.com/modelcontextprotocol/go-sdk
3. **Tool Mapping**: Each binary command should be exposed as an MCP tool
4. **Parameter Validation**: Validate parameters before passing to binary
5. **Error Handling**: Proper error propagation and user-friendly messages
6. **Logging**: Comprehensive logging for debugging using Go's log package or structured logging
7. **Configuration**: Support for configurable binary path and default options

### Technical Requirements

#### Tools to Implement
Create these MCP tools based on dispense commands:

1. **`dispense_create_sandbox`**: Main command to create and setup sandbox
   - Parameters: `name` (string, required), `task` (string, required), `remote` (boolean, optional)
   - Description: Creates a local or remote sandbox for working on GitHub issues
   - Returns: Sandbox creation status, container info, and setup results

2. **`dispense_claude_status`**: Check Claude daemon status in sandbox  
   - Parameters: `sandbox_name` (string, required)
   - Description: Gets current status of Claude working in the specified sandbox
   - Returns: Current working status and sandbox state

3. **`dispense_claude_logs`**: Get Claude logs from sandbox
   - Parameters: `sandbox_name` (string, required)
   - Description: Retrieves recent logs from Claude daemon in sandbox
   - Returns: Log entries with timestamps and progress information

4. **`dispense_wait`**: Wait for sandbox tasks to complete
   - Parameters: `sandbox_names` (array of strings, required)
   - Description: Monitors one or more sandboxes until their tasks are completed
   - Returns: Completion status for each sandbox with progress updates

5. **`dispense_delete`**: Delete sandbox and cleanup resources
   - Parameters: `sandbox_name` (string, required), `force` (boolean, optional)
   - Description: Deletes a sandbox and cleans up associated containers/resources
   - Returns: Deletion confirmation and cleanup status

6. **`dispense_list`**: List all existing sandboxes
   - Parameters: None
   - Description: Gets overview of all sandboxes with their current state and connection info
   - Returns: Structured list with ID, name, type, state, and shell commands

7. **`dispense_shell`**: Connect to sandbox shell (interactive mode notification)
   - Parameters: `sandbox_name` (string, required)
   - Description: Prepares shell connection info for sandbox (note: actual shell connection requires external terminal)
   - Returns: Connection details and shell command for manual execution

For each tool:
- **Validation**: Input validation and sanitization
- **Execution**: Safe subprocess execution with appropriate timeout (longer for wait operations)
- **Output**: Structured response with success/error status and parsed results

#### Server Features
- **Health Check**: Tool to verify binary is accessible
- **Help/Info**: Tool to get command help and available options
- **Configuration**: Runtime configuration for binary path and defaults
- **Security**: Input sanitization and safe command execution
- **Performance**: Efficient subprocess management

#### File Structure
```
mcp-dispense-server/
├── main.go (main server entry point)
├── internal/
│   ├── server/
│   │   ├── server.go (MCP server implementation)
│   │   └── handlers.go (tool handlers)
│   ├── tools/ (individual tool implementations)
│   │   ├── create.go
│   │   ├── status.go
│   │   ├── logs.go
│   │   ├── wait.go
│   │   ├── delete.go
│   │   ├── list.go
│   │   └── shell.go
│   ├── config/
│   │   └── config.go (configuration management)
│   └── executor/
│       └── executor.go (command execution utilities)
├── go.mod
├── go.sum
└── README.md
```

### Implementation Guidelines

#### Command Execution
- Use os/exec package for safe command execution
- Implement proper context.Context with timeout handling
- Capture both stdout and stderr using cmd.Output() and cmd.CombinedOutput()
- Handle process exit codes with proper error type checking
- Use bufio.Scanner for real-time output parsing when needed

#### Error Handling
- Use Go's error interface for proper error propagation
- Create custom error types for different failure modes
- Provide meaningful error messages to the AI assistant
- Use structured logging (log/slog) for debugging while sanitizing user-facing messages

#### Configuration
- Support environment variables using os.Getenv()
- Use embedded structs for configuration management
- Validate binary availability on startup with exec.LookPath()
- Support configuration files using encoding/json or gopkg.in/yaml.v3

#### Testing
- Unit tests for each tool using Go's testing package
- Integration tests with actual dispense binary
- Mock executor for CI/CD environments using interfaces
- Table-driven tests for parameter validation
- Benchmark tests for performance-critical operations

## Example Tool Implementation

Please implement tools following this pattern for the dispense binary using the Go MCP SDK:

```go
// Example tool definition structure
type CreateSandboxTool struct {
    executor CommandExecutor
}

type CreateSandboxParams struct {
    Name   string `json:"name" validate:"required,alphanum"`
    Task   string `json:"task" validate:"required,url"`
    Remote bool   `json:"remote,omitempty"`
}

// Tool schema definition
func (t *CreateSandboxTool) Schema() mcp.ToolSchema {
    return mcp.ToolSchema{
        Name:        "dispense_create_sandbox",
        Description: "Create a new sandbox for working on a GitHub issue with Claude",
        InputSchema: mcp.JSONSchema{
            Type: "object",
            Properties: map[string]mcp.JSONSchema{
                "name": {
                    Type:        "string",
                    Description: "Name for the sandbox (will be used as branch name and container name)",
                    Pattern:     "^[a-zA-Z0-9_-]+$",
                },
                "task": {
                    Type:        "string", 
                    Description: "GitHub issue URL to work on",
                    Pattern:     "^https://github\\.com/.+/issues/\\d+$",
                },
                "remote": {
                    Type:        "boolean",
                    Description: "Create remote Daytona sandbox instead of local Docker container",
                    Default:     false,
                },
            },
            Required: []string{"name", "task"},
        },
    }
}

// Additional tool schemas for:
// - dispense_claude_status
// - dispense_claude_logs  
// - dispense_wait
// - dispense_delete
// - dispense_list
// - dispense_shell
```

## Deliverables

1. **Complete MCP Server**: Fully functional Go server with all tools using github.com/modelcontextprotocol/go-sdk
2. **Documentation**: README with setup, building, and usage instructions
3. **Configuration**: Example configuration files and environment variable support
4. **Tests**: Test suite with good coverage using Go's testing package
5. **Build Configuration**: Proper go.mod with dependencies and build scripts
6. **Type Safety**: Full Go struct definitions and validation using go-playground/validator

## Additional Considerations

- **Performance**: Optimize for multiple concurrent tool calls using goroutines and proper context handling
- **Monitoring**: Add metrics and health monitoring capabilities using expvar or prometheus metrics
- **Output Parsing**: Parse emoji-rich output and extract structured information using regexp and strings packages
- **Long-running Operations**: Handle sandbox creation which may take several minutes using context.WithTimeout
- **Container Management**: Track Docker container lifecycle and cleanup
- **Concurrency**: Use sync.WaitGroup and channels for managing multiple sandbox operations
- **Graceful Shutdown**: Implement proper signal handling for clean server shutdown

## Questions for Implementation

The following details are provided:
1. **Binary name**: `dispense`
2. **Example commands**: Provided above with full output examples
3. **Security considerations**: None - safe to execute all dispense commands
4. **Operating systems**: Linux and macOS support required
5. **MCP client applications**: Claude Code and Codex integration

**Additional parsing requirements:**
- Extract container IDs and names from output
- Parse sandbox states (running, stopped, etc.)
- Handle emoji indicators (✅, 🔧, 🐳, 🔍, ⏳, 🔄, 🗑️, etc.) in output formatting
- Capture GitHub issue information and repository details
- Parse log timestamps and status messages
- Extract tabular data from list command (ID, Name, Type, State, Shell Command)
- Parse completion status and progress indicators from wait command
- Handle shell connection details and docker exec commands
- Parse deletion confirmation messages

Please create a robust, production-ready MCP server that makes my binary application easily accessible to AI assistants while maintaining security and reliability.
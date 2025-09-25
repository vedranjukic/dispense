package mcp

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server provides the MCP server functionality for dispense
type Server struct {
	config   *Config
	executor CommandExecutor
}

// NewServer creates a new MCP server instance
func NewServer() (*Server, error) {
	// Load configuration (from file, env vars, and defaults)
	cfg := LoadConfig()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Create executor (using the same binary we're running in)
	executor := NewDispenseExecutor(cfg.BinaryPath)

	return &Server{
		config:   cfg,
		executor: executor,
	}, nil
}

// Run starts the MCP server with stdio transport
func (s *Server) Run(ctx context.Context) error {
	log.Printf("Starting MCP Dispense Server (bundled mode)")

	// Create server
	server := mcp.NewServer("dispense-mcp", "1.0.0", nil)

	// Add tools
	server.AddTools(
		mcp.NewServerTool(
			"dispense_create_sandbox",
			"Create an isolated development sandbox where Claude can work autonomously on GitHub issues or development tasks. The sandbox provides a safe, isolated environment with full file system access where Claude can execute commands, modify files, and work freely without safety restrictions. Use 'dispense wait <sandbox-name>' to monitor completion, then 'dispense claude <sandbox-name> logs' to see results.",
			CreateSandbox(s.executor, s.config),
			mcp.Input(
				mcp.Property("name", mcp.Description("Sandbox name using alphanumeric characters, hyphens, and underscores (e.g., 'fix-auth-bug', 'feature_123', 'issue-2283'). Will be used as branch name and container identifier."), mcp.Required(true)),
				mcp.Property("task", mcp.Description("GitHub issue URL (e.g., 'https://github.com/owner/repo/issues/123') OR detailed task description (e.g., 'Fix authentication bug in login system'). For GitHub issues, provide the full URL first, followed by any additional context."), mcp.Required(true)),
				mcp.Property("remote", mcp.Description("Set to true for cloud-based Daytona sandbox (recommended for resource-intensive tasks or when Docker is unavailable), false for local Docker container (faster for simple tasks). Defaults to false."), mcp.Required(false)),
				mcp.Property("model", mcp.Description("Anthropic model to use for Claude Code in the sandbox (e.g., 'claude-3-5-sonnet-20241022', 'claude-3-opus-20240229', 'claude-3-5-haiku-20241022'). If not specified, uses the default model."), mcp.Required(false)),
			),
		),
		mcp.NewServerTool(
			"dispense_exec_command",
			"Execute a command in a sandbox and return the output (stdout and stderr) and exit code. Works with both local Docker containers and remote Daytona sandboxes. Example usage: copy files, run scripts, check status, etc.",
			ExecCommand(s.executor, s.config),
			mcp.Input(
				mcp.Property("name", mcp.Description("Sandbox name or ID to execute the command in"), mcp.Required(true)),
				mcp.Property("command", mcp.Description("The command string to execute (e.g., 'ls -la', 'cp -r /source /destination', 'echo \"Hello World\"')"), mcp.Required(true)),
			),
		),
	)

	log.Printf("Registered %d MCP tools", 2)

	// Start the server with stdio transport
	transport := mcp.NewStdioTransport()
	return server.Run(ctx, transport)
}
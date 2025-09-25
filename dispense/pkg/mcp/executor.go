package mcp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// ExecutionResult holds the result of command execution
type ExecutionResult struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	Duration   time.Duration
	ParsedData map[string]interface{} // For structured parsing
}

// CommandExecutor defines the interface for executing commands
type CommandExecutor interface {
	Execute(ctx context.Context, args []string) (*ExecutionResult, error)
	ExecuteWithTimeout(args []string, timeout time.Duration) (*ExecutionResult, error)
}

// DispenseExecutor implements CommandExecutor for the dispense binary
type DispenseExecutor struct {
	binaryPath string
}

// NewDispenseExecutor creates a new executor for the dispense binary
func NewDispenseExecutor(binaryPath string) *DispenseExecutor {
	return &DispenseExecutor{
		binaryPath: binaryPath,
	}
}

// Execute runs a command with the given context
func (e *DispenseExecutor) Execute(ctx context.Context, args []string) (*ExecutionResult, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, e.binaryPath, args...)

	// Set environment to prevent recursive MCP calls
	cmd.Env = append(os.Environ(), "DISPENSE_MCP_MODE=internal")

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	result := &ExecutionResult{
		Stdout:     string(output),
		Stderr:     "",
		Duration:   duration,
		ParsedData: make(map[string]interface{}),
	}

	// Get exit code
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
			result.Stderr = string(exitError.Stderr)
		} else {
			return result, fmt.Errorf("failed to execute command: %w", err)
		}
	}

	return result, nil
}

// ExecuteWithTimeout runs a command with a specific timeout
func (e *DispenseExecutor) ExecuteWithTimeout(args []string, timeout time.Duration) (*ExecutionResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return e.Execute(ctx, args)
}

// CreateResult represents the result of sandbox creation
type CreateResult struct {
	Success      bool     `json:"success"`
	SandboxName  string   `json:"sandbox_name,omitempty"`
	ContainerID  string   `json:"container_id,omitempty"`
	Steps        []string `json:"steps"`
	ErrorMessage string   `json:"error_message,omitempty"`
}

// ParseCreateOutput parses the output from sandbox creation
func ParseCreateOutput(output string) *CreateResult {
	result := &CreateResult{
		Success: false,
		Steps:   []string{},
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for success indicators
		if strings.Contains(line, "✅") || strings.Contains(line, "success") {
			result.Success = true
		}

		// Extract container ID
		if strings.Contains(line, "Container ID:") || strings.Contains(line, "🐳") {
			containerIDRegex := regexp.MustCompile(`[a-f0-9]{12,}`)
			if match := containerIDRegex.FindString(line); match != "" {
				result.ContainerID = match
			}
		}

		// Extract sandbox name
		if strings.Contains(line, "Sandbox:") || strings.Contains(line, "Name:") {
			nameRegex := regexp.MustCompile(`(?:Sandbox|Name):\s*([a-zA-Z0-9_-]+)`)
			if matches := nameRegex.FindStringSubmatch(line); len(matches) > 1 {
				result.SandboxName = matches[1]
			}
		}

		result.Steps = append(result.Steps, line)
	}

	return result
}
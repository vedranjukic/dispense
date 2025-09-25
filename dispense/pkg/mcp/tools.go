package mcp

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateSandboxParams represents the parameters for creating a sandbox
type CreateSandboxParams struct {
	Name   string `json:"name" validate:"required,min=1,max=50"`
	Task   string `json:"task" validate:"required,min=10"`
	Remote bool   `json:"remote,omitempty"`
	Model  string `json:"model,omitempty"`
}

// CreateSandboxResult represents the result of sandbox creation
type CreateSandboxResult struct {
	Success      bool   `json:"success"`
	SandboxName  string `json:"sandbox_name,omitempty"`
	ContainerID  string `json:"container_id,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// CreateSandbox creates a new sandbox for working on GitHub issues
func CreateSandbox(executor CommandExecutor, config *Config) mcp.ToolHandlerFor[CreateSandboxParams, CreateSandboxResult] {
	validate := validator.New()

	return func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[CreateSandboxParams]) (*mcp.CallToolResultFor[CreateSandboxResult], error) {
		p := params.Arguments

		// Validate parameters with user-friendly messages
		if err := validate.Struct(p); err != nil {
			if p.Name == "" {
				return nil, fmt.Errorf("sandbox name is required (e.g., 'fix-auth-bug', 'feature-123')")
			}
			if len(p.Name) > 50 {
				return nil, fmt.Errorf("sandbox name must be 50 characters or less (current: %d characters)", len(p.Name))
			}
			if p.Task == "" {
				return nil, fmt.Errorf("task description or GitHub issue URL is required")
			}
			if len(p.Task) < 10 {
				return nil, fmt.Errorf("task description must be at least 10 characters long (provide more detail about what needs to be done)")
			}
			return nil, fmt.Errorf("parameter validation failed: %w", err)
		}

		// Additional name validation - allow alphanumeric, hyphens, underscores
		namePattern := `^[a-zA-Z0-9_-]+$`
		if matched, _ := regexp.MatchString(namePattern, p.Name); !matched {
			return nil, fmt.Errorf("sandbox name can only contain letters, numbers, hyphens (-), and underscores (_). Invalid name: '%s'", p.Name)
		}

		// Build command arguments for dispense
		args := []string{"new", "--name", p.Name, "--task", p.Task}
		if p.Remote {
			args = append(args, "--remote")
		}
		if p.Model != "" {
			args = append(args, "--model", p.Model)
		}

		// Execute command with appropriate timeout
		result, err := executor.ExecuteWithTimeout(args, config.DefaultTimeout)
		if err != nil {
			return &mcp.CallToolResultFor[CreateSandboxResult]{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("❌ Failed to execute create command: %v", err)},
				},
			}, nil
		}

		// Parse the output
		createResult := parseCreateOutput(result.Stdout)

		// Prepare result
		toolResult := CreateSandboxResult{
			Success:     createResult.Success && result.ExitCode == 0,
			SandboxName: createResult.SandboxName,
			ContainerID: createResult.ContainerID,
		}

		if !toolResult.Success {
			toolResult.ErrorMessage = fmt.Sprintf("Exit Code: %d, Output: %s", result.ExitCode, result.Stdout)
			if result.Stderr != "" {
				toolResult.ErrorMessage += fmt.Sprintf(", Error: %s", result.Stderr)
			}
		}

		// Prepare response content
		var responseText string
		if toolResult.Success {
			responseText = fmt.Sprintf("✅ Sandbox '%s' created successfully!\n", p.Name)
			if createResult.ContainerID != "" {
				responseText += fmt.Sprintf("🐳 Container ID: %s\n", createResult.ContainerID)
			}
			responseText += "\n📋 Creation steps:\n"
			for i, step := range createResult.Steps {
				if step != "" {
					responseText += fmt.Sprintf("%d. %s\n", i+1, step)
				}
			}

			// Add next steps guidance
			responseText += "\n🚀 Next Steps:\n"
			responseText += fmt.Sprintf("• Monitor progress: Use 'dispense wait %s' to wait for task completion\n", p.Name)
			responseText += fmt.Sprintf("• View logs: Use 'dispense claude %s logs' to see Claude's work\n", p.Name)
			responseText += fmt.Sprintf("• Connect to sandbox: Use 'dispense shell %s' for direct access\n", p.Name)

			if p.Remote {
				responseText += "\n☁️ Remote Sandbox: Claude is now working autonomously in the cloud. The task will continue even if you close this session."
			} else {
				responseText += "\n🐳 Local Sandbox: Claude is working in an isolated Docker container on your machine."
			}
		} else {
			responseText = fmt.Sprintf("❌ Sandbox creation failed.\n%s", toolResult.ErrorMessage)

			// Add troubleshooting guidance based on common issues
			if strings.Contains(toolResult.ErrorMessage, "Docker") || strings.Contains(toolResult.ErrorMessage, "docker") {
				responseText += "\n\n💡 Troubleshooting: Docker appears to be unavailable. Try using remote=true for a cloud-based Daytona sandbox instead."
			}
		}

		return &mcp.CallToolResultFor[CreateSandboxResult]{
			Content: []mcp.Content{
				&mcp.TextContent{Text: responseText},
			},
			StructuredContent: toolResult,
		}, nil
	}
}

// parseCreateOutput parses the output from sandbox creation
func parseCreateOutput(output string) *CreateResult {
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

// ExecSandboxParams represents the parameters for executing a command in a sandbox
type ExecSandboxParams struct {
	Name    string `json:"name" validate:"required,min=1"`
	Command string `json:"command" validate:"required,min=1"`
}

// ExecSandboxResult represents the result of command execution
type ExecSandboxResult struct {
	Success      bool   `json:"success"`
	Stdout       string `json:"stdout"`
	Stderr       string `json:"stderr"`
	ExitCode     int    `json:"exit_code"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// ExecCommand executes a command in a sandbox and returns output and exit code
func ExecCommand(executor CommandExecutor, config *Config) mcp.ToolHandlerFor[ExecSandboxParams, ExecSandboxResult] {
	validate := validator.New()

	return func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[ExecSandboxParams]) (*mcp.CallToolResultFor[ExecSandboxResult], error) {
		p := params.Arguments

		// Validate parameters with user-friendly messages
		if err := validate.Struct(p); err != nil {
			if p.Name == "" {
				return nil, fmt.Errorf("sandbox name is required")
			}
			if p.Command == "" {
				return nil, fmt.Errorf("command is required")
			}
			return nil, fmt.Errorf("parameter validation failed: %w", err)
		}

		// Build command arguments for dispense exec
		args := []string{"exec", p.Name, p.Command}

		// Execute command with appropriate timeout
		result, err := executor.ExecuteWithTimeout(args, config.DefaultTimeout)

		// Prepare result - always return a result even if execution fails
		toolResult := ExecSandboxResult{
			Success:  false,
			Stdout:   result.Stdout,
			Stderr:   result.Stderr,
			ExitCode: result.ExitCode,
		}

		if err != nil {
			toolResult.ErrorMessage = fmt.Sprintf("Failed to execute command: %v", err)
		} else {
			toolResult.Success = true
		}

		// Prepare response content
		var responseText string
		if toolResult.Success {
			responseText = fmt.Sprintf("✅ Command executed in sandbox '%s'\n", p.Name)
			responseText += fmt.Sprintf("📋 Command: %s\n", p.Command)
			responseText += fmt.Sprintf("🔢 Exit Code: %d\n", toolResult.ExitCode)

			if toolResult.Stdout != "" {
				responseText += fmt.Sprintf("\n📄 Output:\n%s", toolResult.Stdout)
			}

			if toolResult.Stderr != "" {
				responseText += fmt.Sprintf("\n⚠️ Error Output:\n%s", toolResult.Stderr)
			}
		} else {
			responseText = fmt.Sprintf("❌ Command execution failed in sandbox '%s'\n", p.Name)
			responseText += fmt.Sprintf("📋 Command: %s\n", p.Command)
			responseText += fmt.Sprintf("🔢 Exit Code: %d\n", toolResult.ExitCode)

			if toolResult.ErrorMessage != "" {
				responseText += fmt.Sprintf("💥 Error: %s\n", toolResult.ErrorMessage)
			}

			if toolResult.Stderr != "" {
				responseText += fmt.Sprintf("\n⚠️ Error Output:\n%s", toolResult.Stderr)
			}
		}

		return &mcp.CallToolResultFor[ExecSandboxResult]{
			Content: []mcp.Content{
				&mcp.TextContent{Text: responseText},
			},
			StructuredContent: toolResult,
		}, nil
	}
}
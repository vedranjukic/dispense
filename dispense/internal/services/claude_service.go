package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cli/internal/core/errors"
	"cli/internal/core/models"
	"cli/pkg/sandbox"
	"cli/pkg/sandbox/local"
	"cli/pkg/sandbox/remote"
	pb "cli/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ClaudeService handles Claude integration and daemon communication
type ClaudeService struct {
	sandboxService *SandboxService
}

// NewClaudeService creates a new Claude service
func NewClaudeService(sandboxService *SandboxService) *ClaudeService {
	return &ClaudeService{
		sandboxService: sandboxService,
	}
}

// RunTask executes a task using Claude in the specified sandbox
func (s *ClaudeService) RunTask(req *models.ClaudeTaskRequest) (*models.ClaudeTaskResponse, error) {
	// Find the sandbox
	sandboxInfo, err := s.sandboxService.FindByName(req.SandboxIdentifier)
	if err != nil {
		return nil, err
	}

	// Get API key (fetch fresh each time as it can change)
	apiKey, err := s.getAnthropicAPIKey()
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeAPIKeyMissing, "failed to get Anthropic API key")
	}

	// Get working directory for the sandbox
	workDir, err := s.getWorkingDirectoryFromProvider(sandboxInfo)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeSystemUnavailable, "failed to get working directory")
	}

	// Get daemon connection info
	daemonAddr, err := s.getDaemonAddress(sandboxInfo)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDaemonUnavailable, "failed to get daemon address")
	}

	// Connect to daemon
	conn, err := s.connectToDaemon(daemonAddr)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDaemonUnavailable, "failed to connect to daemon")
	}
	defer conn.Close()

	// Create client and create async task
	client := pb.NewAgentServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // Shorter timeout for task creation
	defer cancel()

	grpcReq := &pb.CreateTaskRequest{
		Prompt:           req.TaskDescription,
		WorkingDirectory: workDir,
		EnvironmentVars:  make(map[string]string), // Add any needed env vars
		AnthropicApiKey:  apiKey,
		Model:           req.Model,
	}

	resp, err := client.CreateTask(ctx, grpcReq)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeSystemUnavailable, "failed to create task")
	}

	if !resp.Success {
		return &models.ClaudeTaskResponse{
			Success:  false,
			ErrorMsg: resp.Message,
		}, nil
	}

	// Return immediately with task ID for async execution
	return &models.ClaudeTaskResponse{
		Success: true,
		TaskID:  resp.TaskId,
		Output:  resp.Message, // Success message from task creation
	}, nil
}

// GetTaskStatus retrieves the status of a specific Claude task
func (s *ClaudeService) GetTaskStatus(req *models.ClaudeTaskStatusRequest) (*models.ClaudeTaskStatusResponse, error) {
	// Find the sandbox
	sandboxInfo, err := s.sandboxService.FindByName(req.SandboxIdentifier)
	if err != nil {
		return nil, err
	}

	// Get daemon connection info
	daemonAddr, err := s.getDaemonAddress(sandboxInfo)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDaemonUnavailable, "failed to get daemon address")
	}

	// Connect to daemon
	conn, err := s.connectToDaemon(daemonAddr)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDaemonUnavailable, "failed to connect to daemon")
	}
	defer conn.Close()

	// Create client and get task status
	client := pb.NewAgentServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcReq := &pb.TaskStatusRequest{
		TaskId: req.TaskID,
	}

	resp, err := client.GetTaskStatus(ctx, grpcReq)
	if err != nil {
		return &models.ClaudeTaskStatusResponse{
			Success:  false,
			ErrorMsg: err.Error(),
		}, nil
	}

	// Convert task state enum to string
	var state string
	switch resp.State {
	case pb.TaskStatusResponse_PENDING:
		state = "PENDING"
	case pb.TaskStatusResponse_RUNNING:
		state = "RUNNING"
	case pb.TaskStatusResponse_COMPLETED:
		state = "COMPLETED"
	case pb.TaskStatusResponse_FAILED:
		state = "FAILED"
	default:
		state = "UNKNOWN"
	}

	return &models.ClaudeTaskStatusResponse{
		Success:    true,
		State:      state,
		Message:    resp.Message,
		ExitCode:   resp.ExitCode,
		StartedAt:  resp.StartedAt,
		FinishedAt: resp.FinishedAt,
		Prompt:     resp.Prompt,
		WorkDir:    resp.WorkingDirectory,
	}, nil
}

// GetStatus retrieves the status of Claude daemon in the specified sandbox
func (s *ClaudeService) GetStatus(req *models.ClaudeStatusRequest) (*models.ClaudeStatusResponse, error) {
	// Find the sandbox
	sandboxInfo, err := s.sandboxService.FindByName(req.SandboxIdentifier)
	if err != nil {
		return nil, err
	}

	// Get daemon connection info
	daemonAddr, err := s.getDaemonAddress(sandboxInfo)
	if err != nil {
		return &models.ClaudeStatusResponse{
			Connected: false,
			ErrorMsg:  err.Error(),
		}, nil
	}

	// Try to connect to daemon
	conn, err := s.connectToDaemon(daemonAddr)
	if err != nil {
		return &models.ClaudeStatusResponse{
			Connected: false,
			ErrorMsg:  err.Error(),
		}, nil
	}
	defer conn.Close()

	// Get daemon info
	client := pb.NewAgentServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.Init(ctx, &pb.InitRequest{})
	if err != nil {
		return &models.ClaudeStatusResponse{
			Connected: false,
			ErrorMsg:  err.Error(),
		}, nil
	}

	// Get working directory
	workDir, _ := s.getWorkingDirectory(sandboxInfo)

	return &models.ClaudeStatusResponse{
		Connected:  resp.Success,
		DaemonInfo: resp.Message,
		WorkDir:    workDir,
	}, nil
}

// GetLogs retrieves Claude logs from the specified sandbox
func (s *ClaudeService) GetLogs(req *models.ClaudeLogsRequest) (*models.ClaudeLogsResponse, error) {
	// Find the sandbox
	sandboxInfo, err := s.sandboxService.FindByName(req.SandboxIdentifier)
	if err != nil {
		return nil, err
	}

	// For now, this is a simple implementation that would need to be enhanced
	// to actually retrieve logs from the sandbox based on the sandbox type
	logs, err := s.retrieveLogsFromSandbox(sandboxInfo, req.TaskID)
	if err != nil {
		return &models.ClaudeLogsResponse{
			Success:  false,
			ErrorMsg: err.Error(),
		}, nil
	}

	return &models.ClaudeLogsResponse{
		Success: true,
		Logs:    logs,
	}, nil
}

// getDaemonAddress gets the daemon address for the sandbox
func (s *ClaudeService) getDaemonAddress(sandboxInfo *models.SandboxInfo) (string, error) {
	if sandboxInfo.Type == models.TypeLocal {
		// For local sandboxes, get the container IP and use port 28080
		containerIP, err := s.getSandboxContainerIP(sandboxInfo)
		if err != nil {
			return "", errors.Wrap(err, errors.ErrCodeDaemonUnavailable, "failed to get container IP")
		}
		return fmt.Sprintf("%s:28080", containerIP), nil
	}

	// For remote sandboxes, use SSH port forwarding through provider
	// This would need to be implemented similarly to CLI's approach
	return "", errors.New(errors.ErrCodeDaemonUnavailable, "remote sandbox daemon connection not yet implemented in dashboard")
}

// connectToDaemon establishes a connection to the daemon
func (s *ClaudeService) connectToDaemon(address string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// getWorkingDirectory gets the working directory for the sandbox
func (s *ClaudeService) getWorkingDirectory(sandboxInfo *models.SandboxInfo) (string, error) {
	// This would involve calling the appropriate provider to get the working directory
	// Implementation depends on how the sandbox providers expose working directory info
	_ = sandboxInfo // Suppress unused parameter warning for now
	return "/workspace", nil // Default for now
}

// getWorkingDirectoryFromProvider gets the working directory from the appropriate provider based on sandbox type
func (s *ClaudeService) getWorkingDirectoryFromProvider(sandboxInfo *models.SandboxInfo) (string, error) {
	// Convert models.SandboxInfo to sandbox.SandboxInfo for provider compatibility
	sbInfo := &sandbox.SandboxInfo{
		ID:           sandboxInfo.ID,
		Name:         sandboxInfo.Name,
		Type:         sandbox.SandboxType(sandboxInfo.Type),
		State:        sandboxInfo.State,
		ShellCommand: sandboxInfo.ShellCommand,
		Metadata:     sandboxInfo.Metadata,
	}

	// Get the appropriate provider based on sandbox type
	if sandboxInfo.Type == models.TypeRemote {
		remoteProvider, err := remote.NewProvider()
		if err != nil {
			return "", fmt.Errorf("failed to create remote provider: %w", err)
		}
		return remoteProvider.GetWorkDir(sbInfo)
	} else {
		// For local sandboxes, we can return the working directory directly
		// without needing to create a new provider instance (which can cause database timeouts)
		// since local sandboxes always use /workspace
		return "/workspace", nil
	}
}

// getAnthropicAPIKey retrieves the Anthropic API key from various sources
func (s *ClaudeService) getAnthropicAPIKey() (string, error) {
	// First try environment variable
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		return apiKey, nil
	}

	// Try app-specific config first
	if apiKey, err := s.loadAppSpecificClaudeAPIKey(); err == nil && apiKey != "" {
		return apiKey, nil
	}

	// Try to read from Claude config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Try reading from Claude config file
	claudeConfigPath := filepath.Join(homeDir, ".claude", "config.toml")
	if content, err := os.ReadFile(claudeConfigPath); err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "api_key") && strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					apiKey := strings.TrimSpace(strings.Trim(parts[1], `"`))
					if apiKey != "" {
						return apiKey, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("no Anthropic API key found. Please set ANTHROPIC_API_KEY environment variable or configure Claude CLI")
}

// loadAppSpecificClaudeAPIKey tries to load API key from app-specific config
func (s *ClaudeService) loadAppSpecificClaudeAPIKey() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(homeDir, ".dispense", "claude", "config")
	content, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}

	// Just return the trimmed content as the API key
	apiKey := strings.TrimSpace(string(content))
	if apiKey == "" {
		return "", fmt.Errorf("API key not found in app-specific config")
	}

	return apiKey, nil
}

// retrieveLogsFromSandbox retrieves logs from the specified sandbox
func (s *ClaudeService) retrieveLogsFromSandbox(sandboxInfo *models.SandboxInfo, taskID string) ([]string, error) {
	logDir := "/home/daytona/.dispense/logs"

	if taskID != "" {
		// Retrieve specific task log
		return s.getTaskLogContent(sandboxInfo, logDir, taskID)
	} else {
		// Retrieve recent logs (list all log files)
		return s.getRecentLogs(sandboxInfo, logDir)
	}
}

// getTaskLogContent retrieves content of a specific task log file
func (s *ClaudeService) getTaskLogContent(sandboxInfo *models.SandboxInfo, logDir, taskID string) ([]string, error) {
	logFile := fmt.Sprintf("%s.log", taskID)
	logPath := fmt.Sprintf("%s/%s", logDir, logFile)

	// Check if log file exists
	_, err := s.executeSandboxCommand(sandboxInfo, []string{"test", "-f", logPath})
	if err != nil {
		return []string{fmt.Sprintf("Log file not found for task: %s", taskID)}, nil
	}

	// Read the log file content
	output, err := s.executeSandboxCommand(sandboxInfo, []string{"cat", logPath})
	if err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	// Split content into lines and add header
	lines := []string{
		fmt.Sprintf("📄 Log file: %s", logFile),
		"─────────────────────────────────────",
	}

	content := string(output)
	if content != "" {
		lines = append(lines, content)
	} else {
		lines = append(lines, "(log file is empty)")
	}

	lines = append(lines, "─────────────────────────────────────")
	return lines, nil
}

// getRecentLogs retrieves list of recent log files and their info
func (s *ClaudeService) getRecentLogs(sandboxInfo *models.SandboxInfo, logDir string) ([]string, error) {
	// List log files in directory
	output, err := s.executeSandboxCommand(sandboxInfo, []string{"ls", "-la", logDir})
	if err != nil {
		if strings.Contains(err.Error(), "exit status 2") || strings.Contains(err.Error(), "No such file or directory") {
			return []string{
				fmt.Sprintf("📋 No log files found in sandbox '%s'", sandboxInfo.Name),
				fmt.Sprintf("💡 Log directory doesn't exist: %s", logDir),
			}, nil
		}
		return nil, fmt.Errorf("failed to list log files: %w", err)
	}

	// Parse output to find .log files
	content := string(output)
	if strings.Contains(content, "No such file or directory") {
		return []string{
			fmt.Sprintf("📋 No log files found in sandbox '%s'", sandboxInfo.Name),
			fmt.Sprintf("💡 Log directory doesn't exist: %s", logDir),
		}, nil
	}

	lines := strings.Split(content, "\n")
	var logFiles []string
	var results []string

	results = append(results, fmt.Sprintf("📋 Recent Claude Logs from sandbox '%s':", sandboxInfo.Name))
	results = append(results, "")

	for _, line := range lines {
		if strings.Contains(line, ".log") && !strings.HasPrefix(line, "total") {
			// Extract filename from ls -la output
			fields := strings.Fields(line)
			if len(fields) >= 9 {
				filename := fields[len(fields)-1]
				if strings.HasSuffix(filename, ".log") {
					logFiles = append(logFiles, filename)
				}
			}
		}
	}

	if len(logFiles) == 0 {
		results = append(results, "📋 No Claude log files found in sandbox")
		return results, nil
	}

	// Show info for each log file
	for _, logFile := range logFiles {
		taskID := strings.TrimSuffix(logFile, ".log")
		results = append(results, fmt.Sprintf("  🔹 %s", taskID))

		// Get file stats
		logPath := fmt.Sprintf("%s/%s", logDir, logFile)
		statOutput, err := s.executeSandboxCommand(sandboxInfo, []string{"stat", "-c", "%Y %s", logPath})
		if err == nil {
			fields := strings.Fields(string(statOutput))
			if len(fields) >= 2 {
				if timestamp, err := strconv.ParseInt(fields[0], 10, 64); err == nil {
					modTime := time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
					results = append(results, fmt.Sprintf("     📅 %s", modTime))
					results = append(results, fmt.Sprintf("     📊 %s bytes", fields[1]))
				}
			}
		}
		results = append(results, "")
	}

	return results, nil
}

// executeSandboxCommand executes a command inside the sandbox based on sandbox type
func (s *ClaudeService) executeSandboxCommand(sandboxInfo *models.SandboxInfo, command []string) ([]byte, error) {
	// Create the appropriate provider based on sandbox type
	var provider sandbox.Provider
	var err error

	if sandboxInfo.Type == models.TypeRemote {
		provider, err = remote.NewProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to create remote provider: %w", err)
		}
	} else {
		provider, err = local.NewProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to create local provider: %w", err)
		}
	}

	// Convert command slice to single command string
	commandStr := strings.Join(command, " ")

	// Convert models.SandboxInfo to sandbox.SandboxInfo
	sbInfo := &sandbox.SandboxInfo{
		ID:           sandboxInfo.ID,
		Name:         sandboxInfo.Name,
		Type:         sandbox.SandboxType(sandboxInfo.Type),
		State:        sandboxInfo.State,
		ShellCommand: sandboxInfo.ShellCommand,
		Metadata:     sandboxInfo.Metadata,
	}

	// Execute the command using the provider
	result, err := provider.ExecuteCommand(sbInfo, commandStr)
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}

	// Return stdout, but if there's an error, include stderr
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("command failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	return []byte(result.Stdout), nil
}

// getSandboxContainerIP gets the IP address of a sandbox container using Docker commands
func (s *ClaudeService) getSandboxContainerIP(sandboxInfo *models.SandboxInfo) (string, error) {
	// Get container name from metadata
	containerName, exists := sandboxInfo.Metadata["container_name"]
	if !exists {
		return "", fmt.Errorf("container_name not found in sandbox metadata")
	}

	// If the container name is truncated in metadata, find the full container name
	containerNameStr := fmt.Sprintf("%v", containerName)
	if len(containerNameStr) > 50 { // Metadata might be truncated
		// Use the sandbox name to find the container
		cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", sandboxInfo.Name), "--format", "{{.Names}}")
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to find container for sandbox %s: %w", sandboxInfo.Name, err)
		}

		containerNames := strings.TrimSpace(string(output))
		if containerNames == "" {
			return "", fmt.Errorf("no running container found for sandbox %s", sandboxInfo.Name)
		}

		// Take the first container name (most recent)
		lines := strings.Split(containerNames, "\n")
		if len(lines) > 0 {
			containerNameStr = strings.TrimSpace(lines[0])
		}
	}

	// Get the IP address of the container
	cmd := exec.Command("docker", "inspect", containerNameStr, "--format", "{{.NetworkSettings.IPAddress}}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get IP address for container %s: %w", containerNameStr, err)
	}

	ip := strings.TrimSpace(string(output))
	if ip == "" {
		return "", fmt.Errorf("no IP address found for container %s", containerNameStr)
	}

	return ip, nil
}
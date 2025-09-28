package services

import (
	"context"
	"fmt"
	"io"
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

	// Create client and execute task
	client := pb.NewAgentServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	grpcReq := &pb.ExecuteClaudeRequest{
		Prompt: req.TaskDescription,
	}

	stream, err := client.ExecuteClaude(ctx, grpcReq)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeSystemUnavailable, "failed to execute task")
	}

	// Collect all streaming responses
	var output, errorMsg string
	success := true

	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			success = false
			errorMsg = err.Error()
			break
		}

		switch resp.Type {
		case pb.ExecuteClaudeResponse_STDOUT:
			output += resp.Content
		case pb.ExecuteClaudeResponse_STDERR:
			errorMsg += resp.Content
		case pb.ExecuteClaudeResponse_ERROR:
			success = false
			errorMsg += resp.Content
		case pb.ExecuteClaudeResponse_STATUS:
			if resp.ExitCode != 0 {
				success = false
			}
		}
	}

	return &models.ClaudeTaskResponse{
		Success:  success,
		Output:   output,
		ErrorMsg: errorMsg,
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
	// This logic would determine the daemon address based on sandbox type
	// For local sandboxes, it might be localhost with a mapped port
	// For remote sandboxes, it might be the sandbox's public endpoint
	
	if sandboxInfo.Type == models.TypeLocal {
		// For local sandboxes, daemon typically runs on a mapped port
		if port, exists := sandboxInfo.Metadata["daemon_port"]; exists {
			return fmt.Sprintf("localhost:%v", port), nil
		}
		return "", errors.New(errors.ErrCodeDaemonUnavailable, "daemon port not found in sandbox metadata")
	}

	// For remote sandboxes
	if addr, exists := sandboxInfo.Metadata["daemon_address"]; exists {
		return fmt.Sprintf("%v", addr), nil
	}

	return "", errors.New(errors.ErrCodeDaemonUnavailable, "daemon address not found in sandbox metadata")
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
	return "/workspace", nil // Default for now
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
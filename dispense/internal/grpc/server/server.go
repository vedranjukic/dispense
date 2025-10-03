package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	pb "cli/internal/grpc/proto"
	daemonpb "cli/proto"
	"cli/internal/services"
	"cli/internal/core/models"
	"cli/internal/core/errors"
	"cli/pkg/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DispenseServer implements the gRPC service
type DispenseServer struct {
	pb.UnimplementedDispenseServiceServer
	ServiceContainer *services.ServiceContainer
	Logger           *log.Logger
	configService    *config.Service
}

// NewDispenseServer creates a new gRPC server instance
func NewDispenseServer(serviceContainer *services.ServiceContainer) *DispenseServer {
	return &DispenseServer{
		ServiceContainer: serviceContainer,
		Logger:           log.New(os.Stdout, "[grpc-server] ", log.LstdFlags),
		configService:    config.NewService(),
	}
}

// CreateSandbox creates a new sandbox
func (s *DispenseServer) CreateSandbox(ctx context.Context, req *pb.CreateSandboxRequest) (*pb.CreateSandboxResponse, error) {
	s.Logger.Printf("CreateSandbox called for: %s", req.Name)

	// Validate request
	if err := s.validateCreateSandboxRequest(req); err != nil {
		return &pb.CreateSandboxResponse{
			Error: s.convertError(err),
		}, nil
	}

	// Convert to internal model
	createReq := s.convertCreateSandboxRequest(req)

	// Call service
	sandboxInfo, err := s.ServiceContainer.SandboxService.Create(createReq)
	if err != nil {
		s.Logger.Printf("Failed to create sandbox: %v", err)
		return &pb.CreateSandboxResponse{
			Error: s.convertError(err),
		}, nil
	}

	// Convert response
	return &pb.CreateSandboxResponse{
		Sandbox: s.convertSandboxInfo(sandboxInfo),
	}, nil
}

// ListSandboxes lists sandboxes
func (s *DispenseServer) ListSandboxes(ctx context.Context, req *pb.ListSandboxesRequest) (*pb.ListSandboxesResponse, error) {
	s.Logger.Printf("ListSandboxes called")

	// Convert to internal model
	opts := &models.SandboxListOptions{
		ShowLocal:  req.ShowLocal,
		ShowRemote: req.ShowRemote,
		Verbose:    req.Verbose,
		Group:      req.Group,
	}

	// Call service
	sandboxes, err := s.ServiceContainer.SandboxService.List(opts)
	if err != nil {
		s.Logger.Printf("Failed to list sandboxes: %v", err)
		return &pb.ListSandboxesResponse{
			Error: s.convertError(err),
		}, nil
	}

	// Convert response
	pbSandboxes := make([]*pb.SandboxInfo, len(sandboxes))
	for i, sb := range sandboxes {
		pbSandboxes[i] = s.convertSandboxInfo(sb)
	}

	return &pb.ListSandboxesResponse{
		Sandboxes: pbSandboxes,
	}, nil
}

// DeleteSandbox deletes a sandbox
func (s *DispenseServer) DeleteSandbox(ctx context.Context, req *pb.DeleteSandboxRequest) (*pb.DeleteSandboxResponse, error) {
	s.Logger.Printf("DeleteSandbox called for: %s", req.Identifier)

	// Convert to internal model
	opts := &models.SandboxDeleteOptions{
		DeleteAll: req.DeleteAll,
		Force:     req.Force,
	}

	// Call service
	err := s.ServiceContainer.SandboxService.Delete(req.Identifier, opts)
	if err != nil {
		s.Logger.Printf("Failed to delete sandbox: %v", err)
		return &pb.DeleteSandboxResponse{
			Success: false,
			Error:   s.convertError(err),
		}, nil
	}

	return &pb.DeleteSandboxResponse{
		Success: true,
		Message: "Sandbox deleted successfully",
	}, nil
}

// GetSandbox gets a specific sandbox
func (s *DispenseServer) GetSandbox(ctx context.Context, req *pb.GetSandboxRequest) (*pb.GetSandboxResponse, error) {
	s.Logger.Printf("GetSandbox called for: %s", req.Identifier)

	// Call service
	sandboxInfo, err := s.ServiceContainer.SandboxService.FindByName(req.Identifier)
	if err != nil {
		s.Logger.Printf("Failed to get sandbox: %v", err)
		return &pb.GetSandboxResponse{
			Error: s.convertError(err),
		}, nil
	}

	return &pb.GetSandboxResponse{
		Sandbox: s.convertSandboxInfo(sandboxInfo),
	}, nil
}

// WaitForSandbox waits for sandbox readiness
func (s *DispenseServer) WaitForSandbox(ctx context.Context, req *pb.WaitForSandboxRequest) (*pb.WaitForSandboxResponse, error) {
	s.Logger.Printf("WaitForSandbox called for: %s", req.Identifier)

	// Convert to internal model
	opts := &models.SandboxWaitOptions{
		Timeout: time.Duration(req.TimeoutSeconds) * time.Second,
		Group:   req.Group,
	}

	// Call service
	err := s.ServiceContainer.SandboxService.Wait(req.Identifier, opts)
	if err != nil {
		s.Logger.Printf("Failed to wait for sandbox: %v", err)
		return &pb.WaitForSandboxResponse{
			Success: false,
			Error:   s.convertError(err),
		}, nil
	}

	return &pb.WaitForSandboxResponse{
		Success: true,
		Message: "Sandbox is ready",
	}, nil
}

// GetProjectSources gets distinct project sources from all sandboxes
func (s *DispenseServer) GetProjectSources(ctx context.Context, req *pb.GetProjectSourcesRequest) (*pb.GetProjectSourcesResponse, error) {
	s.Logger.Printf("GetProjectSources called")

	// Convert to internal model
	opts := &models.SandboxListOptions{
		ShowLocal:  req.ShowLocal,
		ShowRemote: req.ShowRemote,
		Group:      req.Group,
	}

	// Call service
	projectSources, err := s.ServiceContainer.SandboxService.GetProjectSources(opts)
	if err != nil {
		s.Logger.Printf("Failed to get project sources: %v", err)
		return &pb.GetProjectSourcesResponse{
			Error: s.convertError(err),
		}, nil
	}

	return &pb.GetProjectSourcesResponse{
		ProjectSources: projectSources,
	}, nil
}

// RunClaudeTask runs a Claude task with streaming response
func (s *DispenseServer) RunClaudeTask(req *pb.RunClaudeTaskRequest, stream pb.DispenseService_RunClaudeTaskServer) error {
	s.Logger.Printf("RunClaudeTask called for sandbox: %s", req.SandboxIdentifier)

	// Convert to internal model
	claudeReq := &models.ClaudeTaskRequest{
		SandboxIdentifier: req.SandboxIdentifier,
		TaskDescription:   req.TaskDescription,
		Model:             req.Model,
	}

	// Call service (this would need to be modified to support streaming)
	response, err := s.ServiceContainer.ClaudeService.RunTask(claudeReq)
	if err != nil {
		// Send error response
		return stream.Send(&pb.RunClaudeTaskResponse{
			Type:       pb.RunClaudeTaskResponse_ERROR,
			Content:    err.Error(),
			Timestamp:  time.Now().Unix(),
			IsFinished: true,
		})
	}

	// Send success response
	responseType := pb.RunClaudeTaskResponse_STDOUT
	if !response.Success {
		responseType = pb.RunClaudeTaskResponse_ERROR
	}

	return stream.Send(&pb.RunClaudeTaskResponse{
		Type:       responseType,
		Content:    response.Output,
		Timestamp:  time.Now().Unix(),
		IsFinished: true,
	})
}

// GetClaudeStatus gets Claude daemon status
func (s *DispenseServer) GetClaudeStatus(ctx context.Context, req *pb.GetClaudeStatusRequest) (*pb.GetClaudeStatusResponse, error) {
	s.Logger.Printf("GetClaudeStatus called for: %s", req.SandboxIdentifier)

	// Convert to internal model
	statusReq := &models.ClaudeStatusRequest{
		SandboxIdentifier: req.SandboxIdentifier,
	}

	// Call service
	statusResp, err := s.ServiceContainer.ClaudeService.GetStatus(statusReq)
	if err != nil {
		s.Logger.Printf("Failed to get Claude status: %v", err)
		return &pb.GetClaudeStatusResponse{
			Error: s.convertError(err),
		}, nil
	}

	return &pb.GetClaudeStatusResponse{
		Connected:  statusResp.Connected,
		DaemonInfo: statusResp.DaemonInfo,
		WorkDir:    statusResp.WorkDir,
	}, nil
}

// GetClaudeLogs gets Claude logs
func (s *DispenseServer) GetClaudeLogs(ctx context.Context, req *pb.GetClaudeLogsRequest) (*pb.GetClaudeLogsResponse, error) {
	s.Logger.Printf("GetClaudeLogs called for: %s", req.SandboxIdentifier)

	// Convert to internal model
	logsReq := &models.ClaudeLogsRequest{
		SandboxIdentifier: req.SandboxIdentifier,
		TaskID:           req.TaskId,
	}

	// Call service
	logsResp, err := s.ServiceContainer.ClaudeService.GetLogs(logsReq)
	if err != nil {
		s.Logger.Printf("Failed to get Claude logs: %v", err)
		return &pb.GetClaudeLogsResponse{
			Error: s.convertError(err),
		}, nil
	}

	return &pb.GetClaudeLogsResponse{
		Success: logsResp.Success,
		Logs:    logsResp.Logs,
	}, nil
}

// StreamTaskLogs streams logs for a specific task in real-time
func (s *DispenseServer) StreamTaskLogs(req *pb.StreamTaskLogsRequest, stream pb.DispenseService_StreamTaskLogsServer) error {
	s.Logger.Printf("StreamTaskLogs called for task: %s in sandbox: %s", req.TaskId, req.SandboxIdentifier)

	// Get sandbox info using the provided sandbox identifier
	sandboxInfo, err := s.ServiceContainer.SandboxService.FindByName(req.SandboxIdentifier)
	if err != nil {
		s.Logger.Printf("Failed to find sandbox %s: %v", req.SandboxIdentifier, err)
		return stream.Send(&pb.StreamTaskLogsResponse{
			Type:      pb.StreamTaskLogsResponse_ERROR,
			Content:   fmt.Sprintf("Failed to find sandbox: %v", err),
			Timestamp: time.Now().Unix(),
		})
	}

	// Connect directly to the daemon for this sandbox
	daemonConn, err := s.connectToDaemon(sandboxInfo)
	if err != nil {
		s.Logger.Printf("Failed to connect to daemon: %v", err)
		return stream.Send(&pb.StreamTaskLogsResponse{
			Type:      pb.StreamTaskLogsResponse_ERROR,
			Content:   fmt.Sprintf("Failed to connect to daemon: %v", err),
			Timestamp: time.Now().Unix(),
		})
	}
	defer daemonConn.Close()

	// Create daemon client and forward the streaming request
	daemonClient := daemonpb.NewAgentServiceClient(daemonConn)

	// Create daemon request (need to import daemon proto package)
	daemonReq := &daemonpb.StreamTaskLogsRequest{
		TaskId:         req.TaskId,
		Follow:         req.Follow,
		IncludeHistory: req.IncludeHistory,
		FromTimestamp:  req.FromTimestamp,
	}

	// Call daemon streaming method
	daemonStream, err := daemonClient.StreamTaskLogs(stream.Context(), daemonReq)
	if err != nil {
		s.Logger.Printf("Failed to start daemon stream: %v", err)
		return stream.Send(&pb.StreamTaskLogsResponse{
			Type:      pb.StreamTaskLogsResponse_ERROR,
			Content:   fmt.Sprintf("Failed to start daemon stream: %v", err),
			Timestamp: time.Now().Unix(),
		})
	}

	// Forward all responses from daemon to client
	for {
		daemonResp, err := daemonStream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				s.Logger.Printf("Daemon stream completed for task %s", req.TaskId)
				return nil
			}
			s.Logger.Printf("Daemon stream error: %v", err)
			return stream.Send(&pb.StreamTaskLogsResponse{
				Type:      pb.StreamTaskLogsResponse_ERROR,
				Content:   fmt.Sprintf("Daemon stream error: %v", err),
				Timestamp: time.Now().Unix(),
			})
		}

		// Convert daemon response to dispense response format
		dispenseResp := &pb.StreamTaskLogsResponse{
			Type:          pb.StreamTaskLogsResponse_LogType(daemonResp.Type),
			Content:       daemonResp.Content,
			Timestamp:     daemonResp.Timestamp,
			TaskCompleted: daemonResp.TaskCompleted,
			TaskStatus:    daemonResp.TaskStatus,
		}

		// Forward to client
		if err := stream.Send(dispenseResp); err != nil {
			s.Logger.Printf("Failed to send response to client: %v", err)
			return err
		}

		// If task is completed, we're done
		if daemonResp.TaskCompleted {
			s.Logger.Printf("Task %s completed, ending stream", req.TaskId)
			return nil
		}
	}
}

// GetAPIKey gets the API key
func (s *DispenseServer) GetAPIKey(ctx context.Context, req *pb.GetAPIKeyRequest) (*pb.GetAPIKeyResponse, error) {
	s.Logger.Printf("GetAPIKey called")

	apiKey, err := s.configService.GetDaytonaAPIKey(req.Interactive)
	if err != nil {
		s.Logger.Printf("Failed to get API key: %v", err)
		return &pb.GetAPIKeyResponse{
			Error: s.convertError(err),
		}, nil
	}

	return &pb.GetAPIKeyResponse{
		ApiKey: apiKey,
	}, nil
}

// SetAPIKey sets the API key
func (s *DispenseServer) SetAPIKey(ctx context.Context, req *pb.SetAPIKeyRequest) (*pb.SetAPIKeyResponse, error) {
	s.Logger.Printf("SetAPIKey called")

	err := s.configService.SetDaytonaAPIKey(req.ApiKey)
	if err != nil {
		s.Logger.Printf("Failed to set API key: %v", err)
		return &pb.SetAPIKeyResponse{
			Success: false,
			Error:   s.convertError(err),
		}, nil
	}

	return &pb.SetAPIKeyResponse{
		Success: true,
		Message: "API key saved successfully",
	}, nil
}

// ValidateAPIKey validates an API key
func (s *DispenseServer) ValidateAPIKey(ctx context.Context, req *pb.ValidateAPIKeyRequest) (*pb.ValidateAPIKeyResponse, error) {
	s.Logger.Printf("ValidateAPIKey called")

	valid, message := s.configService.ValidateDaytonaAPIKey(req.ApiKey)

	return &pb.ValidateAPIKeyResponse{
		Valid:   valid,
		Message: message,
	}, nil
}

// Helper methods

func (s *DispenseServer) validateCreateSandboxRequest(req *pb.CreateSandboxRequest) error {
	if req.Name == "" && req.BranchName == "" {
		return errors.New(errors.ErrCodeValidationFailed, "name or branch_name must be provided")
	}
	return nil
}

func (s *DispenseServer) convertCreateSandboxRequest(req *pb.CreateSandboxRequest) *models.SandboxCreateRequest {
	createReq := &models.SandboxCreateRequest{
		Name:            req.Name,
		BranchName:      req.BranchName,
		IsRemote:        req.IsRemote,
		Force:           req.Force,
		SkipCopy:        req.SkipCopy,
		SkipDaemon:      req.SkipDaemon,
		Group:           req.Group,
		Model:           req.Model,
		Task:            req.Task,
		SourceDirectory: req.SourceDirectory,
	}

	// Convert resource allocation
	if req.Resources != nil {
		createReq.Snapshot = req.Resources.Snapshot
		createReq.Target = req.Resources.Target
		createReq.CPU = req.Resources.Cpu
		createReq.Memory = req.Resources.Memory
		createReq.Disk = req.Resources.Disk
		createReq.AutoStop = req.Resources.AutoStop
	}

	// Convert task data
	if req.TaskData != nil {
		createReq.TaskData = &models.TaskData{
			Description: req.TaskData.Description,
		}

		if req.TaskData.GithubIssue != nil {
			createReq.TaskData.GitHubIssue = &models.GitHubIssue{
				URL:    req.TaskData.GithubIssue.Url,
				Number: int(req.TaskData.GithubIssue.Number),
				Owner:  req.TaskData.GithubIssue.Owner,
				Repo:   req.TaskData.GithubIssue.Repo,
				Title:  req.TaskData.GithubIssue.Title,
				Body:   req.TaskData.GithubIssue.Body,
			}
		}

		if req.TaskData.GithubPr != nil {
			createReq.TaskData.GitHubPR = &models.GitHubPR{
				URL:    req.TaskData.GithubPr.Url,
				Number: int(req.TaskData.GithubPr.Number),
				Owner:  req.TaskData.GithubPr.Owner,
				Repo:   req.TaskData.GithubPr.Repo,
				Title:  req.TaskData.GithubPr.Title,
				Body:   req.TaskData.GithubPr.Body,
			}
		}
	}

	return createReq
}

func (s *DispenseServer) convertSandboxInfo(sb *models.SandboxInfo) *pb.SandboxInfo {
	pbSb := &pb.SandboxInfo{
		Id:            sb.ID,
		Name:          sb.Name,
		State:         sb.State,
		ShellCommand:  sb.ShellCommand,
		Group:         sb.Group,
		ProjectSource: sb.ProjectSource,
		Metadata:      make(map[string]string),
	}

	// Convert type
	switch sb.Type {
	case models.TypeLocal:
		pbSb.Type = pb.SandboxType_SANDBOX_TYPE_LOCAL
	case models.TypeRemote:
		pbSb.Type = pb.SandboxType_SANDBOX_TYPE_REMOTE
	default:
		pbSb.Type = pb.SandboxType_SANDBOX_TYPE_UNSPECIFIED
	}

	// Convert created at
	if sb.CreatedAt != nil {
		pbSb.CreatedAt = timestamppb.New(*sb.CreatedAt)
	}

	// Convert metadata
	if sb.Metadata != nil {
		for k, v := range sb.Metadata {
			if str, ok := v.(string); ok {
				pbSb.Metadata[k] = str
			} else {
				pbSb.Metadata[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	return pbSb
}

func (s *DispenseServer) convertError(err error) *pb.ErrorResponse {
	if err == nil {
		return nil
	}

	// Try to extract error code if it's a dispense error
	code := "UNKNOWN"
	if dispenseErr, ok := err.(*errors.DispenseError); ok {
		code = string(dispenseErr.Code)
	}

	return &pb.ErrorResponse{
		Code:    code,
		Message: err.Error(),
		Details: make(map[string]string),
	}
}

// findSandboxForTask finds which sandbox a task belongs to
// This is a simplified implementation - in production you might want to store task-to-sandbox mapping
func (s *DispenseServer) findSandboxForTask(taskID string) (string, error) {
	// For now, we'll search through all active sandboxes to find one that has this task
	// This is not very efficient but works for the MVP

	// List all sandboxes
	sandboxes, err := s.ServiceContainer.SandboxService.List(&models.SandboxListOptions{
		ShowLocal:  true,
		ShowRemote: true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list sandboxes: %w", err)
	}

	// Try each sandbox to see if it has this task
	for _, sandbox := range sandboxes {
		if s.sandboxHasTask(sandbox, taskID) {
			return sandbox.Name, nil
		}
	}

	return "", fmt.Errorf("task %s not found in any active sandbox", taskID)
}

// sandboxHasTask checks if a sandbox has a specific task
func (s *DispenseServer) sandboxHasTask(sandbox *models.SandboxInfo, taskID string) bool {
	// Try to connect to the sandbox's daemon and check if it has this task
	daemonConn, err := s.connectToDaemon(sandbox)
	if err != nil {
		// If we can't connect, assume this sandbox doesn't have the task
		return false
	}
	defer daemonConn.Close()

	// Create daemon client and check for task
	daemonClient := daemonpb.NewAgentServiceClient(daemonConn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to get task status - if it exists, this sandbox has the task
	_, err = daemonClient.GetTaskStatus(ctx, &daemonpb.TaskStatusRequest{
		TaskId: taskID,
	})

	// If no error, the task exists in this sandbox
	return err == nil
}

// connectToDaemon connects to a sandbox's daemon using the same logic as ClaudeService
func (s *DispenseServer) connectToDaemon(sandbox *models.SandboxInfo) (*grpc.ClientConn, error) {
	// Use the same logic as ClaudeService.getDaemonAddress
	var daemonAddr string

	if sandbox.Type == models.TypeLocal {
		// For local sandboxes, get the container IP and use port 28080
		containerIP, err := s.getSandboxContainerIP(sandbox)
		if err != nil {
			return nil, fmt.Errorf("failed to get container IP: %w", err)
		}
		daemonAddr = fmt.Sprintf("%s:28080", containerIP)
	} else {
		// For remote sandboxes, use SSH port forwarding through provider
		return nil, fmt.Errorf("remote sandbox daemon connection not yet implemented")
	}

	// Connect to daemon
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, daemonAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon at %s: %w", daemonAddr, err)
	}

	return conn, nil
}

// getSandboxContainerIP gets the IP address of a sandbox container (simplified version)
func (s *DispenseServer) getSandboxContainerIP(sandbox *models.SandboxInfo) (string, error) {
	// Get container name from metadata
	containerName, exists := sandbox.Metadata["container_name"]
	if !exists {
		return "", fmt.Errorf("container_name not found in sandbox metadata")
	}

	containerNameStr := fmt.Sprintf("%v", containerName)

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

// CreateClaudeTask creates a new Claude task and returns the task ID
func (s *DispenseServer) CreateClaudeTask(ctx context.Context, req *pb.CreateClaudeTaskRequest) (*pb.CreateClaudeTaskResponse, error) {
	s.Logger.Printf("CreateClaudeTask called for sandbox: %s", req.SandboxIdentifier)

	// Get sandbox info using the provided sandbox identifier
	sandboxInfo, err := s.ServiceContainer.SandboxService.FindByName(req.SandboxIdentifier)
	if err != nil {
		s.Logger.Printf("Failed to find sandbox %s: %v", req.SandboxIdentifier, err)
		return &pb.CreateClaudeTaskResponse{
			Error: s.convertError(err),
		}, nil
	}

	// Connect to the daemon
	daemonConn, err := s.connectToDaemon(sandboxInfo)
	if err != nil {
		s.Logger.Printf("Failed to connect to daemon: %v", err)
		return &pb.CreateClaudeTaskResponse{
			Error: s.convertError(err),
		}, nil
	}
	defer daemonConn.Close()

	// Create daemon client and forward the request
	daemonClient := daemonpb.NewAgentServiceClient(daemonConn)

	// Create daemon request - let daemon handle API key loading
	daemonReq := &daemonpb.CreateTaskRequest{
		Prompt:           req.TaskDescription,
		WorkingDirectory: req.WorkingDirectory,
		EnvironmentVars:  req.EnvironmentVars,
		AnthropicApiKey:  req.AnthropicApiKey, // Pass through if provided, let daemon load if empty
		Model:            req.Model,
	}

	// Call daemon CreateTask method
	daemonResp, err := daemonClient.CreateTask(ctx, daemonReq)
	if err != nil {
		s.Logger.Printf("Daemon CreateTask failed: %v", err)
		return &pb.CreateClaudeTaskResponse{
			Error: s.convertError(err),
		}, nil
	}

	if !daemonResp.Success {
		s.Logger.Printf("Daemon CreateTask returned failure: %s", daemonResp.Message)
		return &pb.CreateClaudeTaskResponse{
			Success: false,
			Message: daemonResp.Message,
		}, nil
	}

	return &pb.CreateClaudeTaskResponse{
		Success: true,
		TaskId:  daemonResp.TaskId,
		Message: daemonResp.Message,
	}, nil
}
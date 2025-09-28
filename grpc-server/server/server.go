package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	pb "dispense/grpc-server/proto"
	"cli/internal/services"
	"cli/internal/core/models"
	"cli/internal/core/errors"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// DispenseServer implements the gRPC service
type DispenseServer struct {
	pb.UnimplementedDispenseServiceServer
	ServiceContainer *services.ServiceContainer
	Logger           *log.Logger
}

// NewDispenseServer creates a new gRPC server instance
func NewDispenseServer(serviceContainer *services.ServiceContainer) *DispenseServer {
	return &DispenseServer{
		ServiceContainer: serviceContainer,
		Logger:           log.New(os.Stdout, "[grpc-server] ", log.LstdFlags),
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

// GetAPIKey gets the API key
func (s *DispenseServer) GetAPIKey(ctx context.Context, req *pb.GetAPIKeyRequest) (*pb.GetAPIKeyResponse, error) {
	s.Logger.Printf("GetAPIKey called")

	var apiKey string
	var err error

	if req.Interactive {
		apiKey, err = s.ServiceContainer.ConfigManager.GetOrPromptAPIKey()
	} else {
		apiKey, err = s.ServiceContainer.ConfigManager.LoadAPIKeyNonInteractive()
	}

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

	err := s.ServiceContainer.ConfigManager.SaveAPIKey(req.ApiKey)
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

	// For now, this is a basic validation
	// In a real implementation, this would make an API call to validate the key
	valid := req.ApiKey != "" && len(req.ApiKey) > 10

	message := "API key is valid"
	if !valid {
		message = "API key is invalid"
	}

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
		Id:           sb.ID,
		Name:         sb.Name,
		State:        sb.State,
		ShellCommand: sb.ShellCommand,
		Group:        sb.Group,
		Metadata:     make(map[string]string),
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

	// Try to extract error code if it's a custom error
	code := "UNKNOWN"
	if customErr, ok := err.(*errors.CustomError); ok {
		code = string(customErr.Code)
	}

	return &pb.ErrorResponse{
		Code:    code,
		Message: err.Error(),
		Details: make(map[string]string),
	}
}
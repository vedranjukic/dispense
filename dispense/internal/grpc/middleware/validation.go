package middleware

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "cli/internal/grpc/proto"
)

// ValidationInterceptor provides input validation middleware for gRPC
type ValidationInterceptor struct{}

// NewValidationInterceptor creates a new validation interceptor
func NewValidationInterceptor() *ValidationInterceptor {
	return &ValidationInterceptor{}
}

// UnaryServerInterceptor returns a unary server interceptor for validation
func (v *ValidationInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := v.validateRequest(req, info.FullMethod); err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

// validateRequest validates the incoming request based on the method
func (v *ValidationInterceptor) validateRequest(req interface{}, method string) error {
	switch {
	case strings.Contains(method, "CreateSandbox"):
		return v.validateCreateSandboxRequest(req)
	case strings.Contains(method, "DeleteSandbox"):
		return v.validateDeleteSandboxRequest(req)
	case strings.Contains(method, "GetSandbox"):
		return v.validateGetSandboxRequest(req)
	case strings.Contains(method, "WaitForSandbox"):
		return v.validateWaitForSandboxRequest(req)
	case strings.Contains(method, "RunClaudeTask"):
		return v.validateRunClaudeTaskRequest(req)
	case strings.Contains(method, "GetClaudeStatus"):
		return v.validateGetClaudeStatusRequest(req)
	case strings.Contains(method, "GetClaudeLogs"):
		return v.validateGetClaudeLogsRequest(req)
	case strings.Contains(method, "SetAPIKey"):
		return v.validateSetAPIKeyRequest(req)
	case strings.Contains(method, "ValidateAPIKey"):
		return v.validateValidateAPIKeyRequest(req)
	}

	return nil
}

// validateCreateSandboxRequest validates create sandbox request
func (v *ValidationInterceptor) validateCreateSandboxRequest(req interface{}) error {
	r, ok := req.(*pb.CreateSandboxRequest)
	if !ok {
		return status.Error(codes.InvalidArgument, "invalid request type")
	}

	if r.Name == "" && r.BranchName == "" {
		return status.Error(codes.InvalidArgument, "name or branch_name must be provided")
	}

	if strings.TrimSpace(r.Name) == "" && strings.TrimSpace(r.BranchName) == "" {
		return status.Error(codes.InvalidArgument, "name or branch_name cannot be empty")
	}

	// Validate resource allocation for remote sandboxes
	if r.IsRemote && r.Resources != nil {
		if r.Resources.Cpu < 0 || r.Resources.Memory < 0 || r.Resources.Disk < 0 {
			return status.Error(codes.InvalidArgument, "resource values cannot be negative")
		}
	}

	return nil
}

// validateDeleteSandboxRequest validates delete sandbox request
func (v *ValidationInterceptor) validateDeleteSandboxRequest(req interface{}) error {
	r, ok := req.(*pb.DeleteSandboxRequest)
	if !ok {
		return status.Error(codes.InvalidArgument, "invalid request type")
	}

	if !r.DeleteAll && strings.TrimSpace(r.Identifier) == "" {
		return status.Error(codes.InvalidArgument, "identifier must be provided when not deleting all")
	}

	return nil
}

// validateGetSandboxRequest validates get sandbox request
func (v *ValidationInterceptor) validateGetSandboxRequest(req interface{}) error {
	r, ok := req.(*pb.GetSandboxRequest)
	if !ok {
		return status.Error(codes.InvalidArgument, "invalid request type")
	}

	if strings.TrimSpace(r.Identifier) == "" {
		return status.Error(codes.InvalidArgument, "identifier is required")
	}

	return nil
}

// validateWaitForSandboxRequest validates wait for sandbox request
func (v *ValidationInterceptor) validateWaitForSandboxRequest(req interface{}) error {
	r, ok := req.(*pb.WaitForSandboxRequest)
	if !ok {
		return status.Error(codes.InvalidArgument, "invalid request type")
	}

	if strings.TrimSpace(r.Identifier) == "" {
		return status.Error(codes.InvalidArgument, "identifier is required")
	}

	if r.TimeoutSeconds < 0 {
		return status.Error(codes.InvalidArgument, "timeout cannot be negative")
	}

	return nil
}

// validateRunClaudeTaskRequest validates run claude task request
func (v *ValidationInterceptor) validateRunClaudeTaskRequest(req interface{}) error {
	r, ok := req.(*pb.RunClaudeTaskRequest)
	if !ok {
		return status.Error(codes.InvalidArgument, "invalid request type")
	}

	if strings.TrimSpace(r.SandboxIdentifier) == "" {
		return status.Error(codes.InvalidArgument, "sandbox_identifier is required")
	}

	if strings.TrimSpace(r.TaskDescription) == "" {
		return status.Error(codes.InvalidArgument, "task_description is required")
	}

	return nil
}

// validateGetClaudeStatusRequest validates get claude status request
func (v *ValidationInterceptor) validateGetClaudeStatusRequest(req interface{}) error {
	r, ok := req.(*pb.GetClaudeStatusRequest)
	if !ok {
		return status.Error(codes.InvalidArgument, "invalid request type")
	}

	if strings.TrimSpace(r.SandboxIdentifier) == "" {
		return status.Error(codes.InvalidArgument, "sandbox_identifier is required")
	}

	return nil
}

// validateGetClaudeLogsRequest validates get claude logs request
func (v *ValidationInterceptor) validateGetClaudeLogsRequest(req interface{}) error {
	r, ok := req.(*pb.GetClaudeLogsRequest)
	if !ok {
		return status.Error(codes.InvalidArgument, "invalid request type")
	}

	if strings.TrimSpace(r.SandboxIdentifier) == "" {
		return status.Error(codes.InvalidArgument, "sandbox_identifier is required")
	}

	return nil
}

// validateSetAPIKeyRequest validates set API key request
func (v *ValidationInterceptor) validateSetAPIKeyRequest(req interface{}) error {
	r, ok := req.(*pb.SetAPIKeyRequest)
	if !ok {
		return status.Error(codes.InvalidArgument, "invalid request type")
	}

	if strings.TrimSpace(r.ApiKey) == "" {
		return status.Error(codes.InvalidArgument, "api_key is required")
	}

	if len(r.ApiKey) < 10 {
		return status.Error(codes.InvalidArgument, "api_key appears to be too short")
	}

	return nil
}

// validateValidateAPIKeyRequest validates validate API key request
func (v *ValidationInterceptor) validateValidateAPIKeyRequest(req interface{}) error {
	r, ok := req.(*pb.ValidateAPIKeyRequest)
	if !ok {
		return status.Error(codes.InvalidArgument, "invalid request type")
	}

	if strings.TrimSpace(r.ApiKey) == "" {
		return status.Error(codes.InvalidArgument, "api_key is required")
	}

	return nil
}
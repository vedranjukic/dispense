package services

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"cli/internal/core/errors"
	"cli/internal/core/models"
	"cli/pkg/sandbox"
	"cli/pkg/sandbox/local"
	"cli/pkg/sandbox/remote"
)

// SandboxService handles all sandbox-related business logic
type SandboxService struct {
	configManager *ConfigManager
}

// NewSandboxService creates a new sandbox service
func NewSandboxService(configManager *ConfigManager) *SandboxService {
	return &SandboxService{
		configManager: configManager,
	}
}

// Create creates a new sandbox based on the provided request
func (s *SandboxService) Create(req *models.SandboxCreateRequest) (*models.SandboxInfo, error) {
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Parse task data if provided
	taskData, err := s.parseTaskData(req.Task)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeTaskInvalid, "failed to parse task")
	}
	req.TaskData = taskData

	// Auto-configure based on task data
	s.autoConfigureFromTask(req)

	// Create appropriate provider
	provider, err := s.createProvider(req.IsRemote)
	if err != nil {
		return nil, err
	}

	// Convert to sandbox package format
	opts := &sandbox.CreateOptions{
		Name:        req.BranchName,
		Snapshot:    req.Snapshot,
		Target:      req.Target,
		CPU:         req.CPU,
		Memory:      req.Memory,
		Disk:        req.Disk,
		AutoStop:    req.AutoStop,
		Force:       req.Force,
		SkipCopy:    req.SkipCopy,
		SkipDaemon:  req.SkipDaemon,
		BranchName:  req.BranchName,
		SourceDir:   req.SourceDirectory,
		Group:       req.Group,
		Model:       req.Model,
	}

	if req.TaskData != nil {
		taskDataJSON, _ := json.Marshal(req.TaskData)
		opts.TaskData = string(taskDataJSON)
		opts.GitHubIssue = req.TaskData.GitHubIssue != nil
	}

	// Create sandbox
	sandboxInfo, err := provider.Create(opts)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeSandboxCreateFailed, "failed to create sandbox")
	}

	return s.convertSandboxInfo(sandboxInfo), nil
}

// List returns a list of sandboxes based on the provided options
func (s *SandboxService) List(opts *models.SandboxListOptions) ([]*models.SandboxInfo, error) {
	var allSandboxes []*models.SandboxInfo

	// Default to showing both if no specific type requested
	if !opts.ShowLocal && !opts.ShowRemote {
		opts.ShowLocal = true
		opts.ShowRemote = true
	}

	// Get local sandboxes
	if opts.ShowLocal {
		localSandboxes, err := s.getLocalSandboxes(opts)
		if err != nil {
			// Log error but continue with remote sandboxes
			fmt.Fprintf(os.Stderr, "Warning: Failed to get local sandboxes: %v\n", err)
		} else {
			allSandboxes = append(allSandboxes, localSandboxes...)
		}
	}

	// Get remote sandboxes
	if opts.ShowRemote {
		remoteSandboxes, err := s.getRemoteSandboxes(opts)
		if err != nil {
			// Only show error if we're specifically requesting remote sandboxes
			if !opts.ShowLocal {
				return nil, err
			}
			// Otherwise just log warning
			fmt.Fprintf(os.Stderr, "Warning: Failed to get remote sandboxes: %v\n", err)
		} else {
			allSandboxes = append(allSandboxes, remoteSandboxes...)
		}
	}

	// Filter by group if specified
	if opts.Group != "" {
		filtered := make([]*models.SandboxInfo, 0)
		for _, sb := range allSandboxes {
			if sb.Group == opts.Group {
				filtered = append(filtered, sb)
			}
		}
		allSandboxes = filtered
	}

	return allSandboxes, nil
}

// Delete deletes a sandbox or all sandboxes based on the provided options
func (s *SandboxService) Delete(identifier string, opts *models.SandboxDeleteOptions) error {
	if opts.DeleteAll {
		return s.deleteAllSandboxes(opts.Force)
	}

	return s.deleteSingleSandbox(identifier, opts.Force)
}

// FindByName searches for a sandbox across providers by name or ID
func (s *SandboxService) FindByName(sandboxName string) (*models.SandboxInfo, error) {
	// Try local provider first
	if localProvider, err := local.NewProvider(); err == nil {
		if sandboxes, err := localProvider.List(); err == nil {
			for _, sb := range sandboxes {
				if sb.Name == sandboxName || sb.ID == sandboxName {
					return s.convertSandboxInfo(sb), nil
				}
			}
		}
	}

	// Try remote provider
	if remoteProvider, err := remote.NewProviderNonInteractive(); err == nil {
		if sandboxes, err := remoteProvider.List(); err == nil {
			for _, sb := range sandboxes {
				if sb.Name == sandboxName || sb.ID == sandboxName {
					return s.convertSandboxInfo(sb), nil
				}
			}
		}
	}

	return nil, errors.NewWithDetails(errors.ErrCodeSandboxNotFound, "sandbox not found", sandboxName)
}

// Wait waits for sandbox readiness
func (s *SandboxService) Wait(identifier string, opts *models.SandboxWaitOptions) error {
	// Implementation for wait logic
	// This would contain the wait logic from the wait command
	return errors.New(errors.ErrCodeSystemUnavailable, "wait functionality not yet implemented")
}

// validateCreateRequest validates the create request parameters
func (s *SandboxService) validateCreateRequest(req *models.SandboxCreateRequest) error {
	if req.BranchName == "" && req.Name == "" {
		return errors.New(errors.ErrCodeValidationFailed, "branch name or name must be provided")
	}

	// Additional validation logic
	return nil
}

// parseTaskData parses task description and extracts structured data
func (s *SandboxService) parseTaskData(taskDescription string) (*models.TaskData, error) {
	if taskDescription == "" {
		return nil, nil
	}

	taskData := &models.TaskData{
		Description: taskDescription,
	}

	// Parse GitHub issue URL
	githubIssueRegex := regexp.MustCompile(`https://github\.com/([^/]+)/([^/]+)/issues/(\d+)`)
	if matches := githubIssueRegex.FindStringSubmatch(taskDescription); len(matches) == 4 {
		taskData.GitHubIssue = &models.GitHubIssue{
			URL:   matches[0],
			Owner: matches[1],
			Repo:  matches[2],
		}
		// Parse number would go here
	}

	// Parse GitHub PR URL  
	githubPRRegex := regexp.MustCompile(`https://github\.com/([^/]+)/([^/]+)/pull/(\d+)`)
	if matches := githubPRRegex.FindStringSubmatch(taskDescription); len(matches) == 4 {
		taskData.GitHubPR = &models.GitHubPR{
			URL:   matches[0],
			Owner: matches[1],
			Repo:  matches[2],
		}
	}

	return taskData, nil
}

// autoConfigureFromTask automatically configures request based on parsed task data
func (s *SandboxService) autoConfigureFromTask(req *models.SandboxCreateRequest) {
	if req.TaskData == nil {
		return
	}

	// Auto-skip file copy for remote sandboxes with GitHub issues
	if !req.SkipCopy && req.TaskData.GitHubIssue != nil && req.IsRemote {
		req.SkipCopy = true
	}

	// Set empty source directory for GitHub issues
	if req.TaskData.GitHubIssue != nil {
		req.SourceDirectory = ""
	}
}

// createProvider creates appropriate sandbox provider
func (s *SandboxService) createProvider(isRemote bool) (sandbox.Provider, error) {
	if isRemote {
		provider, err := remote.NewProvider()
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrCodeProviderUnavailable, "failed to create remote provider")
		}
		return provider, nil
	}

	provider, err := local.NewProvider()
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeProviderUnavailable, "failed to create local provider")
	}
	return provider, nil
}

// getLocalSandboxes retrieves local sandboxes
func (s *SandboxService) getLocalSandboxes(opts *models.SandboxListOptions) ([]*models.SandboxInfo, error) {
	provider, err := local.NewProvider()
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeProviderUnavailable, "failed to create local provider")
	}

	sandboxes, err := provider.List()
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeProviderUnavailable, "failed to list local sandboxes")
	}

	result := make([]*models.SandboxInfo, len(sandboxes))
	for i, sb := range sandboxes {
		result[i] = s.convertSandboxInfo(sb)
	}

	return result, nil
}

// getRemoteSandboxes retrieves remote sandboxes
func (s *SandboxService) getRemoteSandboxes(opts *models.SandboxListOptions) ([]*models.SandboxInfo, error) {
	var provider sandbox.Provider
	var err error

	// Use appropriate provider based on context
	if opts.ShowLocal {
		// Both are being shown (default case) - use non-interactive to avoid API key prompt
		provider, err = remote.NewProviderNonInteractive()
	} else {
		// Only remote explicitly requested - use interactive
		provider, err = remote.NewProvider()
	}

	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeProviderUnavailable, "failed to create remote provider")
	}

	sandboxes, err := provider.List()
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeProviderUnavailable, "failed to list remote sandboxes")
	}

	result := make([]*models.SandboxInfo, len(sandboxes))
	for i, sb := range sandboxes {
		result[i] = s.convertSandboxInfo(sb)
	}

	return result, nil
}

// deleteAllSandboxes deletes all sandboxes
func (s *SandboxService) deleteAllSandboxes(force bool) error {
	// Implementation for deleting all sandboxes
	return errors.New(errors.ErrCodeSystemUnavailable, "delete all functionality not yet implemented")
}

// deleteSingleSandbox deletes a single sandbox
func (s *SandboxService) deleteSingleSandbox(identifier string, force bool) error {
	// Find the sandbox first
	sandboxInfo, err := s.FindByName(identifier)
	if err != nil {
		return err
	}

	// Create appropriate provider
	provider, err := s.createProvider(sandboxInfo.Type == models.TypeRemote)
	if err != nil {
		return err
	}

	// Delete the sandbox
	if err := provider.Delete(sandboxInfo.ID); err != nil {
		return errors.Wrap(err, errors.ErrCodeSandboxDeleteFailed, "failed to delete sandbox")
	}

	return nil
}

// convertSandboxInfo converts from sandbox package format to models format
func (s *SandboxService) convertSandboxInfo(sb *sandbox.SandboxInfo) *models.SandboxInfo {
	return &models.SandboxInfo{
		ID:           sb.ID,
		Name:         sb.Name,
		Type:         models.SandboxType(sb.Type),
		State:        sb.State,
		ShellCommand: sb.ShellCommand,
		Metadata:     sb.Metadata,
	}
}
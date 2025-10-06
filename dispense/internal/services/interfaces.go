package services

import (
	"cli/internal/core/models"
)

// SandboxServiceInterface defines the contract for sandbox operations
type SandboxServiceInterface interface {
	Create(req *models.SandboxCreateRequest) (*models.SandboxInfo, error)
	List(opts *models.SandboxListOptions) ([]*models.SandboxInfo, error)
	GetProjectSources(opts *models.SandboxListOptions) ([]string, error)
	Delete(identifier string, opts *models.SandboxDeleteOptions) error
	FindByName(sandboxName string) (*models.SandboxInfo, error)
	Wait(identifier string, opts *models.SandboxWaitOptions) error
}

// ClaudeServiceInterface defines the contract for Claude operations
type ClaudeServiceInterface interface {
	RunTask(req *models.ClaudeTaskRequest) (*models.ClaudeTaskResponse, error)
	GetStatus(req *models.ClaudeStatusRequest) (*models.ClaudeStatusResponse, error)
	GetLogs(req *models.ClaudeLogsRequest) (*models.ClaudeLogsResponse, error)
	ListTasks(req *models.ClaudeTaskListRequest) (*models.ClaudeTaskListResponse, error)
}

// ConfigManagerInterface defines the contract for configuration management
type ConfigManagerInterface interface {
	LoadAPIKey() (string, error)
	SaveAPIKey(apiKey string) error
	GetOrPromptAPIKey() (string, error)
	LoadAPIKeyNonInteractive() (string, error)
	PromptForAPIKey() (string, error)
	LoadAnthropicAPIKey() (string, error)
}

// ServiceContainer holds all services for dependency injection
type ServiceContainer struct {
	ConfigManager  ConfigManagerInterface
	SandboxService SandboxServiceInterface
	ClaudeService  ClaudeServiceInterface
}

// NewServiceContainer creates a new service container with all dependencies wired
func NewServiceContainer() *ServiceContainer {
	// Create config manager
	configManager := NewConfigManager()

	// Create sandbox service with config manager dependency
	sandboxService := NewSandboxService(configManager)

	// Create Claude service with sandbox service dependency
	claudeService := NewClaudeService(sandboxService)

	return &ServiceContainer{
		ConfigManager:  configManager,
		SandboxService: sandboxService,
		ClaudeService:  claudeService,
	}
}
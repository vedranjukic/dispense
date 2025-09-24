package sandbox

import (
	"fmt"
)

// SandboxType represents the type of sandbox
type SandboxType string

const (
	TypeLocal  SandboxType = "local"
	TypeRemote SandboxType = "remote"
)

// CreateOptions contains options for creating a sandbox
type CreateOptions struct {
	Name         string
	Snapshot     string
	Target       string
	CPU          int32
	Memory       int32
	Disk         int32
	AutoStop     int32
	Force        bool
	SkipCopy     bool
	SkipDaemon   bool
	BranchName   string
	SourceDir    string
	TaskData     string  // JSON serialized task data
	GitHubIssue  bool    // Indicates if this is for a GitHub issue (affects project setup)
	Group        string  // Optional group parameter for organizing sandboxes
	Model        string  // Optional model parameter
}

// SandboxInfo contains information about a created sandbox
type SandboxInfo struct {
	ID           string
	Name         string
	Type         SandboxType
	State        string
	ShellCommand string
	Metadata     map[string]interface{}
}

// Provider defines the interface for sandbox providers
type Provider interface {
	// Create creates a new sandbox
	Create(opts *CreateOptions) (*SandboxInfo, error)

	// CopyFiles copies files from local directory to sandbox
	CopyFiles(sandboxInfo *SandboxInfo, localPath string) error

	// InstallDaemon installs the embedded daemon to the sandbox
	InstallDaemon(sandboxInfo *SandboxInfo) error

	// CloneGitHubRepo clones a GitHub repository into the sandbox workspace
	CloneGitHubRepo(sandboxInfo *SandboxInfo, owner, repo, branchName string) error

	// GetInfo retrieves information about a sandbox
	GetInfo(id string) (*SandboxInfo, error)

	// List lists all sandboxes managed by this provider
	List() ([]*SandboxInfo, error)

	// Delete removes a sandbox
	Delete(id string) error

	// GetType returns the provider type
	GetType() SandboxType

	// ExecuteShell starts an interactive shell session in the sandbox
	ExecuteShell(sandboxInfo *SandboxInfo) error

	// GetWorkDir returns the working directory path for the sandbox environment
	GetWorkDir(sandboxInfo *SandboxInfo) (string, error)
}

// These will be implemented by importing the specific provider packages
// For now, they return not-implemented errors

// ProviderFactory creates sandbox providers
type ProviderFactory struct{}

// NewProvider creates a new sandbox provider based on type
func (f *ProviderFactory) NewProvider(sandboxType SandboxType) (Provider, error) {
	switch sandboxType {
	case TypeLocal:
		// Will be implemented by importing local provider
		return nil, fmt.Errorf("local provider not yet implemented")
	case TypeRemote:
		// Will be implemented by importing remote provider
		return nil, fmt.Errorf("remote provider not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported sandbox type: %s", sandboxType)
	}
}
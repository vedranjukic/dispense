package models

import "time"

// SandboxType represents the type of sandbox
type SandboxType string

const (
	TypeLocal  SandboxType = "local"
	TypeRemote SandboxType = "remote"
)

// SandboxCreateRequest contains all parameters for creating a sandbox
type SandboxCreateRequest struct {
	Name         string
	BranchName   string
	IsRemote     bool
	Force        bool
	SkipCopy     bool
	SkipDaemon   bool
	Group        string
	Model        string
	Task         string
	
	// Resource allocation (remote only)
	Snapshot     string
	Target       string
	CPU          int32
	Memory       int32
	Disk         int32
	AutoStop     int32
	
	// Project context
	SourceDirectory string
	TaskData        *TaskData
}

// TaskData represents parsed task information
type TaskData struct {
	Description  string        `json:"description"`
	GitHubIssue  *GitHubIssue  `json:"github_issue,omitempty"`
	GitHubPR     *GitHubPR     `json:"github_pr,omitempty"`
}

// GitHubIssue represents GitHub issue information
type GitHubIssue struct {
	URL    string `json:"url"`
	Number int    `json:"number"`
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// GitHubPR represents GitHub pull request information  
type GitHubPR struct {
	URL    string `json:"url"`
	Number int    `json:"number"`
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// SandboxInfo contains information about a created or existing sandbox
type SandboxInfo struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         SandboxType            `json:"type"`
	State        string                 `json:"state"`
	ShellCommand string                 `json:"shell_command"`
	CreatedAt    *time.Time             `json:"created_at,omitempty"`
	Group        string                 `json:"group,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// SandboxListOptions contains options for listing sandboxes
type SandboxListOptions struct {
	ShowLocal   bool
	ShowRemote  bool
	Verbose     bool
	Group       string
}

// SandboxDeleteOptions contains options for deleting sandboxes
type SandboxDeleteOptions struct {
	DeleteAll bool
	Force     bool
}

// SandboxExecResult contains the result of command execution
type SandboxExecResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// SandboxWaitOptions contains options for waiting for sandbox readiness
type SandboxWaitOptions struct {
	Timeout time.Duration
	Group   string
}
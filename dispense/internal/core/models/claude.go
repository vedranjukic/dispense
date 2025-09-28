package models

// ClaudeTaskRequest represents a request to run a Claude task
type ClaudeTaskRequest struct {
	SandboxIdentifier string
	TaskDescription   string
	Model             string
}

// ClaudeStatusRequest represents a request to get Claude status
type ClaudeStatusRequest struct {
	SandboxIdentifier string
}

// ClaudeTaskResponse represents the response from a Claude task
type ClaudeTaskResponse struct {
	Success   bool   `json:"success"`
	Output    string `json:"output"`
	ErrorMsg  string `json:"error,omitempty"`
}

// ClaudeStatusResponse represents Claude daemon status
type ClaudeStatusResponse struct {
	Connected    bool   `json:"connected"`
	DaemonInfo   string `json:"daemon_info,omitempty"`
	WorkDir      string `json:"work_dir,omitempty"`
	ErrorMsg     string `json:"error,omitempty"`
}

// ClaudeLogsRequest represents a request to get Claude logs
type ClaudeLogsRequest struct {
	SandboxIdentifier string
	TaskID           string // optional, if empty returns recent logs
}

// ClaudeLogsResponse represents the response from getting Claude logs
type ClaudeLogsResponse struct {
	Success bool     `json:"success"`
	Logs    []string `json:"logs"`
	ErrorMsg string  `json:"error,omitempty"`
}
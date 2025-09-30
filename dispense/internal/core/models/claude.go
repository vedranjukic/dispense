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
	TaskID    string `json:"task_id,omitempty"`  // For async tasks
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

// ClaudeTaskStatusRequest represents a request to get task status
type ClaudeTaskStatusRequest struct {
	SandboxIdentifier string
	TaskID            string
}

// ClaudeTaskStatusResponse represents task status information
type ClaudeTaskStatusResponse struct {
	Success      bool   `json:"success"`
	State        string `json:"state"`        // PENDING, RUNNING, COMPLETED, FAILED
	Message      string `json:"message"`
	ExitCode     int32  `json:"exit_code"`
	StartedAt    int64  `json:"started_at"`
	FinishedAt   int64  `json:"finished_at"`
	Prompt       string `json:"prompt"`
	WorkDir      string `json:"work_dir"`
	ErrorMsg     string `json:"error,omitempty"`
}

// ClaudeTaskListRequest represents a request to list Claude tasks
type ClaudeTaskListRequest struct {
	SandboxIdentifier string
}

// ClaudeTaskInfo represents information about a Claude task
type ClaudeTaskInfo struct {
	TaskID      string `json:"task_id"`
	Prompt      string `json:"prompt"`
	State       string `json:"state"`
	StartedAt   int64  `json:"started_at"`
	FinishedAt  int64  `json:"finished_at"`
	ExitCode    int32  `json:"exit_code"`
	Error       string `json:"error"`
	WorkDir     string `json:"work_dir"`
}

// ClaudeTaskListResponse represents the response from listing Claude tasks
type ClaudeTaskListResponse struct {
	Success  bool              `json:"success"`
	Tasks    []ClaudeTaskInfo  `json:"tasks"`
	ErrorMsg string            `json:"error,omitempty"`
}
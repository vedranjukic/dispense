package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"daemon/proto"
)

// Task represents a running Claude task
type Task struct {
	ID            string
	Prompt        string
	WorkingDir    string
	StartedAt     time.Time
	FinishedAt    *time.Time
	State         proto.TaskStatusResponse_TaskState
	ExitCode      *int32
	Error         *string
	Process       *exec.Cmd
	LogFile       *os.File
	StdoutPipe    io.ReadCloser
	StderrPipe    io.ReadCloser
	cancel        context.CancelFunc
	ctx           context.Context
}

// TaskManager manages Claude execution tasks
type TaskManager struct {
	tasks  map[string]*Task
	mutex  sync.RWMutex
	logDir string
}

// NewTaskManager creates a new task manager
func NewTaskManager(logDir string) *TaskManager {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Warning: Failed to create log directory %s: %v", logDir, err)
	}

	return &TaskManager{
		tasks:  make(map[string]*Task),
		logDir: logDir,
	}
}

// StartClaudeTask starts a new Claude execution task
func (tm *TaskManager) StartClaudeTask(prompt, workingDir, apiKey, model string, envVars map[string]string) (string, error) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// Generate task ID
	taskID := fmt.Sprintf("claude_%d", time.Now().UnixNano())

	// Create context for task cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Create log file
	logFileName := fmt.Sprintf("%s.log", taskID)
	logFilePath := filepath.Join(tm.logDir, logFileName)
	logFile, err := os.Create(logFilePath)
	if err != nil {
		cancel()
		return "", fmt.Errorf("failed to create log file: %w", err)
	}

	// Set working directory (default to /workspace if not specified)
	if workingDir == "" {
		workingDir = "/workspace"
	}

	// Configure Claude on first run
	if err := tm.ensureClaudeConfigured(workingDir); err != nil {
		logFile.Close()
		cancel()
		return "", fmt.Errorf("failed to configure Claude: %w", err)
	}

	// Create task (mark as running initially)
	task := &Task{
		ID:         taskID,
		Prompt:     prompt,
		WorkingDir: workingDir,
		StartedAt:  time.Now(),
		State:      proto.TaskStatusResponse_RUNNING,
		LogFile:    logFile,
		cancel:     cancel,
		ctx:        ctx,
	}

	// Store task
	tm.tasks[taskID] = task

	log.Printf("Starting Claude task %s with prompt: %s", taskID, prompt)
	if apiKey != "" {
		log.Printf("API key provided: length %d, starts with: %.10s...", len(apiKey), apiKey)
	} else {
		log.Printf("No API key provided")
	}
	if model != "" {
		log.Printf("Model specified: %s", model)
	} else {
		log.Printf("No model specified")
	}

	// Execute Claude synchronously and log output
	go func() {
		defer func() {
			logFile.Close()
			tm.completeTask(taskID)
			cancel() // Cancel context to signal completion
		}()
		// Prepare Claude command for synchronous execution
		cmd := exec.CommandContext(ctx, "claude", "--dangerously-skip-permissions", "--print", "--output-format=stream-json", "--include-partial-messages", "--verbose", prompt)
		cmd.Dir = workingDir

		// Set environment variables
		cmd.Env = os.Environ()

		// Add ANTHROPIC_API_KEY if provided
		if apiKey != "" {
			log.Printf("Setting ANTHROPIC_API_KEY environment variable for task %s", taskID)
			cmd.Env = append(cmd.Env, fmt.Sprintf("ANTHROPIC_API_KEY=%s", apiKey))
		} else {
			log.Printf("No API key provided for task %s", taskID)
		}

		// Add ANTHROPIC_MODEL if provided
		if model != "" {
			log.Printf("Setting ANTHROPIC_MODEL environment variable for task %s", taskID)
			cmd.Env = append(cmd.Env, fmt.Sprintf("ANTHROPIC_MODEL=%s", model))
		} else {
			log.Printf("No model provided for task %s", taskID)
		}

		for key, value := range envVars {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}

		// Get stdout and stderr pipes
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("Task %s [ERROR]: Failed to create stdout pipe: %v", taskID, err)
			errorMsg := fmt.Sprintf("Failed to create stdout pipe: %v", err)
			tm.markTaskFailed(taskID, &errorMsg, 1)
			return
		}

		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			log.Printf("Task %s [ERROR]: Failed to create stderr pipe: %v", taskID, err)
			errorMsg := fmt.Sprintf("Failed to create stderr pipe: %v", err)
			tm.markTaskFailed(taskID, &errorMsg, 1)
			return
		}

		// Store pipes in task for potential cleanup
		tm.mutex.Lock()
		if task, exists := tm.tasks[taskID]; exists {
			task.Process = cmd
			task.StdoutPipe = stdoutPipe
			task.StderrPipe = stderrPipe
		}
		tm.mutex.Unlock()

		// Start the command
		if err := cmd.Start(); err != nil {
			log.Printf("Task %s [ERROR]: Failed to start command: %v", taskID, err)
			errorMsg := fmt.Sprintf("Failed to start command: %v", err)
			tm.markTaskFailed(taskID, &errorMsg, 1)
			return
		}

		log.Printf("Task %s started with PID: %d", taskID, cmd.Process.Pid)

		// Start goroutines to stream stdout and stderr
		var wg sync.WaitGroup
		wg.Add(2)

		// Stream stdout
		go func() {
			defer wg.Done()
			tm.streamOutput(taskID, stdoutPipe, "STDOUT")
		}()

		// Stream stderr
		go func() {
			defer wg.Done()
			tm.streamOutput(taskID, stderrPipe, "STDERR")
		}()

		// Wait for streaming to complete
		wg.Wait()

		// Wait for the process to finish
		err = cmd.Wait()
		timestamp := time.Now().Format(time.RFC3339)

		tm.mutex.Lock()
		if task, exists := tm.tasks[taskID]; exists {
			finishedAt := time.Now()
			task.FinishedAt = &finishedAt

			if err != nil {
				// Process failed
				var exitCode int32 = 1
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = int32(exitError.ExitCode())
				}
				task.ExitCode = &exitCode
				task.State = proto.TaskStatusResponse_FAILED
				errorMsg := err.Error()
				task.Error = &errorMsg

				logEntry := fmt.Sprintf("[%s] [ERROR] Claude execution failed: %v\n", timestamp, err)
				logFile.WriteString(logEntry)
				log.Printf("Task %s [ERROR]: Claude execution failed: %v", taskID, err)
			} else {
				// Process succeeded
				exitCode := int32(0)
				task.ExitCode = &exitCode
				task.State = proto.TaskStatusResponse_COMPLETED

				logEntry := fmt.Sprintf("[%s] [STATUS] Task completed successfully\n", timestamp)
				logFile.WriteString(logEntry)
				log.Printf("Task %s completed successfully", taskID)
			}
		}
		tm.mutex.Unlock()

		// Log completion
		completionEntry := fmt.Sprintf("[%s] [STATUS] Task completed with exit code %d\n",
			timestamp,
			func() int32 {
				if err != nil {
					if exitError, ok := err.(*exec.ExitError); ok {
						return int32(exitError.ExitCode())
					}
					return 1
				}
				return 0
			}())
		logFile.WriteString(completionEntry)
		logFile.Sync()
	}()

	return taskID, nil
}

// completeTask handles task completion logging
func (tm *TaskManager) completeTask(taskID string) {
	tm.mutex.RLock()
	task, exists := tm.tasks[taskID]
	tm.mutex.RUnlock()

	if !exists {
		log.Printf("Task %s not found during completion", taskID)
		return
	}

	// Log completion to file if it's still open
	if task.LogFile != nil {
		logEntry := fmt.Sprintf("[%s] [STATUS] Task %s completed\n", time.Now().Format(time.RFC3339), taskID)
		task.LogFile.WriteString(logEntry)
		task.LogFile.Sync()
	}

	log.Printf("Task %s completed", taskID)
}

// streamOutput streams output from a pipe to the log file in real-time
func (tm *TaskManager) streamOutput(taskID string, pipe io.ReadCloser, streamType string) {
	defer pipe.Close()

	// Get the task and log file
	tm.mutex.RLock()
	task, exists := tm.tasks[taskID]
	tm.mutex.RUnlock()

	if !exists {
		log.Printf("Task %s not found during output streaming", taskID)
		return
	}

	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		timestamp := time.Now().Format(time.RFC3339)

		// Write to log file
		logEntry := fmt.Sprintf("[%s] [%s] %s\n", timestamp, streamType, line)
		task.LogFile.WriteString(logEntry)
		task.LogFile.Sync() // Ensure data is written to disk immediately

		// Log to daemon logs as well
		log.Printf("Task %s [%s]: %s", taskID, streamType, line)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading %s for task %s: %v", streamType, taskID, err)
		// Log the error to the file as well
		timestamp := time.Now().Format(time.RFC3339)
		errorEntry := fmt.Sprintf("[%s] [ERROR] Error reading %s: %v\n", timestamp, streamType, err)
		task.LogFile.WriteString(errorEntry)
		task.LogFile.Sync()
	}
}

// markTaskFailed marks a task as failed with the given error message and exit code
func (tm *TaskManager) markTaskFailed(taskID string, errorMsg *string, exitCode int32) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	if task, exists := tm.tasks[taskID]; exists {
		task.State = proto.TaskStatusResponse_FAILED
		task.ExitCode = &exitCode
		task.Error = errorMsg
		finishedAt := time.Now()
		task.FinishedAt = &finishedAt

		// Log to file if available
		if task.LogFile != nil {
			timestamp := time.Now().Format(time.RFC3339)
			logEntry := fmt.Sprintf("[%s] [ERROR] %s\n", timestamp, *errorMsg)
			task.LogFile.WriteString(logEntry)
			task.LogFile.Sync()
		}
	}
}


// GetTaskStatus returns the current status of a task
func (tm *TaskManager) GetTaskStatus(taskID string) (*proto.TaskStatusResponse, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	// If taskID is empty, return the status of the most recent task
	if taskID == "" {
		return tm.getLatestTaskStatus()
	}

	task, exists := tm.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	response := &proto.TaskStatusResponse{
		State:            task.State,
		StartedAt:        task.StartedAt.Unix(),
		Prompt:           task.Prompt,
		WorkingDirectory: task.WorkingDir,
	}

	if task.FinishedAt != nil {
		response.FinishedAt = task.FinishedAt.Unix()
	}

	if task.ExitCode != nil {
		response.ExitCode = *task.ExitCode
	}

	if task.Error != nil {
		response.Error = *task.Error
	}

	switch task.State {
	case proto.TaskStatusResponse_RUNNING:
		response.Message = "Task is currently running"
	case proto.TaskStatusResponse_COMPLETED:
		response.Message = "Task completed successfully"
	case proto.TaskStatusResponse_FAILED:
		response.Message = "Task failed"
	default:
		response.Message = "Task status unknown"
	}

	return response, nil
}

// getLatestTaskStatus returns the status of the most recent task (helper method - assumes mutex is already held)
func (tm *TaskManager) getLatestTaskStatus() (*proto.TaskStatusResponse, error) {
	var latestTask *Task
	var latestStartTime time.Time

	// Find the most recent task
	for _, task := range tm.tasks {
		if latestTask == nil || task.StartedAt.After(latestStartTime) {
			latestTask = task
			latestStartTime = task.StartedAt
		}
	}

	// If no tasks found, return idle status
	if latestTask == nil {
		return &proto.TaskStatusResponse{
			State:   proto.TaskStatusResponse_PENDING,
			Message: "No tasks found - daemon is ready",
		}, nil
	}

	// Build response for the latest task
	response := &proto.TaskStatusResponse{
		State:            latestTask.State,
		StartedAt:        latestTask.StartedAt.Unix(),
		Prompt:           latestTask.Prompt,
		WorkingDirectory: latestTask.WorkingDir,
	}

	if latestTask.FinishedAt != nil {
		response.FinishedAt = latestTask.FinishedAt.Unix()
	}

	if latestTask.ExitCode != nil {
		response.ExitCode = *latestTask.ExitCode
	}

	if latestTask.Error != nil {
		response.Error = *latestTask.Error
	}

	switch latestTask.State {
	case proto.TaskStatusResponse_RUNNING:
		response.Message = "Task is currently running"
	case proto.TaskStatusResponse_COMPLETED:
		response.Message = "Task completed successfully"
	case proto.TaskStatusResponse_FAILED:
		response.Message = "Task failed"
	default:
		response.Message = "Task status unknown"
	}

	return response, nil
}

// StreamTaskOutput streams the output of a Claude task (simplified synchronous version)
func (tm *TaskManager) StreamTaskOutput(taskID string, stream proto.AgentService_ExecuteClaudeServer) error {
	tm.mutex.RLock()
	task, exists := tm.tasks[taskID]
	tm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Wait for task to complete
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-task.ctx.Done():
			// Task completed, send all output at once and final status
			tm.mutex.RLock()
			currentTask, exists := tm.tasks[taskID]
			tm.mutex.RUnlock()

			if !exists {
				return fmt.Errorf("task not found during streaming: %s", taskID)
			}

			// Read and send the log file content (if available)
			if currentTask.LogFile != nil {
				logFileName := currentTask.LogFile.Name()
				if content, err := os.ReadFile(logFileName); err == nil {
					lines := strings.Split(string(content), "\n")
					for _, line := range lines {
						if strings.TrimSpace(line) == "" {
							continue
						}

						// Parse log entry and determine type
						responseType := proto.ExecuteClaudeResponse_STDOUT
						content := line
						if strings.Contains(line, "[ERROR]") {
							responseType = proto.ExecuteClaudeResponse_ERROR
						} else if strings.Contains(line, "[STATUS]") {
							responseType = proto.ExecuteClaudeResponse_STATUS
						}

						if err := stream.Send(&proto.ExecuteClaudeResponse{
							Type:      responseType,
							Content:   content,
							Timestamp: time.Now().Unix(),
						}); err != nil {
							log.Printf("Error sending stream data: %v", err)
							return err
						}
					}
				}
			}

			// Send final completion status
			var exitCode int32
			if currentTask.ExitCode != nil {
				exitCode = *currentTask.ExitCode
			}

			return stream.Send(&proto.ExecuteClaudeResponse{
				Type:       proto.ExecuteClaudeResponse_STATUS,
				Content:    "Task completed",
				Timestamp:  time.Now().Unix(),
				ExitCode:   exitCode,
				IsFinished: true,
			})
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}


// StopTask stops a running task
func (tm *TaskManager) StopTask(taskID string) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if task.State == proto.TaskStatusResponse_RUNNING {
		// Cancel context to stop the process
		task.cancel()

		// If we have a process, try to kill it gracefully
		if task.Process != nil && task.Process.Process != nil {
			if err := task.Process.Process.Kill(); err != nil {
				log.Printf("Error killing process for task %s: %v", taskID, err)
			}
		}

		// Close pipes if they exist
		if task.StdoutPipe != nil {
			task.StdoutPipe.Close()
		}
		if task.StderrPipe != nil {
			task.StderrPipe.Close()
		}

		task.State = proto.TaskStatusResponse_FAILED
		finishedAt := time.Now()
		task.FinishedAt = &finishedAt
		exitCode := int32(-1) // Indicate terminated
		task.ExitCode = &exitCode
		errorMsg := "Task was stopped by user"
		task.Error = &errorMsg

		// Log termination
		if task.LogFile != nil {
			timestamp := time.Now().Format(time.RFC3339)
			logEntry := fmt.Sprintf("[%s] [STATUS] Task stopped by user\n", timestamp)
			task.LogFile.WriteString(logEntry)
			task.LogFile.Sync()
		}

		log.Printf("Stopped task %s", taskID)
	}

	return nil
}

// CleanupTask removes a completed task from memory
func (tm *TaskManager) CleanupTask(taskID string) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Only cleanup completed or failed tasks
	if task.State == proto.TaskStatusResponse_RUNNING {
		return fmt.Errorf("cannot cleanup running task")
	}

	delete(tm.tasks, taskID)
	log.Printf("Cleaned up task %s", taskID)
	return nil
}

// ensureClaudeConfigured configures Claude with necessary settings and tools on first run
func (tm *TaskManager) ensureClaudeConfigured(workingDir string) error {
	// List of configuration commands to run
	configCommands := [][]string{
		{"claude", "config", "set", "hasCompletedProjectOnboarding", "true"},
		{"claude", "config", "set", "hasTrustDialogAccepted", "true"},
		{"claude", "config", "add", "allowedTools", "Bash"},
		{"claude", "config", "add", "allowedTools", "Read"},
		{"claude", "config", "add", "allowedTools", "Write"},
		{"claude", "config", "add", "allowedTools", "Edit"},
		{"claude", "config", "add", "allowedTools", "Create"},
	}

	log.Printf("Configuring Claude with necessary settings and tools...")

	for _, cmdArgs := range configCommands {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Dir = workingDir
		cmd.Env = os.Environ()

		output, err := cmd.CombinedOutput()
		if err != nil {
			// Log warning but don't fail - some configs might already be set
			log.Printf("Warning: Claude config command failed (%v): %s", cmdArgs, string(output))
		} else {
			log.Printf("Claude config applied: %v", cmdArgs[2:])
		}
	}

	log.Printf("Claude configuration completed")
	return nil
}

// ListTasks returns a list of all tasks, optionally filtered by state
func (tm *TaskManager) ListTasks(stateFilter *proto.TaskStatusResponse_TaskState) ([]*proto.TaskInfo, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	var tasks []*proto.TaskInfo

	for _, task := range tm.tasks {
		// Apply state filter if provided
		if stateFilter != nil && task.State != *stateFilter {
			continue
		}

		taskInfo := &proto.TaskInfo{
			TaskId:           task.ID,
			Prompt:           task.Prompt,
			State:            task.State,
			StartedAt:        task.StartedAt.Unix(),
			WorkingDirectory: task.WorkingDir,
		}

		if task.FinishedAt != nil {
			taskInfo.FinishedAt = task.FinishedAt.Unix()
		}

		if task.ExitCode != nil {
			taskInfo.ExitCode = *task.ExitCode
		}

		if task.Error != nil {
			taskInfo.Error = *task.Error
		}

		tasks = append(tasks, taskInfo)
	}

	return tasks, nil
}
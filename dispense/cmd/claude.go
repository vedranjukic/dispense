package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"cli/pkg/sandbox"
	"cli/pkg/sandbox/local"
	"cli/pkg/sandbox/remote"
	"cli/pkg/utils"
	pb "cli/proto"
)


// findSandboxByName searches for a sandbox across providers by name
func findSandboxByName(sandboxName string) (*sandbox.SandboxInfo, error) {
	utils.DebugPrintf("Searching for sandbox: %s\n", sandboxName)

	// Try local provider first
	localProvider, err := local.NewProvider()
	if err == nil {
		sandboxes, err := localProvider.List()
		if err == nil {
			utils.DebugPrintf("Found %d local sandboxes\n", len(sandboxes))
			for _, sb := range sandboxes {
				utils.DebugPrintf("Checking local sandbox: ID='%s', Name='%s'\n", sb.ID, sb.Name)
				if sb.Name == sandboxName || sb.ID == sandboxName {
					utils.DebugPrintf("Found matching local sandbox: %s\n", sb.ID)
					return sb, nil
				}
			}
		} else {
			utils.DebugPrintf("Failed to list local sandboxes: %v\n", err)
		}
	} else {
		utils.DebugPrintf("Failed to create local provider: %v\n", err)
	}

	// Try remote provider (non-interactive to avoid prompting for API key)
	remoteProvider, err := remote.NewProviderNonInteractive()
	if err == nil {
		sandboxes, err := remoteProvider.List()
		if err == nil {
			utils.DebugPrintf("Found %d remote sandboxes\n", len(sandboxes))
			for _, sb := range sandboxes {
				utils.DebugPrintf("Checking remote sandbox: ID='%s', Name='%s'\n", sb.ID, sb.Name)
				if sb.Name == sandboxName || sb.ID == sandboxName {
					utils.DebugPrintf("Found matching remote sandbox: %s\n", sb.ID)
					return sb, nil
				}
			}
		} else {
			utils.DebugPrintf("Failed to list remote sandboxes: %v\n", err)
		}
	} else {
		utils.DebugPrintf("Failed to create remote provider: %v\n", err)
	}

	return nil, fmt.Errorf("sandbox not found: %s", sandboxName)
}

// getDaemonConnection gets connection details for daemon (works with both local and remote)
func getDaemonConnection(sandboxName string) (string, func(), error) {
	// Find the sandbox to determine its type
	sandboxInfo, err := findSandboxByName(sandboxName)
	if err != nil {
		return "", nil, fmt.Errorf("failed to find sandbox: %w", err)
	}

	// Handle connection based on sandbox type
	if sandboxInfo.Type == sandbox.TypeRemote {
		// Remote sandbox - use SSH port forwarding
		remoteProvider, err := remote.NewProvider()
		if err != nil {
			return "", nil, fmt.Errorf("failed to create remote provider: %w", err)
		}

		return remoteProvider.GetDaemonConnection(sandboxInfo)
	} else {
		// Local sandbox - use direct IP connection
		ip, err := getSandboxIP(sandboxName)
		if err != nil {
			return "", nil, fmt.Errorf("failed to get sandbox IP: %w", err)
		}

		daemonAddr := fmt.Sprintf("%s:28080", ip)
		return daemonAddr, func() {}, nil // No cleanup needed for local connections
	}
}

// executeSandboxCommand runs a command in a sandbox (both local Docker and remote Daytona)
func executeSandboxCommand(sandboxName string, command []string) ([]byte, error) {
	// Find the sandbox to determine its type
	sandboxInfo, err := findSandboxByName(sandboxName)
	if err != nil {
		return nil, fmt.Errorf("failed to find sandbox: %w", err)
	}

	if sandboxInfo.Type == sandbox.TypeRemote {
		// Remote sandbox - use Daytona API
		remoteProvider, err := remote.NewProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to create remote provider: %w", err)
		}

		// Convert command slice to single command string
		commandStr := strings.Join(command, " ")

		// Use the public method to run commands
		response, err := remoteProvider.RunCommandInSandbox(sandboxInfo.ID, commandStr, "")
		if err != nil {
			return nil, fmt.Errorf("failed to run remote command: %w", err)
		}

		return []byte(response), nil
	} else {
		// Local sandbox - use Docker exec
		containerName, err := getContainerName(sandboxName)
		if err != nil {
			return nil, fmt.Errorf("failed to get container name: %w", err)
		}

		// Prepend docker exec command
		dockerCmd := append([]string{"docker", "exec", containerName}, command...)
		cmd := exec.Command(dockerCmd[0], dockerCmd[1:]...)
		return cmd.Output()
	}
}

// getSandboxIP gets the IP address of a sandbox container
func getSandboxIP(sandboxName string) (string, error) {
	// Get container names for the sandbox, with creation time for sorting
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", sandboxName), "--format", "{{.CreatedAt}}\t{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to find container for sandbox %s: %w", sandboxName, err)
	}

	containerNames := strings.TrimSpace(string(output))
	if containerNames == "" {
		return "", fmt.Errorf("no running container found for sandbox %s", sandboxName)
	}

	// Parse lines and find the most recent container
	lines := strings.Split(containerNames, "\n")
	var mostRecentContainer string

	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) >= 2 {
			containerName := strings.TrimSpace(parts[1])
			if mostRecentContainer == "" {
				mostRecentContainer = containerName
			}
			// Since docker ps shows newest first by default, take the first one
			break
		}
	}

	if mostRecentContainer == "" {
		return "", fmt.Errorf("no valid container found for sandbox %s", sandboxName)
	}

	// Get the IP address of the container
	cmd = exec.Command("docker", "inspect", mostRecentContainer, "--format", "{{.NetworkSettings.IPAddress}}")
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get IP address for container %s: %w", mostRecentContainer, err)
	}

	ip := strings.TrimSpace(string(output))
	if ip == "" {
		return "", fmt.Errorf("no IP address found for container %s", mostRecentContainer)
	}

	return ip, nil
}

// getContainerName gets the container name for a sandbox
func getContainerName(sandboxName string) (string, error) {
	// Get container names for the sandbox, with creation time for sorting
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", sandboxName), "--format", "{{.CreatedAt}}\t{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to find container for sandbox %s: %w", sandboxName, err)
	}

	containerNames := strings.TrimSpace(string(output))
	if containerNames == "" {
		return "", fmt.Errorf("no running container found for sandbox %s", sandboxName)
	}

	// Parse lines and find the most recent container
	lines := strings.Split(containerNames, "\n")
	var mostRecentContainer string

	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) >= 2 {
			containerName := strings.TrimSpace(parts[1])
			if mostRecentContainer == "" {
				mostRecentContainer = containerName
			}
			// Since docker ps shows newest first by default, take the first one
			break
		}
	}

	if mostRecentContainer == "" {
		return "", fmt.Errorf("no valid container found for sandbox %s", sandboxName)
	}

	return mostRecentContainer, nil
}

var claudeCmd = &cobra.Command{
	Use:   "claude [sandbox-name] [subcommand]",
	Short: "Claude Code management commands for sandboxes",
	Long:  `Commands for managing Claude Code execution and tasks in sandboxes.
All claude commands require a sandbox name as the first argument.

Usage:
  cli claude <sandbox-name> status
  cli claude <sandbox-name> run "prompt"
  cli claude <sandbox-name> tasks [task-id]
  cli claude <sandbox-name> logs [task-id] [--format=human|raw] [--follow]`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "❌ Both sandbox name and subcommand are required\n\n")
			cmd.Help()
			return
		}

		sandboxName := args[0]
		subcommand := args[1]

		switch subcommand {
		case "status":
			if err := checkClaudeDaemonStatus(sandboxName); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Claude daemon status check failed: %s\n", err)
				os.Exit(1)
			}
		case "run":
			if len(args) < 3 {
				fmt.Fprintf(os.Stderr, "❌ Prompt is required for 'run' command\n")
				os.Exit(1)
			}
			prompt := args[2]
			workDir, err := getWorkDirFromProvider(sandboxName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Failed to get working directory: %s\n", err)
				os.Exit(1)
			}
			modelFlag := cmd.Root().Flag("model").Value.String()
			if err := runClaudeWithPrompt(prompt, workDir, sandboxName, modelFlag); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Claude execution failed: %s\n", err)
				os.Exit(1)
			}
		case "tasks":
			var taskID string
			if len(args) > 2 {
				taskID = args[2]
			}
			if err := listClaudeTasks(sandboxName, taskID); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Failed to list Claude tasks: %s\n", err)
				os.Exit(1)
			}
		case "issue":
			if err := runClaudeOnGitHubIssue(sandboxName); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Failed to start Claude on GitHub issue: %s\n", err)
				os.Exit(1)
			}
		case "logs":
			var taskID string
			if len(args) > 2 {
				taskID = args[2]
			}
			formatFlag, _ := cmd.Flags().GetString("format")
			followFlag, _ := cmd.Flags().GetBool("follow")
			if err := showClaudeLogs(taskID, sandboxName, formatFlag == "human", followFlag); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Failed to show Claude logs: %s\n", err)
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "❌ Unknown subcommand: %s\n", subcommand)
			fmt.Fprintf(os.Stderr, "Available subcommands: status, run, tasks, logs\n")
			os.Exit(1)
		}
	},
}

// Removed old subcommand definitions - now handled inline in main claude command

// checkClaudeDaemonStatus checks if the daemon is running and reachable
func checkClaudeDaemonStatus(sandboxName string) error {
	if sandboxName == "" {
		return fmt.Errorf("sandbox name is required. Use --sandbox flag to specify which sandbox to check")
	}

	utils.DebugPrintf("Checking Claude daemon status for sandbox: %s\n", sandboxName)

	// Get daemon connection (works with both local and remote)
	daemonAddr, cleanup, err := getDaemonConnection(sandboxName)
	if err != nil {
		return fmt.Errorf("failed to get daemon connection: %w", err)
	}
	defer cleanup()

	utils.DebugPrintf("Connecting to daemon at: %s\n", daemonAddr)

	// Try to connect to the daemon
	conn, err := grpc.NewClient(daemonAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("daemon not reachable at %s: %w", daemonAddr, err)
	}
	defer conn.Close()

	// Test connection with a simple health check using the AgentService
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to create an AgentService client and test connectivity
	client := pb.NewAgentServiceClient(conn)

	// Get status of the most recent task (empty task ID returns latest task status)
	status, err := client.GetTaskStatus(ctx, &pb.TaskStatusRequest{TaskId: ""})
	if err != nil {
		// If there's an error (e.g., no tasks found), daemon is ready but no tasks
		utils.DebugPrintf("GetTaskStatus error: %v\n", err)
		fmt.Printf("🟢 Claude is ready in sandbox '%s'\n", sandboxName)
		return nil
	}

	if status != nil {
		switch status.State {
		case pb.TaskStatusResponse_RUNNING:
			fmt.Printf("🟡 Working in sandbox '%s'", sandboxName)
		case pb.TaskStatusResponse_COMPLETED:
			fmt.Printf("🟢 Done in sandbox '%s'", sandboxName)
		case pb.TaskStatusResponse_FAILED:
			fmt.Printf("🔴 Error in sandbox '%s'", sandboxName)
			if status.Error != "" {
				fmt.Printf(": %s", status.Error)
			}
		case pb.TaskStatusResponse_PENDING:
			fmt.Printf("🟢 Claude is ready in sandbox '%s'", sandboxName)
		default:
			fmt.Printf("🟢 Claude is ready in sandbox '%s'", sandboxName)
		}
	} else {
		fmt.Printf("🟢 Claude is ready in sandbox '%s'", sandboxName)
	}

	return nil
}

// getTaskStatus returns the current task status for the sandbox
func getTaskStatus(sandboxName, taskID string) (pb.TaskStatusResponse_TaskState, error) {
	// Get daemon connection (works with both local and remote)
	daemonAddr, cleanup, err := getDaemonConnection(sandboxName)
	if err != nil {
		return pb.TaskStatusResponse_PENDING, fmt.Errorf("failed to get daemon connection: %w", err)
	}
	defer cleanup()

	// Try to connect to the daemon
	conn, err := grpc.NewClient(daemonAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return pb.TaskStatusResponse_PENDING, fmt.Errorf("daemon not reachable at %s: %w", daemonAddr, err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := pb.NewAgentServiceClient(conn)

	// Get status of the specific task (empty task ID returns latest task status)
	status, err := client.GetTaskStatus(ctx, &pb.TaskStatusRequest{TaskId: taskID})
	if err != nil {
		// If there's an error (e.g., no tasks found), assume completed
		return pb.TaskStatusResponse_COMPLETED, nil
	}

	if status != nil {
		return status.State, nil
	}

	return pb.TaskStatusResponse_COMPLETED, nil
}

// runClaudeWithPrompt executes Claude with the given prompt
func runClaudeWithPrompt(prompt, workDir, sandboxName, model string) error {
	if sandboxName == "" {
		return fmt.Errorf("sandbox name is required. Use --sandbox flag to specify which sandbox to use")
	}

	utils.DebugPrintf("Running Claude with prompt: %s in sandbox: %s\n", prompt, sandboxName)

	// Get daemon connection (works with both local and remote)
	daemonAddr, cleanup, err := getDaemonConnection(sandboxName)
	if err != nil {
		return fmt.Errorf("failed to get daemon connection: %w", err)
	}
	defer cleanup()

	utils.DebugPrintf("Connecting to daemon at: %s\n", daemonAddr)

	// Connect to daemon
	conn, err := grpc.NewClient(daemonAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to daemon at %s: %w", daemonAddr, err)
	}
	defer conn.Close()

	// Create AgentService client
	client := pb.NewAgentServiceClient(conn)

	// Get Anthropic API key
	apiKey, err := utils.GetAnthropicAPIKey()
	if err != nil {
		return fmt.Errorf("failed to get Anthropic API key: %w", err)
	}

	// Prepare the request
	req := &pb.ExecuteClaudeRequest{
		Prompt:           prompt,
		WorkingDirectory: workDir,
		EnvironmentVars:  make(map[string]string), // Add any needed env vars
		AnthropicApiKey:  apiKey,
		Model:           model,
	}

	fmt.Printf("🟡 Claude is working...\n")

	// Execute Claude and stream the output
	ctx := context.Background()
	stream, err := client.ExecuteClaude(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to start Claude execution: %w", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error receiving stream: %w", err)
		}

		// Handle different response types
		switch resp.Type {
		case pb.ExecuteClaudeResponse_STDOUT:
			fmt.Print(resp.Content)
		case pb.ExecuteClaudeResponse_STDERR:
			fmt.Fprintf(os.Stderr, "%s", resp.Content)
		case pb.ExecuteClaudeResponse_STATUS:
			if resp.IsFinished {
				if resp.ExitCode == 0 {
					fmt.Printf("🟢 Done\n")
				} else {
					fmt.Printf("🔴 Error\n")
				}
			}
		case pb.ExecuteClaudeResponse_ERROR:
			fmt.Fprintf(os.Stderr, "❌ Error: %s\n", resp.Content)
		}

		if resp.IsFinished {
			break
		}
	}

	return nil
}

// listClaudeTasks shows running and recent Claude tasks, or details for a specific task
func listClaudeTasks(sandboxName, taskID string) error {
	if sandboxName == "" {
		return fmt.Errorf("sandbox name is required. Use --sandbox flag to specify which sandbox to connect to")
	}

	utils.DebugPrintf("Listing Claude tasks from sandbox: %s\n", sandboxName)

	// Try to get tasks from daemon first
	daemonAddr, cleanup, err := getDaemonConnection(sandboxName)
	if err != nil {
		return fmt.Errorf("failed to get daemon connection: %w", err)
	}
	defer cleanup()

	conn, err := grpc.NewClient(daemonAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		utils.DebugPrintf("Failed to connect to daemon, will check log files in sandbox: %v\n", err)
	} else {
		defer conn.Close()

		client := pb.NewAgentServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if taskID != "" {
			// Show details for a specific task
			taskStatus, err := client.GetTaskStatus(ctx, &pb.TaskStatusRequest{TaskId: taskID})
			if err != nil {
				utils.DebugPrintf("Failed to get task details from daemon: %v\n", err)
			} else {
				return showTaskDetails(sandboxName, taskID, taskStatus)
			}
		} else {
			// List all tasks
			tasksResp, err := client.ListTasks(ctx, &pb.ListTasksRequest{})
			if err != nil {
				utils.DebugPrintf("Failed to get tasks from daemon: %v\n", err)
			} else {
				fmt.Printf("📋 Tasks from sandbox '%s' daemon:\n", sandboxName)

				if len(tasksResp.Tasks) == 0 {
					fmt.Printf("💡 No tasks found\n")
					return nil
				}

				fmt.Printf("\n")
				for _, task := range tasksResp.Tasks {
					stateEmoji := getTaskStateEmoji(task.State)
					stateText := getTaskStateText(task.State)

					startTime := time.Unix(task.StartedAt, 0).Format("2006-01-02 15:04:05")

					fmt.Printf("  %s %s\n", stateEmoji, task.TaskId)
					fmt.Printf("     📝 %s\n", truncatePrompt(task.Prompt, 100))
					fmt.Printf("     📅 Started: %s\n", startTime)
					fmt.Printf("     📊 State: %s\n", stateText)

					if task.FinishedAt > 0 {
						endTime := time.Unix(task.FinishedAt, 0).Format("2006-01-02 15:04:05")
						duration := time.Unix(task.FinishedAt, 0).Sub(time.Unix(task.StartedAt, 0))
						fmt.Printf("     ⏱️  Duration: %s (finished: %s)\n", duration.String(), endTime)
					}

					if task.Error != "" {
						fmt.Printf("     ❌ Error: %s\n", task.Error)
					}

					fmt.Println()
				}
				return nil
			}
		}
	}

	// Fall back to log files in sandbox (not host)
	if taskID != "" {
		// Show details for a specific task using log files
		return showTaskDetailsFromLogFile(sandboxName, taskID)
	} else {
		// List all tasks using log files
		return listTasksFromLogFiles(sandboxName)
	}
}

// showClaudeLogs displays logs for a specific task or recent logs
func showClaudeLogs(taskID string, sandboxName string, humanFormat bool, follow bool) error {
	if sandboxName == "" {
		return fmt.Errorf("sandbox name is required. Use --sandbox flag to specify which sandbox to get logs from")
	}

	utils.DebugPrintf("Showing Claude logs for task: %s in sandbox: %s\n", taskID, sandboxName)

	sandboxLogDir := "/home/daytona/.dispense/logs"

	if taskID == "" {
		// Show recent logs from all tasks
		fmt.Printf("📋 Recent Claude Logs from sandbox '%s':\n\n", sandboxName)

		// List log files in sandbox
		output, err := executeSandboxCommand(sandboxName, []string{"ls", "-la", sandboxLogDir})
		if err != nil {
			// Check if directory doesn't exist
			if strings.Contains(err.Error(), "exit status 2") {
				fmt.Printf("📋 No log files found in sandbox\n")
				fmt.Printf("💡 Log directory doesn't exist: %s\n", sandboxLogDir)
				return nil
			}
			return fmt.Errorf("failed to list log files in sandbox: %w", err)
		}

		if strings.Contains(string(output), "No such file or directory") {
			fmt.Printf("📋 No log files found in sandbox\n")
			fmt.Printf("💡 Log directory doesn't exist: %s\n", sandboxLogDir)
			return nil
		}

		// Parse the output to find log files
		lines := strings.Split(string(output), "\n")
		var logFiles []string

		for _, line := range lines {
			if strings.Contains(line, "claude_") && strings.HasSuffix(line, ".log") {
				fields := strings.Fields(line)
				if len(fields) >= 9 {
					filename := fields[8]
					logFiles = append(logFiles, filename)
				}
			}
		}

		if len(logFiles) > 0 {
			// Take the most recent log file (first in the list since ls -la shows newest first by default)
			mostRecentFile := logFiles[0]
			// Extract taskID from filename (remove .log extension)
			mostRecentTaskID := strings.TrimSuffix(mostRecentFile, ".log")
			return showLogFileFromSandbox(sandboxName, sandboxLogDir, mostRecentFile, humanFormat, follow, mostRecentTaskID)
		} else {
			fmt.Printf("📋 No Claude log files found in sandbox\n")
			return nil
		}
	} else {
		// Show specific task logs
		logFile := fmt.Sprintf("%s.log", taskID)
		return showLogFileFromSandbox(sandboxName, sandboxLogDir, logFile, humanFormat, follow, taskID)
	}
}


// showLogFileFromSandbox displays the contents of a log file from inside a sandbox
func showLogFileFromSandbox(sandboxName, logDir, filename string, humanFormat bool, follow bool, taskID string) error {
	logPath := filepath.Join(logDir, filename)

	// Check if log file exists in sandbox
	_, err := executeSandboxCommand(sandboxName, []string{"test", "-f", logPath})
	if err != nil {
		return fmt.Errorf("log file not found in sandbox: %s", filename)
	}

	if follow {
		return followLogFile(sandboxName, logPath, filename, humanFormat, taskID)
	}

	if humanFormat {
		fmt.Printf("📄 Human-readable log: %s (from sandbox)\n", filename)
		fmt.Printf("─────────────────────────────────────\n")

		// Read and display the log file content with human formatting
		output, err := executeSandboxCommand(sandboxName, []string{"cat", logPath})
		if err != nil {
			return fmt.Errorf("failed to read log file from sandbox: %w", err)
		}

		if err := formatLogOutput(string(output)); err != nil {
			fmt.Printf("❌ Error formatting log: %v\n", err)
			fmt.Print(string(output)) // Fall back to raw output
		}
	} else {
		fmt.Printf("📄 Raw log file: %s (from sandbox)\n", filename)
		fmt.Printf("─────────────────────────────────────\n")

		// Read and display the log file content
		output, err := executeSandboxCommand(sandboxName, []string{"cat", logPath})
		if err != nil {
			return fmt.Errorf("failed to read log file from sandbox: %w", err)
		}

		fmt.Print(string(output))
	}

	fmt.Printf("─────────────────────────────────────\n")
	return nil
}

// followLogFile follows a log file in real-time until the task completes
func followLogFile(sandboxName, logPath, filename string, humanFormat bool, taskID string) error {
	fmt.Printf("📄 Following log: %s (from sandbox) - Press Ctrl+C to stop\n", filename)
	fmt.Printf("─────────────────────────────────────\n")

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Printf("\r\033[K📋 Log following stopped by user\n")
		cancel()
	}()

	var lastSize int64 = 0
	var lastActivity time.Time = time.Now()
	var isWorking bool = false
	var workingState WorkingState = WorkingStateProcessing

	// Start spinner animation
	spinnerDone := make(chan struct{})
	defer close(spinnerDone)
	go startSpinner(&isWorking, &workingState, spinnerDone)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Check if file still exists
			_, err := executeSandboxCommand(sandboxName, []string{"test", "-f", logPath})
			if err != nil {
				// If file doesn't exist, wait and retry
				time.Sleep(1 * time.Second)
				continue
			}

			// Get current file size
			sizeOutput, err := executeSandboxCommand(sandboxName, []string{"stat", "-c", "%s", logPath})
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}

			currentSize, err := strconv.ParseInt(strings.TrimSpace(string(sizeOutput)), 10, 64)
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}

			// If file has grown, read the new content
			if currentSize > lastSize {
				// Use tail to read new content from the last position
				tailOutput, err := executeSandboxCommand(sandboxName, []string{"tail", "-c", fmt.Sprintf("+%d", lastSize+1), logPath})
				if err != nil {
					time.Sleep(1 * time.Second)
					continue
				}

				newContent := string(tailOutput)
				if newContent != "" {
					if humanFormat {
						// Process each line for follow mode with minimal output
						lines := strings.Split(strings.TrimRight(newContent, "\n"), "\n")
						for _, line := range lines {
							if strings.TrimSpace(line) != "" {
								processLogLineForFollow(line, &lastActivity, &isWorking, &workingState)
							}
						}
					} else {
						// Print raw content but clear the working indicator first
						if isWorking {
							fmt.Print("\r\033[K")
						}
						fmt.Print(newContent)
						// Raw mode doesn't use advanced working states
						if isWorking {
							fmt.Print("⠋ Working...")
						}
					}

					// No need to parse content for completion - we'll check status below
				}

				lastSize = currentSize
			}

			// Check task status to see if it's still running
			taskStatus, err := getTaskStatus(sandboxName, taskID)
			if err != nil {
				// If we can't get status, assume task is completed and exit
				utils.DebugPrintf("Failed to get task status: %v\n", err)
				fmt.Printf("\r\033[K🏁 Task completed - log following ended\n")
				fmt.Printf("─────────────────────────────────────\n")
				return nil
			}

			// Exit if task is no longer running
			switch taskStatus {
			case pb.TaskStatusResponse_COMPLETED:
				fmt.Printf("\r\033[K🟢 Task completed - log following ended\n")
				fmt.Printf("─────────────────────────────────────\n")
				return nil
			case pb.TaskStatusResponse_FAILED:
				fmt.Printf("\r\033[K🔴 Task failed - log following ended\n")
				fmt.Printf("─────────────────────────────────────\n")
				return nil
			case pb.TaskStatusResponse_RUNNING:
				// Continue following
			default:
				// For PENDING or unknown states, assume completed
				fmt.Printf("\r\033[K🏁 Task finished - log following ended\n")
				fmt.Printf("─────────────────────────────────────\n")
				return nil
			}

			// Sleep before next check
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// processLogLineForFollow processes a single log line for follow mode with minimal output
func processLogLineForFollow(line string, lastActivity *time.Time, isWorking *bool, workingState *WorkingState) {
	// Use regex to parse log line format: [timestamp] [type] content
	logEntryRegex := regexp.MustCompile(`^\[([^\]]+)\] \[([^\]]+)\] (.*)$`)
	matches := logEntryRegex.FindStringSubmatch(line)

	if len(matches) != 4 {
		// Not a structured log entry, print as-is only if it's important
		if strings.Contains(line, "ERROR") || strings.Contains(line, "FATAL") {
			fmt.Println(line)
		}
		return
	}

	timestamp := matches[1]
	logType := matches[2]
	content := matches[3]

	// Parse timestamp for display
	var currentTime string
	if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
		currentTime = t.Format("15:04:05")
	} else {
		currentTime = timestamp
	}

	// Update last activity time
	*lastActivity = time.Now()

	switch logType {
	case "STDOUT":
		formatClaudeJSONForFollow(content, currentTime, isWorking, workingState)
	case "STDERR":
		// Show errors and warnings
		fmt.Printf("\r\033[K[%s] ⚠️  %s\n", currentTime, content)
	case "ERROR":
		fmt.Printf("\r\033[K[%s] ❌ %s\n", currentTime, content)
	default:
		// Skip other log types to reduce noise
	}
}

// formatClaudeJSONForFollow formats Claude messages for follow mode with minimal output
func formatClaudeJSONForFollow(jsonLine string, timestamp string, isWorking *bool, workingState *WorkingState) {
	// Skip empty lines
	if strings.TrimSpace(jsonLine) == "" {
		return
	}

	// Try to parse as Claude message
	var claudeMsg map[string]interface{}
	if err := json.Unmarshal([]byte(jsonLine), &claudeMsg); err != nil {
		// If it's not JSON and looks important, show it
		if len(jsonLine) > 10 && !strings.Contains(jsonLine, "thinking") {
			fmt.Printf("\r\033[K[%s] 💬 %s\n", timestamp, jsonLine)
			showWorkingIndicator(*isWorking)
		}
		return
	}

	msgType, _ := claudeMsg["type"].(string)
	switch msgType {
	case "user":
		// Don't show "Task Started" - just update working state
		*isWorking = true
		*workingState = WorkingStateThinking

	case "assistant":
		if message, ok := claudeMsg["message"].(map[string]interface{}); ok {
			if content, ok := message["content"].([]interface{}); ok {
				for _, item := range content {
					if contentItem, ok := item.(map[string]interface{}); ok {
						contentType, _ := contentItem["type"].(string)
						switch contentType {
						case "text":
							if text, ok := contentItem["text"].(string); ok && text != "" {
								// Show actual Claude responses (thinking/planning/outcomes)
								fmt.Printf("\r\033[K[%s] 💭 %s\n", timestamp, text)
								*workingState = WorkingStateThinking
							}
						case "tool_use":
							// Update working state for tool usage
							*isWorking = true
							*workingState = WorkingStateTooling
						}
					}
				}
			}
		}

	case "stream_event":
		if event, ok := claudeMsg["event"].(map[string]interface{}); ok {
			eventType, _ := event["type"].(string)
			switch eventType {
			case "message_start":
				*isWorking = true
				*workingState = WorkingStateProcessing
			case "message_stop":
				// Don't show "Claude Finished" - just update state
				*isWorking = false
				*workingState = WorkingStateIdle
			case "content_block_start":
				*isWorking = true
				*workingState = WorkingStateThinking
			case "content_block_stop":
				// Keep working indicator until message stops
			}
		}

	case "system":
		// Don't show system messages - just update state
		*isWorking = true
		*workingState = WorkingStateProcessing

	case "result":
		// Show only meaningful result information
		if message, ok := claudeMsg["message"].(map[string]interface{}); ok {
			if durationMs, ok := message["duration_ms"].(float64); ok {
				duration := time.Duration(durationMs) * time.Millisecond
				fmt.Printf("\r\033[K[%s] ✅ Task completed in %s\n", timestamp, duration.String())
			} else {
				fmt.Printf("\r\033[K[%s] ✅ Task completed\n", timestamp)
			}
		}
		*isWorking = false
		*workingState = WorkingStateIdle
	}
}

// showWorkingIndicator shows a working indicator if task is active
func showWorkingIndicator(isWorking bool) {
	if isWorking {
		fmt.Print("🔄 Working...")
	}
}

// WorkingState represents different types of activity
type WorkingState int

const (
	WorkingStateIdle WorkingState = iota
	WorkingStateProcessing
	WorkingStateTooling
	WorkingStateThinking
)

// startSpinner starts a background spinner animation with different indicators
func startSpinner(isWorking *bool, workingState *WorkingState, done chan struct{}) {
	spinners := map[WorkingState][]string{
		WorkingStateProcessing: {"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		WorkingStateTooling:    {"🔧", "⚙️", "🛠️", "🔨"},
		WorkingStateThinking:   {"🤔", "💭", "🧠", "💡"},
	}

	messages := map[WorkingState]string{
		WorkingStateProcessing: "Working...",
		WorkingStateTooling:    "Using tools...",
		WorkingStateThinking:   "Thinking...",
	}

	i := 0
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if *isWorking {
				currentSpinner := spinners[*workingState]
				currentMessage := messages[*workingState]

				if len(currentSpinner) > 0 {
					fmt.Printf("\r%s %s", currentSpinner[i%len(currentSpinner)], currentMessage)
					i++
				}
			}
		}
	}
}

// formatLogOutput formats raw log output into human-readable format
func formatLogOutput(rawLog string) error {
	lines := strings.Split(rawLog, "\n")
	logEntryRegex := regexp.MustCompile(`^\[([^\]]+)\] \[([^\]]+)\] (.*)$`)

	var taskPrompt string
	var currentTime string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Match log entry format: [timestamp] [type] content
		matches := logEntryRegex.FindStringSubmatch(line)
		if len(matches) != 4 {
			// Not a structured log entry, print as-is
			fmt.Println(line)
			continue
		}

		timestamp := matches[1]
		logType := matches[2]
		content := matches[3]

		// Parse timestamp for display
		if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
			currentTime = t.Format("15:04:05")
		} else {
			currentTime = timestamp
		}

		switch logType {
		case "STDOUT":
			formatClaudeJSON(content, taskPrompt, currentTime)
		case "STDERR":
			fmt.Printf("[%s] ⚠️  %s\n", currentTime, content)
		case "ERROR":
			fmt.Printf("[%s] ❌ %s\n", currentTime, content)
		case "STATUS":
			fmt.Printf("[%s] ℹ️  %s\n", currentTime, content)
		default:
			fmt.Printf("[%s] [%s] %s\n", currentTime, logType, content)
		}
	}

	return nil
}

// formatClaudeJSON formats a Claude JSON output line into human-readable format
func formatClaudeJSON(jsonLine string, taskPrompt string, timestamp string) {
	// Skip empty lines
	if strings.TrimSpace(jsonLine) == "" {
		return
	}

	// Try to parse as Claude message
	var claudeMsg map[string]interface{}
	if err := json.Unmarshal([]byte(jsonLine), &claudeMsg); err != nil {
		// If it's not JSON, treat as plain text output
		fmt.Printf("[%s] 💬 %s\n", timestamp, jsonLine)
		return
	}

	msgType, _ := claudeMsg["type"].(string)
	switch msgType {
	case "user":
		fmt.Printf("[%s] 👤 **Task Started**\n", timestamp)
		if taskPrompt != "" {
			fmt.Printf("[%s] 📝 **Prompt**: %s\n", timestamp, taskPrompt)
		}
	case "assistant":
		if message, ok := claudeMsg["message"].(map[string]interface{}); ok {
			if content, ok := message["content"].([]interface{}); ok {
				for _, item := range content {
					if contentItem, ok := item.(map[string]interface{}); ok {
						contentType, _ := contentItem["type"].(string)
						switch contentType {
						case "text":
							if text, ok := contentItem["text"].(string); ok && text != "" {
								fmt.Printf("[%s] 🤖 %s\n", timestamp, text)
							}
						case "tool_use":
							if name, ok := contentItem["name"].(string); ok {
								fmt.Printf("[%s] 🛠️  **Using %s**\n", timestamp, name)
							}
						}
					}
				}
			}
		}
	case "stream_event":
		if event, ok := claudeMsg["event"].(map[string]interface{}); ok {
			eventType, _ := event["type"].(string)
			switch eventType {
			case "content_block_start":
				fmt.Printf("[%s] ⏳ Claude is thinking...\n", timestamp)
			case "content_block_stop":
				fmt.Printf("\n[%s] ✅ Response complete\n", timestamp)
			case "message_start":
				fmt.Printf("[%s] 🚀 **Claude Started Working**\n", timestamp)
			case "message_stop":
				fmt.Printf("[%s] 🏁 **Claude Finished**\n", timestamp)
			}
		}
	case "system":
		fmt.Printf("[%s] ⚙️  **System Initialized**\n", timestamp)
		if message, ok := claudeMsg["message"].(map[string]interface{}); ok {
			if workDir, ok := message["current_working_dir"].(string); ok && workDir != "" {
				fmt.Printf("[%s] 📂 Working Directory: %s\n", timestamp, workDir)
			}
			if model, ok := message["model"].(string); ok && model != "" {
				fmt.Printf("[%s] 🧠 Model: %s\n", timestamp, model)
			}
		}
	case "result":
		fmt.Printf("[%s] 📊 **Task Summary**\n", timestamp)
		if message, ok := claudeMsg["message"].(map[string]interface{}); ok {
			if durationMs, ok := message["duration_ms"].(float64); ok {
				duration := time.Duration(durationMs) * time.Millisecond
				fmt.Printf("[%s] ⏱️  Duration: %s\n", timestamp, duration.String())
			}
			if inputTokens, ok := message["input_tokens"].(float64); ok {
				if outputTokens, ok := message["output_tokens"].(float64); ok {
					fmt.Printf("[%s] 🔤 Tokens: %.0f input, %.0f output\n", timestamp, inputTokens, outputTokens)
				}
			}
			if totalCost, ok := message["total_cost"].(float64); ok && totalCost > 0 {
				fmt.Printf("[%s] 💰 Cost: $%.4f\n", timestamp, totalCost)
			}
		}
	default:
		fmt.Printf("[%s] ❓ %s\n", timestamp, jsonLine)
	}
}

// runClaudeOnGitHubIssue starts Claude working on the GitHub issue from the sandbox's task data
func runClaudeOnGitHubIssue(sandboxName string) error {
	utils.DebugPrintf("Starting Claude on GitHub issue for sandbox %s\n", sandboxName)

	// Read task data from the sandbox file
	taskData, err := readTaskDataFromSandbox(sandboxName)
	if err != nil {
		return fmt.Errorf("failed to read GitHub issue data: %w", err)
	}

	if taskData.GitHubIssue == nil {
		return fmt.Errorf("no GitHub issue found in task data")
	}

	// Construct the comprehensive prompt for the GitHub issue
	var taskPrompt strings.Builder

	taskPrompt.WriteString("I need help with this GitHub issue:\n\n")
	taskPrompt.WriteString(fmt.Sprintf("**Issue**: %s\n", taskData.GitHubIssue.Title))
	taskPrompt.WriteString(fmt.Sprintf("**Repository**: %s/%s\n", taskData.GitHubIssue.Owner, taskData.GitHubIssue.Repo))
	taskPrompt.WriteString(fmt.Sprintf("**Issue #%d**: %s\n\n", taskData.GitHubIssue.Number, taskData.GitHubIssue.URL))

	if taskData.GitHubIssue.Body != "" {
		taskPrompt.WriteString(fmt.Sprintf("**Description**:\n%s\n\n", taskData.GitHubIssue.Body))
	}

	if taskData.AdditionalText != "" {
		taskPrompt.WriteString(fmt.Sprintf("**Additional context**:\n%s\n\n", taskData.AdditionalText))
	}

	taskPrompt.WriteString("Please help me work on this issue. Start by analyzing the codebase and understanding the problem, then propose a solution. Take your time to examine the relevant files and provide a comprehensive analysis.")

	// Execute Claude with the constructed prompt
	workDir, err := getWorkDirFromProvider(sandboxName)
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	return runClaudeWithPrompt(taskPrompt.String(), workDir, sandboxName, "")
}

// readTaskDataFromSandbox reads the GitHub issue task data from the sandbox
func readTaskDataFromSandbox(sandboxName string) (*TaskData, error) {
	utils.DebugPrintf("Reading task data from sandbox %s\n", sandboxName)

	workDir, err := getWorkDirFromProvider(sandboxName)
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Read the task data file from the sandbox
	taskDataBytes, err := executeSandboxCommand(sandboxName, []string{"cat", fmt.Sprintf("%s/.claude_task.json", workDir)})
	if err != nil {
		return nil, fmt.Errorf("failed to read task data file (this sandbox may not have GitHub issue data): %w", err)
	}

	// Parse the JSON data
	var taskData TaskData
	if err := json.Unmarshal(taskDataBytes, &taskData); err != nil {
		return nil, fmt.Errorf("failed to parse task data: %w", err)
	}

	return &taskData, nil
}

// getTaskStateEmoji returns an emoji for the task state
func getTaskStateEmoji(state pb.TaskStatusResponse_TaskState) string {
	switch state {
	case pb.TaskStatusResponse_PENDING:
		return "⏳"
	case pb.TaskStatusResponse_RUNNING:
		return "🟡"
	case pb.TaskStatusResponse_COMPLETED:
		return "🟢"
	case pb.TaskStatusResponse_FAILED:
		return "🔴"
	default:
		return "❓"
	}
}

// getTaskStateText returns human-readable text for the task state
func getTaskStateText(state pb.TaskStatusResponse_TaskState) string {
	switch state {
	case pb.TaskStatusResponse_PENDING:
		return "Pending"
	case pb.TaskStatusResponse_RUNNING:
		return "Running"
	case pb.TaskStatusResponse_COMPLETED:
		return "Completed"
	case pb.TaskStatusResponse_FAILED:
		return "Failed"
	default:
		return "Unknown"
	}
}

// truncatePrompt truncates a prompt to a specified length
func truncatePrompt(prompt string, maxLength int) string {
	if len(prompt) <= maxLength {
		return prompt
	}
	return prompt[:maxLength-3] + "..."
}

// showTaskDetails displays detailed information about a specific task
func showTaskDetails(sandboxName, taskID string, taskStatus *pb.TaskStatusResponse) error {
	stateEmoji := getTaskStateEmoji(taskStatus.State)
	stateText := getTaskStateText(taskStatus.State)

	fmt.Printf("📋 Task Details for '%s' in sandbox '%s':\n\n", taskID, sandboxName)
	fmt.Printf("🆔 Task ID: %s\n", taskID)
	fmt.Printf("%s  State: %s\n", stateEmoji, stateText)

	if taskStatus.Prompt != "" {
		fmt.Printf("📝 Prompt: %s\n", taskStatus.Prompt)
	}

	if taskStatus.WorkingDirectory != "" {
		fmt.Printf("📂 Working Directory: %s\n", taskStatus.WorkingDirectory)
	}

	fmt.Printf("📅 Started: %s\n", time.Unix(taskStatus.StartedAt, 0).Format("2006-01-02 15:04:05"))

	if taskStatus.FinishedAt > 0 {
		endTime := time.Unix(taskStatus.FinishedAt, 0)
		startTime := time.Unix(taskStatus.StartedAt, 0)
		duration := endTime.Sub(startTime)
		fmt.Printf("🏁 Finished: %s\n", endTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("⏱️  Duration: %s\n", duration.String())
	}

	if taskStatus.ExitCode != 0 {
		fmt.Printf("🔢 Exit Code: %d\n", taskStatus.ExitCode)
	}

	if taskStatus.Error != "" {
		fmt.Printf("❌ Error: %s\n", taskStatus.Error)
	}

	if taskStatus.Message != "" {
		fmt.Printf("💬 Message: %s\n", taskStatus.Message)
	}

	fmt.Println()
	return nil
}

// showTaskDetailsFromLogFile displays detailed information about a specific task from log files
func showTaskDetailsFromLogFile(sandboxName, taskID string) error {
	sandboxLogDir := "/home/daytona/.dispense/logs"
	logFile := fmt.Sprintf("%s.log", taskID)
	logPath := filepath.Join(sandboxLogDir, logFile)

	// Check if log file exists
	_, err := executeSandboxCommand(sandboxName, []string{"test", "-f", logPath})
	if err != nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	fmt.Printf("📋 Task Details for '%s' in sandbox '%s':\n\n", taskID, sandboxName)
	fmt.Printf("🆔 Task ID: %s\n", taskID)

	// Get file info
	statOutput, err := executeSandboxCommand(sandboxName, []string{"stat", "-c", "%Y %s", logPath})
	if err == nil {
		fields := strings.Fields(string(statOutput))
		if len(fields) >= 2 {
			timestamp := fields[0]
			size := fields[1]
			if ts, err := strconv.ParseInt(timestamp, 10, 64); err == nil {
				modTime := time.Unix(ts, 0).Format("2006-01-02 15:04:05")
				fmt.Printf("📅 Last Modified: %s\n", modTime)
				fmt.Printf("📊 Log Size: %s bytes\n", size)
			}
		}
	}

	fmt.Printf("\n📄 Full Task Log:\n")
	fmt.Printf("─────────────────────────────────────\n")

	// Read and display the log file content
	output, err := executeSandboxCommand(sandboxName, []string{"cat", logPath})
	if err != nil {
		return fmt.Errorf("failed to read log file: %w", err)
	}

	fmt.Print(string(output))
	fmt.Printf("─────────────────────────────────────\n")
	return nil
}

// listTasksFromLogFiles lists all tasks using log files fallback
func listTasksFromLogFiles(sandboxName string) error {
	sandboxLogDir := "/home/daytona/.dispense/logs"
	output, err := executeSandboxCommand(sandboxName, []string{"ls", "-la", sandboxLogDir})
	if err != nil {
		if strings.Contains(err.Error(), "exit status 2") {
			fmt.Printf("📋 No Claude tasks found for sandbox '%s'\n", sandboxName)
			fmt.Printf("💡 Log directory doesn't exist in sandbox: %s\n", sandboxLogDir)
			return nil
		}
		return fmt.Errorf("failed to list log files in sandbox: %w", err)
	}

	if strings.Contains(string(output), "No such file or directory") {
		fmt.Printf("📋 No Claude tasks found for sandbox '%s'\n", sandboxName)
		fmt.Printf("💡 Log directory doesn't exist in sandbox: %s\n", sandboxLogDir)
		return nil
	}

	// Parse the output to count log files
	lines := strings.Split(string(output), "\n")
	var logFiles []string

	for _, line := range lines {
		if strings.Contains(line, "claude_") && strings.HasSuffix(line, ".log") {
			fields := strings.Fields(line)
			if len(fields) >= 9 {
				filename := fields[8]
				logFiles = append(logFiles, filename)
			}
		}
	}

	if len(logFiles) == 0 {
		fmt.Printf("📋 No Claude tasks found for sandbox '%s'\n", sandboxName)
		return nil
	}

	fmt.Printf("📋 Claude Tasks in sandbox '%s' (%d found):\n\n", sandboxName, len(logFiles))

	for _, filename := range logFiles {
		// Extract task ID from filename (remove .log extension)
		taskIDFromFile := filename
		if len(taskIDFromFile) > 4 && taskIDFromFile[len(taskIDFromFile)-4:] == ".log" {
			taskIDFromFile = taskIDFromFile[:len(taskIDFromFile)-4]
		}

		// Get file info from sandbox
		statOutput, err := executeSandboxCommand(sandboxName, []string{"stat", "-c", "%Y %s", filepath.Join(sandboxLogDir, filename)})
		if err == nil {
			fields := strings.Fields(string(statOutput))
			if len(fields) >= 2 {
				timestamp := fields[0]
				size := fields[1]

				if ts, err := strconv.ParseInt(timestamp, 10, 64); err == nil {
					modTime := time.Unix(ts, 0).Format("2006-01-02 15:04:05")
					fmt.Printf("  🔹 %s\n", taskIDFromFile)
					fmt.Printf("     📅 %s\n", modTime)
					fmt.Printf("     📊 %s bytes\n", size)
					fmt.Println()
				}
			}
		} else {
			// Fallback without detailed info
			fmt.Printf("  🔹 %s\n", taskIDFromFile)
			fmt.Println()
		}
	}

	return nil
}

// getWorkDirFromProvider gets the working directory from the appropriate provider based on sandbox type
func getWorkDirFromProvider(sandboxName string) (string, error) {
	// Find the sandbox to determine its type
	sandboxInfo, err := findSandboxByName(sandboxName)
	if err != nil {
		return "", fmt.Errorf("failed to find sandbox: %w", err)
	}

	// Get the appropriate provider based on sandbox type
	if sandboxInfo.Type == sandbox.TypeRemote {
		remoteProvider, err := remote.NewProvider()
		if err != nil {
			return "", fmt.Errorf("failed to create remote provider: %w", err)
		}
		return remoteProvider.GetWorkDir(sandboxInfo)
	} else {
		// For local sandboxes, we can return the working directory directly
		// without needing to create a new provider instance (which can cause database timeouts)
		// since local sandboxes always use /workspace
		return "/workspace", nil
	}
}

func init() {
	// Add format flag specifically to claude command
	claudeCmd.Flags().String("format", "human", "Log output format: raw or human")
	claudeCmd.Flags().BoolP("follow", "f", false, "Follow log output in real-time until task completes")
}
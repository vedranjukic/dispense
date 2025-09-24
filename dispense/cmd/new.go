package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cli/pkg/sandbox"
	"cli/pkg/sandbox/local"
	"cli/pkg/sandbox/remote"
	"cli/pkg/utils"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"

	pb "cli/proto"
)

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new sandbox with a branch name",
	Long:  `Create a new sandbox using either local Docker containers (default) or remote Daytona API (--remote).`,
	Run: func(cmd *cobra.Command, args []string) {
		utils.DebugPrintf("Starting new command\n")

		var err error

		// Get command flags
		isRemote, _ := cmd.Flags().GetBool("remote")
		snapshot, _ := cmd.Flags().GetString("snapshot")
		target, _ := cmd.Flags().GetString("target")
		cpu, _ := cmd.Flags().GetInt32("cpu")
		memory, _ := cmd.Flags().GetInt32("memory")
		disk, _ := cmd.Flags().GetInt32("disk")
		autoStop, _ := cmd.Flags().GetInt32("auto-stop")
		force, _ := cmd.Flags().GetBool("force")
		name, _ := cmd.Flags().GetString("name")
		skipCopy, _ := cmd.Flags().GetBool("skip-copy")
		skipDaemon, _ := cmd.Flags().GetBool("skip-daemon")
		group, _ := cmd.Flags().GetString("group")

		// Get branch name - either from flag or prompt
		var branchName string
		if name != "" {
			branchName = name
			fmt.Printf("Using provided branch name: %s\n", branchName)
		} else {
			branchName, err = promptForBranchName()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting branch name: %s\n", err)
				os.Exit(1)
			}
		}

		// Prompt for task description
		taskDescription, err := promptForTask()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting task description: %s\n", err)
			os.Exit(1)
		}

		// Parse task for GitHub issue if provided
		taskData, err := parseTaskData(taskDescription)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing task: %s\n", err)
			os.Exit(1)
		}

		// Auto-skip file copy only for remote sandboxes when GitHub issue is provided
		if !skipCopy && taskData != nil && taskData.GitHubIssue != nil && isRemote {
			skipCopy = true
			fmt.Printf("🔗 GitHub issue detected for remote sandbox - automatically skipping file copy\n")
			fmt.Printf("   Working on: %s\n", taskData.GitHubIssue.URL)
		} else if taskData != nil && taskData.GitHubIssue != nil && !isRemote {
			fmt.Printf("🔗 GitHub issue detected for local sandbox - creating empty workspace\n")
			fmt.Printf("   Working on: %s\n", taskData.GitHubIssue.URL)
		}

		// The source directory is the directory from which the files will be copied
		// When task source is a GitHub issue, the source directory is empty
		sourceDirectory := ""

		if !skipCopy {
			// Get the source directory
			sourceDirectory, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting current directory: %s\n", err)
				os.Exit(1)
			}

			// Validate git repository (unless force flag is used or GitHub issue provided)
			if !force && !isGitRepository(sourceDirectory) {
				if taskData != nil && taskData.GitHubIssue != nil {
					fmt.Printf("ℹ️  Note: Working on GitHub issue - local git repository not required\n")
				} else {
					fmt.Printf("⚠️  Warning: Current directory does not appear to be a git repository\n")
					if !confirmContinue() {
						fmt.Println("Sandbox creation cancelled.")
						os.Exit(0)
					}
				}
			}
		}

		// Determine sandbox type
		var sandboxType sandbox.SandboxType
		var provider sandbox.Provider

		if isRemote {
			sandboxType = sandbox.TypeRemote
			fmt.Println("🌐 Creating remote sandbox using Daytona API...")
			provider, err = remote.NewProvider()
		} else {
			// Default to local Docker sandboxes
			sandboxType = sandbox.TypeLocal
			fmt.Println("🐳 Creating local sandbox using Docker containers...")
			fmt.Println("   Use --remote flag to create a remote Daytona sandbox instead")
			provider, err = local.NewProvider()
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating %s provider: %s\n", sandboxType, err)
			os.Exit(1)
		}

		// Serialize task data to JSON for storage
		var taskDataJSON string
		if taskData != nil {
			taskDataBytes, err := json.Marshal(taskData)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to serialize task data: %s\n", err)
			} else {
				taskDataJSON = string(taskDataBytes)
			}
		}

		// Create sandbox options
		opts := &sandbox.CreateOptions{
			Name:        branchName,
			Snapshot:    snapshot,
			Target:      target,
			CPU:         cpu,
			Memory:      memory,
			Disk:        disk,
			AutoStop:    autoStop,
			Force:       force,
			SkipCopy:    skipCopy,
			SkipDaemon:  skipDaemon,
			BranchName:  branchName,
			SourceDir:   sourceDirectory,
			TaskData:    taskDataJSON,
			GitHubIssue: taskData != nil && taskData.GitHubIssue != nil,
			Group:       group,
		}

		// Create sandbox
		fmt.Printf("Creating %s sandbox...\n", sandboxType)
		sandboxInfo, err := provider.Create(opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating sandbox: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Sandbox created successfully!\n")
		fmt.Printf("Sandbox ID: %s\n", sandboxInfo.ID)
		fmt.Printf("Name: %s\n", sandboxInfo.Name)
		fmt.Printf("Type: %s\n", sandboxInfo.Type)
		fmt.Printf("State: %s\n", sandboxInfo.State)

		// Handle file copying or GitHub repo cloning
		if taskData != nil && taskData.GitHubIssue != nil {
			// Clone GitHub repository instead of copying files
			fmt.Printf("📥 Cloning GitHub repository %s/%s...\n", taskData.GitHubIssue.Owner, taskData.GitHubIssue.Repo)
			err = provider.CloneGitHubRepo(sandboxInfo, taskData.GitHubIssue.Owner, taskData.GitHubIssue.Repo, branchName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Failed to clone repository: %s\n", err)
				os.Exit(1)
			} else {
				fmt.Printf("✅ Repository cloned and branch '%s' created!\n", branchName)
			}

			// Save task data to a file in the sandbox for later use by claude commands
			err = saveTaskDataToSandbox(sandboxInfo, taskData)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Could not save task data: %s\n", err)
			}
		} else if !skipCopy {
			// Normal file copy logic
			if sandboxInfo.Type == sandbox.TypeRemote {
				fmt.Println("📁 Copying files to sandbox...")
				err = provider.CopyFiles(sandboxInfo, sourceDirectory)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Could not copy files: %s\n", err)
					printManualCopyInstructions(sandboxInfo)
				} else {
					fmt.Println("✅ Files copied successfully!")
				}
			} else {
				fmt.Println("📁 Files available via volume mount in container")
			}
		} else {
			fmt.Println("⏭️  Skipping file copy (--skip-copy flag used)")
		}

		// Install daemon unless skipped
		if !skipDaemon {
			fmt.Println("🔧 Installing embedded daemon...")
			err = provider.InstallDaemon(sandboxInfo)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Could not install daemon: %s\n", err)
				fmt.Fprintf(os.Stderr, "You can manually install the daemon later using 'cli daemon' commands\n")
			} else {
				fmt.Println("✅ Daemon installed and started successfully as 'dispensed'!")
			}
		} else {
			fmt.Println("⏭️  Skipping daemon installation (--skip-daemon flag used)")
		}

		// Get Claude API key for background tasks - check saved config first
		var claudeApiKey string
		if savedApiKey, err := loadSandboxClaudeConfig(); err == nil && savedApiKey != "" {
			fmt.Printf("Using saved Anthropic API key (length: %d, starts with: %.10s...).\n", len(savedApiKey), savedApiKey)
			claudeApiKey = savedApiKey
		} else {
			fmt.Printf("Enter your Anthropic API key for Claude: ")
			reader := bufio.NewReader(os.Stdin)
			inputApiKey, err := reader.ReadString('\n')
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to read API key: %s\n", err)
				claudeApiKey = ""
			} else {
				claudeApiKey = strings.TrimSpace(inputApiKey)
				// Save the API key for future use
				if claudeApiKey != "" {
					if err := saveSandboxClaudeConfig(claudeApiKey); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: Failed to save API key: %s\n", err)
					} else {
						fmt.Printf("✅ Anthropic API key saved for future use!\n")
					}
				}
			}
		}

		// Verify Claude daemon and show issue-specific instructions
		if !skipDaemon {
			fmt.Println("🧪 Verifying Claude daemon connectivity...")

			// Show container name for debugging
			if containerName, ok := sandboxInfo.Metadata["container_name"].(string); ok {
				fmt.Printf("   • Debug container: %s\n", containerName)
				fmt.Printf("   • Debug command: docker exec -it %s /bin/bash\n", containerName)
			}

			err = verifyClaudeDaemonReady(sandboxInfo)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Claude daemon not ready: %s\n", err)

				// Even if daemon is not immediately ready, start the retry process for GitHub issues
				if taskData != nil && taskData.GitHubIssue != nil {
					fmt.Printf("🤖 Waiting for daemon to be ready and starting Claude on GitHub issue #%d...\n", taskData.GitHubIssue.Number)

					// Create the GitHub issue prompt
					prompt := createGitHubIssuePrompt(taskData)

					// Wait for daemon to be ready and then start Claude in background
					if err := waitForDaemonAndStartClaude(sandboxInfo, prompt, claudeApiKey); err != nil {
						fmt.Fprintf(os.Stderr, "❌ Failed to start Claude on GitHub issue: %s\n", err)
					} else {
						fmt.Printf("✅ Claude has started working on GitHub issue #%d in background!\n", taskData.GitHubIssue.Number)
					}

					fmt.Printf("📋 Claude is working in background. Monitor progress with:\n")
					fmt.Printf("   claude %s logs\n", sandboxInfo.Name)
				}
			} else {
				if taskData != nil && taskData.GitHubIssue != nil {
					// Automatically start Claude working on the GitHub issue
					fmt.Printf("✅ Claude is ready to work on the issue!\n")
					fmt.Printf("🤖 Starting Claude to work on GitHub issue #%d...\n", taskData.GitHubIssue.Number)

					// Create the GitHub issue prompt
					prompt := createGitHubIssuePrompt(taskData)

					// Start Claude in background since daemon is already ready
					if err := startClaudeCommandInBackground(sandboxInfo, prompt, claudeApiKey); err != nil {
						fmt.Fprintf(os.Stderr, "❌ Failed to start Claude on GitHub issue: %s\n", err)
					} else {
						fmt.Printf("✅ Claude has started working on GitHub issue #%d in background!\n", taskData.GitHubIssue.Number)
					}

					fmt.Printf("📋 Claude is working in background. Monitor progress with: claude %s logs\n", sandboxInfo.Name)
				} else {
					fmt.Printf("✅ Claude daemon is ready! Test with: claude %s run \"your prompt\"\n", sandboxInfo.Name)
				}
			}
		}

		// Show connection information
		fmt.Printf("\n🎉 Sandbox setup complete!\n")
		fmt.Printf("Connect with: %s\n", sandboxInfo.ShellCommand)

		// Show additional information based on type
		if sandboxType == sandbox.TypeLocal {
			fmt.Println("\n📋 Local sandbox information:")
			fmt.Printf("  • Container access: %s\n", sandboxInfo.ShellCommand)
			fmt.Printf("  • Type: Docker container\n")
		} else {
			fmt.Println("\n📋 Remote sandbox information:")
			fmt.Printf("  • SSH access: %s\n", sandboxInfo.ShellCommand)
			fmt.Printf("  • Type: Daytona remote sandbox\n")
		}
	},
}

// Helper functions

func promptForBranchName() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	defaultBranchName := generateAutoBranchName()

	fmt.Printf("Enter branch name (press Enter for auto-generated: %s): ", defaultBranchName)
	branchName, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	branchName = strings.TrimSpace(branchName)
	if branchName == "" {
		fmt.Printf("Using auto-generated branch name: %s\n", defaultBranchName)
		return defaultBranchName, nil
	}

	return branchName, nil
}

func generateAutoBranchName() string {
	adjectives := []string{
		"swift", "bright", "clever", "bold", "quick", "sharp", "smart", "fast",
		"cool", "warm", "fresh", "new", "shiny", "smooth", "clean", "happy",
	}

	nouns := []string{
		"feature", "update", "fix", "patch", "enhancement", "improvement",
		"refactor", "optimization", "addition", "change", "modification",
	}

	seed := time.Now().UnixNano()
	adjIndex := int(seed) % len(adjectives)
	nounIndex := int(seed/2) % len(nouns)

	return fmt.Sprintf("%s-%s", adjectives[adjIndex], nouns[nounIndex])
}

func isGitRepository(path string) bool {
	_, err := os.Stat(path + "/.git")
	return err == nil
}

func confirmContinue() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Continue anyway? (y/N): ")
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func printManualCopyInstructions(sandboxInfo *sandbox.SandboxInfo) {
	fmt.Fprintf(os.Stderr, "\n📁 Manual file copy instructions:\n")

	if sandboxInfo.Type == sandbox.TypeLocal {
		fmt.Fprintf(os.Stderr, "For local Docker sandbox:\n")
		fmt.Fprintf(os.Stderr, "1. Copy files: docker cp . %s:/workspace\n", sandboxInfo.ID)
		fmt.Fprintf(os.Stderr, "2. Connect: %s\n", sandboxInfo.ShellCommand)
	} else {
		fmt.Fprintf(os.Stderr, "For remote sandbox:\n")
		fmt.Fprintf(os.Stderr, "1. Connect to your sandbox: %s\n", sandboxInfo.ShellCommand)
		fmt.Fprintf(os.Stderr, "2. Create workspace directory: mkdir -p /home/daytona/workspace\n")
		fmt.Fprintf(os.Stderr, "3. Copy files using tar: tar -czf - . | ssh %s 'cd /home/daytona/workspace && tar -xzf -'\n", sandboxInfo.Name)
	}
}

// getClaudeApiKey gets the Claude API key from various sources
func getClaudeApiKey() (string, error) {
	// Check if we have a saved sandbox configuration
	if savedApiKey, err := loadSandboxClaudeConfig(); err == nil && savedApiKey != "" {
		fmt.Printf("Found existing sandbox Claude API key configuration.\n")
		fmt.Printf("Would you like to use the saved API key? (Y/n): ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read user input: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response == "" || response == "y" || response == "yes" {
			return savedApiKey, nil
		}
	}


	// Ask for sandbox-specific API key
	fmt.Printf("Enter Claude API key for sandbox (will be used only for sandboxes): ")
	reader := bufio.NewReader(os.Stdin)
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read API key: %w", err)
	}

	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", fmt.Errorf("API key cannot be empty")
	}

	// Save the API key for future sandbox use
	if err := saveSandboxClaudeConfig(apiKey); err != nil {
		return "", fmt.Errorf("failed to save sandbox configuration: %w", err)
	}

	return apiKey, nil
}



// getSandboxClaudeConfigPath returns the path where sandbox Claude config is stored
func getSandboxClaudeConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".dispense", "claude")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config"), nil
}

// saveSandboxClaudeConfig saves the API key for sandbox use
func saveSandboxClaudeConfig(apiKey string) error {
	configPath, err := getSandboxClaudeConfigPath()
	if err != nil {
		return err
	}

	// Create a simple config file with just the API key
	if err := os.WriteFile(configPath, []byte(apiKey), 0600); err != nil {
		return err
	}

	utils.DebugPrintf("Saved sandbox Claude config to: %s\n", configPath)
	return nil
}

// loadSandboxClaudeConfig loads the saved API key for sandbox use
func loadSandboxClaudeConfig() (string, error) {
	configPath, err := getSandboxClaudeConfigPath()
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}

	// Just return the trimmed content as the API key
	apiKey := strings.TrimSpace(string(content))
	if apiKey == "" {
		return "", fmt.Errorf("API key not found in config")
	}
	return apiKey, nil
}

// createGitHubIssuePrompt creates a comprehensive prompt for working on a GitHub issue
func createGitHubIssuePrompt(taskData *TaskData) string {
	if taskData == nil || taskData.GitHubIssue == nil {
		return ""
	}

	var taskPrompt strings.Builder

	taskPrompt.WriteString(fmt.Sprintf("I need help with this GitHub issue:\n\n"))
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

	return taskPrompt.String()
}

// waitForDaemonAndStartClaude waits for daemon to be ready then starts Claude with the given prompt in background
func waitForDaemonAndStartClaude(sandboxInfo *sandbox.SandboxInfo, prompt string, apiKey string) error {
	utils.DebugPrintf("Waiting for daemon and starting Claude with prompt for sandbox: %s\n", sandboxInfo.Name)

	// Wait up to 60 seconds for daemon to be ready
	maxRetries := 12 // 12 retries * 5 seconds = 60 seconds
	for i := 0; i < maxRetries; i++ {
		time.Sleep(5 * time.Second)

		// Check if daemon is ready
		if isDaemonReady(sandboxInfo) {
			utils.DebugPrintf("Daemon ready, starting Claude in background\n")
			return startClaudeCommandInBackground(sandboxInfo, prompt, apiKey)
		}

		utils.DebugPrintf("Daemon not ready yet, retry %d/%d\n", i+1, maxRetries)
	}

	return fmt.Errorf("daemon not ready after 60 seconds")
}

// TaskData represents parsed task information
type TaskData struct {
	OriginalText  string      `json:"original_text"`
	GitHubIssue   *GitHubIssue `json:"github_issue,omitempty"`
	AdditionalText string      `json:"additional_text,omitempty"`
}

// GitHubIssue represents GitHub issue information
type GitHubIssue struct {
	URL         string `json:"url"`
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	State       string `json:"state"`
	User        string `json:"user"`
	CreatedAt   string `json:"created_at"`
}

// promptForTask prompts the user to enter a task description
func promptForTask() (string, error) {
	fmt.Println("\n📋 Task Description")
	fmt.Println("Enter a description of what needs to be done in this sandbox.")
	fmt.Println("You can start with a GitHub issue URL (e.g., https://github.com/owner/repo/issues/123)")
	fmt.Println("followed by additional instructions.")
	fmt.Print("Task: ")

	reader := bufio.NewReader(os.Stdin)
	task, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read task: %w", err)
	}

	task = strings.TrimSpace(task)
	if task == "" {
		return "", fmt.Errorf("task description cannot be empty")
	}

	return task, nil
}

// parseTaskData parses task description and fetches GitHub issue if present
func parseTaskData(taskDescription string) (*TaskData, error) {
	utils.DebugPrintf("Parsing task description: %s\n", taskDescription)

	taskData := &TaskData{
		OriginalText: taskDescription,
	}

	// Check if task starts with GitHub issue URL
	githubIssueRegex := regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+)/issues/(\d+)`)
	matches := githubIssueRegex.FindStringSubmatch(taskDescription)

	if len(matches) == 4 {
		owner := matches[1]
		repo := matches[2]
		issueNumber := matches[3]

		utils.DebugPrintf("Found GitHub issue: %s/%s#%s\n", owner, repo, issueNumber)

		// Fetch GitHub issue
		issue, err := fetchGitHubIssue(owner, repo, issueNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch GitHub issue: %w", err)
		}

		taskData.GitHubIssue = issue

		// Extract additional text after the URL
		urlEnd := githubIssueRegex.FindStringIndex(taskDescription)
		if urlEnd != nil && len(taskDescription) > urlEnd[1] {
			additionalText := strings.TrimSpace(taskDescription[urlEnd[1]:])
			if additionalText != "" {
				taskData.AdditionalText = additionalText
			}
		}

		fmt.Printf("✅ Fetched GitHub issue: %s - %s\n", issue.Title, issue.State)
		if taskData.AdditionalText != "" {
			fmt.Printf("📝 Additional instructions: %s\n", taskData.AdditionalText)
		}
	}

	return taskData, nil
}

// fetchGitHubIssue fetches issue details from GitHub API
func fetchGitHubIssue(owner, repo, issueNumber string) (*GitHubIssue, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%s", owner, repo, issueNumber)
	utils.DebugPrintf("Fetching GitHub issue from: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d for issue %s/%s#%s", resp.StatusCode, owner, repo, issueNumber)
	}

	var issueResponse struct {
		URL       string `json:"html_url"`
		Title     string `json:"title"`
		Body      string `json:"body"`
		State     string `json:"state"`
		Number    int    `json:"number"`
		User      struct {
			Login string `json:"login"`
		} `json:"user"`
		CreatedAt string `json:"created_at"`
	}

	err = json.NewDecoder(resp.Body).Decode(&issueResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to decode GitHub response: %w", err)
	}

	issue := &GitHubIssue{
		URL:       issueResponse.URL,
		Owner:     owner,
		Repo:      repo,
		Number:    issueResponse.Number,
		Title:     issueResponse.Title,
		Body:      issueResponse.Body,
		State:     issueResponse.State,
		User:      issueResponse.User.Login,
		CreatedAt: issueResponse.CreatedAt,
	}

	return issue, nil
}


// verifyClaudeDaemonReady verifies that the Claude daemon is ready to accept commands
func verifyClaudeDaemonReady(sandboxInfo *sandbox.SandboxInfo) error {
	utils.DebugPrintf("Verifying Claude daemon readiness in sandbox %s\n", sandboxInfo.Name)

	// Wait a moment for daemon to fully start
	time.Sleep(2 * time.Second)

	var daemonAddr string
	var cleanup func()

	// Check if this is a remote sandbox and use appropriate connection method
	if sandboxInfo.Type == sandbox.TypeRemote {
		utils.DebugPrintf("Setting up SSH port forwarding for remote sandbox\n")

		// Get remote provider from metadata
		remoteProvider, err := remote.NewProvider()
		if err != nil {
			return fmt.Errorf("failed to create remote provider: %w", err)
		}

		// Get daemon connection with SSH port forwarding
		addr, cleanupFunc, err := remoteProvider.GetDaemonConnection(sandboxInfo)
		if err != nil {
			return fmt.Errorf("failed to setup daemon connection: %w", err)
		}

		daemonAddr = addr
		cleanup = cleanupFunc
		defer cleanup()
	} else {
		// Local sandbox - use existing logic
		containerIP, err := getSandboxIP(sandboxInfo.Name)
		if err != nil {
			return fmt.Errorf("failed to get sandbox IP: %w", err)
		}
		daemonAddr = fmt.Sprintf("%s:28080", containerIP)
	}

	utils.DebugPrintf("Testing connection to daemon at %s\n", daemonAddr)

	// Use context with timeout for connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(daemonAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to create grpc client: %w", err)
	}
	defer conn.Close()

	// Wait for connection to be ready - keep checking until ready or timeout
	for {
		state := conn.GetState()
		utils.DebugPrintf("Current connection state: %v\n", state)

		if state == connectivity.Ready || state == connectivity.Idle {
			break
		}
		if state == connectivity.TransientFailure || state == connectivity.Shutdown {
			return fmt.Errorf("daemon connection failed, state: %s", state.String())
		}

		// Wait for state to change or timeout
		if !conn.WaitForStateChange(ctx, state) {
			return fmt.Errorf("connection timeout while waiting for daemon (state: %s)", state.String())
		}
	}

	// Test connection state
	state := conn.GetState()
	utils.DebugPrintf("Connection state: %v\n", state)

	if state == connectivity.Ready || state == connectivity.Idle {
		utils.DebugPrintf("Claude daemon is ready\n")
		return nil
	}

	return fmt.Errorf("daemon connection state: %s", state.String())
}

// saveTaskDataToSandbox saves the task data to a file in the sandbox
func saveTaskDataToSandbox(sandboxInfo *sandbox.SandboxInfo, taskData *TaskData) error {
	// Convert task data to JSON
	taskDataBytes, err := json.Marshal(taskData)
	if err != nil {
		return fmt.Errorf("failed to marshal task data: %w", err)
	}

	// For local sandboxes, write to the host directory that's mounted in the container
	if sandboxInfo.Type == sandbox.TypeLocal {
		// Get container name from metadata
		containerID, ok := sandboxInfo.Metadata["container_id"].(string)
		if !ok {
			return fmt.Errorf("container ID not found in sandbox metadata")
		}

		// Write task data file to the container's /workspace
		taskDataFile := "/tmp/task_data.json"
		if err := os.WriteFile(taskDataFile, taskDataBytes, 0644); err != nil {
			return fmt.Errorf("failed to create temp task data file: %w", err)
		}

		// Copy the file to the container
		copyCmd := exec.Command("docker", "cp", taskDataFile, containerID+":/workspace/.claude_task.json")
		if err := copyCmd.Run(); err != nil {
			return fmt.Errorf("failed to copy task data to container: %w", err)
		}

		// Clean up temp file
		os.Remove(taskDataFile)

		utils.DebugPrintf("Saved task data to sandbox: /workspace/.claude_task.json\n")
		return nil
	}

	// For remote sandboxes, would need SSH upload - implement later if needed
	return fmt.Errorf("saving task data to remote sandboxes not yet implemented")
}

// startClaudeCommandInBackground starts a Claude command in the background without waiting for completion
func startClaudeCommandInBackground(sandboxInfo *sandbox.SandboxInfo, prompt string, apiKey string) error {
	utils.DebugPrintf("Starting Claude command in background for sandbox %s with prompt: %s\n", sandboxInfo.Name, prompt)

	var daemonAddr string
	var cleanup func()

	// Check if this is a remote sandbox and use appropriate connection method
	if sandboxInfo.Type == sandbox.TypeRemote {
		// Get remote provider from metadata
		remoteProvider, err := remote.NewProvider()
		if err != nil {
			return fmt.Errorf("failed to create remote provider: %w", err)
		}

		// Get daemon connection with SSH port forwarding
		addr, cleanupFunc, err := remoteProvider.GetDaemonConnection(sandboxInfo)
		if err != nil {
			return fmt.Errorf("failed to setup daemon connection: %w", err)
		}

		daemonAddr = addr
		cleanup = cleanupFunc
		defer cleanup()
	} else {
		// Local sandbox - use existing logic
		containerIP, err := getSandboxIP(sandboxInfo.Name)
		if err != nil {
			return fmt.Errorf("failed to get sandbox IP: %w", err)
		}
		daemonAddr = fmt.Sprintf("%s:28080", containerIP)
	}

	utils.DebugPrintf("Connecting to daemon at %s\n", daemonAddr)

	conn, err := grpc.NewClient(daemonAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer conn.Close()

	client := pb.NewAgentServiceClient(conn)

	workDir, err := getWorkDirFromProvider(sandboxInfo.Name)
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Execute Claude command
	_, err = client.ExecuteClaude(context.Background(), &pb.ExecuteClaudeRequest{
		Prompt:           prompt,
		WorkingDirectory: workDir,
		EnvironmentVars:  make(map[string]string),
		AnthropicApiKey:  apiKey,
	})
	if err != nil {
		return fmt.Errorf("failed to start Claude command: %w", err)
	}

	time.Sleep(1 * time.Second)

	return nil
}

func init() {
	// Add flags for the new command
	newCmd.Flags().BoolP("remote", "r", false, "Create remote sandbox using Daytona API (default: local Docker)")
	newCmd.Flags().StringP("name", "n", "", "Branch name to use (skips prompt)")
	newCmd.Flags().BoolP("force", "f", false, "Skip git repository check and force sandbox creation")
	newCmd.Flags().StringP("snapshot", "s", "", "Snapshot ID/name or Docker image to use")
	newCmd.Flags().StringP("target", "t", "", "Target region for remote sandbox")
	newCmd.Flags().Int32P("cpu", "", 0, "CPU allocation")
	newCmd.Flags().Int32P("memory", "", 0, "Memory allocation (MB)")
	newCmd.Flags().Int32P("disk", "", 0, "Disk allocation (GB)")
	newCmd.Flags().Int32P("auto-stop", "a", 60, "Auto-stop interval in minutes (0 = disabled, remote only)")
	newCmd.Flags().Bool("skip-copy", false, "Skip copying files to sandbox")
	newCmd.Flags().Bool("skip-daemon", false, "Skip installing daemon to sandbox")
	newCmd.Flags().StringP("group", "g", "", "Optional group parameter for organizing sandboxes")
}


// isDaemonReady checks if the Claude daemon is ready for connections
func isDaemonReady(sandboxInfo *sandbox.SandboxInfo) bool {
	var daemonAddr string
	var cleanup func()

	// Check if this is a remote sandbox and use appropriate connection method
	if sandboxInfo.Type == sandbox.TypeRemote {
		// Get remote provider from metadata
		remoteProvider, err := remote.NewProvider()
		if err != nil {
			return false
		}

		// Get daemon connection with SSH port forwarding
		addr, cleanupFunc, err := remoteProvider.GetDaemonConnection(sandboxInfo)
		if err != nil {
			return false
		}

		daemonAddr = addr
		cleanup = cleanupFunc
		defer cleanup()
	} else {
		// Local sandbox - use existing logic
		ip, err := getSandboxIP(sandboxInfo.Name)
		if err != nil {
			return false
		}
		daemonAddr = fmt.Sprintf("%s:28080", ip)
	}

	// Use context with timeout for connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to connect to the daemon
	conn, err := grpc.NewClient(daemonAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return false
	}
	defer conn.Close()

	// Wait for connection to be ready - keep checking until ready or timeout
	for {
		state := conn.GetState()

		if state == connectivity.Ready || state == connectivity.Idle {
			break
		}
		if state == connectivity.TransientFailure || state == connectivity.Shutdown {
			return false
		}

		// Wait for state to change or timeout
		if !conn.WaitForStateChange(ctx, state) {
			return false
		}
	}

	// Check connection state
	state := conn.GetState()
	return state == connectivity.Ready || state == connectivity.Idle
}
package local

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"cli/pkg/daemon"
	"cli/pkg/database"
	"cli/pkg/project"
	"cli/pkg/sandbox"
	"cli/pkg/utils"
)

// Provider implements the local sandbox provider using Docker
type Provider struct {
	db             *database.SandboxDB
	projectManager *project.Manager
}

// NewProvider creates a new local sandbox provider
func NewProvider() (*Provider, error) {
	// Check if Docker is available
	if err := checkDockerAvailable(); err != nil {
		return nil, fmt.Errorf("Docker not available: %w", err)
	}

	// Initialize database using singleton pattern
	db, err := database.GetSandboxDB()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize sandbox database: %w", err)
	}

	// Initialize project manager
	projectManager, err := project.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize project manager: %w", err)
	}

	return &Provider{
		db:             db,
		projectManager: projectManager,
	}, nil
}

// GetType returns the provider type
func (p *Provider) GetType() sandbox.SandboxType {
	return sandbox.TypeLocal
}

// Close closes the provider and cleans up resources
func (p *Provider) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// Create creates a new local sandbox using Docker containers
func (p *Provider) Create(opts *sandbox.CreateOptions) (*sandbox.SandboxInfo, error) {
	utils.DebugPrintf("Creating local sandbox with name: %s\n", opts.Name)

	// Generate container name from branch name
	containerName := p.generateContainerName(opts.BranchName)
	utils.DebugPrintf("Generated container name: %s\n", containerName)

	// Setup project directory
	var projectPath string
	var err error

	if opts.GitHubIssue {
		// For GitHub issues, create an empty directory - repo will be cloned into container later
		utils.DebugPrintf("GitHub issue detected, creating empty project directory for repo clone\n")
		projectPath, err = p.projectManager.SetupEmptyProject(containerName)
	} else if opts.SkipCopy || opts.SourceDir == "" {
		// For skip-copy or empty source directory, create an empty project directory
		utils.DebugPrintf("Skip copy or empty source directory, creating empty project directory\n")
		projectPath, err = p.projectManager.SetupEmptyProject(containerName)
	} else {
		// Normal setup: git worktree or file copy
		// Use the original branch name for git branch creation, containerName for directory
		projectPath, err = p.projectManager.SetupProjectWithBranch(opts.SourceDir, containerName, opts.BranchName)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to setup project: %w", err)
	}
	utils.DebugPrintf("Project setup completed at: %s\n", projectPath)

	// Create Docker container with project volume mounted
	containerID, err := p.createContainer(containerName, projectPath, opts)
	if err != nil {
		// Cleanup project directory if container creation fails
		if cleanupErr := p.projectManager.CleanupProject(containerName); cleanupErr != nil {
			utils.DebugPrintf("Warning: Failed to cleanup project after container creation failure: %s\n", cleanupErr)
		}
		return nil, fmt.Errorf("failed to create Docker container: %w", err)
	}
	utils.DebugPrintf("Docker container created with ID: %s\n", containerID)

	// Log container details for debugging (visible to user)
	fmt.Printf("🐳 Docker container created:\n")
	fmt.Printf("   • Container ID: %s\n", containerID[:12]) // Show short ID
	fmt.Printf("   • Container Name: %s\n", containerName)
	fmt.Printf("   • Debug Command: docker exec -it %s /bin/bash\n", containerName)

	// Get container state
	containerState := "running" // Assume running for now

	metadata := map[string]interface{}{
		"container_name": containerName,
		"container_id":   containerID,
		"image":         opts.Snapshot, // Use snapshot as Docker image
		"project_path":  projectPath,
		"ports":         []string{}, // TODO: Add port mappings
	}

	// Add group to metadata if specified
	if opts.Group != "" {
		metadata["group"] = opts.Group
	}

	// Add model to metadata if specified
	if opts.Model != "" {
		metadata["model"] = opts.Model
	}

	sandboxInfo := &sandbox.SandboxInfo{
		ID:            opts.BranchName, // Use user-friendly branch name as ID
		Name:          opts.BranchName, // Use user-friendly branch name as name
		Type:          sandbox.TypeLocal,
		State:         containerState,
		ShellCommand:  fmt.Sprintf("docker exec -it %s /bin/bash", containerName),
		ProjectSource: "", // Will be set by service layer
		Metadata:      metadata,
	}

	// Save to database for future reference
	localSandbox := database.FromSandboxInfo(sandboxInfo, containerID, opts.Snapshot, opts.TaskData)
	if localSandbox.Image == "" {
		localSandbox.Image = "vedranjukic/dispense-sandbox:0.0.1" // Default dispense sandbox image
	}

	err = p.db.Save(localSandbox)
	if err != nil {
		utils.DebugPrintf("Warning: Failed to save sandbox to database: %s\n", err)
	}

	return sandboxInfo, nil
}

// CopyFiles copies files from local directory to local sandbox (Docker container)
func (p *Provider) CopyFiles(sandboxInfo *sandbox.SandboxInfo, localPath string) error {
	utils.DebugPrintf("Copying files to local sandbox %s from %s\n", sandboxInfo.ID, localPath)

	// TODO: Implement file copying to Docker container
	// 1. Use docker cp command or Docker API to copy files
	// 2. Handle file permissions and ownership
	// 3. Support selective copying (exclude patterns)

	return fmt.Errorf("file copying to local sandbox not yet implemented")
}

// InstallDaemon installs the embedded daemon to the local sandbox
func (p *Provider) InstallDaemon(sandboxInfo *sandbox.SandboxInfo) error {
	utils.DebugPrintf("Installing daemon to local sandbox %s\n", sandboxInfo.ID)

	// Get container ID from metadata
	containerID, ok := sandboxInfo.Metadata["container_id"].(string)
	if !ok {
		return fmt.Errorf("container ID not found in sandbox metadata")
	}

	// Get container name for user-friendly logging
	containerName, ok := sandboxInfo.Metadata["container_name"].(string)
	if !ok {
		containerName = containerID[:12] // Fallback to short container ID
	}

	fmt.Printf("🔧 Installing daemon in container: %s\n", containerName)

	// Extract the embedded daemon binary
	embeddedDaemon := daemon.NewEmbeddedDaemon()
	err := embeddedDaemon.Extract()
	if err != nil {
		return fmt.Errorf("failed to extract daemon binary: %w", err)
	}
	defer embeddedDaemon.Cleanup()

	daemonPath := embeddedDaemon.GetPath()

	utils.DebugPrintf("Daemon binary extracted to: %s\n", daemonPath)

	// Copy daemon to container using Docker cp
	if err := p.copyFileToContainer(containerID, daemonPath, "/tmp/dispensed"); err != nil {
		return fmt.Errorf("failed to copy daemon to container: %w", err)
	}

	// Install daemon and set permissions
	// First, change ownership to the current user (daytona), then make it executable and move it
	commands := []string{
		"sudo chown $(whoami):$(whoami) /tmp/dispensed", // Change ownership to current user
		"chmod +x /tmp/dispensed",                       // Make executable
		"sudo mv /tmp/dispensed /usr/local/bin/dispensed", // Move to system path
		"which dispensed", // Verify installation
		"nohup /usr/local/bin/dispensed > /dev/null 2>&1 &", // Start daemon in background
	}

	for _, cmd := range commands {
		if err := p.execInContainer(containerID, cmd); err != nil {
			return fmt.Errorf("failed to execute command '%s': %w", cmd, err)
		}
	}

	// Give the daemon a few seconds to start up after being launched with nohup
	utils.DebugPrintf("Waiting for daemon to start up...\n")
	time.Sleep(5 * time.Second)

	// Verify daemon process is running
	checkCmd := "pgrep dispensed"
	if err := p.execInContainer(containerID, checkCmd); err != nil {
		utils.DebugPrintf("Warning: Daemon process may not be running: %s\n", err)
		// Don't fail here as daemon might still be starting
	} else {
		utils.DebugPrintf("Daemon process confirmed running\n")
	}

	utils.DebugPrintf("Daemon installed successfully as /usr/local/bin/dispensed\n")
	return nil
}

// CloneGitHubRepo clones a GitHub repository directly into the container's /workspace
func (p *Provider) CloneGitHubRepo(sandboxInfo *sandbox.SandboxInfo, owner, repo, branchName string) error {
	utils.DebugPrintf("Cloning GitHub repo %s/%s into sandbox %s\n", owner, repo, sandboxInfo.ID)

	// Get container ID from metadata
	containerID, ok := sandboxInfo.Metadata["container_id"].(string)
	if !ok {
		return fmt.Errorf("container ID not found in sandbox metadata")
	}

	repoURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
	utils.DebugPrintf("Repository URL: %s\n", repoURL)

	// Fix ownership of /workspace directory first
	chownCmd := "sudo chown -R $(whoami):$(whoami) /workspace"
	if err := p.execInContainer(containerID, chownCmd); err != nil {
		return fmt.Errorf("failed to fix workspace ownership: %w", err)
	}

	// Clone the repository directly into /workspace (without creating a subfolder)
	cloneCmd := fmt.Sprintf("cd /workspace && git clone %s .", repoURL)
	if err := p.execInContainer(containerID, cloneCmd); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Create and checkout a new branch named after the sandbox
	branchCmd := fmt.Sprintf("cd /workspace && git checkout -b %s", branchName)
	if err := p.execInContainer(containerID, branchCmd); err != nil {
		return fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}

	utils.DebugPrintf("Successfully cloned repo and created branch %s\n", branchName)
	return nil
}

// GetInfo retrieves information about a local sandbox
func (p *Provider) GetInfo(id string) (*sandbox.SandboxInfo, error) {
	utils.DebugPrintf("Getting info for local sandbox %s\n", id)

	// Try to get sandbox from database first
	localSandbox, err := p.db.GetByID(id)
	if err != nil {
		// Try by name if ID lookup failed
		localSandbox, err = p.db.GetByName(id)
		if err != nil {
			return nil, fmt.Errorf("local sandbox not found: %w", err)
		}
	}

	// TODO: Sync with actual Docker container state
	// 1. Use Docker API to inspect container
	// 2. Update database if container state has changed
	// 3. Handle cases where container exists but not in database

	return localSandbox.ToSandboxInfo(), nil
}

// List lists all local sandboxes
func (p *Provider) List() ([]*sandbox.SandboxInfo, error) {
	utils.DebugPrintf("Listing local sandboxes\n")

	// Get all sandboxes from database
	localSandboxes, err := p.db.List()
	if err != nil {
		utils.DebugPrintf("Failed to list from database: %v, falling back to Docker containers\n", err)
		localSandboxes = []*database.LocalSandbox{}
	}

	// Convert to SandboxInfo slice
	var sandboxes []*sandbox.SandboxInfo
	for _, localSandbox := range localSandboxes {
		sandboxes = append(sandboxes, localSandbox.ToSandboxInfo())
	}

	return sandboxes, nil
}

// ListByGroup lists all local sandboxes in a specific group
func (p *Provider) ListByGroup(group string) ([]*sandbox.SandboxInfo, error) {
	utils.DebugPrintf("Listing local sandboxes in group: %s\n", group)

	// Get sandboxes by group from database
	localSandboxes, err := p.db.ListByGroup(group)
	if err != nil {
		utils.DebugPrintf("Failed to list by group from database: %v\n", err)
		return []*sandbox.SandboxInfo{}, nil
	}

	// Convert to SandboxInfo slice
	var sandboxes []*sandbox.SandboxInfo
	for _, localSandbox := range localSandboxes {
		sandboxes = append(sandboxes, localSandbox.ToSandboxInfo())
	}

	return sandboxes, nil
}

// Delete removes a local sandbox
func (p *Provider) Delete(id string) error {
	utils.DebugPrintf("Deleting local sandbox %s\n", id)

	// Get sandbox info from database first
	localSandbox, err := p.db.GetByID(id)
	if err != nil {
		// Try by name if ID lookup failed
		localSandbox, err = p.db.GetByName(id)
		if err != nil {
			return fmt.Errorf("sandbox not found in database: %w", err)
		}
	}

	// Delete actual Docker container
	containerID := localSandbox.ContainerID
	if containerID != "" {
		if err := p.deleteContainer(containerID); err != nil {
			utils.DebugPrintf("Warning: Failed to delete container %s: %s\n", containerID, err)
		} else {
			utils.DebugPrintf("Container %s deleted successfully\n", containerID)
		}
	}

	// Cleanup project directory (git worktree or regular directory)
	// Use container name from metadata for directory cleanup
	containerName := ""
	if localSandbox.Metadata != nil {
		if name, ok := localSandbox.Metadata["container_name"].(string); ok {
			containerName = name
		}
	}
	if containerName != "" {
		err = p.projectManager.CleanupProject(containerName)
		if err != nil {
			utils.DebugPrintf("Warning: Failed to cleanup project directory: %s\n", err)
		} else {
			utils.DebugPrintf("Project directory cleaned up successfully\n")
		}
	}

	// Remove from database
	err = p.db.Delete(localSandbox.ID)
	if err != nil {
		return fmt.Errorf("failed to remove sandbox from database: %w", err)
	}

	utils.DebugPrintf("Successfully deleted sandbox %s from database\n", localSandbox.ID)
	return nil
}

// Helper methods

// generateContainerName creates a Docker-friendly container name from branch name
func (p *Provider) generateContainerName(branchName string) string {
	// Docker container names can only contain [a-zA-Z0-9][a-zA-Z0-9_.-]
	name := strings.ToLower(branchName)

	// Replace invalid characters with underscores
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "/", "_")

	// Ensure it starts with alphanumeric
	if len(name) > 0 && !isAlphaNumeric(name[0]) {
		name = "dispense_" + name
	}

	// Add timestamp to ensure uniqueness
	timestamp := time.Now().Format("20060102_150405")
	name = fmt.Sprintf("dispense_%s_%s", name, timestamp)

	// Limit length (Docker has a limit)
	if len(name) > 63 {
		name = name[:63]
	}

	return name
}

func isAlphaNumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// createContainer creates and starts a Docker container with the project volume mounted
func (p *Provider) createContainer(containerName, projectPath string, opts *sandbox.CreateOptions) (string, error) {
	// Determine Docker image to use
	imageName := "vedranjukic/dispense-sandbox:0.0.1" // Default dispense sandbox image
	if opts.Snapshot != "" {
		imageName = opts.Snapshot
	}

	utils.DebugPrintf("Using Docker image: %s\n", imageName)

	// Pull the image if it doesn't exist locally
	if err := p.pullImageIfNeeded(imageName); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	utils.DebugPrintf("Creating container with name: %s, image: %s, project mount: %s -> /workspace\n",
		containerName, imageName, projectPath)

	// Create and start the container using Docker CLI
	args := []string{
		"run",
		"-d", // Run in background
		"--name", containerName,
		"-v", fmt.Sprintf("%s:/workspace", projectPath),
		"-w", "/workspace", // Set working directory
		"-t", // Allocate a pseudo-TTY
		"--label", "dispense.sandbox=true",
		"--label", fmt.Sprintf("dispense.name=%s", opts.BranchName),
		"--label", "dispense.type=local",
		imageName,
		// "/bin/bash", "-c", "while true; do sleep 30; done", // Keep container running
	}

	// Add group label if specified
	if opts.Group != "" {
		args = append(args, "--label", fmt.Sprintf("dispense.group=%s", opts.Group))
	}

	// Add resource limits if specified
	if opts.CPU > 0 {
		args = append(args, "--cpus", fmt.Sprintf("%.1f", float64(opts.CPU)))
	}
	if opts.Memory > 0 {
		args = append(args, "--memory", fmt.Sprintf("%dM", opts.Memory))
	}

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w\nOutput: %s", err, string(output))
	}

	containerID := strings.TrimSpace(string(output))
	utils.DebugPrintf("Container created with ID: %s\n", containerID)

	return containerID, nil
}

// pullImageIfNeeded pulls the Docker image if it doesn't exist locally
func (p *Provider) pullImageIfNeeded(imageName string) error {
	// Check if image exists locally
	cmd := exec.Command("docker", "image", "inspect", imageName)
	if err := cmd.Run(); err == nil {
		utils.DebugPrintf("Image %s already exists locally\n", imageName)
		return nil
	}

	utils.DebugPrintf("Pulling Docker image: %s\n", imageName)

	// Pull the image
	cmd = exec.Command("docker", "pull", imageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w\nOutput: %s", imageName, err, string(output))
	}

	utils.DebugPrintf("Image %s pulled successfully\n", imageName)
	return nil
}

// checkDockerAvailable checks if Docker is available
func checkDockerAvailable() error {
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker not found or not running: %w", err)
	}
	return nil
}

// copyFileToContainer copies a file from the host to a Docker container
func (p *Provider) copyFileToContainer(containerID, srcPath, destPath string) error {
	utils.DebugPrintf("Copying file from %s to container %s:%s\n", srcPath, containerID, destPath)

	// Use docker cp command
	cmd := exec.Command("docker", "cp", srcPath, fmt.Sprintf("%s:%s", containerID, destPath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to copy file to container: %w\nOutput: %s", err, string(output))
	}

	return nil
}


// execInContainer executes a command inside a Docker container
func (p *Provider) execInContainer(containerID, command string) error {
	utils.DebugPrintf("Executing command in container %s: %s\n", containerID, command)

	// Use docker exec command
	cmd := exec.Command("docker", "exec", containerID, "/bin/sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute command: %w\nOutput: %s", err, string(output))
	}

	utils.DebugPrintf("Command executed successfully\n")
	return nil
}

// deleteContainer stops and removes a Docker container
func (p *Provider) deleteContainer(containerID string) error {
	utils.DebugPrintf("Force removing container %s\n", containerID)

	// Force remove the container (kills running containers and removes them)
	removeCmd := exec.Command("docker", "rm", "-f", "-v", containerID)
	output, err := removeCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove container: %w\nOutput: %s", err, string(output))
	}

	utils.DebugPrintf("Container %s force removed successfully\n", containerID)
	return nil
}

// fixFilePermissions fixes the permissions of a single file
func (p *Provider) fixFilePermissions(containerID, defaultUser, filePath string) {
	chownCmd := exec.Command("docker", "exec", containerID, "chown", defaultUser+":"+defaultUser, filePath)
	chownOutput, chownErr := chownCmd.CombinedOutput()
	if chownErr != nil {
		utils.DebugPrintf("chown failed for file, trying with sudo: %s\n", string(chownOutput))
		// Try with sudo as fallback
		sudoChownCmd := exec.Command("docker", "exec", containerID, "sudo", "chown", defaultUser+":"+defaultUser, filePath)
		sudoOutput, sudoErr := sudoChownCmd.CombinedOutput()
		if sudoErr != nil {
			utils.DebugPrintf("Warning: failed to fix file permissions: %s\n", string(sudoOutput))
		} else {
			utils.DebugPrintf("Fixed file permissions with sudo\n")
		}
	} else {
		utils.DebugPrintf("Fixed file permissions\n")
	}
}

// getContainerDefaultUser detects the default user and home directory in a Docker container
func (p *Provider) getContainerDefaultUser(containerID string) (string, string, error) {
	// First, try to get the user that's actually configured in the image
	inspectCmd := exec.Command("docker", "inspect", "--format", "{{.Config.User}}", containerID)
	inspectOutput, err := inspectCmd.CombinedOutput()
	if err == nil {
		configUser := strings.TrimSpace(string(inspectOutput))
		if configUser != "" && configUser != "<no value>" {
			// If it's a numeric UID, try to resolve it
			if p.isNumeric(configUser) {
				resolvedUser, resolvedHome := p.resolveNumericUser(containerID, configUser)
				if resolvedUser != "" {
					return resolvedUser, resolvedHome, nil
				}
			} else {
				// It's already a username, get the home directory
				homeCmd := exec.Command("docker", "exec", "-u", configUser, containerID, "sh", "-c", "echo $HOME")
				homeOutput, homeErr := homeCmd.CombinedOutput()
				if homeErr == nil {
					userHome := strings.TrimSpace(string(homeOutput))
					if userHome != "" {
						return configUser, userHome, nil
					}
				}
				// Fallback to standard home path
				if configUser == "root" {
					return configUser, "/root", nil
				}
				return configUser, "/home/" + configUser, nil
			}
		}
	}

	// Try to get the current user inside the container
	whoamiCmd := exec.Command("docker", "exec", containerID, "whoami")
	output, err := whoamiCmd.CombinedOutput()
	if err != nil {
		utils.DebugPrintf("Failed to run whoami: %s\n", err)
		// Fallback to common users
		return p.detectUserFallback(containerID)
	}

	defaultUser := strings.TrimSpace(string(output))
	if defaultUser == "" {
		return p.detectUserFallback(containerID)
	}

	// Get the home directory for this user
	homeCmd := exec.Command("docker", "exec", containerID, "sh", "-c", "echo $HOME")
	homeOutput, err := homeCmd.CombinedOutput()
	if err != nil {
		// Fallback to standard paths
		if defaultUser == "root" {
			return defaultUser, "/root", nil
		}
		return defaultUser, "/home/" + defaultUser, nil
	}

	userHome := strings.TrimSpace(string(homeOutput))
	if userHome == "" {
		// Fallback to standard paths
		if defaultUser == "root" {
			userHome = "/root"
		} else {
			userHome = "/home/" + defaultUser
		}
	}

	return defaultUser, userHome, nil
}

// detectUserFallback tries to detect user by checking common non-root users
func (p *Provider) detectUserFallback(containerID string) (string, string, error) {
	// Common non-root users to try (in order of preference)
	commonUsers := []string{"daytona", "ubuntu", "debian", "alpine", "node", "python", "vscode"}

	for _, user := range commonUsers {
		// Check if user exists by trying to get their home directory
		checkCmd := exec.Command("docker", "exec", containerID, "sh", "-c", fmt.Sprintf("getent passwd %s", user))
		if err := checkCmd.Run(); err == nil {
			// User exists, get their home directory
			homeCmd := exec.Command("docker", "exec", containerID, "sh", "-c", fmt.Sprintf("eval echo ~%s", user))
			homeOutput, homeErr := homeCmd.CombinedOutput()
			if homeErr == nil {
				userHome := strings.TrimSpace(string(homeOutput))
				if userHome != "" {
					utils.DebugPrintf("Detected user via fallback: %s, home: %s\n", user, userHome)
					return user, userHome, nil
				}
			}
			// Fallback to standard home path
			return user, "/home/" + user, nil
		}
	}

	// Ultimate fallback to root
	utils.DebugPrintf("No non-root user found, falling back to root\n")
	return "root", "/root", nil
}

// isNumeric checks if a string contains only digits
func (p *Provider) isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// resolveNumericUser tries to resolve a numeric UID to username and home directory
func (p *Provider) resolveNumericUser(containerID, uid string) (string, string) {
	// Try to get username from /etc/passwd
	passwdCmd := exec.Command("docker", "exec", containerID, "sh", "-c", fmt.Sprintf("getent passwd %s | cut -d: -f1,6", uid))
	output, err := passwdCmd.CombinedOutput()
	if err != nil {
		utils.DebugPrintf("Failed to resolve UID %s: %s\n", uid, err)
		return "", ""
	}

	parts := strings.Split(strings.TrimSpace(string(output)), ":")
	if len(parts) >= 2 {
		username := parts[0]
		homeDir := parts[1]
		utils.DebugPrintf("Resolved UID %s to user %s, home %s\n", uid, username, homeDir)
		return username, homeDir
	}

	return "", ""
}

// ExecuteShell starts an interactive shell session in the local sandbox container
func (p *Provider) ExecuteShell(sandboxInfo *sandbox.SandboxInfo) error {
	utils.DebugPrintf("Starting interactive shell in local sandbox %s\n", sandboxInfo.ID)

	// Get container ID from metadata
	containerID, ok := sandboxInfo.Metadata["container_id"].(string)
	if !ok {
		return fmt.Errorf("container ID not found in sandbox metadata")
	}

	// Try bash first, fallback to sh
	shells := []string{"/bin/bash", "/bin/sh"}
	var shellToUse string

	for _, shell := range shells {
		// Check if shell exists in container
		checkCmd := exec.Command("docker", "exec", containerID, "which", shell)
		if err := checkCmd.Run(); err == nil {
			shellToUse = shell
			break
		}
	}

	if shellToUse == "" {
		return fmt.Errorf("no suitable shell found in container (tried: %s)", strings.Join(shells, ", "))
	}

	utils.DebugPrintf("Using shell: %s\n", shellToUse)

	// Create interactive docker exec command
	cmd := exec.Command("docker", "exec", "-it", containerID, shellToUse)

	// Connect stdin, stdout, stderr for interactive use
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start interactive shell: %w", err)
	}

	return nil
}

// listDockerContainers lists Docker containers with dispense labels
func (p *Provider) listDockerContainers() ([]*sandbox.SandboxInfo, error) {
	// List containers with dispense labels
	cmd := exec.Command("docker", "ps", "--filter", "label=dispense.sandbox=true", "--format", "{{.ID}}\t{{.Names}}\t{{.Label \"dispense.name\"}}\t{{.Status}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list Docker containers: %w", err)
	}

	var sandboxes []*sandbox.SandboxInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) >= 4 {
			containerID := parts[0]
			containerName := parts[1]
			dispenseName := parts[2]
			status := parts[3]

			// Use dispense.name label if available, otherwise fall back to container name
			sandboxName := dispenseName
			if sandboxName == "" {
				sandboxName = containerName
			}

			sandboxInfo := &sandbox.SandboxInfo{
				ID:            sandboxName,
				Name:          sandboxName,
				Type:          sandbox.TypeLocal,
				State:         p.parseContainerStatus(status),
				ShellCommand:  fmt.Sprintf("docker exec -it %s /bin/bash", containerName),
				ProjectSource: "", // Not available during listing
				Metadata: map[string]interface{}{
					"container_name": containerName,
					"container_id":   containerID,
				},
			}
			sandboxes = append(sandboxes, sandboxInfo)
		}
	}

	return sandboxes, nil
}

// parseContainerStatus converts Docker status to sandbox state
func (p *Provider) parseContainerStatus(status string) string {
	if strings.Contains(status, "Up") {
		return "running"
	}
	return "stopped"
}

// GetWorkDir returns the working directory path for the local sandbox environment
func (p *Provider) GetWorkDir(sandboxInfo *sandbox.SandboxInfo) (string, error) {
	// Local provider always uses /workspace as the working directory
	return "/workspace", nil
}

// ExecuteCommand executes a command in the local sandbox container
func (p *Provider) ExecuteCommand(sandboxInfo *sandbox.SandboxInfo, command string) (*sandbox.ExecResult, error) {
	utils.DebugPrintf("Executing command in local sandbox %s: %s\n", sandboxInfo.ID, command)

	// Get container ID from metadata
	containerID, ok := sandboxInfo.Metadata["container_id"].(string)
	if !ok {
		return nil, fmt.Errorf("container ID not found in sandbox metadata")
	}

	// Execute command using docker exec
	cmd := exec.Command("docker", "exec", containerID, "/bin/sh", "-c", command)

	// Capture stdout and stderr separately
	stdout, stderr, exitCode, err := p.execCommandWithOutput(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}

	result := &sandbox.ExecResult{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
	}

	utils.DebugPrintf("Command completed with exit code: %d\n", exitCode)
	return result, nil
}

// execCommandWithOutput executes a command and captures stdout, stderr, and exit code
func (p *Provider) execCommandWithOutput(cmd *exec.Cmd) (string, string, int, error) {
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			// If we can't get the exit code, return the error
			return "", "", -1, err
		}
	}

	return stdout.String(), stderr.String(), exitCode, nil
}
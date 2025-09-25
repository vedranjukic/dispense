package remote

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"apiclient"
	"cli/pkg/client"
	"cli/pkg/daemon"
	"cli/pkg/sandbox"
	"cli/pkg/utils"

	"golang.org/x/crypto/ssh"
)

// Provider implements the remote sandbox provider using Daytona API
type Provider struct {
	apiClient *client.Client
}

// NewProvider creates a new remote sandbox provider
func NewProvider() (*Provider, error) {
	apiClient, err := client.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	return &Provider{
		apiClient: apiClient,
	}, nil
}

// NewProviderNonInteractive creates a new remote sandbox provider without prompting for API key
// Returns an error if no API key is available
func NewProviderNonInteractive() (*Provider, error) {
	apiClient, err := client.NewClientNonInteractive()
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	return &Provider{
		apiClient: apiClient,
	}, nil
}

// GetType returns the provider type
func (p *Provider) GetType() sandbox.SandboxType {
	return sandbox.TypeRemote
}

// Create creates a new remote sandbox using Daytona API
func (p *Provider) Create(opts *sandbox.CreateOptions) (*sandbox.SandboxInfo, error) {
	utils.DebugPrintf("Creating remote sandbox with name: %s\n", opts.Name)

	// Generate slug from branch name
	slug := generateSlug(opts.BranchName)
	utils.DebugPrintf("Generated slug: %s\n", slug)

	// Create sandbox with the slug as dispense-name label
	remoteSandbox, err := p.createSandboxWithLabel(slug, opts.Snapshot, opts.Target, opts.CPU, opts.Memory, opts.Disk, opts.AutoStop, opts.Group, opts.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create remote sandbox: %w", err)
	}

	metadata := map[string]interface{}{
		"daytona_sandbox": remoteSandbox,
		"api_client":      p.apiClient,
	}

	// Add group to metadata if specified
	if opts.Group != "" {
		metadata["group"] = opts.Group
	}

	// Add model to metadata if specified
	if opts.Model != "" {
		metadata["model"] = opts.Model
	}

	// Convert to our SandboxInfo format
	sandboxInfo := &sandbox.SandboxInfo{
		ID:           remoteSandbox.Id,
		Name:         slug,
		Type:         sandbox.TypeRemote,
		State:        p.getSandboxState(remoteSandbox),
		ShellCommand: fmt.Sprintf("ssh %s", slug),
		Metadata:     metadata,
	}

	return sandboxInfo, nil
}

// CopyFiles copies files from local directory to remote sandbox
func (p *Provider) CopyFiles(sandboxInfo *sandbox.SandboxInfo, localPath string) error {
	utils.DebugPrintf("Copying files to remote sandbox %s from %s\n", sandboxInfo.ID, localPath)

	// Wait for sandbox to be ready
	err := p.waitForSandboxReady(sandboxInfo.ID)
	if err != nil {
		return fmt.Errorf("sandbox not ready: %w", err)
	}


	// Check if we're in a git repository and create bundle if so
	if p.isGitRepository(localPath) {
		utils.DebugPrintf("Git repository detected, creating git bundle...\n")
		err = p.copyFilesWithGitBundle(sandboxInfo.ID, localPath)
		if err != nil {
			utils.DebugPrintf("Git bundle copy failed, trying regular file copy: %v\n", err)
			// Fallback to regular file copying
			err = p.copyFilesWithAPI(sandboxInfo.ID, localPath)
			if err != nil {
				utils.DebugPrintf("API-based copy failed, trying API file-by-file fallback: %v\n", err)
				// Fallback to API file-by-file approach
				err = p.copyFilesOneByOne(sandboxInfo.ID, localPath)
				if err != nil {
					return fmt.Errorf("failed to copy files: %w", err)
				}
			}
		}
	} else {
		// Use API-based file copying
		utils.DebugPrintf("Using regular file copying (no git repository detected)\n")
		err = p.copyFilesWithAPI(sandboxInfo.ID, localPath)
		if err != nil {
			utils.DebugPrintf("API-based copy failed, trying API file-by-file fallback: %v\n", err)
			// Fallback to API file-by-file approach
			err = p.copyFilesOneByOne(sandboxInfo.ID, localPath)
			if err != nil {
				return fmt.Errorf("failed to copy files: %w", err)
			}
		}
	}

	return nil
}

// CopyClaudeConfig copies and modifies .claude.json from host to remote sandbox
func (p *Provider) CopyClaudeConfig(sandboxInfo *sandbox.SandboxInfo) error {
	utils.DebugPrintf("Copying .claude.json to remote sandbox %s\n", sandboxInfo.ID)

	// Wait for sandbox to be ready first
	err := p.waitForSandboxReady(sandboxInfo.ID)
	if err != nil {
		return fmt.Errorf("sandbox not ready: %w", err)
	}

	// Get the remote home directory and workspace path
	remoteHomeDir, remoteWorkspacePath, err := p.getRemotePaths(sandboxInfo.ID)
	if err != nil {
		return fmt.Errorf("failed to get remote paths: %w", err)
	}

	utils.DebugPrintf("Remote home directory: %s, workspace: %s\n", remoteHomeDir, remoteWorkspacePath)

	// Create a modified .claude.json file locally in a temporary file
	tempClaudeJsonPath, err := utils.CreateModifiedClaudeConfig(remoteWorkspacePath)
	if err != nil {
		utils.DebugPrintf("No .claude.json found or failed to create modified version: %s\n", err)
		return nil // Not an error - file might not exist
	}
	defer os.Remove(tempClaudeJsonPath) // Clean up temporary file

	utils.DebugPrintf("Created modified .claude.json in temporary file: %s\n", tempClaudeJsonPath)

	// Upload the pre-modified .claude.json to the remote sandbox home directory
	remoteClaudeJsonPath := filepath.Join(remoteHomeDir, ".claude.json")
	err = p.apiClient.UploadFile(sandboxInfo.ID, tempClaudeJsonPath, remoteClaudeJsonPath)
	if err != nil {
		return fmt.Errorf("failed to upload .claude.json: %w", err)
	}

	utils.DebugPrintf("Successfully uploaded modified .claude.json to remote sandbox: %s\n", remoteClaudeJsonPath)
	return nil
}

// InstallDaemon installs the embedded daemon to the remote sandbox
func (p *Provider) InstallDaemon(sandboxInfo *sandbox.SandboxInfo) error {
	utils.DebugPrintf("Installing daemon to remote sandbox %s\n", sandboxInfo.ID)

	// Create embedded daemon instance
	embeddedDaemon := daemon.NewEmbeddedDaemon()

	// Check if daemon is available
	if embeddedDaemon.Size() == 0 {
		return fmt.Errorf("no embedded daemon binary found")
	}

	utils.DebugPrintf("Installing embedded daemon (%d bytes) to sandbox %s\n", embeddedDaemon.Size(), sandboxInfo.ID)

	// Extract daemon binary locally first
	err := embeddedDaemon.Extract()
	if err != nil {
		return fmt.Errorf("failed to extract daemon binary: %w", err)
	}
	defer embeddedDaemon.Cleanup()

	// Upload daemon binary to /tmp using Daytona API
	tempDaemonPath := "/tmp/dispensed"
	utils.DebugPrintf("Uploading daemon binary via API to: %s\n", tempDaemonPath)
	err = p.apiClient.UploadFile(sandboxInfo.ID, embeddedDaemon.GetPath(), tempDaemonPath)
	if err != nil {
		return fmt.Errorf("failed to upload daemon binary: %w", err)
	}

	// Move daemon to /usr/local/bin and make executable using API
	utils.DebugPrintf("Installing daemon to /usr/local/bin/dispensed\n")
	err = p.installDaemonRemotely(sandboxInfo.ID, tempDaemonPath, "/usr/local/bin/dispensed")
	if err != nil {
		return fmt.Errorf("failed to install daemon: %w", err)
	}

	utils.DebugPrintf("Daemon installed and started successfully\n")
	return nil
}

// GetInfo retrieves information about a remote sandbox
func (p *Provider) GetInfo(id string) (*sandbox.SandboxInfo, error) {
	remoteSandbox, err := p.apiClient.GetSandbox(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get sandbox info: %w", err)
	}

	// Extract name from labels
	name := id
	if remoteSandbox.Labels != nil && remoteSandbox.Labels["dispense-name"] != "" {
		name = remoteSandbox.Labels["dispense-name"]
	}

	sandboxInfo := &sandbox.SandboxInfo{
		ID:           remoteSandbox.Id,
		Name:         name,
		Type:         sandbox.TypeRemote,
		State:        p.getSandboxState(remoteSandbox),
		ShellCommand: fmt.Sprintf("ssh %s", name),
		Metadata: map[string]interface{}{
			"daytona_sandbox": remoteSandbox,
		},
	}

	return sandboxInfo, nil
}

// List lists all remote sandboxes
func (p *Provider) List() ([]*sandbox.SandboxInfo, error) {
	sandboxes, err := p.apiClient.ListSandboxes()
	if err != nil {
		return nil, fmt.Errorf("failed to list remote sandboxes: %w", err)
	}

	var result []*sandbox.SandboxInfo
	for _, sb := range sandboxes {
		info, err := p.GetInfo(sb.Id)
		if err != nil {
			utils.DebugPrintf("Failed to get info for sandbox %s: %v\n", sb.Id, err)
			continue
		}
		result = append(result, info)
	}

	return result, nil
}

// Delete removes a remote sandbox
func (p *Provider) Delete(id string) error {
	utils.DebugPrintf("Deleting remote sandbox: %s\n", id)

	err := p.apiClient.DeleteSandbox(id)
	if err != nil {
		return fmt.Errorf("failed to delete remote sandbox: %w", err)
	}

	utils.DebugPrintf("Successfully deleted remote sandbox: %s\n", id)
	return nil
}

// Helper methods (extracted from default.go)

func (p *Provider) createSandboxWithLabel(dispenseName, snapshot, target string, cpu, memory, disk, autoStop int32, group, model string) (*apiclient.Sandbox, error) {
	// Set default values if not provided
	if snapshot == "" {
		snapshot = "dispense-sandbox-001" // Default dispense snapshot
	}
	if target == "" {
		target = "us" // Default target
	}

	// Create labels map with dispense-name
	labels := map[string]string{
		"dispense-name": dispenseName,
	}

	// Add group label if specified
	if group != "" {
		labels["dispense-group"] = group
	}

	// Add model label if specified
	if model != "" {
		labels["dispense-model"] = model
	}

	// Create environment variables
	env := map[string]string{
		"DISPENSE_NAME": dispenseName,
	}

	// Create the sandbox request
	createSandbox := apiclient.NewCreateSandbox()
	createSandbox.SetSnapshot(snapshot)
	createSandbox.SetTarget(target)
	createSandbox.SetLabels(labels)
	createSandbox.SetEnv(env)
	createSandbox.SetPublic(false)
	createSandbox.SetNetworkBlockAll(false)

	// Set auto-stop if provided
	if autoStop > 0 {
		createSandbox.SetAutoStopInterval(autoStop)
	}

	// Create the sandbox
	return p.apiClient.CreateSandbox(createSandbox)
}

func (p *Provider) getSandboxState(sandbox *apiclient.Sandbox) string {
	if sandbox.State == nil {
		return "Unknown"
	}
	return string(*sandbox.State)
}

func (p *Provider) waitForSandboxReady(sandboxId string) error {
	maxAttempts := 30 // 30 attempts with 2 second intervals = 1 minute max
	for i := 0; i < maxAttempts; i++ {
		sandbox, err := p.apiClient.GetSandbox(sandboxId)
		if err != nil {
			return fmt.Errorf("failed to get sandbox status: %w", err)
		}

		if sandbox.State != nil {
			state := string(*sandbox.State)
			utils.DebugPrintf("Sandbox state: %s\n", state)

			if state == string(apiclient.SANDBOXSTATE_STARTED) {
				return nil
			}

			if state == string(apiclient.SANDBOXSTATE_DESTROYED) ||
			   state == string(apiclient.SANDBOXSTATE_ERROR) {
				return fmt.Errorf("sandbox is in %s state", state)
			}
		}

		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("sandbox did not become ready within timeout")
}

// generateSlug creates a URL-friendly slug from the input string
func generateSlug(input string) string {
	// Convert to lowercase
	slug := strings.ToLower(input)

	// Replace spaces and underscores with hyphens
	slug = regexp.MustCompile(`[\s_]+`).ReplaceAllString(slug, "-")

	// Remove all non-alphanumeric characters except hyphens
	slug = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(slug, "")

	// Replace multiple consecutive hyphens with single hyphen
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")

	// Remove leading and trailing hyphens
	slug = strings.Trim(slug, "-")

	// If empty after processing, generate a fallback
	if slug == "" {
		slug = fmt.Sprintf("branch-%d", time.Now().Unix())
	}

	return slug
}

// Helper methods for file operations

// isGitRepository checks if a directory contains a .git folder
func (p *Provider) isGitRepository(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	if stat, err := os.Stat(gitDir); err == nil {
		return stat.IsDir()
	}
	return false
}

// copyFilesWithAPI copies files using Daytona's API
func (p *Provider) copyFilesWithAPI(sandboxId, localPath string) error {
	utils.DebugPrintf("Starting API-based file copy\n")

	// Count files first to decide between individual uploads vs tar approach
	fileCount := 0
	var totalSize int64

	err := filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip hidden files and directories (like .git)
		if strings.HasPrefix(filepath.Base(path), ".") && path != localPath {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Count regular files only
		if info.Mode().IsRegular() {
			fileCount++
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to count files: %w", err)
	}

	utils.DebugPrintf("Found %d files, total size: %d bytes\n", fileCount, totalSize)

	// Use tar approach for many files (>10) or large total size (>10MB)
	useTar := fileCount > 10 || totalSize > 10*1024*1024
	if useTar {
		utils.DebugPrintf("Using tar-based upload for efficiency (%d files, %d bytes)\n", fileCount, totalSize)
		return p.copyFilesWithTar(sandboxId, localPath)
	}

	// Use individual file uploads for few files
	utils.DebugPrintf("Using individual file uploads (%d files, %d bytes)\n", fileCount, totalSize)
	return p.copyFilesIndividually(sandboxId, localPath)
}

// copyFilesWithTar creates a tar archive and uploads it via API
func (p *Provider) copyFilesWithTar(sandboxId, localPath string) error {
	utils.DebugPrintf("Creating temporary tar file\n")

	// Create temporary tar file
	tempFile, err := os.CreateTemp("", "dispense-upload-*.tar.gz")
	if err != nil {
		return fmt.Errorf("failed to create temp tar file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	err = p.createTarArchive(tempFile, localPath)
	if err != nil {
		return fmt.Errorf("failed to create tar archive: %w", err)
	}

	// Get dynamic remote workspace path
	remotePath, err := p.getRemoteWorkspacePath(sandboxId)
	if err != nil {
		return fmt.Errorf("failed to get remote workspace path: %w", err)
	}

	utils.DebugPrintf("Uploading tar file to sandbox %s, extracting to %s\n", sandboxId, remotePath)

	err = p.apiClient.UploadTarFile(sandboxId, tempFile.Name(), remotePath)
	if err != nil {
		return fmt.Errorf("failed to upload and extract tar file: %w", err)
	}

	return nil
}

// copyFilesIndividually uploads files one by one via API
func (p *Provider) copyFilesIndividually(sandboxId, localPath string) error {
	utils.DebugPrintf("Starting individual file uploads\n")

	// Get dynamic remote workspace path
	remoteWorkspacePath, err := p.getRemoteWorkspacePath(sandboxId)
	if err != nil {
		return fmt.Errorf("failed to get remote workspace path: %w", err)
	}

	fileCount := 0
	err = filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip hidden files and directories (like .git)
		if strings.HasPrefix(filepath.Base(path), ".") && path != localPath {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Upload regular files only
		if info.Mode().IsRegular() {
			// Get relative path
			relPath, err := filepath.Rel(localPath, path)
			if err != nil {
				return nil
			}

			// Upload file to dynamic remote workspace path
			remotePath := filepath.Join(remoteWorkspacePath, filepath.ToSlash(relPath))
			err = p.apiClient.UploadFile(sandboxId, path, remotePath)
			if err != nil {
				return err
			}
			fileCount++
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to upload files: %w", err)
	}

	utils.DebugPrintf("Successfully uploaded %d files\n", fileCount)
	return nil
}

// copyFilesOneByOne copies files using API client as fallback
func (p *Provider) copyFilesOneByOne(sandboxId, cwd string) error {
	utils.DebugPrintf("Starting API-based file copy fallback\n")

	// Get dynamic workspace path
	remoteWorkspacePath, err := p.getRemoteWorkspacePath(sandboxId)
	if err != nil {
		return fmt.Errorf("failed to get remote workspace path: %w", err)
	}

	// Create remote directory
	err = p.createRemoteDirectory(sandboxId, remoteWorkspacePath)
	if err != nil {
		return fmt.Errorf("failed to create remote directory: %w", err)
	}

	// Copy files one by one using file upload API
	fileCount := 0
	err = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil || strings.HasPrefix(filepath.Base(path), ".") {
			return nil // Skip errors and hidden files
		}

		if info.Mode().IsRegular() {
			relPath, _ := filepath.Rel(cwd, path)
			remotePath := filepath.Join(remoteWorkspacePath, filepath.ToSlash(relPath))

			// Upload file using API client
			err = p.apiClient.UploadFile(sandboxId, path, remotePath)
			if err != nil {
				return err
			}
			fileCount++
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to copy files via API: %w", err)
	}

	utils.DebugPrintf("Successfully copied %d files via API\n", fileCount)
	return nil
}

// copyFilesWithGitBundle creates git bundle and uploads it
func (p *Provider) copyFilesWithGitBundle(sandboxId, localPath string) error {
	utils.DebugPrintf("Starting git bundle creation\n")

	// Get current git branch
	currentBranch, err := p.getCurrentGitBranch(localPath)
	if err != nil {
		return fmt.Errorf("failed to get current git branch: %w", err)
	}

	// Create git bundle
	bundlePath, err := p.createGitBundle(localPath, currentBranch)
	if err != nil {
		return fmt.Errorf("failed to create git bundle: %w", err)
	}
	defer os.Remove(bundlePath)

	// Get dynamic remote workspace path
	remoteWorkspacePath, err := p.getRemoteWorkspacePath(sandboxId)
	if err != nil {
		return fmt.Errorf("failed to get remote workspace path: %w", err)
	}

	// Upload git bundle to workspace directory
	remoteBundlePath := filepath.Join(remoteWorkspacePath, "project.bundle")
	err = p.apiClient.UploadFile(sandboxId, bundlePath, remoteBundlePath)
	if err != nil {
		return fmt.Errorf("failed to upload git bundle: %w", err)
	}

	// Extract git bundle remotely
	return p.extractGitBundleRemotely(sandboxId, remoteBundlePath, remoteWorkspacePath)
}


// createSSHClient creates SSH connection to Daytona (used only for interactive shell)
func (p *Provider) createSSHClient(sshToken string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User:            sshToken,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	client, err := ssh.Dial("tcp", "ssh.app.daytona.io:22", config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SSH: %w", err)
	}

	return client, nil
}

// createRemoteDirectory creates directory via API client
func (p *Provider) createRemoteDirectory(sandboxId, remotePath string) error {
	cmd := fmt.Sprintf("mkdir -p %s", remotePath)
	_, err := p.apiClient.RunCommand(sandboxId, cmd, "")
	return err
}


// createTarArchive creates compressed tar of local directory
func (p *Provider) createTarArchive(writer io.Writer, localPath string) error {
	gzipWriter := gzip.NewWriter(writer)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	return filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || strings.HasPrefix(filepath.Base(path), ".") {
			return nil // Skip errors and hidden files
		}

		relPath, err := filepath.Rel(localPath, path)
		if err != nil || relPath == "." {
			return nil
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		err = tarWriter.WriteHeader(header)
		if err != nil {
			return err
		}

		// Write file content for regular files
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(tarWriter, file)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// getCurrentGitBranch gets current branch name
func (p *Provider) getCurrentGitBranch(repoPath string) (string, error) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	os.Chdir(repoPath)
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" {
		// Fallback for detached HEAD
		cmd = exec.Command("git", "rev-parse", "--short", "HEAD")
		output, err = cmd.Output()
		if err != nil {
			return "", err
		}
		branch = "detached-" + strings.TrimSpace(string(output))
	}
	return branch, nil
}

// createGitBundle creates git bundle file
func (p *Provider) createGitBundle(repoPath, branchName string) (string, error) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	os.Chdir(repoPath)

	tempFile, err := os.CreateTemp("", "dispense-bundle-*.bundle")
	if err != nil {
		return "", err
	}
	tempFile.Close()

	cmd := exec.Command("git", "bundle", "create", tempFile.Name(), branchName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to create git bundle: %w\nOutput: %s", err, string(output))
	}

	return tempFile.Name(), nil
}

// extractGitBundleRemotely extracts git bundle on remote system
func (p *Provider) extractGitBundleRemotely(sandboxId, remoteBundlePath, remoteWorkspacePath string) error {
    // Get remote home directory dynamically via workspace path
    homeDir := filepath.Dir(remoteWorkspacePath)

    // Resolve git path and use it for clone
    gitWhichCmd := "which git"
    gitPathResp, err := p.apiClient.RunCommand(sandboxId, gitWhichCmd, "")
    if err != nil {
        return err
    }
    gitPath := strings.TrimSpace(gitPathResp.Output)
    if gitPath == "" {
        return fmt.Errorf("git not found on remote system")
    }

    // Create workspace and extract bundle
    cmd := fmt.Sprintf("cd %s && rm -rf workspace && mkdir workspace && cd workspace && %s clone --recursive %s .", homeDir, gitPath, remoteBundlePath)
    _, err = p.apiClient.RunCommand(sandboxId, cmd, "")
    return err
}

// installDaemonRemotely installs daemon binary via API client and starts it
func (p *Provider) installDaemonRemotely(sandboxId, tempPath, finalPath string) error {
	commands := []string{
		fmt.Sprintf("chmod +x %s", tempPath),
		fmt.Sprintf("sudo mv %s %s", tempPath, finalPath),
		"which dispensed", // Verify installation
	}

	for _, cmd := range commands {
		_, err := p.apiClient.RunCommand(sandboxId, cmd, "")
		if err != nil {
			return fmt.Errorf("failed to execute command '%s': %w", cmd, err)
		}
	}

	// Start the daemon in the background using the installed path
	utils.DebugPrintf("Running daemon asynchronously: %s\n", finalPath)

	err := p.apiClient.RunAsyncCommand(sandboxId, finalPath)
	if err != nil {
		return fmt.Errorf("failed to start daemon asynchronously: %w", err)
	}

	// Wait a moment for daemon to start
	time.Sleep(3 * time.Second)

	// Verify daemon is running by checking if port 28080 is listening
	portCheckCmd := "netstat -plunt | grep :28080 || echo 'Port 28080 not listening'"
	utils.DebugPrintf("Checking daemon port with command: %s\n", portCheckCmd)
	response, err := p.apiClient.RunCommand(sandboxId, portCheckCmd, "")
	if err != nil {
		utils.DebugPrintf("Daemon port check failed: %v\n", err)
		return fmt.Errorf("failed to check daemon port: %w", err)
	}
	utils.DebugPrintf("Daemon port check result: %s\n", response.Output)

	if strings.Contains(response.Output, "Port 28080 not listening") {
		return fmt.Errorf("daemon failed to start - port 28080 not listening")
	}

	utils.DebugPrintf("Daemon verification: %s\n", response.Output)

	return nil
}

// getRemoteWorkspacePath gets the workspace path for the sandbox, using dynamic user detection
func (p *Provider) getRemoteWorkspacePath(sandboxId string) (string, error) {
	remoteHomeDir, workspacePath, err := p.getRemotePaths(sandboxId)
	if err != nil {
		return "", err
	}

	// Try to use an existing workspace directory or default to home/workspace
	if workspacePath != "" {
		return workspacePath, nil
	}

	return filepath.Join(remoteHomeDir, "workspace"), nil
}

// getRemoteHomeDirectory gets the remote home directory using API
func (p *Provider) getRemoteHomeDirectory(sandboxId string) (string, error) {
	utils.DebugPrintf("Detecting remote home directory for sandbox %s\n", sandboxId)

	// Try multiple approaches to get the home directory
	commands := []string{
		"echo $HOME",           // Environment variable
		"pwd && cd ~ && pwd",   // Change to home and print working directory
		"eval echo ~$(whoami)", // Evaluate tilde expansion for current user
	}

	var remoteHomeDir string
	for _, homeCmd := range commands {
		utils.DebugPrintf("Trying command: %s\n", homeCmd)
		output, err := p.executeCommand(sandboxId, homeCmd, "")
		if err != nil {
			utils.DebugPrintf("Command failed: %v\n", err)
			continue
		}

		// For the compound command, take the last line (the home directory)
		lines := strings.Split(strings.TrimSpace(output), "\n")
		candidateHome := strings.TrimSpace(lines[len(lines)-1])

		utils.DebugPrintf("Command output: %s\n", candidateHome)

		// Validate the result looks like a home directory path
		if candidateHome != "" &&
		   strings.HasPrefix(candidateHome, "/") &&
		   !strings.Contains(candidateHome, "$HOME") &&
		   len(candidateHome) > 1 {
			remoteHomeDir = candidateHome
			utils.DebugPrintf("Successfully detected remote home directory: %s\n", remoteHomeDir)
			break
		}
	}

	if remoteHomeDir == "" {
		// Ultimate fallback - try common home directories
		fallbacks := []string{"/home/daytona", "/home/ubuntu", "/home/user", "/root"}
		for _, fallback := range fallbacks {
			checkCmd := fmt.Sprintf("[ -d '%s' ] && echo '%s' || echo ''", fallback, fallback)
			output, err := p.executeCommand(sandboxId, checkCmd, "")
			if err == nil && strings.TrimSpace(output) == fallback {
				remoteHomeDir = fallback
				utils.DebugPrintf("Using fallback home directory: %s\n", remoteHomeDir)
				break
			}
		}
	}

	if remoteHomeDir == "" {
		// Last resort fallback
		remoteHomeDir = "/home/user"
		utils.DebugPrintf("Using last resort fallback home directory: %s\n", remoteHomeDir)
	}

	return remoteHomeDir, nil
}

// getRemotePaths detects the remote home directory and workspace path
func (p *Provider) getRemotePaths(sandboxId string) (string, string, error) {
	utils.DebugPrintf("Detecting remote paths for sandbox %s\n", sandboxId)

	// Get remote home directory using API
	remoteHomeDir, err := p.getRemoteHomeDirectory(sandboxId)
	if err != nil {
		return "", "", fmt.Errorf("failed to get remote home directory: %w", err)
	}

	// Try common workspace locations within user's home directory
	workspaceLocations := []string{
		filepath.Join(remoteHomeDir, "workspace"),
		remoteHomeDir, // Fallback to home directory
	}

	var remoteWorkspacePath string
	for _, location := range workspaceLocations {
		checkCmd := fmt.Sprintf("[ -d '%s' ] && echo 'exists' || echo 'not_exists'", location)
		output, err := p.executeCommand(sandboxId, checkCmd, "")
		if err == nil && strings.TrimSpace(output) == "exists" {
			remoteWorkspacePath = location
			break
		}
	}

	// If no workspace found, use the first option as default
	if remoteWorkspacePath == "" {
		remoteWorkspacePath = workspaceLocations[0]
		utils.DebugPrintf("No existing workspace found, using default: %s\n", remoteWorkspacePath)
	} else {
		utils.DebugPrintf("Found existing workspace: %s\n", remoteWorkspacePath)
	}

	return remoteHomeDir, remoteWorkspacePath, nil
}

// ExecuteShell starts an interactive SSH shell session to the remote sandbox
func (p *Provider) ExecuteShell(sandboxInfo *sandbox.SandboxInfo) error {
	utils.DebugPrintf("Starting interactive SSH shell to remote sandbox %s\n", sandboxInfo.ID)

	// Get SSH access token from Daytona API
	sshAccess, err := p.apiClient.CreateSshAccess(sandboxInfo.ID, 10)
	if err != nil {
		return fmt.Errorf("failed to create SSH access: %w", err)
	}

	// Create SSH client
	sshClient, err := p.createSSHClient(sshAccess.GetToken())
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}
	defer sshClient.Close()

	// Create interactive session
	session, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Request a PTY for interactive shell
	err = session.RequestPty("xterm-256color", 80, 24, ssh.TerminalModes{})
	if err != nil {
		return fmt.Errorf("failed to request PTY: %w", err)
	}

	// Connect stdin, stdout, stderr for interactive use
	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	// Start the shell
	err = session.Shell()
	if err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	// Wait for session to complete
	if err := session.Wait(); err != nil {
		// Check if this is a normal exit (exit status 0)
		if exitError, ok := err.(*ssh.ExitError); ok {
			// If the exit code is 0, this is a clean exit
			if exitError.ExitStatus() == 0 {
				return nil
			}
			// For non-zero exit codes, still return an error but with more context
			return fmt.Errorf("SSH session exited with status %d", exitError.ExitStatus())
		}

		// Check if this is a missing exit status (common with clean exits)
		if err.Error() == "wait: remote command exited without exit status or exit signal" {
			// This typically happens with clean exits - treat as success
			return nil
		}

		// Other types of errors
		return fmt.Errorf("SSH session error: %w", err)
	}

	return nil
}

// CloneGitHubRepo clones a GitHub repository into the remote sandbox workspace
func (p *Provider) CloneGitHubRepo(sandboxInfo *sandbox.SandboxInfo, owner, repo, branchName string) error {
	utils.DebugPrintf("Cloning GitHub repo %s/%s into remote sandbox %s\n", owner, repo, sandboxInfo.ID)

	repoURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
	utils.DebugPrintf("Repository URL: %s\n", repoURL)

	// Get dynamic workspace path
	remoteWorkspacePath, err := p.getRemoteWorkspacePath(sandboxInfo.ID)
	if err != nil {
		return fmt.Errorf("failed to get remote workspace path: %w", err)
	}
	utils.DebugPrintf("Using workspace directory: %s\n", remoteWorkspacePath)

	// Ensure workspace directory exists and has proper permissions
	prepCmd := fmt.Sprintf("mkdir -p %s && cd %s && pwd", remoteWorkspacePath, remoteWorkspacePath)
	if _, err := p.apiClient.RunCommand(sandboxInfo.ID, prepCmd, ""); err != nil {
		return fmt.Errorf("failed to prepare workspace directory: %w", err)
	}

    gitPathCmd := "which git"
    gitPathResp, err := p.apiClient.RunCommand(sandboxInfo.ID, gitPathCmd, "")
    if err != nil {
        return fmt.Errorf("git is not available in the sandbox: %w", err)
    }
    gitPath := strings.TrimSpace(gitPathResp.Output)
    if gitPath == "" {
        return fmt.Errorf("git is not available in the sandbox: empty path")
    }

    // Check if git is available
    if _, err := p.apiClient.RunCommand(sandboxInfo.ID, fmt.Sprintf("%s --version", gitPath), ""); err != nil {
        return fmt.Errorf("git is not available in the sandbox: %w", err)
    }

    // Clone the repository directly into workspace directory (without creating a subfolder)
    cloneCmd := fmt.Sprintf("%s clone %s .", gitPath, repoURL)
	if _, err := p.apiClient.RunCommand(sandboxInfo.ID, cloneCmd, remoteWorkspacePath); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

    // Create and checkout a new branch named after the sandbox
    branchCmd := fmt.Sprintf("%s checkout -b %s", gitPath, branchName)
	if _, err := p.apiClient.RunCommand(sandboxInfo.ID, branchCmd, remoteWorkspacePath); err != nil {
		return fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}

	utils.DebugPrintf("Successfully cloned repo and created branch %s\n", branchName)
	return nil
}

// executeCommand executes a command via Daytona API
func (p *Provider) executeCommand(sandboxId, command, cwd string) (string, error) {
	utils.DebugPrintf("Executing command: %s\n", command)
	response, err := p.apiClient.RunCommand(sandboxId, command, cwd)
	if err != nil {
		return "", fmt.Errorf("command failed: %w", err)
	}

	if response.ExitCode != 0 {
		return "", fmt.Errorf("command failed with exit code %d: %s", response.ExitCode, response.Output)
	}

	utils.DebugPrintf("Command executed successfully\n")

	return response.Output, nil
}

// RunCommandInSandbox executes a command in the sandbox and returns the output (public method)
func (p *Provider) RunCommandInSandbox(sandboxId, command, cwd string) (string, error) {
	return p.executeCommand(sandboxId, command, cwd)
}


// GetDaemonConnection returns connection details for the daemon in the remote sandbox
// Instead of using direct IP connectivity, it uses SSH port forwarding
func (p *Provider) GetDaemonConnection(sandboxInfo *sandbox.SandboxInfo) (string, func(), error) {
	utils.DebugPrintf("Setting up daemon connection for remote sandbox %s\n", sandboxInfo.ID)

	// Get SSH access token from Daytona API
	sshAccess, err := p.apiClient.CreateSshAccess(sandboxInfo.ID, 10)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create SSH access: %w", err)
	}

	// Create SSH client for port forwarding
	config := &ssh.ClientConfig{
		User:            sshAccess.GetToken(),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	client, err := ssh.Dial("tcp", "ssh.app.daytona.io:22", config)
	if err != nil {
		return "", nil, fmt.Errorf("failed to dial SSH: %w", err)
	}

	// Find an available local port for port forwarding
	localPort, err := p.findAvailablePort()
	if err != nil {
		client.Close()
		return "", nil, fmt.Errorf("failed to find available port: %w", err)
	}

	// Set up port forwarding from local port to remote daemon port (28080)
	localAddr := fmt.Sprintf("localhost:%d", localPort)
	remoteAddr := "localhost:28080" // Daemon listens on 28080

	// Start port forwarding in a goroutine
	done := make(chan error, 1)
	ready := make(chan bool, 1)

	go func() {
		done <- p.startPortForwarding(client, localAddr, remoteAddr, ready)
	}()

	// Wait for port forwarding to be ready or timeout
	select {
	case <-ready:
		utils.DebugPrintf("Port forwarding is ready\n")
	case err := <-done:
		client.Close()
		return "", nil, fmt.Errorf("port forwarding failed to start: %w", err)
	case <-time.After(10 * time.Second):
		client.Close()
		return "", nil, fmt.Errorf("timeout waiting for port forwarding to start")
	}

	// Return the local address to connect to and cleanup function
	cleanup := func() {
		client.Close()
	}

	return localAddr, cleanup, nil
}

// findAvailablePort finds an available port for port forwarding
func (p *Provider) findAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port, nil
}

// startPortForwarding handles the SSH port forwarding
func (p *Provider) startPortForwarding(client *ssh.Client, localAddr, remoteAddr string, ready chan<- bool) error {
	// Listen on local port
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		utils.DebugPrintf("Failed to listen on %s: %v\n", localAddr, err)
		return err
	}
	defer listener.Close()

	utils.DebugPrintf("Port forwarding established: %s -> %s\n", localAddr, remoteAddr)

	// Signal that port forwarding is ready
	if ready != nil {
		ready <- true
		close(ready)
	}

	for {
		// Accept local connections
		localConn, err := listener.Accept()
		if err != nil {
			utils.DebugPrintf("Failed to accept local connection: %v\n", err)
			break
		}

		// For each local connection, create a connection to remote
		go func(localConn net.Conn) {
			defer localConn.Close()

			// Create connection through SSH tunnel
			remoteConn, err := client.Dial("tcp", remoteAddr)
			if err != nil {
				utils.DebugPrintf("Failed to dial remote: %v\n", err)
				return
			}
			defer remoteConn.Close()

			// Start copying data between local and remote connections
			go io.Copy(remoteConn, localConn)
			io.Copy(localConn, remoteConn)
		}(localConn)
	}

	return nil
}

// GetWorkDir returns the working directory path for the remote sandbox environment
func (p *Provider) GetWorkDir(sandboxInfo *sandbox.SandboxInfo) (string, error) {
	// For remote provider, get the workspace path dynamically
	workspacePath, err := p.getRemoteWorkspacePath(sandboxInfo.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get remote workspace path: %w", err)
	}
	return workspacePath, nil
}

// ExecuteCommand executes a command in the remote sandbox using Daytona API
func (p *Provider) ExecuteCommand(sandboxInfo *sandbox.SandboxInfo, command string) (*sandbox.ExecResult, error) {
	utils.DebugPrintf("Executing command in remote sandbox %s: %s\n", sandboxInfo.ID, command)

	// Execute command using Daytona API
	response, err := p.apiClient.RunCommand(sandboxInfo.ID, command, "")
	if err != nil {
		return nil, fmt.Errorf("failed to execute command via Daytona API: %w", err)
	}

	result := &sandbox.ExecResult{
		Stdout:   response.Output,
		Stderr:   "", // Daytona API combines stdout and stderr in Output
		ExitCode: response.ExitCode,
	}

	utils.DebugPrintf("Command completed with exit code: %d\n", response.ExitCode)
	return result, nil
}
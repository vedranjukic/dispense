package project

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cli/pkg/utils"
)

// Manager handles project setup for local sandboxes
type Manager struct {
	projectsRoot string
}

// NewManager creates a new project manager
func NewManager() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	projectsRoot := filepath.Join(homeDir, ".dispense", "projects")
	return &Manager{
		projectsRoot: projectsRoot,
	}, nil
}

// GetProjectsRoot returns the root directory for all projects
func (m *Manager) GetProjectsRoot() string {
	return m.projectsRoot
}

// SetupProject sets up a project directory for a sandbox
// Returns the path to the created project directory
func (m *Manager) SetupProject(workingDir, sandboxName string) (string, error) {
	return m.SetupProjectWithBranch(workingDir, sandboxName, sandboxName)
}

// SetupProjectWithBranch sets up a project directory for a sandbox with a specific branch name
// Returns the path to the created project directory
func (m *Manager) SetupProjectWithBranch(workingDir, sandboxName, branchName string) (string, error) {
	utils.DebugPrintf("Setting up project for sandbox '%s' with branch '%s' from working dir '%s'\n", sandboxName, branchName, workingDir)

	// Ensure projects root directory exists
	if err := os.MkdirAll(m.projectsRoot, 0755); err != nil {
		return "", fmt.Errorf("failed to create projects root directory: %w", err)
	}

	projectPath := filepath.Join(m.projectsRoot, sandboxName)

	// Check if .git folder exists in working directory
	gitDir := filepath.Join(workingDir, ".git")
	if isGitRepository(gitDir) {
		utils.DebugPrintf("Git repository detected, creating worktree\n")
		return m.createGitWorktree(workingDir, projectPath, branchName)
	} else {
		utils.DebugPrintf("No git repository detected, copying files\n")
		return m.copyProjectFiles(workingDir, projectPath)
	}
}

// SetupEmptyProject creates an empty project directory for GitHub issues
// Returns the path to the created project directory
func (m *Manager) SetupEmptyProject(sandboxName string) (string, error) {
	utils.DebugPrintf("Setting up empty project directory for sandbox '%s'\n", sandboxName)

	// Ensure projects root directory exists
	if err := os.MkdirAll(m.projectsRoot, 0755); err != nil {
		return "", fmt.Errorf("failed to create projects root directory: %w", err)
	}

	projectPath := filepath.Join(m.projectsRoot, sandboxName)

	// Remove project path if it already exists
	if _, err := os.Stat(projectPath); err == nil {
		utils.DebugPrintf("Removing existing project directory: %s\n", projectPath)
		if err := os.RemoveAll(projectPath); err != nil {
			return "", fmt.Errorf("failed to remove existing project directory: %w", err)
		}
	}

	// Create empty directory
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create empty project directory: %w", err)
	}

	utils.DebugPrintf("Empty project directory created successfully: %s\n", projectPath)
	return projectPath, nil
}

// createGitWorktree creates a new git worktree for the project
func (m *Manager) createGitWorktree(gitRoot, projectPath, branchName string) (string, error) {
	utils.DebugPrintf("Creating git worktree at '%s' with branch '%s'\n", projectPath, branchName)

	// Remove project path if it already exists
	if _, err := os.Stat(projectPath); err == nil {
		utils.DebugPrintf("Removing existing project directory: %s\n", projectPath)
		if err := os.RemoveAll(projectPath); err != nil {
			return "", fmt.Errorf("failed to remove existing project directory: %w", err)
		}
	}

	// Create git worktree - let git create a new branch automatically
	// This approach avoids conflicts with existing branches
	worktreeCmd := exec.Command("git", "worktree", "add", "-b", branchName, projectPath)
	worktreeCmd.Dir = gitRoot

	output, err := worktreeCmd.CombinedOutput()
	if err != nil {
		// If creating with new branch fails, try using current branch
		utils.DebugPrintf("Worktree creation with new branch failed: %s\n", string(output))

		// Clean up any partially created directory
		os.RemoveAll(projectPath)

		worktreeCmd = exec.Command("git", "worktree", "add", projectPath)
		worktreeCmd.Dir = gitRoot

		output, err = worktreeCmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to create git worktree: %w\\nOutput: %s", err, string(output))
		}
	}

	utils.DebugPrintf("Git worktree created successfully: %s\n", string(output))
	return projectPath, nil
}

// copyProjectFiles copies all files from source to destination
func (m *Manager) copyProjectFiles(sourceDir, destDir string) (string, error) {
	utils.DebugPrintf("Copying files from '%s' to '%s'\n", sourceDir, destDir)

	// Remove destination if it already exists
	if _, err := os.Stat(destDir); err == nil {
		utils.DebugPrintf("Removing existing project directory: %s\n", destDir)
		if err := os.RemoveAll(destDir); err != nil {
			return "", fmt.Errorf("failed to remove existing project directory: %w", err)
		}
	}

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Copy files using cp command for better performance and preservation of permissions
	// Use a more robust approach to copy directory contents
	var copyCmd *exec.Cmd
	if sourceDir == "/" {
		// Special case for root directory - we don't want to copy the entire filesystem
		return "", fmt.Errorf("cannot copy from root directory '/' - this would copy the entire filesystem")
	} else {
		// For normal directories, copy all contents using shell glob expansion
		copyCmd = exec.Command("sh", "-c", fmt.Sprintf("cp -r %s/* %s/ 2>/dev/null || true && cp -r %s/.[^.]* %s/ 2>/dev/null || true",
			sourceDir, destDir, sourceDir, destDir))
	}

	output, err := copyCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to copy files: %w\\nOutput: %s", err, string(output))
	}

	utils.DebugPrintf("Files copied successfully\n")
	return destDir, nil
}

// CleanupProject removes a project directory (and git worktree if applicable)
func (m *Manager) CleanupProject(sandboxName string) error {
	projectPath := filepath.Join(m.projectsRoot, sandboxName)

	utils.DebugPrintf("Cleaning up project directory: %s\n", projectPath)

	// Check if it's a git worktree
	if isGitWorktree(projectPath) {
		utils.DebugPrintf("Removing git worktree: %s\n", projectPath)

		// Remove git worktree
		worktreeCmd := exec.Command("git", "worktree", "remove", projectPath, "--force")
		output, err := worktreeCmd.CombinedOutput()
		if err != nil {
			utils.DebugPrintf("Warning: Failed to remove git worktree cleanly: %s\n", string(output))
			// Fall back to regular directory removal
		}
	}

	// Remove directory (either after worktree removal or for non-git projects)
	if _, err := os.Stat(projectPath); err == nil {
		if err := os.RemoveAll(projectPath); err != nil {
			return fmt.Errorf("failed to remove project directory: %w", err)
		}
	}

	utils.DebugPrintf("Project directory cleaned up successfully\n")
	return nil
}

// Helper functions

// isGitRepository checks if a directory contains a .git folder
func isGitRepository(gitDir string) bool {
	if stat, err := os.Stat(gitDir); err == nil {
		return stat.IsDir()
	}
	return false
}

// isGitWorktree checks if a directory is a git worktree
func isGitWorktree(dir string) bool {
	gitFile := filepath.Join(dir, ".git")
	if stat, err := os.Stat(gitFile); err == nil {
		// In worktrees, .git is a file pointing to the git directory, not a directory
		if !stat.IsDir() {
			// Read the .git file to confirm it's a worktree
			content, err := os.ReadFile(gitFile)
			if err == nil && strings.HasPrefix(string(content), "gitdir:") {
				return true
			}
		}
	}
	return false
}

// GetProjectPath returns the full path for a project
func (m *Manager) GetProjectPath(sandboxName string) string {
	return filepath.Join(m.projectsRoot, sandboxName)
}

// ListProjects returns a list of all project directories
func (m *Manager) ListProjects() ([]string, error) {
	if _, err := os.Stat(m.projectsRoot); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(m.projectsRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to read projects directory: %w", err)
	}

	var projects []string
	for _, entry := range entries {
		if entry.IsDir() {
			projects = append(projects, entry.Name())
		}
	}

	return projects, nil
}
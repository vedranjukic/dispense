package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DebugMode is the global debug flag
var DebugMode bool

// DebugPrintf prints debug messages only when debug mode is enabled
func DebugPrintf(format string, args ...interface{}) {
	if DebugMode {
		fmt.Printf("DEBUG: "+format, args...)
	}
}

// loadAppSpecificClaudeAPIKey loads the Claude API key from the app-specific config file
func loadAppSpecificClaudeAPIKey() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(homeDir, ".dispense", "claude", "config")
	content, err := os.ReadFile(configPath)
	if err != nil {
		DebugPrintf("No app-specific Claude config found at: %s\n", configPath)
		return "", err
	}

	// Just return the trimmed content as the API key
	apiKey := strings.TrimSpace(string(content))
	if apiKey == "" {
		return "", fmt.Errorf("API key not found in app-specific config")
	}
	DebugPrintf("Found app-specific Claude API key in config\n")
	return apiKey, nil
}

// CreateModifiedClaudeConfig reads the original .claude.json from the host home directory,
// modifies it for sandbox use (including adding app-specific API key), and creates a temporary file with the modified content.
// Returns the path to the temporary file, or error if the operation fails.
// The caller is responsible for cleaning up the temporary file.
func CreateModifiedClaudeConfig(workspacePath string) (string, error) {
	// Get the host home directory
	hostHomeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get host home directory: %w", err)
	}

	// Check if .claude.json exists in the host home directory
	hostClaudeJsonPath := filepath.Join(hostHomeDir, ".claude.json")
	if _, err := os.Stat(hostClaudeJsonPath); err != nil {
		DebugPrintf("No .claude.json found in host home directory: %s\n", hostClaudeJsonPath)
		return "", fmt.Errorf("no .claude.json found: %w", err)
	}

	DebugPrintf("Found .claude.json in host home directory: %s\n", hostClaudeJsonPath)

	// Read the original .claude.json content
	originalFile, err := os.Open(hostClaudeJsonPath)
	if err != nil {
		return "", fmt.Errorf("failed to open .claude.json: %w", err)
	}
	defer originalFile.Close()

	originalContent, err := io.ReadAll(originalFile)
	if err != nil {
		return "", fmt.Errorf("failed to read .claude.json: %w", err)
	}

	// Parse the JSON
	var claudeConfig map[string]interface{}
	if err := json.Unmarshal(originalContent, &claudeConfig); err != nil {
		return "", fmt.Errorf("failed to parse .claude.json: %w", err)
	}

	DebugPrintf("Original .claude.json parsed successfully\n")

	// Try to load app-specific Claude API key
	appApiKey, err := loadAppSpecificClaudeAPIKey()
	if err != nil {
		DebugPrintf("Warning: Could not load app-specific Claude API key: %s\n", err)
	} else {
		DebugPrintf("Loaded app-specific Claude API key successfully\n")
		// Add the API key to the config if found
		claudeConfig["apiKey"] = appApiKey
	}

	// Modify the configuration for sandbox use
	// 1. Empty the cachedChangelog field
	claudeConfig["cachedChangelog"] = ""

	// 2. Create trusted project configuration template
	trustedProjectConfig := map[string]interface{}{
		"allowedTools":                             []string{},
		"history":                                  []string{},
		"mcpContextUris":                           []string{},
		"mcpServers":                              map[string]interface{}{},
		"enableedMcpjsonServers":                  []string{},
		"disabledMcpjsonServers":                  []string{},
		"hasTrustDialogAccepted":                  true,
		"projectOnboardingSeenCount":              0,
		"hasClaudeMdExternalIncludesApproved":     false,
		"hasClaudeMdExternalIncludesWarningShown": false,
		"lastTotalWebSearchRequests":              0,
		"exampleFiles":                            []string{},
		"exampleFilesGeneratedAt":                 0,
	}

	// Create new projects object with both home directory and workspace entries
	projects := map[string]interface{}{
		workspacePath: trustedProjectConfig,
	}

	// Also add the parent directory of workspace as trusted (e.g., /home/daytona if workspace is /home/daytona/workspace)
	workspaceParent := filepath.Dir(workspacePath)
	if workspaceParent != "." && workspaceParent != "/" && workspaceParent != workspacePath {
		projects[workspaceParent] = trustedProjectConfig
		DebugPrintf("Added parent directory as trusted project: %s\n", workspaceParent)
	}

	claudeConfig["projects"] = projects

	// 3. Add workspace configuration for sandbox environment
	workspaceConfig := map[string]interface{}{
		//  "cwd": workspacePath,
		"cwd": "/workspace",
	}
	claudeConfig["workspace"] = workspaceConfig

	DebugPrintf("Modified .claude.json configuration for sandbox use\n")

	// Marshal the modified configuration back to JSON
	modifiedJSON, err := json.MarshalIndent(claudeConfig, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal modified .claude.json: %w", err)
	}

	// Create temporary file for the modified content
	tempFile, err := os.CreateTemp("", "dispense-claude-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer tempFile.Close()

	// Write the modified content to the temporary file
	_, err = tempFile.Write(modifiedJSON)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to write to temporary file: %w", err)
	}

	DebugPrintf("Created modified .claude.json in temporary file: %s\n", tempFile.Name())
	return tempFile.Name(), nil
}

// GetAnthropicAPIKey gets the Anthropic API key from environment variable or Claude config
func GetAnthropicAPIKey() (string, error) {
	// First try environment variable
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		return apiKey, nil
	}

	// Try app-specific config first
	if apiKey, err := loadAppSpecificClaudeAPIKey(); err == nil && apiKey != "" {
		return apiKey, nil
	}

	// Try to read from Claude config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Try reading from Claude config file
	claudeConfigPath := filepath.Join(homeDir, ".claude", "config.toml")
	if content, err := os.ReadFile(claudeConfigPath); err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "api_key") && strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					apiKey := strings.TrimSpace(strings.Trim(parts[1], `"`))
					if apiKey != "" {
						return apiKey, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("ANTHROPIC_API_KEY not found in environment or Claude config")
}
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cli/internal/core/errors"
)

const (
	ConfigDirName  = ".dispense"
	APIKeyFileName = "api_key"
)

// Manager handles all configuration operations
type Manager struct{}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	return &Manager{}
}

// GetConfigDir returns the path to the .dispense directory in the user's home folder
func (m *Manager) GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, errors.ErrCodeConfigInvalid, "failed to get home directory")
	}
	
	configDir := filepath.Join(homeDir, ConfigDirName)
	return configDir, nil
}

// EnsureConfigDir creates the .dispense directory if it doesn't exist
func (m *Manager) EnsureConfigDir() error {
	configDir, err := m.GetConfigDir()
	if err != nil {
		return err
	}
	
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return errors.Wrap(err, errors.ErrCodeConfigInvalid, "failed to create config directory")
		}
	}
	
	return nil
}

// GetAPIKeyPath returns the full path to the API key file
func (m *Manager) GetAPIKeyPath() (string, error) {
	configDir, err := m.GetConfigDir()
	if err != nil {
		return "", err
	}
	
	return filepath.Join(configDir, APIKeyFileName), nil
}

// LoadAPIKey loads the API key from environment variable or config file
func (m *Manager) LoadAPIKey() (string, error) {
	// First try environment variable
	if apiKey := os.Getenv("DAYTONA_API_KEY"); apiKey != "" {
		return apiKey, nil
	}

	apiKeyPath, err := m.GetAPIKeyPath()
	if err != nil {
		return "", err
	}

	// Check if the API key file exists
	if _, err := os.Stat(apiKeyPath); os.IsNotExist(err) {
		return "", errors.New(errors.ErrCodeAPIKeyMissing, "API key not found in environment variable DAYTONA_API_KEY or config file")
	}

	// Read the API key from the file
	keyBytes, err := os.ReadFile(apiKeyPath)
	if err != nil {
		return "", errors.Wrap(err, errors.ErrCodeAPIKeyInvalid, "failed to read API key file")
	}

	apiKey := strings.TrimSpace(string(keyBytes))
	if apiKey == "" {
		return "", errors.New(errors.ErrCodeAPIKeyInvalid, "API key is empty")
	}

	return apiKey, nil
}

// SaveAPIKey saves the API key to the config file
func (m *Manager) SaveAPIKey(apiKey string) error {
	// Ensure the config directory exists
	if err := m.EnsureConfigDir(); err != nil {
		return err
	}
	
	apiKeyPath, err := m.GetAPIKeyPath()
	if err != nil {
		return err
	}
	
	// Write the API key to the file
	if err := os.WriteFile(apiKeyPath, []byte(apiKey), 0600); err != nil {
		return errors.Wrap(err, errors.ErrCodeConfigInvalid, "failed to write API key file")
	}
	
	return nil
}

// PromptForAPIKey prompts the user to enter their Daytona API key
func (m *Manager) PromptForAPIKey() (string, error) {
	fmt.Print("Please enter your Daytona API Key: ")
	
	reader := bufio.NewReader(os.Stdin)
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return "", errors.Wrap(err, errors.ErrCodeInputInvalid, "failed to read input")
	}
	
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", errors.New(errors.ErrCodeInputInvalid, "API key cannot be empty")
	}
	
	return apiKey, nil
}

// GetOrPromptAPIKey gets the API key from storage or prompts the user if not found
func (m *Manager) GetOrPromptAPIKey() (string, error) {
	// Try to load existing API key
	apiKey, err := m.LoadAPIKey()
	if err == nil {
		return apiKey, nil
	}

	// If API key not found, prompt user
	fmt.Println("Daytona API Key not found in configuration.")
	apiKey, err = m.PromptForAPIKey()
	if err != nil {
		return "", err
	}

	// Save the API key for future use
	if err := m.SaveAPIKey(apiKey); err != nil {
		return "", errors.Wrap(err, errors.ErrCodeConfigInvalid, "failed to save API key")
	}

	fmt.Println("API key saved successfully!")
	return apiKey, nil
}

// LoadAPIKeyNonInteractive gets the API key from storage without prompting
// Returns an error if no API key is found, avoiding any user interaction
func (m *Manager) LoadAPIKeyNonInteractive() (string, error) {
	return m.LoadAPIKey()
}

// LoadAnthropicAPIKey loads the Anthropic API key from environment variable or Claude config
func (m *Manager) LoadAnthropicAPIKey() (string, error) {
	// First try environment variable
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		return apiKey, nil
	}

	// Try app-specific config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, errors.ErrCodeConfigInvalid, "failed to get home directory")
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

	return "", errors.New(errors.ErrCodeAPIKeyMissing, "ANTHROPIC_API_KEY not found in environment variable or Claude config")
}
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ConfigDirName = ".dispense"
	APIKeyFileName = "api_key"
)

// GetConfigDir returns the path to the .dispense directory in the user's home folder
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	
	configDir := filepath.Join(homeDir, ConfigDirName)
	return configDir, nil
}

// EnsureConfigDir creates the .dispense directory if it doesn't exist
func EnsureConfigDir() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}
	
	return nil
}

// GetAPIKeyPath returns the full path to the API key file
func GetAPIKeyPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	
	return filepath.Join(configDir, APIKeyFileName), nil
}

// LoadAPIKey loads the API key from environment variable or config file
func LoadAPIKey() (string, error) {
	// First try environment variable
	if apiKey := os.Getenv("DAYTONA_API_KEY"); apiKey != "" {
		return apiKey, nil
	}

	apiKeyPath, err := GetAPIKeyPath()
	if err != nil {
		return "", err
	}

	// Check if the API key file exists
	if _, err := os.Stat(apiKeyPath); os.IsNotExist(err) {
		return "", fmt.Errorf("API key not found in environment variable DAYTONA_API_KEY or config file")
	}

	// Read the API key from the file
	keyBytes, err := os.ReadFile(apiKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read API key file: %w", err)
	}

	apiKey := strings.TrimSpace(string(keyBytes))
	if apiKey == "" {
		return "", fmt.Errorf("API key is empty")
	}

	return apiKey, nil
}

// SaveAPIKey saves the API key to the config file
func SaveAPIKey(apiKey string) error {
	// Ensure the config directory exists
	if err := EnsureConfigDir(); err != nil {
		return err
	}
	
	apiKeyPath, err := GetAPIKeyPath()
	if err != nil {
		return err
	}
	
	// Write the API key to the file
	if err := os.WriteFile(apiKeyPath, []byte(apiKey), 0600); err != nil {
		return fmt.Errorf("failed to write API key file: %w", err)
	}
	
	return nil
}

// PromptForAPIKey prompts the user to enter their Daytona API key
func PromptForAPIKey() (string, error) {
	fmt.Print("Please enter your Daytona API Key: ")
	
	reader := bufio.NewReader(os.Stdin)
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", fmt.Errorf("API key cannot be empty")
	}
	
	return apiKey, nil
}

// GetOrPromptAPIKey gets the API key from storage or prompts the user if not found
func GetOrPromptAPIKey() (string, error) {
	// Try to load existing API key
	apiKey, err := LoadAPIKey()
	if err == nil {
		return apiKey, nil
	}

	// If API key not found, prompt user
	fmt.Println("Daytona API Key not found in configuration.")
	apiKey, err = PromptForAPIKey()
	if err != nil {
		return "", err
	}

	// Save the API key for future use
	if err := SaveAPIKey(apiKey); err != nil {
		return "", fmt.Errorf("failed to save API key: %w", err)
	}

	fmt.Println("API key saved successfully!")
	return apiKey, nil
}

// GetAPIKeyNonInteractive gets the API key from storage without prompting
// Returns an error if no API key is found, avoiding any user interaction
func GetAPIKeyNonInteractive() (string, error) {
	return LoadAPIKey()
}


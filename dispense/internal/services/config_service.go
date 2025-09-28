package services

import (
	"cli/internal/core/config"
)

// ConfigManager wraps the core config manager to implement the interface
type ConfigManager struct {
	manager *config.Manager
}

// NewConfigManager creates a new config manager service
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		manager: config.NewManager(),
	}
}

// LoadAPIKey loads the API key from environment variable or config file
func (c *ConfigManager) LoadAPIKey() (string, error) {
	return c.manager.LoadAPIKey()
}

// SaveAPIKey saves the API key to the config file
func (c *ConfigManager) SaveAPIKey(apiKey string) error {
	return c.manager.SaveAPIKey(apiKey)
}

// GetOrPromptAPIKey gets the API key from storage or prompts the user if not found
func (c *ConfigManager) GetOrPromptAPIKey() (string, error) {
	return c.manager.GetOrPromptAPIKey()
}

// LoadAPIKeyNonInteractive gets the API key from storage without prompting
func (c *ConfigManager) LoadAPIKeyNonInteractive() (string, error) {
	return c.manager.LoadAPIKeyNonInteractive()
}

// PromptForAPIKey prompts the user to enter their Daytona API key
func (c *ConfigManager) PromptForAPIKey() (string, error) {
	return c.manager.PromptForAPIKey()
}
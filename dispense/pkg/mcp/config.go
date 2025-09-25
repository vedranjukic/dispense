package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Config holds the configuration for the MCP server
type Config struct {
	// BinaryPath is the path to the dispense binary
	BinaryPath string `json:"binary_path"`

	// DefaultTimeout is the default timeout for command execution
	DefaultTimeout time.Duration `json:"default_timeout"`

	// LongTimeout is the timeout for long-running operations like wait
	LongTimeout time.Duration `json:"long_timeout"`

	// LogLevel controls the verbosity of logging
	LogLevel string `json:"log_level"`

	// MaxConcurrentOps limits the number of concurrent operations
	MaxConcurrentOps int `json:"max_concurrent_ops"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	// Since we're now bundled in the dispense binary, we can determine our own path
	execPath, _ := os.Executable()

	return &Config{
		BinaryPath:       execPath,
		DefaultTimeout:   30 * time.Second,
		LongTimeout:      10 * time.Minute, // For wait operations
		LogLevel:         "info",
		MaxConcurrentOps: 10,
	}
}

// LoadFromEnv loads configuration from environment variables
func (c *Config) LoadFromEnv() {
	if path := os.Getenv("DISPENSE_BINARY_PATH"); path != "" {
		c.BinaryPath = path
	}

	if timeout := os.Getenv("DISPENSE_DEFAULT_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			c.DefaultTimeout = d
		}
	}

	if timeout := os.Getenv("DISPENSE_LONG_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			c.LongTimeout = d
		}
	}

	if level := os.Getenv("DISPENSE_LOG_LEVEL"); level != "" {
		c.LogLevel = level
	}
}

// Validate checks if the configuration is valid and the binary is accessible
func (c *Config) Validate() error {
	// Check if binary exists and is executable
	if !filepath.IsAbs(c.BinaryPath) {
		// Check if it's in PATH
		if _, err := exec.LookPath(c.BinaryPath); err != nil {
			return fmt.Errorf("dispense binary not found in PATH: %w", err)
		}
	} else {
		// Check absolute path
		if _, err := os.Stat(c.BinaryPath); err != nil {
			return fmt.Errorf("dispense binary not found at %s: %w", c.BinaryPath, err)
		}
	}

	return nil
}

// GetConfigPath returns the path where the MCP config file should be stored
func GetConfigPath() string {
	// Use XDG_CONFIG_HOME if set, otherwise default to ~/.config
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		homeDir, _ := os.UserHomeDir()
		configDir = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(configDir, "dispense", "mcp.json")
}

// LoadFromFile loads configuration from a JSON file
func (c *Config) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, use defaults
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

// SaveToFile saves the current configuration to a JSON file
func (c *Config) SaveToFile(path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadConfig loads configuration from file, environment variables, and defaults
func LoadConfig() *Config {
	config := DefaultConfig()

	// Try to load from config file
	configPath := GetConfigPath()
	config.LoadFromFile(configPath)

	// Override with environment variables
	config.LoadFromEnv()

	return config
}
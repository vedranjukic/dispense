package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigDir(t *testing.T) {
	configDir, err := GetConfigDir()
	if err != nil {
		t.Fatalf("GetConfigDir() failed: %v", err)
	}
	
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}
	
	expected := filepath.Join(homeDir, ConfigDirName)
	if configDir != expected {
		t.Errorf("GetConfigDir() = %v, want %v", configDir, expected)
	}
}

func TestEnsureConfigDir(t *testing.T) {
	// Clean up any existing test directory
	configDir, _ := GetConfigDir()
	os.RemoveAll(configDir)
	
	err := EnsureConfigDir()
	if err != nil {
		t.Fatalf("EnsureConfigDir() failed: %v", err)
	}
	
	// Check if directory was created
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Errorf("Config directory was not created")
	}
	
	// Clean up
	os.RemoveAll(configDir)
}

func TestAPIKeyOperations(t *testing.T) {
	// Clean up any existing test directory
	configDir, _ := GetConfigDir()
	os.RemoveAll(configDir)
	defer os.RemoveAll(configDir)
	
	// Test saving and loading API key
	testKey := "test-api-key-12345"
	
	err := SaveAPIKey(testKey)
	if err != nil {
		t.Fatalf("SaveAPIKey() failed: %v", err)
	}
	
	loadedKey, err := LoadAPIKey()
	if err != nil {
		t.Fatalf("LoadAPIKey() failed: %v", err)
	}
	
	if loadedKey != testKey {
		t.Errorf("LoadAPIKey() = %v, want %v", loadedKey, testKey)
	}
}

func TestLoadAPIKeyNotFound(t *testing.T) {
	// Clean up any existing test directory
	configDir, _ := GetConfigDir()
	os.RemoveAll(configDir)
	defer os.RemoveAll(configDir)
	
	_, err := LoadAPIKey()
	if err == nil {
		t.Errorf("LoadAPIKey() should fail when file doesn't exist")
	}
}


package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cli/pkg/sandbox"

	"github.com/asdine/storm/v3"
)

// LocalSandbox represents a local sandbox stored in the database
type LocalSandbox struct {
	ID          string                 `storm:"id" json:"id"`
	Name        string                 `storm:"unique" json:"name"`
	ContainerID string                 `json:"container_id"`
	Image       string                 `json:"image"`
	State       string                 `json:"state"`
	Group       string                 `json:"group,omitempty"` // Optional group parameter for querying
	Model       string                 `json:"model,omitempty"` // Optional model parameter
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Ports       map[string]string      `json:"ports"`
	Metadata    map[string]interface{} `json:"metadata"`
	TaskData    string                 `json:"task_data,omitempty"` // JSON serialized task data
}

// SandboxDB provides database operations for local sandboxes
type SandboxDB struct {
	db *storm.DB
}

var (
	sandboxDBInstance *SandboxDB
	sandboxDBMutex    sync.RWMutex
	initError         error
)

// NewSandboxDB creates a new sandbox database instance using singleton pattern
func NewSandboxDB() (*SandboxDB, error) {
	sandboxDBMutex.Lock()
	defer sandboxDBMutex.Unlock()

	// Return existing instance if already initialized
	if sandboxDBInstance != nil {
		return sandboxDBInstance, nil
	}

	// Return previous initialization error if any
	if initError != nil {
		return nil, initError
	}

	// Create database directory in user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		initError = fmt.Errorf("failed to get user home directory: %w", err)
		return nil, initError
	}

	dbDir := filepath.Join(homeDir, ".dispense")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		initError = fmt.Errorf("failed to create database directory: %w", err)
		return nil, initError
	}

	dbPath := filepath.Join(dbDir, "sandboxes.db")

	// Try to open the database with retry logic for timeout issues
	var db *storm.DB
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		db, err = storm.Open(dbPath)
		if err == nil {
			break
		}

		// Check if it's a timeout or lock error
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "locked") {
			// If this isn't the last attempt, wait a bit and try again
			if i < maxRetries-1 {
				time.Sleep(time.Duration(i+1) * 200 * time.Millisecond)
				continue
			}
		}

		// For non-timeout errors, don't retry
		break
	}

	if err != nil {
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "locked") {
			initError = fmt.Errorf("database is locked by another process. Please wait and try again, or use 'dispense db info' to check database status: %w", err)
		} else {
			initError = fmt.Errorf("failed to open database after %d attempts: %w", maxRetries, err)
		}
		return nil, initError
	}

	sandboxDBInstance = &SandboxDB{db: db}
	return sandboxDBInstance, nil
}

// GetSandboxDB returns the singleton instance if it exists, otherwise creates a new one
func GetSandboxDB() (*SandboxDB, error) {
	sandboxDBMutex.RLock()
	if sandboxDBInstance != nil {
		defer sandboxDBMutex.RUnlock()
		return sandboxDBInstance, nil
	}
	sandboxDBMutex.RUnlock()

	return NewSandboxDB()
}

// Close closes the database connection and resets the singleton
func (sdb *SandboxDB) Close() error {
	sandboxDBMutex.Lock()
	defer sandboxDBMutex.Unlock()

	if sdb.db != nil {
		err := sdb.db.Close()
		sandboxDBInstance = nil
		initError = nil // Reset the initialization error
		return err
	}
	return nil
}

// ResetSingleton resets the singleton instance (useful for testing or error recovery)
func ResetSingleton() {
	sandboxDBMutex.Lock()
	defer sandboxDBMutex.Unlock()

	if sandboxDBInstance != nil && sandboxDBInstance.db != nil {
		sandboxDBInstance.db.Close()
	}
	sandboxDBInstance = nil
	initError = nil
}

// Save saves a sandbox to the database
func (sdb *SandboxDB) Save(sandbox *LocalSandbox) error {
	sandbox.UpdatedAt = time.Now()
	if sandbox.CreatedAt.IsZero() {
		sandbox.CreatedAt = time.Now()
	}
	return sdb.db.Save(sandbox)
}

// GetByID retrieves a sandbox by ID
func (sdb *SandboxDB) GetByID(id string) (*LocalSandbox, error) {
	var sandbox LocalSandbox
	err := sdb.db.One("ID", id, &sandbox)
	if err != nil {
		return nil, err
	}
	return &sandbox, nil
}

// GetByName retrieves a sandbox by name
func (sdb *SandboxDB) GetByName(name string) (*LocalSandbox, error) {
	var sandbox LocalSandbox
	err := sdb.db.One("Name", name, &sandbox)
	if err != nil {
		return nil, err
	}
	return &sandbox, nil
}

// List retrieves all sandboxes
func (sdb *SandboxDB) List() ([]*LocalSandbox, error) {
	var sandboxes []*LocalSandbox
	err := sdb.db.All(&sandboxes)
	if err != nil {
		return nil, err
	}
	return sandboxes, nil
}

// ListByGroup retrieves all sandboxes in a specific group
func (sdb *SandboxDB) ListByGroup(group string) ([]*LocalSandbox, error) {
	var sandboxes []*LocalSandbox
	err := sdb.db.Find("Group", group, &sandboxes)
	if err != nil {
		return nil, err
	}
	return sandboxes, nil
}

// Delete removes a sandbox from the database
func (sdb *SandboxDB) Delete(id string) error {
	var sandbox LocalSandbox
	err := sdb.db.One("ID", id, &sandbox)
	if err != nil {
		return err
	}
	return sdb.db.DeleteStruct(&sandbox)
}

// DeleteByName removes a sandbox by name from the database
func (sdb *SandboxDB) DeleteByName(name string) error {
	var sandbox LocalSandbox
	err := sdb.db.One("Name", name, &sandbox)
	if err != nil {
		return err
	}
	return sdb.db.DeleteStruct(&sandbox)
}

// Update updates an existing sandbox
func (sdb *SandboxDB) Update(sandbox *LocalSandbox) error {
	sandbox.UpdatedAt = time.Now()
	return sdb.db.Update(sandbox)
}

// ToSandboxInfo converts a LocalSandbox to sandbox.SandboxInfo
func (ls *LocalSandbox) ToSandboxInfo() *sandbox.SandboxInfo {
	// Use container name from metadata for shell command, but keep user-friendly name for display
	containerName := ""
	if ls.Metadata != nil {
		if name, ok := ls.Metadata["container_name"].(string); ok {
			containerName = name
		}
	}

	shellCommand := fmt.Sprintf("docker exec -it %s /bin/bash", containerName)

	return &sandbox.SandboxInfo{
		ID:           ls.ID,
		Name:         ls.Name,
		Type:         sandbox.TypeLocal,
		State:        ls.State,
		ShellCommand: shellCommand,
		Metadata:     ls.Metadata,
	}
}

// FromSandboxInfo creates a LocalSandbox from sandbox.SandboxInfo
func FromSandboxInfo(info *sandbox.SandboxInfo, containerID, image, taskData string) *LocalSandbox {
	ports := make(map[string]string)
	if portData, exists := info.Metadata["ports"]; exists {
		if portMap, ok := portData.(map[string]string); ok {
			ports = portMap
		}
	}

	// Extract group from metadata if present
	group := ""
	if groupData, exists := info.Metadata["group"]; exists {
		if groupStr, ok := groupData.(string); ok {
			group = groupStr
		}
	}

	// Extract model from metadata if present
	model := ""
	if modelData, exists := info.Metadata["model"]; exists {
		if modelStr, ok := modelData.(string); ok {
			model = modelStr
		}
	}

	return &LocalSandbox{
		ID:          info.ID,
		Name:        info.Name,
		ContainerID: containerID,
		Image:       image,
		State:       info.State,
		Group:       group,
		Model:       model,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Ports:       ports,
		Metadata:    info.Metadata,
		TaskData:    taskData,
	}
}
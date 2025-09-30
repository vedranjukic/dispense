package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"syscall"

	"cli/pkg/sandbox"

	"github.com/asdine/storm/v3"
)

// LocalSandbox represents a local sandbox stored in the database
type LocalSandbox struct {
	ID            string                 `storm:"id" json:"id"`
	Name          string                 `storm:"unique" json:"name"`
	ContainerID   string                 `json:"container_id"`
	Image         string                 `json:"image"`
	State         string                 `json:"state"`
	Group         string                 `json:"group,omitempty"`         // Optional group parameter for querying
	Model         string                 `json:"model,omitempty"`         // Optional model parameter
	ProjectSource string                 `json:"project_source,omitempty"` // Project source (SourceDirectory or GitHub repo URL)
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	Ports         map[string]string      `json:"ports"`
	Metadata      map[string]interface{} `json:"metadata"`
	TaskData      string                 `json:"task_data,omitempty"` // JSON serialized task data
}

// SandboxDB provides database operations for local sandboxes
type SandboxDB struct {
	dbPath string
}

var (
	dbPathOnce sync.Once
	dbPath     string
	dbPathErr  error
)

// NewSandboxDB creates a new sandbox database instance
func NewSandboxDB() (*SandboxDB, error) {
	path, err := getDBPath()
	if err != nil {
		return nil, err
	}
	return &SandboxDB{dbPath: path}, nil
}

// GetSandboxDB creates a new sandbox database instance
func GetSandboxDB() (*SandboxDB, error) {
	return NewSandboxDB()
}

// getDBPath returns the database file path, initializing it once
func getDBPath() (string, error) {
	dbPathOnce.Do(func() {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			dbPathErr = fmt.Errorf("failed to get user home directory: %w", err)
			return
		}

		dbDir := filepath.Join(homeDir, ".dispense")
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			dbPathErr = fmt.Errorf("failed to create database directory: %w", err)
			return
		}

		dbPath = filepath.Join(dbDir, "sandboxes.db")
	})

	return dbPath, dbPathErr
}

// openDB opens a database connection with retry logic
func (sdb *SandboxDB) openDB() (*storm.DB, error) {
	maxRetries := 3
	baseDelay := 50 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		db, err := storm.Open(sdb.dbPath)
		if err == nil {
			return db, nil
		}

		// Check if it's a timeout or lock error
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "locked") {
			// If this isn't the last attempt, wait with exponential backoff
			if i < maxRetries-1 {
				delay := baseDelay * time.Duration(1<<uint(i)) // exponential backoff
				time.Sleep(delay)
				continue
			}
		}

		// For non-timeout errors or final retry, return error
		if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "locked") {
			return nil, fmt.Errorf("database is temporarily locked by another process. Please try again in a moment: %w", err)
		}
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return nil, fmt.Errorf("failed to open database after %d attempts", maxRetries)
}

// withDB executes a function with a database connection, handling open/close automatically
func (sdb *SandboxDB) withDB(fn func(*storm.DB) error) error {
	db, err := sdb.openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	return fn(db)
}

// withWriteLock executes a function with file locking for write operations
func (sdb *SandboxDB) withWriteLock(fn func(*storm.DB) error) error {
	// Create lock file for write operations
	lockPath := sdb.dbPath + ".lock"
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create lock file: %w", err)
	}
	defer func() {
		lockFile.Close()
		os.Remove(lockPath) // Clean up lock file
	}()

	// Acquire exclusive file lock with timeout
	maxRetries := 3
	baseDelay := 100 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}

		if err == syscall.EWOULDBLOCK || err == syscall.EAGAIN {
			if i < maxRetries-1 {
				delay := baseDelay * time.Duration(1<<uint(i))
				time.Sleep(delay)
				continue
			}
			return fmt.Errorf("database is busy, please try again")
		}

		return fmt.Errorf("failed to acquire write lock: %w", err)
	}

	// Execute operation with database
	return sdb.withDB(fn)
}

// GetDBPath returns the database file path (for external utilities)
func GetDBPath() (string, error) {
	return getDBPath()
}

// Save saves a sandbox to the database
func (sdb *SandboxDB) Save(sandbox *LocalSandbox) error {
	sandbox.UpdatedAt = time.Now()
	if sandbox.CreatedAt.IsZero() {
		sandbox.CreatedAt = time.Now()
	}

	return sdb.withWriteLock(func(db *storm.DB) error {
		return db.Save(sandbox)
	})
}

// GetByID retrieves a sandbox by ID
func (sdb *SandboxDB) GetByID(id string) (*LocalSandbox, error) {
	var sandbox LocalSandbox

	err := sdb.withDB(func(db *storm.DB) error {
		return db.One("ID", id, &sandbox)
	})

	if err != nil {
		return nil, err
	}
	return &sandbox, nil
}

// GetByName retrieves a sandbox by name
func (sdb *SandboxDB) GetByName(name string) (*LocalSandbox, error) {
	var sandbox LocalSandbox

	err := sdb.withDB(func(db *storm.DB) error {
		return db.One("Name", name, &sandbox)
	})

	if err != nil {
		return nil, err
	}
	return &sandbox, nil
}

// List retrieves all sandboxes
func (sdb *SandboxDB) List() ([]*LocalSandbox, error) {
	var sandboxes []*LocalSandbox

	err := sdb.withDB(func(db *storm.DB) error {
		return db.All(&sandboxes)
	})

	if err != nil {
		return nil, err
	}
	return sandboxes, nil
}

// ListByGroup retrieves all sandboxes in a specific group
func (sdb *SandboxDB) ListByGroup(group string) ([]*LocalSandbox, error) {
	var sandboxes []*LocalSandbox

	err := sdb.withDB(func(db *storm.DB) error {
		return db.Find("Group", group, &sandboxes)
	})

	if err != nil {
		return nil, err
	}
	return sandboxes, nil
}

// Delete removes a sandbox from the database
func (sdb *SandboxDB) Delete(id string) error {
	return sdb.withWriteLock(func(db *storm.DB) error {
		var sandbox LocalSandbox
		err := db.One("ID", id, &sandbox)
		if err != nil {
			return err
		}
		return db.DeleteStruct(&sandbox)
	})
}

// DeleteByName removes a sandbox by name from the database
func (sdb *SandboxDB) DeleteByName(name string) error {
	return sdb.withWriteLock(func(db *storm.DB) error {
		var sandbox LocalSandbox
		err := db.One("Name", name, &sandbox)
		if err != nil {
			return err
		}
		return db.DeleteStruct(&sandbox)
	})
}

// Update updates an existing sandbox
func (sdb *SandboxDB) Update(sandbox *LocalSandbox) error {
	sandbox.UpdatedAt = time.Now()

	return sdb.withWriteLock(func(db *storm.DB) error {
		return db.Update(sandbox)
	})
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
		ID:            ls.ID,
		Name:          ls.Name,
		Type:          sandbox.TypeLocal,
		State:         ls.State,
		ShellCommand:  shellCommand,
		ProjectSource: ls.ProjectSource,
		Metadata:      ls.Metadata,
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
		ID:            info.ID,
		Name:          info.Name,
		ContainerID:   containerID,
		Image:         image,
		State:         info.State,
		Group:         group,
		Model:         model,
		ProjectSource: info.ProjectSource,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Ports:         ports,
		Metadata:      info.Metadata,
		TaskData:      taskData,
	}
}
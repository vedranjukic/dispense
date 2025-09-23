package daemon

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Embed the daemon binary (will be populated during build)
//go:embed daemon-linux-amd64
var daemonBinaryLinux []byte

// EmbeddedDaemon handles the embedded daemon binary
type EmbeddedDaemon struct {
	extractedPath string
}

// NewEmbeddedDaemon creates a new embedded daemon instance
func NewEmbeddedDaemon() *EmbeddedDaemon {
	return &EmbeddedDaemon{}
}

// Extract extracts the embedded daemon binary to a temporary location
func (d *EmbeddedDaemon) Extract() error {
	// Create temp directory for daemon binary
	tempDir, err := os.MkdirTemp("", "dispense-daemon-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	daemonPath := filepath.Join(tempDir, "daemon")

	// Write the embedded binary to temporary file
	err = os.WriteFile(daemonPath, daemonBinaryLinux, 0755)
	if err != nil {
		os.RemoveAll(tempDir)
		return fmt.Errorf("failed to write daemon binary: %w", err)
	}

	d.extractedPath = daemonPath
	return nil
}

// GetPath returns the path to the extracted daemon binary
func (d *EmbeddedDaemon) GetPath() string {
	return d.extractedPath
}

// Start starts the daemon process
func (d *EmbeddedDaemon) Start() (*exec.Cmd, error) {
	if d.extractedPath == "" {
		return nil, fmt.Errorf("daemon not extracted yet, call Extract() first")
	}

	// Verify binary exists and is executable
	if err := d.verifyBinary(); err != nil {
		return nil, err
	}

	// Start the daemon
	cmd := exec.Command(d.extractedPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start daemon: %w", err)
	}

	return cmd, nil
}

// verifyBinary verifies the extracted binary is valid and executable
func (d *EmbeddedDaemon) verifyBinary() error {
	stat, err := os.Stat(d.extractedPath)
	if err != nil {
		return fmt.Errorf("daemon binary not found: %w", err)
	}

	// Check if file is executable
	mode := stat.Mode()
	if mode&0111 == 0 {
		return fmt.Errorf("daemon binary is not executable")
	}

	return nil
}

// Cleanup removes the extracted daemon binary and temp directory
func (d *EmbeddedDaemon) Cleanup() error {
	if d.extractedPath == "" {
		return nil
	}

	tempDir := filepath.Dir(d.extractedPath)
	return os.RemoveAll(tempDir)
}

// IsSupported checks if the current platform supports the embedded daemon
func (d *EmbeddedDaemon) IsSupported() bool {
	// The embedded daemon is compiled for Linux AMD64
	// It can run on Linux systems or via compatibility layers
	return runtime.GOOS == "linux" ||
		   runtime.GOOS == "darwin" || // macOS can run Linux binaries via Docker/containers
		   runtime.GOOS == "windows"   // Windows can run Linux binaries via WSL
}

// GetVersion returns version info about the embedded daemon
func (d *EmbeddedDaemon) GetVersion() string {
	return "embedded-linux-amd64"
}

// Size returns the size of the embedded daemon binary in bytes
func (d *EmbeddedDaemon) Size() int {
	return len(daemonBinaryLinux)
}
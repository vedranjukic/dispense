package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// BuildInfo contains build metadata
type BuildInfo struct {
	Version   string
	GitCommit string
	BuildTime string
	GoVersion string
}

func main() {
	buildInfo := getBuildInfo()

	// Ensure dist directory exists
	if err := os.MkdirAll("../dist/dispense", 0755); err != nil {
		fmt.Printf("Error creating dist directory: %v\n", err)
		os.Exit(1)
	}

	// Build daemon binary first
	if err := buildDaemonBinary(); err != nil {
		fmt.Printf("Error building daemon binary: %v\n", err)
		os.Exit(1)
	}

	// Define build targets
	targets := []struct {
		os   string
		arch string
		name string
	}{
		{"linux", "amd64", "dispense-linux-amd64"},
		{"darwin", "arm64", "dispense-darwin-arm64"},
	}

	fmt.Println("Building Dispense binaries for cross-platform distribution...")
	fmt.Printf("Version: %s\n", buildInfo.Version)
	fmt.Printf("Git Commit: %s\n", buildInfo.GitCommit)
	fmt.Printf("Build Time: %s\n", buildInfo.BuildTime)
	fmt.Printf("Go Version: %s\n", buildInfo.GoVersion)
	fmt.Println()

	// Build for each target
	for _, target := range targets {
		fmt.Printf("Building %s (%s/%s)...\n", target.name, target.os, target.arch)
		
		// Set environment variables for cross-compilation
		env := os.Environ()
		env = append(env, fmt.Sprintf("GOOS=%s", target.os))
		env = append(env, fmt.Sprintf("GOARCH=%s", target.arch))
		env = append(env, "CGO_ENABLED=0") // Disable CGO for static binaries

		// Get all Go files in cmd directory
		cmdFiles, err := filepath.Glob("./cmd/*.go")
		if err != nil {
			fmt.Printf("Error finding cmd files: %v\n", err)
			os.Exit(1)
		}

		// Build command
		args := []string{"build", "-ldflags", buildLdflags(buildInfo), "-o", filepath.Join("../dist/dispense", target.name)}
		args = append(args, cmdFiles...)
		cmd := exec.Command("go", args...)

		cmd.Env = env
		cmd.Dir = "."

		// Run the build
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Error building %s: %v\n", target.name, err)
			fmt.Printf("Output: %s\n", string(output))
			os.Exit(1)
		}

		// Get file size for verification
		filePath := filepath.Join("../dist/dispense", target.name)
		if stat, err := os.Stat(filePath); err == nil {
			fmt.Printf("✓ Built %s (%d bytes)\n", target.name, stat.Size())
		}
	}

	fmt.Println("\nAll builds completed successfully!")
	fmt.Println("Binaries are available in the ../dist/dispense/ directory:")
	
	// List built files
	files, err := filepath.Glob("../dist/dispense/dispense-*")
	if err == nil {
		for _, file := range files {
			if stat, err := os.Stat(file); err == nil {
				fmt.Printf("  - %s (%d bytes)\n", filepath.Base(file), stat.Size())
			}
		}
	}
}

func getBuildInfo() BuildInfo {
	version := getGitTag()
	if version == "" {
		version = "dev"
	}

	gitCommit := getGitCommit()
	if gitCommit == "" {
		gitCommit = "unknown"
	}

	return BuildInfo{
		Version:   version,
		GitCommit: gitCommit,
		BuildTime: time.Now().Format(time.RFC3339),
		GoVersion: runtime.Version(),
	}
}

func getGitTag() string {
	cmd := exec.Command("git", "describe", "--tags", "--exact-match", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func getGitCommit() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func buildLdflags(info BuildInfo) string {
	return fmt.Sprintf(
		"-X main.Version=%s -X main.GitCommit=%s -X main.BuildTime=%s",
		info.Version, info.GitCommit, info.BuildTime,
	)
}

func buildDaemonBinary() error {
	fmt.Println("Building daemon binary (Linux AMD64)...")

	// Ensure daemon embed directory exists
	embedDir := "pkg/daemon"
	if err := os.MkdirAll(embedDir, 0755); err != nil {
		return fmt.Errorf("failed to create embed directory: %w", err)
	}

	daemonBinaryPath := filepath.Join(embedDir, "daemon-linux-amd64")

	// Build daemon for Linux AMD64
	cmd := exec.Command("go", "build",
		"-ldflags", "-s -w", // Strip debug info for smaller binary
		"-o", daemonBinaryPath,
		"./cmd/main.go")

	// Set environment for Linux AMD64 cross-compilation
	env := os.Environ()
	env = append(env, "GOOS=linux")
	env = append(env, "GOARCH=amd64")
	env = append(env, "CGO_ENABLED=0") // Static binary
	cmd.Env = env
	cmd.Dir = "../daemon" // Build from daemon directory

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to build daemon: %w\nOutput: %s", err, string(output))
	}

	// Verify the binary was created
	if stat, err := os.Stat(daemonBinaryPath); err != nil {
		return fmt.Errorf("daemon binary not found after build: %w", err)
	} else {
		fmt.Printf("✓ Built daemon binary (%d bytes)\n", stat.Size())
	}

	return nil
}

package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"cli/pkg/daemon"

	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the embedded daemon",
	Long:  `Commands to manage the embedded daemon binary.`,
}

var startDaemonCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the embedded daemon",
	Long:  `Extract and start the embedded daemon binary.`,
	Run: func(cmd *cobra.Command, args []string) {
		embeddedDaemon := daemon.NewEmbeddedDaemon()

		// Check if daemon is supported on current platform
		if !embeddedDaemon.IsSupported() {
			fmt.Fprintf(os.Stderr, "Error: Embedded daemon is not supported on this platform\n")
			os.Exit(1)
		}

		fmt.Printf("Starting embedded daemon (%s)...\n", embeddedDaemon.GetVersion())
		fmt.Printf("Daemon binary size: %d bytes\n", embeddedDaemon.Size())

		// Extract the daemon binary
		fmt.Println("Extracting daemon binary...")
		if err := embeddedDaemon.Extract(); err != nil {
			fmt.Fprintf(os.Stderr, "Error extracting daemon: %s\n", err)
			os.Exit(1)
		}

		// Cleanup on exit
		defer func() {
			fmt.Println("Cleaning up daemon binary...")
			if err := embeddedDaemon.Cleanup(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to cleanup daemon binary: %s\n", err)
			}
		}()

		// Start the daemon
		fmt.Printf("Starting daemon from: %s\n", embeddedDaemon.GetPath())
		daemonProcess, err := embeddedDaemon.Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error starting daemon: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("Daemon started with PID: %d\n", daemonProcess.Process.Pid)
		fmt.Println("Daemon is running on port 28080")
		fmt.Println("Press Ctrl+C to stop the daemon...")

		// Set up signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Wait for shutdown signal
		<-sigChan
		fmt.Println("\nShutdown signal received, stopping daemon...")

		// Stop the daemon process
		if err := daemonProcess.Process.Signal(syscall.SIGTERM); err != nil {
			fmt.Printf("Warning: Failed to send SIGTERM to daemon: %s\n", err)
			// Force kill if SIGTERM fails
			if err := daemonProcess.Process.Kill(); err != nil {
				fmt.Printf("Warning: Failed to kill daemon process: %s\n", err)
			}
		}

		// Wait for daemon to exit (with timeout)
		done := make(chan error, 1)
		go func() {
			done <- daemonProcess.Wait()
		}()

		select {
		case err := <-done:
			if err != nil {
				fmt.Printf("Daemon stopped with error: %s\n", err)
			} else {
				fmt.Println("Daemon stopped successfully")
			}
		case <-time.After(5 * time.Second):
			fmt.Println("Daemon did not stop within 5 seconds, force killing...")
			if err := daemonProcess.Process.Kill(); err != nil {
				fmt.Printf("Warning: Failed to force kill daemon: %s\n", err)
			}
		}
	},
}

var infoDaemonCmd = &cobra.Command{
	Use:   "info",
	Short: "Show embedded daemon information",
	Long:  `Display information about the embedded daemon binary.`,
	Run: func(cmd *cobra.Command, args []string) {
		embeddedDaemon := daemon.NewEmbeddedDaemon()

		fmt.Println("Embedded Daemon Information:")
		fmt.Printf("  Version: %s\n", embeddedDaemon.GetVersion())
		fmt.Printf("  Binary size: %d bytes\n", embeddedDaemon.Size())
		fmt.Printf("  Supported on current platform: %t\n", embeddedDaemon.IsSupported())
		fmt.Printf("  Current platform: %s\n", getPlatformInfo())

		if embeddedDaemon.Size() > 0 {
			fmt.Println("  Status: ✓ Daemon binary is embedded and ready to use")
		} else {
			fmt.Println("  Status: ✗ Daemon binary is not embedded (build may have failed)")
		}
	},
}

func init() {
	// Add daemon subcommands
	daemonCmd.AddCommand(startDaemonCmd)
	daemonCmd.AddCommand(infoDaemonCmd)

	// Add daemon command to root
	rootCmd.AddCommand(daemonCmd)
}

func getPlatformInfo() string {
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"cli/pkg/sandbox"
	"cli/pkg/sandbox/local"
	"cli/pkg/sandbox/remote"
	"cli/pkg/utils"
	pb "cli/proto"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var waitCmd = &cobra.Command{
	Use:   "wait [sandbox-names...]",
	Short: "Wait for one or more sandboxes or groups to be ready",
	Long: `Wait for one or more sandboxes to be ready. You can specify:
- Individual sandbox names as arguments
- Group names using the --group flag

Examples:
  dispense wait sandbox1 sandbox2    # Wait for specific sandboxes
  dispense wait --group backend      # Wait for all sandboxes in backend group
  dispense wait --group frontend backend  # Wait for sandboxes in multiple groups
  dispense wait sandbox1 --group backend  # Wait for sandbox1 and all sandboxes in backend group`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get command flags
		groups, _ := cmd.Flags().GetStringSlice("group")

		// Validate that at least one sandbox or group is specified
		if len(args) == 0 && len(groups) == 0 {
			fmt.Printf("Error: Please specify at least one sandbox name or group (use --group flag)\n")
			fmt.Printf("Usage: %s\n", cmd.Use)
			return
		}

		// Resolve all sandboxes to wait for
		fmt.Printf("🔍 Resolving sandboxes...\n")
		sandboxes, err := resolveSandboxes(args, groups)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to resolve sandboxes: %s\n", err)
			return
		}

		if len(sandboxes) == 0 {
			fmt.Printf("❌ No sandboxes found to wait for\n")
			return
		}

		// Show what we're waiting for
		fmt.Printf("📋 Waiting for %d sandbox(es):\n", len(sandboxes))
		for _, sb := range sandboxes {
			fmt.Printf("  - %s (%s)\n", sb.Name, sb.Type)
		}
		fmt.Printf("\n")

		// Wait for all sandboxes to complete their tasks
		err = waitForSandboxes(sandboxes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Wait failed: %s\n", err)
			return
		}

		fmt.Printf("\n✅ All sandboxes have completed their tasks!\n")
	},
}

func init() {
	// Add flags for the wait command
	waitCmd.Flags().StringSliceP("group", "g", []string{}, "Group names to wait for (can be specified multiple times)")
}

// resolveSandboxes resolves sandbox names and group names to actual sandbox instances
func resolveSandboxes(sandboxNames []string, groupNames []string) ([]*sandbox.SandboxInfo, error) {
	var allSandboxes []*sandbox.SandboxInfo
	var providers []sandbox.Provider

	// Create providers
	localProvider, err := local.NewProvider()
	if err != nil {
		utils.DebugPrintf("Warning: Could not create local provider: %s\n", err)
	} else {
		providers = append(providers, localProvider)
	}

	remoteProvider, err := remote.NewProviderNonInteractive()
	if err != nil {
		utils.DebugPrintf("Warning: Could not create remote provider: %s\n", err)
	} else {
		providers = append(providers, remoteProvider)
	}

	// Collect all sandboxes from providers
	for _, provider := range providers {
		sandboxes, err := provider.List()
		if err != nil {
			utils.DebugPrintf("Warning: Error listing %s sandboxes: %s\n", provider.GetType(), err)
			continue
		}
		allSandboxes = append(allSandboxes, sandboxes...)
	}

	var targetSandboxes []*sandbox.SandboxInfo
	seenIDs := make(map[string]bool)

	// Add sandboxes by name
	for _, name := range sandboxNames {
		found := false
		for _, sb := range allSandboxes {
			if (sb.Name == name || sb.ID == name) && !seenIDs[sb.ID] {
				targetSandboxes = append(targetSandboxes, sb)
				seenIDs[sb.ID] = true
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("sandbox not found: %s", name)
		}
	}

	// Add sandboxes by group
	for _, groupName := range groupNames {
		groupFound := false
		for _, sb := range allSandboxes {
			if groupValue, exists := sb.Metadata["group"]; exists {
				if groupStr, ok := groupValue.(string); ok && groupStr == groupName && !seenIDs[sb.ID] {
					targetSandboxes = append(targetSandboxes, sb)
					seenIDs[sb.ID] = true
					groupFound = true
				}
			}
		}
		if !groupFound {
			return nil, fmt.Errorf("no sandboxes found in group: %s", groupName)
		}
	}

	return targetSandboxes, nil
}

// waitForSandboxes waits for all sandboxes to complete their tasks
func waitForSandboxes(sandboxes []*sandbox.SandboxInfo) error {
	fmt.Printf("⏳ Monitoring sandbox tasks...\n")

	// Keep track of sandbox statuses for display
	statuses := make([]pb.TaskStatusResponse_TaskState, len(sandboxes))

	for {
		allCompleted := true

		// Check status for all sandboxes
		for i, sb := range sandboxes {
			status, err := getSandboxTaskStatus(sb)
			if err != nil {
				utils.DebugPrintf("Failed to get status for %s: %s\n", sb.Name, err)
				continue
			}

			statuses[i] = status

			// Check if still working
			if status == pb.TaskStatusResponse_RUNNING || status == pb.TaskStatusResponse_PENDING {
				allCompleted = false
			}
		}

		// Clear the current display and show updated status
		clearLines(len(sandboxes))

		// Display status for each sandbox
		for i, sb := range sandboxes {
			emoji, statusText := formatTaskStatus(statuses[i])
			fmt.Printf("🔄 [%d/%d] %s: %s %s\n", i+1, len(sandboxes), sb.Name, emoji, statusText)
		}

		// Move cursor back up to overwrite on next iteration
		if !allCompleted {
			moveCursorUp(len(sandboxes))
			time.Sleep(2 * time.Second) // Poll every 2 seconds
		} else {
			break
		}
	}

	return nil
}

// clearLines clears the specified number of lines from the current position
func clearLines(numLines int) {
	for i := 0; i < numLines; i++ {
		fmt.Printf("\033[2K") // Clear current line
		if i < numLines-1 {
			fmt.Printf("\n") // Move to next line (except for the last one)
		}
	}
	fmt.Printf("\r") // Move cursor to beginning of line
}

// moveCursorUp moves the cursor up by the specified number of lines
func moveCursorUp(numLines int) {
	if numLines > 0 {
		fmt.Printf("\033[%dA", numLines) // Move cursor up
	}
}

// getSandboxTaskStatus gets the current task status for a sandbox
func getSandboxTaskStatus(sandboxInfo *sandbox.SandboxInfo) (pb.TaskStatusResponse_TaskState, error) {
	// Get daemon connection
	daemonAddr, cleanup, err := getDaemonConnection(sandboxInfo.Name)
	if err != nil {
		return pb.TaskStatusResponse_FAILED, fmt.Errorf("failed to get daemon connection: %w", err)
	}
	defer cleanup()

	// Connect to daemon
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(daemonAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return pb.TaskStatusResponse_FAILED, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer conn.Close()

	client := pb.NewAgentServiceClient(conn)

	// Get status of the most recent task
	status, err := client.GetTaskStatus(ctx, &pb.TaskStatusRequest{TaskId: ""})
	if err != nil {
		// If there's an error (e.g., no tasks found), consider it completed
		utils.DebugPrintf("GetTaskStatus error for %s: %v (treating as completed)\n", sandboxInfo.Name, err)
		return pb.TaskStatusResponse_COMPLETED, nil
	}

	return status.State, nil
}

// formatTaskStatus formats a task status for display
func formatTaskStatus(state pb.TaskStatusResponse_TaskState) (string, string) {
	switch state {
	case pb.TaskStatusResponse_PENDING:
		return "⏸️", "Pending"
	case pb.TaskStatusResponse_RUNNING:
		return "🟡", "Working"
	case pb.TaskStatusResponse_COMPLETED:
		return "✅", "Completed"
	case pb.TaskStatusResponse_FAILED:
		return "❌", "Failed"
	default:
		return "❓", "Unknown"
	}
}
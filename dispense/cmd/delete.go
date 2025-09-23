package main

import (
	"fmt"
	"os"
	"strings"

	"cli/pkg/sandbox"
	"cli/pkg/sandbox/local"
	"cli/pkg/sandbox/remote"
	"cli/pkg/utils"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [sandbox-name-or-id]",
	Short: "Delete a sandbox",
	Long:  `Delete a sandbox by name or ID, or delete all sandboxes with --all flag. This will remove the sandbox(es) and all associated resources.`,
	Args: func(cmd *cobra.Command, args []string) error {
		deleteAll, _ := cmd.Flags().GetBool("all")
		if deleteAll {
			// When --all flag is used, no arguments should be provided
			if len(args) > 0 {
				return fmt.Errorf("cannot specify sandbox name/id when using --all flag")
			}
			return nil
		}
		// When --all flag is not used, exactly one argument is required
		return cobra.ExactArgs(1)(cmd, args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		deleteAll, _ := cmd.Flags().GetBool("all")
		force, _ := cmd.Flags().GetBool("force")

		if deleteAll {
			utils.DebugPrintf("Deleting all sandboxes\n")
			deleteAllSandboxes(force)
			return
		}

		sandboxIdentifier := args[0]
		utils.DebugPrintf("Deleting sandbox: %s\n", sandboxIdentifier)

		// Try to find the sandbox (check both local and remote)
		var sandboxInfo *sandbox.SandboxInfo
		var provider sandbox.Provider
		var err error

		// First try local provider
		localProvider, err := local.NewProvider()
		if err == nil {
			sandboxInfo, err = findLocalSandbox(localProvider, sandboxIdentifier)
			if err == nil {
				provider = localProvider
				utils.DebugPrintf("Found local sandbox: %s\n", sandboxInfo.Name)
			}
		}

		// If not found locally, try remote provider
		if sandboxInfo == nil {
			remoteProvider, err := remote.NewProvider()
			if err == nil {
				sandboxInfo, err = findRemoteSandbox(remoteProvider, sandboxIdentifier)
				if err == nil {
					provider = remoteProvider
					utils.DebugPrintf("Found remote sandbox: %s\n", sandboxInfo.Name)
				}
			}
		}

		if sandboxInfo == nil {
			fmt.Fprintf(os.Stderr, "Error: Sandbox '%s' not found\n", sandboxIdentifier)
			os.Exit(1)
		}

		// Confirm deletion unless force flag is used
		if !force {
			if !confirmDeletion(sandboxInfo) {
				fmt.Println("Deletion cancelled.")
				return
			}
		}

		// Delete the sandbox
		fmt.Printf("🗑️  Deleting %s sandbox: %s\n", sandboxInfo.Type, sandboxInfo.Name)
		err = provider.Delete(sandboxInfo.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to delete sandbox: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Successfully deleted sandbox: %s\n", sandboxInfo.Name)
	},
}

// findLocalSandbox tries to find a sandbox in the local provider by name or ID
func findLocalSandbox(provider *local.Provider, identifier string) (*sandbox.SandboxInfo, error) {
	sandboxes, err := provider.List()
	if err != nil {
		return nil, err
	}

	for _, sb := range sandboxes {
		if sb.Name == identifier || sb.ID == identifier {
			return sb, nil
		}
	}

	return nil, fmt.Errorf("sandbox not found")
}

// findRemoteSandbox tries to find a sandbox in the remote provider by name or ID
func findRemoteSandbox(provider *remote.Provider, identifier string) (*sandbox.SandboxInfo, error) {
	sandboxes, err := provider.List()
	if err != nil {
		return nil, err
	}

	for _, sb := range sandboxes {
		if sb.Name == identifier || sb.ID == identifier {
			return sb, nil
		}
	}

	return nil, fmt.Errorf("sandbox not found")
}

// confirmDeletion asks the user to confirm deletion
func confirmDeletion(sandboxInfo *sandbox.SandboxInfo) bool {
	fmt.Printf("Are you sure you want to delete the %s sandbox '%s' (%s)? [y/N]: ",
		sandboxInfo.Type, sandboxInfo.Name, sandboxInfo.ID)

	var response string
	fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// deleteAllSandboxes deletes all sandboxes from both local and remote providers
func deleteAllSandboxes(force bool) {
	var allSandboxes []*sandbox.SandboxInfo
	var providers []sandbox.Provider

	// Collect all sandboxes from local provider
	localProvider, err := local.NewProvider()
	if err == nil {
		localSandboxes, err := localProvider.List()
		if err == nil {
			for _, sb := range localSandboxes {
				allSandboxes = append(allSandboxes, sb)
				providers = append(providers, localProvider)
			}
		} else {
			utils.DebugPrintf("Warning: Failed to list local sandboxes: %s\n", err)
		}
	} else {
		utils.DebugPrintf("Warning: Failed to create local provider: %s\n", err)
	}

	// Collect all sandboxes from remote provider
	remoteProvider, err := remote.NewProvider()
	if err == nil {
		remoteSandboxes, err := remoteProvider.List()
		if err == nil {
			for _, sb := range remoteSandboxes {
				allSandboxes = append(allSandboxes, sb)
				providers = append(providers, remoteProvider)
			}
		} else {
			utils.DebugPrintf("Warning: Failed to list remote sandboxes: %s\n", err)
		}
	} else {
		utils.DebugPrintf("Warning: Failed to create remote provider: %s\n", err)
	}

	if len(allSandboxes) == 0 {
		fmt.Println("No sandboxes found to delete.")
		return
	}

	// Show all sandboxes that will be deleted
	fmt.Printf("Found %d sandbox(es) to delete:\n\n", len(allSandboxes))
	for _, sb := range allSandboxes {
		fmt.Printf("  %s - %s (%s) [%s]\n", sb.Type, sb.Name, sb.ID, sb.State)
	}
	fmt.Println()

	// Confirm deletion unless force flag is used
	if !force {
		if !confirmAllDeletion(len(allSandboxes)) {
			fmt.Println("Deletion cancelled.")
			return
		}
	}

	// Delete all sandboxes
	fmt.Println("🗑️  Deleting all sandboxes...")
	successCount := 0
	failureCount := 0

	for i, sb := range allSandboxes {
		provider := providers[i]
		fmt.Printf("[%d/%d] Deleting %s sandbox: %s", i+1, len(allSandboxes), sb.Type, sb.Name)

		err := provider.Delete(sb.ID)
		if err != nil {
			fmt.Printf(" ❌ FAILED: %s\n", err)
			failureCount++
		} else {
			fmt.Printf(" ✅ SUCCESS\n")
			successCount++
		}
	}

	// Summary
	fmt.Printf("\n📊 Summary: %d successful, %d failed\n", successCount, failureCount)
	if failureCount > 0 {
		os.Exit(1)
	}
}

// confirmAllDeletion asks the user to confirm deletion of all sandboxes
func confirmAllDeletion(count int) bool {
	fmt.Printf("⚠️  WARNING: This will permanently delete ALL %d sandbox(es) and their associated resources!\n", count)
	fmt.Printf("Are you sure you want to proceed? [y/N]: ")

	var response string
	fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func init() {
	// Add flags for the delete command
	deleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	deleteCmd.Flags().BoolP("all", "a", false, "Delete all sandboxes from both local and remote providers")
}
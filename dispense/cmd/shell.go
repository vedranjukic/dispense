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

var shellCmd = &cobra.Command{
	Use:   "shell <sandboxId|name>",
	Short: "Connect to a sandbox shell",
	Long:  `Connect to a sandbox shell using either the sandbox ID or name. Works with both local Docker containers and remote Daytona sandboxes.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		identifier := args[0]

		// Get command flags
		preferLocal, _ := cmd.Flags().GetBool("local")
		preferRemote, _ := cmd.Flags().GetBool("remote")

		// Find the sandbox using providers
		sandboxInfo, provider, err := findSandbox(identifier, preferLocal, preferRemote)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding sandbox: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("Found %s sandbox: %s (%s)\n", sandboxInfo.Type, sandboxInfo.Name, sandboxInfo.ID)
		fmt.Printf("State: %s\n", sandboxInfo.State)

		// Use the provider's ExecuteShell method
		fmt.Printf("Connecting to sandbox shell...\n")
		err = provider.ExecuteShell(sandboxInfo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error connecting to shell: %s\n", err)
			os.Exit(1)
		}
	},
}

// findSandbox searches for a sandbox across providers
func findSandbox(identifier string, preferLocal, preferRemote bool) (*sandbox.SandboxInfo, sandbox.Provider, error) {
	var providers []sandbox.Provider
	var searchOrder []string

	// Determine search order based on preferences
	if preferLocal && !preferRemote {
		// Only search local
		localProvider, err := local.NewProvider()
		if err == nil {
			providers = append(providers, localProvider)
			searchOrder = append(searchOrder, "local")
		}
	} else if preferRemote && !preferLocal {
		// Only search remote
		remoteProvider, err := remote.NewProvider()
		if err == nil {
			providers = append(providers, remoteProvider)
			searchOrder = append(searchOrder, "remote")
		}
	} else {
		// Search both, with preference order
		if preferLocal {
			// Search local first, then remote
			if localProvider, err := local.NewProvider(); err == nil {
				providers = append(providers, localProvider)
				searchOrder = append(searchOrder, "local")
			}
			// Use non-interactive remote provider to avoid prompting for API key
			if remoteProvider, err := remote.NewProviderNonInteractive(); err == nil {
				providers = append(providers, remoteProvider)
				searchOrder = append(searchOrder, "remote")
			}
		} else {
			// Default: search local first, then remote (to avoid API key prompt for local sandboxes)
			if localProvider, err := local.NewProvider(); err == nil {
				providers = append(providers, localProvider)
				searchOrder = append(searchOrder, "local")
			}
			// Use non-interactive remote provider to avoid prompting for API key
			if remoteProvider, err := remote.NewProviderNonInteractive(); err == nil {
				providers = append(providers, remoteProvider)
				searchOrder = append(searchOrder, "remote")
			}
		}
	}

	if len(providers) == 0 {
		return nil, nil, fmt.Errorf("no sandbox providers available")
	}

	// Search through providers
	for i, provider := range providers {
		utils.DebugPrintf("Searching for sandbox '%s' in %s provider\n", identifier, searchOrder[i])

		// First try to get by ID
		if sandboxInfo, err := provider.GetInfo(identifier); err == nil {
			return sandboxInfo, provider, nil
		}

		// Then search through all sandboxes by name
		sandboxes, err := provider.List()
		if err != nil {
			utils.DebugPrintf("Warning: Could not list sandboxes from %s provider: %s\n", searchOrder[i], err)
			continue
		}

		for _, sb := range sandboxes {
			if sb.Name == identifier || sb.ID == identifier {
				return sb, provider, nil
			}
		}
	}

	// Build error message with search details
	searchedProviders := strings.Join(searchOrder, ", ")
	return nil, nil, fmt.Errorf("no sandbox found with identifier '%s' (searched: %s)", identifier, searchedProviders)
}

func init() {
	// Add flags
	shellCmd.Flags().Bool("local", false, "Prefer local Docker sandboxes")
	shellCmd.Flags().Bool("remote", false, "Prefer remote Daytona sandboxes")
}
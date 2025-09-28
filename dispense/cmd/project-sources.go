package main

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"cli/pkg/sandbox"
	"cli/pkg/sandbox/local"
	"cli/pkg/sandbox/remote"
	"cli/pkg/utils"

	"github.com/spf13/cobra"
)

var projectSourcesCmd = &cobra.Command{
	Use:   "project-sources",
	Short: "List all distinct project sources from sandboxes",
	Long:  `List all distinct project sources from both local Docker containers and remote Daytona API sandboxes.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get command flags
		showLocal, _ := cmd.Flags().GetBool("local")
		showRemote, _ := cmd.Flags().GetBool("remote")
		group, _ := cmd.Flags().GetString("group")

		// If no specific flags, show both
		if !showLocal && !showRemote {
			showLocal = true
			showRemote = true
		}

		var allSandboxes []*sandbox.SandboxInfo
		var providers []sandbox.Provider

		// Create providers based on flags
		if showRemote {
			// Use non-interactive if both local and remote are being shown (default case)
			// Use interactive if only remote is explicitly requested
			var remoteProvider sandbox.Provider
			var err error
			if showLocal {
				// Both are being shown (default case) - use non-interactive to avoid API key prompt
				remoteProvider, err = remote.NewProviderNonInteractive()
			} else {
				// Only remote explicitly requested - use interactive
				remoteProvider, err = remote.NewProvider()
			}

			if err != nil {
				if showLocal {
					utils.DebugPrintf("No remote provider available (no API key): %s\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "Warning: Could not create remote provider: %s\n", err)
				}
			} else {
				providers = append(providers, remoteProvider)
			}
		}

		if showLocal {
			localProvider, err := local.NewProvider()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Could not create local provider: %s\n", err)
			} else {
				providers = append(providers, localProvider)
			}
		}

		// Collect sandboxes from all providers
		for _, provider := range providers {
			utils.DebugPrintf("Listing sandboxes from %s provider\n", provider.GetType())

			var sandboxes []*sandbox.SandboxInfo
			var err error

			// Check if group filtering is requested and provider supports it
			if group != "" {
				if localProvider, ok := provider.(*local.Provider); ok {
					utils.DebugPrintf("Filtering by group: %s\n", group)
					sandboxes, err = localProvider.ListByGroup(group)
				} else {
					// For remote providers, get all and filter manually for now
					allProviderSandboxes, listErr := provider.List()
					if listErr != nil {
						fmt.Fprintf(os.Stderr, "Warning: Error listing %s sandboxes: %s\n", provider.GetType(), listErr)
						continue
					}
					// Filter by group metadata
					for _, sb := range allProviderSandboxes {
						if groupValue, exists := sb.Metadata["group"]; exists {
							if groupStr, ok := groupValue.(string); ok && groupStr == group {
								sandboxes = append(sandboxes, sb)
							}
						}
					}
				}
			} else {
				sandboxes, err = provider.List()
			}

			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Error listing %s sandboxes: %s\n", provider.GetType(), err)
				continue
			}

			allSandboxes = append(allSandboxes, sandboxes...)
		}

		// Extract distinct project sources
		projectSourcesMap := make(map[string]bool)
		for _, sb := range allSandboxes {
			if sb.ProjectSource != "" {
				projectSourcesMap[sb.ProjectSource] = true
			}
		}

		// Convert to sorted slice
		var projectSources []string
		for projectSource := range projectSourcesMap {
			projectSources = append(projectSources, projectSource)
		}
		sort.Strings(projectSources)

		// Display results
		if len(projectSources) == 0 {
			fmt.Println("No project sources found.")
			return
		}

		// Create tab writer for formatted output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()

		// Print header
		fmt.Fprintln(w, "Project Source")
		fmt.Fprintln(w, "--------------")

		// Print project sources
		for _, projectSource := range projectSources {
			fmt.Fprintln(w, projectSource)
		}

		// Print summary
		fmt.Printf("\nTotal: %d distinct project sources\n", len(projectSources))
	},
}

func init() {
	// Add flags
	projectSourcesCmd.Flags().Bool("local", false, "Show only local Docker sandboxes")
	projectSourcesCmd.Flags().Bool("remote", false, "Show only remote Daytona sandboxes")
	projectSourcesCmd.Flags().StringP("group", "g", "", "Filter sandboxes by group")
}
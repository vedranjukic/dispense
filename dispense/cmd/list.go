package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"cli/pkg/sandbox"
	"cli/pkg/sandbox/local"
	"cli/pkg/sandbox/remote"
	"cli/pkg/utils"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sandboxes",
	Long:  `List all sandboxes from both local Docker containers and remote Daytona API.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get command flags
		verbose, _ := cmd.Flags().GetBool("verbose")
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

		// Display results
		if len(allSandboxes) == 0 {
			fmt.Println("No sandboxes found.")
			if showLocal && showRemote {
				fmt.Println("No local Docker containers or remote Daytona sandboxes were found.")
			} else if showLocal {
				fmt.Println("No local Docker containers were found.")
			} else if showRemote {
				fmt.Println("No remote Daytona sandboxes were found.")
			}
			return
		}

		// Create tab writer for formatted output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()

		// Print header
		if verbose {
			fmt.Fprintln(w, "ID\tName\tType\tState\tShell Command\tMetadata")
		} else {
			fmt.Fprintln(w, "ID\tName\tType\tState\tShell Command")
		}

		// Print sandboxes
		for _, sb := range allSandboxes {
			if verbose {
				metadataStr := formatMetadata(sb.Metadata)
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					truncateString(sb.ID, 36),
					sb.Name,
					sb.Type,
					sb.State,
					sb.ShellCommand,
					metadataStr,
				)
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					truncateString(sb.ID, 36),
					sb.Name,
					sb.Type,
					sb.State,
					sb.ShellCommand,
				)
			}
		}

		// Print summary
		fmt.Printf("\nTotal: %d sandboxes", len(allSandboxes))

		// Count by type
		localCount := 0
		remoteCount := 0
		for _, sb := range allSandboxes {
			if sb.Type == sandbox.TypeLocal {
				localCount++
			} else if sb.Type == sandbox.TypeRemote {
				remoteCount++
			}
		}

		if localCount > 0 && remoteCount > 0 {
			fmt.Printf(" (%d local, %d remote)", localCount, remoteCount)
		}
		fmt.Println()
	},
}

// Helper functions

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatMetadata(metadata map[string]interface{}) string {
	if len(metadata) == 0 {
		return "-"
	}

	var parts []string
	for key, value := range metadata {
		// Skip large objects, just show basic info
		switch key {
		case "container_name", "image", "ports", "group":
			parts = append(parts, fmt.Sprintf("%s=%v", key, value))
		case "daytona_sandbox":
			parts = append(parts, "daytona=yes")
		}
	}

	if len(parts) == 0 {
		return fmt.Sprintf("%d keys", len(metadata))
	}

	result := strings.Join(parts, ",")
	if len(result) > 50 {
		return result[:47] + "..."
	}
	return result
}

func init() {
	// Add flags
	listCmd.Flags().BoolP("verbose", "v", false, "Show detailed information")
	listCmd.Flags().Bool("local", false, "Show only local Docker sandboxes")
	listCmd.Flags().Bool("remote", false, "Show only remote Daytona sandboxes")
	listCmd.Flags().StringP("group", "g", "", "Filter sandboxes by group")
}
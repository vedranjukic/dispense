package main

import (
	"fmt"
	"os"
	"strings"

	"cli/pkg/utils"

	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec <sandboxId|name> <command>",
	Short: "Execute a command in a sandbox",
	Long: `Execute a command in a sandbox and return the output (stdout/stderr) and exit code.
Works with both local Docker containers and remote Daytona sandboxes.

Examples:
  dispense exec my-sandbox "ls -la"
  dispense exec my-sandbox "echo 'Hello World'"
  dispense exec my-sandbox "cp -r /source /destination"`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		identifier := args[0]
		command := strings.Join(args[1:], " ")

		// Get command flags
		preferLocal, _ := cmd.Flags().GetBool("local")
		preferRemote, _ := cmd.Flags().GetBool("remote")

		// Find the sandbox using providers
		sandboxInfo, provider, err := findSandbox(identifier, preferLocal, preferRemote)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding sandbox: %s\n", err)
			os.Exit(1)
		}

		utils.DebugPrintf("Found %s sandbox: %s (%s)\n", sandboxInfo.Type, sandboxInfo.Name, sandboxInfo.ID)
		utils.DebugPrintf("Executing command: %s\n", command)

		// Execute the command
		result, err := provider.ExecuteCommand(sandboxInfo, command)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error executing command: %s\n", err)
			os.Exit(1)
		}

		// Output the results
		if result.Stdout != "" {
			fmt.Print(result.Stdout)
		}
		if result.Stderr != "" {
			fmt.Fprint(os.Stderr, result.Stderr)
		}

		// Exit with the same code as the executed command
		os.Exit(result.ExitCode)
	},
}

func init() {
	// Add flags
	execCmd.Flags().Bool("local", false, "Prefer local Docker sandboxes")
	execCmd.Flags().Bool("remote", false, "Prefer remote Daytona sandboxes")
}
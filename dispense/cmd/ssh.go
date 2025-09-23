package main

import (
	"github.com/spf13/cobra"
)

// sshCmd is an alias for the shell command for backward compatibility
var sshCmd = &cobra.Command{
	Use:   "ssh <sandboxId|name>",
	Short: "Connect to a sandbox shell (alias for 'shell')",
	Long:  `Connect to a sandbox shell using either the sandbox ID or name. This is an alias for the 'shell' command for backward compatibility.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Simply delegate to the shell command
		shellCmd.Run(cmd, args)
	},
}

func init() {
	// Add the same flags as shell command
	sshCmd.Flags().Bool("local", false, "Prefer local Docker sandboxes")
	sshCmd.Flags().Bool("remote", false, "Prefer remote Daytona sandboxes")
}
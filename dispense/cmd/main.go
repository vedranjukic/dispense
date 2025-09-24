package main

import (
	"fmt"
	"os"

	"cli/pkg/utils"

	"github.com/spf13/cobra"
)

// Build metadata - set during compilation
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "cli",
	Short: "A CLI application built with Go",
	Long:  `A command line interface application built with Go and managed in an Nx monorepo.`,
	Run: func(cmd *cobra.Command, args []string) {
		utils.DebugPrintf("rootCmd.Run called with args: %v\n", args)

		// Check for version flag
		if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
			versionCmd.Run(cmd, args)
			return
		}

		// If no arguments provided, run the new command (default behavior)
		utils.DebugPrintf("No specific command, running new command\n")
		// Execute the new command
		newCmd.Run(cmd, args)
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print version, git commit, and build time information.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		fmt.Printf("Build Time: %s\n", BuildTime)
	},
}


func init() {
	// Add global debug flag
	rootCmd.PersistentFlags().BoolVarP(&utils.DebugMode, "debug", "d", false, "Enable debug output")

	// Add version flag
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")

	// Add subcommands
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(shellCmd)
	rootCmd.AddCommand(sshCmd) // Alias for shell command
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(claudeCmd)
	rootCmd.AddCommand(waitCmd)
	rootCmd.AddCommand(versionCmd)

	// Add flags from newCmd to rootCmd so they work without specifying "new"
	rootCmd.Flags().BoolP("remote", "r", false, "Create remote sandbox using Daytona API (default: local Docker)")
	rootCmd.Flags().StringP("name", "n", "", "Branch name to use (skips prompt)")
	rootCmd.Flags().BoolP("force", "f", false, "Skip git repository check and force sandbox creation")
	rootCmd.Flags().StringP("snapshot", "s", "", "Snapshot ID/name or Docker image to use")
	rootCmd.Flags().StringP("target", "t", "", "Target region for remote sandbox")
	rootCmd.Flags().Int32P("cpu", "", 0, "CPU allocation")
	rootCmd.Flags().Int32P("memory", "", 0, "Memory allocation (MB)")
	rootCmd.Flags().Int32P("disk", "", 0, "Disk allocation (GB)")
	rootCmd.Flags().Int32P("auto-stop", "a", 60, "Auto-stop interval in minutes (0 = disabled, remote only)")
	rootCmd.Flags().Bool("skip-copy", false, "Skip copying files to sandbox")
	rootCmd.Flags().Bool("skip-daemon", false, "Skip installing daemon to sandbox")
	rootCmd.Flags().String("model", "", "Anthropic model to use (e.g., claude-3-opus-20240229)")
	rootCmd.Flags().String("task", "", "Task description (skips task prompt)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func main() {
	utils.DebugPrintf("main() function called\n")
	Execute()
}
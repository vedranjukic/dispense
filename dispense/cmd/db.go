package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"cli/pkg/database"
	"cli/pkg/utils"

	"github.com/spf13/cobra"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database management commands",
	Long:  `Manage the local sandbox database (for development and debugging).`,
}

var dbListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sandboxes in database",
	Long:  `List all local sandboxes stored in the Storm database.`,
	Run: func(cmd *cobra.Command, args []string) {
		db, err := database.GetSandboxDB()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening database: %s\n", err)
			os.Exit(1)
		}
		defer db.Close()

		sandboxes, err := db.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing sandboxes: %s\n", err)
			os.Exit(1)
		}

		if len(sandboxes) == 0 {
			fmt.Println("No sandboxes found in database.")
			return
		}

		// Create tab writer for formatted output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()

		// Print header
		fmt.Fprintln(w, "ID\tName\tContainer ID\tImage\tState\tCreated\tUpdated")

		// Print sandboxes
		for _, sb := range sandboxes {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				truncateString(sb.ID, 36),
				sb.Name,
				truncateString(sb.ContainerID, 12),
				sb.Image,
				sb.State,
				sb.CreatedAt.Format("2006-01-02 15:04"),
				sb.UpdatedAt.Format("2006-01-02 15:04"),
			)
		}

		fmt.Printf("\nTotal: %d sandboxes in database\n", len(sandboxes))
	},
}

var dbDeleteCmd = &cobra.Command{
	Use:   "delete <sandbox-id|name>",
	Short: "Delete sandbox from database",
	Long:  `Delete a sandbox record from the Storm database (does not affect Docker containers).`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		identifier := args[0]

		db, err := database.GetSandboxDB()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening database: %s\n", err)
			os.Exit(1)
		}
		defer db.Close()

		// Try to delete by ID first
		err = db.Delete(identifier)
		if err != nil {
			// Try by name
			err = db.DeleteByName(identifier)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error deleting sandbox: %s\n", err)
				os.Exit(1)
			}
		}

		fmt.Printf("Successfully deleted sandbox '%s' from database\n", identifier)
	},
}

var dbAddCmd = &cobra.Command{
	Use:   "add <name> <container-id> [image]",
	Short: "Add sandbox to database",
	Long:  `Add a sandbox record to the Storm database (for testing purposes).`,
	Args:  cobra.RangeArgs(2, 3),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		containerID := args[1]
		image := "ubuntu:latest"
		if len(args) > 2 {
			image = args[2]
		}

		db, err := database.GetSandboxDB()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening database: %s\n", err)
			os.Exit(1)
		}
		defer db.Close()

		// Create sandbox record
		sandbox := &database.LocalSandbox{
			ID:          containerID, // Use container ID as sandbox ID
			Name:        name,
			ContainerID: containerID,
			Image:       image,
			State:       "running",
			Metadata: map[string]interface{}{
				"container_name": name,
				"image":         image,
			},
		}

		err = db.Save(sandbox)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error saving sandbox: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully added sandbox '%s' to database\n", name)
		utils.DebugPrintf("Sandbox details: ID=%s, ContainerID=%s, Image=%s\n",
			sandbox.ID, sandbox.ContainerID, sandbox.Image)
	},
}

var dbInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show database information",
	Long:  `Show information about the Storm database location and status.`,
	Run: func(cmd *cobra.Command, args []string) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %s\n", err)
			os.Exit(1)
		}

		dbPath := fmt.Sprintf("%s/.dispense/sandboxes.db", homeDir)
		fmt.Printf("Database path: %s\n", dbPath)

		// Check if database exists
		if stat, err := os.Stat(dbPath); err == nil {
			fmt.Printf("Database size: %d bytes\n", stat.Size())
			fmt.Printf("Last modified: %s\n", stat.ModTime().Format("2006-01-02 15:04:05"))

			// Try to open and get count
			db, err := database.GetSandboxDB()
			if err != nil {
				fmt.Printf("Database status: Error opening (%s)\n", err)
			} else {
				defer db.Close()
				sandboxes, err := db.List()
				if err != nil {
					fmt.Printf("Database status: Error reading (%s)\n", err)
				} else {
					fmt.Printf("Database status: OK (%d sandboxes)\n", len(sandboxes))
				}
			}
		} else {
			fmt.Printf("Database status: Not found (will be created on first use)\n")
		}
	},
}

func init() {
	// Add subcommands
	dbCmd.AddCommand(dbListCmd)
	dbCmd.AddCommand(dbDeleteCmd)
	dbCmd.AddCommand(dbAddCmd)
	dbCmd.AddCommand(dbInfoCmd)

	// Add to root command
	rootCmd.AddCommand(dbCmd)
}
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"cli/pkg/mcp"

	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP (Model Context Protocol) server mode",
	Long: `Start the MCP server to allow AI assistants to interact with dispense commands.

The MCP server communicates via stdin/stdout and is designed to be launched by MCP clients
like Claude Code. It exposes dispense commands as structured MCP tools.

Example usage in MCP client configuration:
{
  "mcpServers": {
    "dispense": {
      "command": "/path/to/dispense",
      "args": ["mcp"],
      "env": {
        "DISPENSE_LOG_LEVEL": "info"
      }
    }
  }
}

Development usage with Nx:
  yarn nx server dispense                        # Start MCP server with debug
  yarn nx server dispense --configuration=debug # Extra verbose logging
  yarn nx server dispense --configuration=production # Use built binary`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check if user wants to save config
		if saveConfig, _ := cmd.Flags().GetBool("save-config"); saveConfig {
			config := mcp.LoadConfig()
			configPath := mcp.GetConfigPath()

			if err := config.SaveToFile(configPath); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Configuration saved to: %s\n", configPath)
			return
		}

		// Check if user wants to see config
		if showConfig, _ := cmd.Flags().GetBool("config"); showConfig {
			config := mcp.LoadConfig()
			configPath := mcp.GetConfigPath()

			fmt.Printf("MCP Configuration:\n")
			fmt.Printf("  Config File: %s\n", configPath)

			// Check if config file exists
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				fmt.Printf("  Status: Config file does not exist (using defaults)\n")
			} else {
				fmt.Printf("  Status: Using config file\n")
			}

			fmt.Printf("  Binary Path: %s\n", config.BinaryPath)
			fmt.Printf("  Default Timeout: %s\n", config.DefaultTimeout)
			fmt.Printf("  Long Timeout: %s\n", config.LongTimeout)
			fmt.Printf("  Log Level: %s\n", config.LogLevel)
			fmt.Printf("  Max Concurrent Operations: %d\n", config.MaxConcurrentOps)
			fmt.Printf("\nEnvironment Variables (override config file):\n")
			fmt.Printf("  DISPENSE_BINARY_PATH=%s\n", os.Getenv("DISPENSE_BINARY_PATH"))
			fmt.Printf("  DISPENSE_DEFAULT_TIMEOUT=%s\n", os.Getenv("DISPENSE_DEFAULT_TIMEOUT"))
			fmt.Printf("  DISPENSE_LONG_TIMEOUT=%s\n", os.Getenv("DISPENSE_LONG_TIMEOUT"))
			fmt.Printf("  DISPENSE_LOG_LEVEL=%s\n", os.Getenv("DISPENSE_LOG_LEVEL"))
			fmt.Printf("\nMCP Client Configuration Example:\n")
			clientConfig := map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"dispense": map[string]interface{}{
						"command": config.BinaryPath,
						"args": []string{"mcp"},
						"env": map[string]string{
							"DISPENSE_LOG_LEVEL": "info",
						},
					},
				},
			}
			configJSON, _ := json.MarshalIndent(clientConfig, "", "  ")
			fmt.Printf("%s\n", string(configJSON))
			fmt.Printf("\nTo save current config: dispense mcp --save-config\n")
			return
		}

		// Check if we're in a recursive call to prevent infinite loops
		if os.Getenv("DISPENSE_MCP_MODE") == "internal" {
			fmt.Fprintf(os.Stderr, "Error: MCP mode cannot be called recursively\n")
			os.Exit(1)
		}

		// Create and run MCP server
		server, err := mcp.NewServer()
		if err != nil {
			log.Fatalf("Failed to create MCP server: %v", err)
		}

		// Run the MCP server
		if err := server.Run(context.Background()); err != nil {
			log.Fatalf("MCP server error: %v", err)
		}
	},
}

func init() {
	// Add MCP-specific flags if needed
	mcpCmd.Flags().String("log-level", "info", "Set logging level (debug, info, warn, error)")
	mcpCmd.Flags().Bool("config", false, "Show MCP configuration, file location, and client config example")
	mcpCmd.Flags().Bool("save-config", false, "Save current MCP configuration to config file")
}
# Dispense - Isolated Claude Code Environments

**Safely run multiple Claude Code instances in isolated sandboxes**

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)]()

## Project Overview

Dispense enables you to run multiple Claude Code instances in completely isolated environments, both locally using Docker and remotely using Daytona. Each Claude Code instance operates in its own sandbox, keeping your host machine safe from any security issues while allowing Claude Code more freedom in terms of file system access.

## Motivation

**Parallel Task Execution:** When working on complex projects, you often need to run multiple tasks simultaneously - fixing bugs, implementing features, writing tests, or refactoring code. Traditional development environments force you to wait for one task to complete before starting another. Dispense allows you to spin up multiple isolated sandboxes and have Claude work on different aspects of your project in parallel.

**Unrestricted AI Operation:** Claude Code's safety mechanisms require frequent user confirmations for file operations, which interrupts workflow and slows down development. In Dispense's isolated sandboxes, Claude can operate freely without constant permission requests since the host system is protected by containerization.

**Multi-Agent Future:** We're building toward a future where multiple AI agents can work on the same project simultaneously. You'll be able to assign competing tasks to different agents (Claude, GPT, Gemini, etc.) and choose the best implementation, or have agents collaborate on different parts of your codebase in real-time.

## 🔒 Security Benefits

- **Host Isolation** - Your main system is protected from any code execution
- **Clean Slate** - Each sandbox starts fresh without system contamination

## Sandbox Types

### Local Sandboxes (Docker)

Local sandboxes provide isolation on your machine using Docker containers:

1. **Git Worktree Creation** - Dispense creates a new git worktree from your current repository, allowing multiple working directories from the same repo. If the project has no git config, Dispense will copy the files to the sandbox FS.
2. **Container Setup** - A Docker container is launched with the worktree folder mounted inside
3. **Claude Installation** - Claude Code is installed and configured within the container
4. **Isolated Execution** - Claude runs completely isolated from your host machine, with full access to the mounted project files

This approach lets Claude operate freely within the sandbox while keeping your host system completely protected. You can run multiple local sandboxes simultaneously without conflicts.

### Remote Sandboxes (Daytona)

Remote sandboxes run in the cloud using Daytona's infrastructure:

1. **Cloud Provisioning** - Sandboxes are created on [Daytona](https://www.daytona.io/)
2. **Real-time Collaboration** - Share sandbox access with team members for collaborative development
3. **Live Preview** - Preview web applications and services running in the sandbox
4. **Background Operation** - Tasks continue running even when your local machine is offline
5. **Scalability** - Run dozens of Claude Code instances simultaneously without impacting your computer's performance

Remote sandboxes are ideal for resource-intensive tasks, long-running operations, or when you want to free up your local machine while Claude works in the background.

## The Workflow

1. **Create a sandbox** from a local project directory or GitHub issue
2. **Launch Claude Code** inside the isolated environment
3. **Run tasks** safely in the background or remotely
4. **Access results** without compromising your host system

## Accessing Sandboxes

### Shell Access

Dispense provides direct shell access to your sandboxes for debugging, monitoring, or manual operations:

**Local Sandboxes:** Connect directly to Docker containers running on your machine. You get full shell access to the isolated environment where Claude Code is running.

**Remote Sandboxes:** SSH into cloud-based Daytona instances. These connections are secured and provide the same terminal experience as local development, but running in the cloud.

### Use Cases for Shell Access
- **Monitor Claude's Progress** - Watch real-time file changes and command execution
- **Debug Issues** - Investigate problems or examine logs when tasks encounter errors
- **Manual Operations** - Run additional commands, install packages, or configure the environment
- **Code Review** - Examine Claude's generated code before merging back to your main branch
- **Collaborative Development** - Multiple team members can access the same remote sandbox simultaneously

## 🔗 SSH Configuration & IDE Integration

### Saving SSH Configuration (Remote Sandboxes Only)

For remote Daytona sandboxes, you can save SSH connection details to your local SSH config file for easy access from terminals and IDEs:

```bash
# Save SSH config for a remote sandbox
dispense ssh my-project --save-config

# This adds an entry to ~/.ssh/config like:
# Host dispense-my-project
#   HostName ssh.daytona.io
#   User SSH_TOKEN
```

### Connecting from Host Terminal

Once SSH config is saved, you can connect directly from any terminal:

```bash
# Connect using the saved SSH config
ssh dispense-my-project
```

### IDE Integration (VSCode/Cursor)
Sometimes it's useful to connect to a sandbox with your favorite IDE in order to supplement a task that Claude has completed or for testing/debugging problems.

#### VSCode with Remote-SSH Extension

1. **Install the Remote-SSH extension** in VSCode
2. **Save SSH config** for your sandbox:
   ```bash
   dispense ssh my-project --save-config
   ```
3. **Connect in VSCode**:
   - Press `Ctrl+Shift+P` (or `Cmd+Shift+P` on macOS)
   - Type "Remote-SSH: Connect to Host"
   - Select `dispense-my-project` from the list
   - VSCode opens a new window connected to the sandbox

#### Cursor with Remote Development

1. **Save SSH config** for your sandbox:
   ```bash
   dispense ssh my-project --save-config
   ```
2. **Connect in Cursor**:
   - Open Cursor
   - Use `File > Connect to Server` or `Ctrl+Shift+P` → "Connect to Server"
   - Enter the SSH connection: `dispense-my-project`
   - Cursor opens connected to the remote sandbox

## ✨ Key Features

- **🔒 Isolated Environments** - Run Claude Code safely in Docker containers or remote sandboxes
- **🚀 Local & Remote Support** - Use Docker locally or Daytona for remote environments
- **🌳 GitHub Integration** - Create sandboxes directly from GitHub issues
- **⚡ Background Tasks** - Leave tasks running while you work on other things
- **🤖 MCP Integration** - Built-in Model Context Protocol server for AI assistant integration

## 🤖 MCP Integration

Dispense includes built-in **Model Context Protocol (MCP)** support, allowing AI assistants like Claude Code to interact with dispense commands through structured function calls. The MCP server is bundled directly into the dispense binary and provides configuration management for easy setup.

### Quick Setup

1. **Generate MCP configuration:**
   ```bash
   dispense mcp --config
   ```

2. **Save configuration to file:**
   ```bash
   dispense mcp --save-config
   ```

3. **Copy the generated JSON to your MCP client configuration**

### Configuration Management

#### Config File Location
- **Linux/macOS**: `~/.config/dispense/mcp.json`
- **Custom location**: Set `XDG_CONFIG_HOME` environment variable

#### Configuration Priority (highest to lowest)
1. **Environment variables** (override everything)
2. **Config file** (`~/.config/dispense/mcp.json`)
3. **Default values**

#### Available Configuration Options
```json
{
  "binary_path": "/path/to/dispense",
  "default_timeout": "30s",
  "long_timeout": "10m0s",
  "log_level": "info",
  "max_concurrent_ops": 10
}
```

#### Environment Variables
Override any config file setting:
- `DISPENSE_BINARY_PATH` - Path to dispense binary
- `DISPENSE_DEFAULT_TIMEOUT` - Default command timeout (e.g., "45s")
- `DISPENSE_LONG_TIMEOUT` - Long operation timeout (e.g., "15m")
- `DISPENSE_LOG_LEVEL` - Logging level (debug, info, warn, error)

### Configuring Claude Code

#### Step 1: Get Your Configuration
```bash
# Show current config and get MCP client JSON
dispense mcp --config
```

This will display your configuration and provide a ready-to-use JSON snippet.

#### Step 2: Add to Claude Code

**Location of Claude Code MCP config:**
- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`

**Add the dispense MCP server:**
```json
{
  "mcpServers": {
    "dispense": {
      "command": "/usr/local/bin/dispense",
      "args": ["mcp"],
      "env": {
        "DISPENSE_LOG_LEVEL": "info"
      }
    }
  }
}
```

**Example complete configuration:**
```json
{
  "mcpServers": {
    "dispense": {
      "command": "/usr/local/bin/dispense",
      "args": ["mcp"],
      "env": {
        "DISPENSE_LOG_LEVEL": "info"
      }
    },
    "other-mcp-server": {
      "command": "/path/to/other-server",
      "args": ["--stdio"]
    }
  }
}
```

#### Step 3: Restart Claude Code
After adding the configuration, restart Claude Code to load the new MCP server.

#### Step 4: Verify Integration
Claude Code should now have access to dispense tools. You can ask Claude to:
- "Create a new sandbox for this GitHub issue"
- "Set up an isolated environment for development"

### Available MCP Tools

When running in MCP mode, dispense exposes these tools to AI assistants:

- **`dispense_create_sandbox`** - Create new sandboxes for GitHub issues or local development
  - Parameters: `name` (required), `task` (required), `remote` (optional)
  - Creates isolated Docker containers or remote Daytona environments

### MCP Server Commands

#### Start MCP Server
```bash
# Production mode
dispense mcp

# With debug logging
dispense mcp --log-level debug
```

#### Development with Nx
```bash
# Start MCP server in development mode
yarn nx server dispense

# With extra debug logging
yarn nx server dispense --configuration=debug

# Using the built binary (faster startup)
yarn nx server dispense --configuration=production
```

#### Configuration Commands
```bash
# Show current configuration and file location
dispense mcp --config

# Save current configuration to file
dispense mcp --save-config

# Start server with custom log level
dispense mcp --log-level debug
```

### Troubleshooting MCP Integration

#### Common Issues

1. **Claude Code doesn't see dispense tools**
   - Verify the binary path in your config: `dispense mcp --config`
   - Check Claude Code logs for MCP connection errors
   - Ensure the dispense binary is executable: `chmod +x /path/to/dispense`

2. **Permission denied errors**
   - Make sure the dispense binary path is correct and executable
   - On macOS, you may need to allow the binary in Security & Privacy settings

3. **Configuration not loading**
   - Check if config file exists: `dispense mcp --config`
   - Verify config file syntax is valid JSON
   - Use environment variables to override config: `DISPENSE_LOG_LEVEL=debug dispense mcp`

#### Debug Mode
Enable debug logging to troubleshoot issues:
```bash
# Via command line
dispense mcp --log-level debug

# Via environment variable
DISPENSE_LOG_LEVEL=debug dispense mcp

# In Claude Code config
{
  "mcpServers": {
    "dispense": {
      "command": "/path/to/dispense",
      "args": ["mcp", "--log-level", "debug"]
    }
  }
}
```

### Benefits of MCP Integration

- **Structured Communication** - AI assistants can reliably create sandboxes with proper validation
- **Error Handling** - Clear success/failure feedback with detailed error messages
- **Type Safety** - Parameter validation ensures correct sandbox creation
- **Configuration Management** - Easy setup and customization through config files
- **Development Workflow** - Seamless integration with development and production environments
- **Integration Ready** - Works with Claude Code and other MCP-compatible tools

## 🚀 Quick Start

### Prerequisites

- Docker (for local sandboxes)
- Git
- Go 1.19+ (if building from source)

### Installation

#### Download Binaries
Download the latest release from the [releases page](../../releases) for your platform.

**macOS Setup:**
```bash
# Make the binary executable
chmod +x dispense

# Move to a directory in your PATH (choose one)
sudo mv dispense /usr/local/bin/
# or
mkdir -p ~/.local/bin && mv dispense ~/.local/bin/
# then add to your shell profile: export PATH="$HOME/.local/bin:$PATH"

# Allow execution (if blocked by Gatekeeper)
sudo xattr -rd com.apple.quarantine /usr/local/bin/dispense
```

**Antivirus Protection:**
If your antivirus software quarantines or deletes the binary:
- **Windows Defender**: Add the dispense binary to exclusions in Windows Security
- **macOS**: System Preferences → Security & Privacy → Allow apps downloaded from "App Store and identified developers"
- **General**: Add the binary location to your antivirus whitelist/exclusions

#### Build from Source
```bash
# Clone the repository
git clone https://github.com/your-org/dispense.git
cd dispense

# Install dependencies
yarn

# Build the CLI
yarn build
```

## 📖 Commands Reference

### Basic Usage

**Note:** Most flags can be used directly with the main `dispense` command (equivalent to `dispense new`) or with the explicit `dispense new` subcommand.

#### Create a New Sandbox
Dispense Sandbox can be created from an existing directory on the host machine, or from a Github issue.

```bash
# Create a local Docker sandbox
dispense new --name my-project

# Alternative: Use main command directly (equivalent to 'new')
dispense --name my-project --remote

# Alternative: Use main command directly (equivalent to 'new')
dispense --name my-project --remote

# Create a remote Daytona sandbox
dispense --name my-project --remote

# Force creation without git check
dispense --name my-project --force

# Create sandbox with group for organization
dispense --name my-project --group issue-331

# Create sandbox with model parameter
dispense --name my-project --model claude-3-5-sonnet

# Create sandbox with both group and model
dispense --name my-project --group issue-331 --model claude-3-5-haiku --remote

```

#### Create from existing directory
If creating from an existing directory, start dispense from the project directory and write a prompt for the Claude when asked.

#### Create from GH issue
If creating from a GH issue you can start dispense from any directory. In the task prompt make sure that the GH issue link is provided first. Additional task notes can be added after the link.

#### List Sandboxes
```bash
# List all sandboxes
dispense list

# List only local sandboxes
dispense list --local

# List only remote sandboxes
dispense list --remote

# Show detailed information
dispense list --verbose
```

#### Connect to Sandbox Shell
```bash
# Connect to sandbox by name or ID
dispense shell my-project
dispense ssh my-project  # alias for shell

# Prefer local sandbox
dispense shell my-project --local

# Prefer remote sandbox
dispense shell my-project --remote
```

#### Wait for Sandboxes
Waits for all sandboxes to complete running tasks before exiting.

```bash
# Wait for specific sandboxes to be ready
dispense wait my-project another-project

# Wait for all sandboxes in a group
dispense wait --group issue-331

# Wait for sandboxes in multiple groups
dispense wait --group issue-331 issue-332

# Wait for specific sandboxes and groups
dispense wait my-project --group issue-331
```

#### Delete Sandbox
```bash
# Delete a specific sandbox
dispense delete my-project

# Delete with force (skip confirmation)
dispense delete my-project --force

# Delete all sandboxes
dispense delete --all

# Delete all with force
dispense delete --all --force
```

### Claude Code Task Management

#### Check Claude Status
```bash
# Check if Claude Code is running in sandbox
dispense claude my-project status
```

#### Run Claude Tasks
Besides the initial Claude Code task, we can run other tasks at any time. Dispense keeps a track of all the tasks.

```bash
# Run a prompt in Claude Code
dispense claude my-project run "Fix the bug in main.go"

# List tasks
dispense claude my-project tasks

# Task details
dispense claude my-project tasks <task-id>

# View task logs
dispense claude my-project logs <task-id>
```

### MCP Server Mode

Start the built-in MCP server for AI assistant integration:

```bash
# Start MCP server (communicates via stdin/stdout)
dispense mcp

# Start with debug logging
dispense mcp --log-level debug

# Show current configuration and get client config JSON
dispense mcp --config

# Save current configuration to file (~/.config/dispense/mcp.json)
dispense mcp --save-config
```

The MCP server allows AI assistants like Claude Code to create and manage sandboxes through structured function calls. For complete setup instructions including Claude Code configuration, see the [MCP Integration](#-mcp-integration) section.

### Shell Completion
```bash
# Generate bash completion
dispense completion bash

# Generate zsh completion
dispense completion zsh

# Generate fish completion
dispense completion fish

# Generate PowerShell completion
dispense completion powershell
```

### Version Information
```bash
# Show version information
dispense version
```

## 🔧 Configuration Options

### Sandbox Organization

### Global Flags
- `-d, --debug` - Enable debug output
- `-v, --version` - Show version information
- `-h, --help` - Show help information

### Sandbox Creation Flags (`new` command)
- `-n, --name <string>` - Specify sandbox name
- `-r, --remote` - Use Daytona instead of Docker
- `-g, --group <string>` - Optional group parameter for organizing sandboxes
- `-m, --model <string>` - Optional model parameter for the sandbox
- `--skip-copy` - Don't copy files to sandbox
- `--skip-daemon` - Don't install daemon in sandbo
- `--cpu` - Limit cpu instances (local only)
- `--memory` - Limit memory allocation (local only)

### Wait Command Flags
- `--group <strings>` - Wait for all sandboxes in specified groups

### Delete Command Flags
- `-a, --all` - Delete all sandboxes from both local and remote providers
- `-f, --force` - Skip confirmation prompt

### List Command Flags
- `--local` - Show only local Docker sandboxes
- `--remote` - Show only remote Daytona sandboxes
- `-v, --verbose` - Show detailed information

### Shell/SSH Command Flags
- `--local` - Prefer local Docker sandboxes
- `--remote` - Prefer remote Daytona sandboxes

## 🏗️ Building from Source

```bash
# Clone repository
git clone https://github.com/your-org/dispense.git
cd dispense

# Install dependencies
yarn install

# Build all components
yarn build

# Run in development
yarn nx serve cli
```

## 🤝 Contributing

Comming soon

## 📄 License

TBA - for now under "All Rights Reserved" licence

package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"apiclient"
	"cli/pkg/config"
	"cli/pkg/utils"
)

const DaytonaSourceHeader = "X-Daytona-Source"

// Client wraps the API client with authentication
type Client struct {
	apiClient *apiclient.APIClient
	apiKey    string
}

type RunCommandResponse struct {
	Output string
	ExitCode  int
}

// NewClient creates a new authenticated API client
func NewClient() (*Client, error) {
	// Get API key from config
	apiKey, err := config.GetOrPromptAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return createClientWithKey(apiKey)
}

// NewClientNonInteractive creates a new authenticated API client without prompting for API key
// Returns an error if no API key is available
func NewClientNonInteractive() (*Client, error) {
	// Get API key from config without prompting
	apiKey, err := config.GetAPIKeyNonInteractive()
	if err != nil {
		return nil, fmt.Errorf("API key not available: %w", err)
	}

	return createClientWithKey(apiKey)
}

// createClientWithKey creates a client with the given API key
func createClientWithKey(apiKey string) (*Client, error) {
	// Create configuration
	cfg := apiclient.NewConfiguration()
	cfg.Servers = apiclient.ServerConfigurations{
		{
			URL:         "https://app.daytona.io/api", // Default Daytona API URL
			Description: "Daytona API",
		},
	}

	// Set up authentication using Bearer token in headers
	cfg.AddDefaultHeader("Authorization", "Bearer "+apiKey)
	cfg.AddDefaultHeader(DaytonaSourceHeader, "cli")

	// Create API client
	apiClient := apiclient.NewAPIClient(cfg)

	// Set up HTTP client
	apiClient.GetConfig().HTTPClient = &http.Client{
		Transport: http.DefaultTransport,
	}

	client := &Client{
		apiClient: apiClient,
		apiKey:    apiKey,
	}

	// Note: API key validation is skipped as the health endpoint may not be available
	// Authentication will be validated when making actual API calls

	return client, nil
}

// validateAPIKey validates the API key by making a simple API call
func (c *Client) validateAPIKey() error {
	ctx := c.getAuthenticatedContext()
	
	// Try to get health status to validate the API key
	// This should return 401/403 for invalid keys and 200 for valid keys
	request := c.apiClient.HealthAPI.HealthControllerCheck(ctx)
	_, response, err := request.Execute()
	
	if err != nil {
		if response != nil {
			switch response.StatusCode {
			case http.StatusUnauthorized:
				return fmt.Errorf("authentication failed: invalid API key")
			case http.StatusForbidden:
				return fmt.Errorf("access forbidden: insufficient permissions")
			case http.StatusNotFound:
				// Health endpoint returns 404, which might mean invalid API key
				return fmt.Errorf("authentication failed: invalid API key or endpoint not found")
			}
		}
		return fmt.Errorf("failed to validate API key: %w", err)
	}
	
	return nil
}

// getAuthenticatedContext returns a context for API calls
func (c *Client) getAuthenticatedContext() context.Context {
	return context.Background()
}

// ListSandboxes lists all sandboxes
func (c *Client) ListSandboxes() ([]apiclient.Sandbox, error) {
	ctx := c.getAuthenticatedContext()
	
	// Call the ListSandboxes API
	request := c.apiClient.SandboxAPI.ListSandboxes(ctx)
	sandboxes, response, err := request.Execute()
	
	if err != nil {
		// Check if the error contains authentication-related information
		if response != nil {
			switch response.StatusCode {
			case http.StatusUnauthorized:
				return nil, fmt.Errorf("authentication failed: invalid API key")
			case http.StatusForbidden:
				return nil, fmt.Errorf("access forbidden: insufficient permissions")
			case http.StatusNotFound:
				// For 404, we need to distinguish between "no sandboxes" and "invalid API key"
				// Since the API returns 404 for both cases, we'll check the error message
				errorMsg := err.Error()
				if strings.Contains(strings.ToLower(errorMsg), "unauthorized") || 
				   strings.Contains(strings.ToLower(errorMsg), "invalid") ||
				   strings.Contains(strings.ToLower(errorMsg), "auth") {
					return nil, fmt.Errorf("authentication failed: invalid API key")
				}
				// For now, we'll assume 404 means no sandboxes found
				// Note: The API returns 404 for both "no sandboxes" and "invalid API key"
				// If you're sure you have sandboxes, check your API key
				return []apiclient.Sandbox{}, nil // No sandboxes found
			default:
				return nil, fmt.Errorf("API returned status %d: %s", response.StatusCode, response.Status)
			}
		}
		return nil, fmt.Errorf("failed to list sandboxes: %w", err)
	}
	
	// Handle different HTTP status codes appropriately
	switch response.StatusCode {
	case http.StatusOK:
		return sandboxes, nil
	case http.StatusNotFound:
		return []apiclient.Sandbox{}, nil // No sandboxes found
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("authentication failed: invalid API key")
	case http.StatusForbidden:
		return nil, fmt.Errorf("access forbidden: insufficient permissions")
	default:
		return nil, fmt.Errorf("API returned status %d: %s", response.StatusCode, response.Status)
	}
}

// ListSandboxesWithOptions lists sandboxes with additional options
func (c *Client) ListSandboxesWithOptions(verbose bool, labels string, includeErroredDeleted bool) ([]apiclient.Sandbox, error) {
	ctx := c.getAuthenticatedContext()
	
	// Call the ListSandboxes API with options
	request := c.apiClient.SandboxAPI.ListSandboxes(ctx)
	
	if verbose {
		request = request.Verbose(verbose)
	}
	
	if labels != "" {
		request = request.Labels(labels)
	}
	
	if includeErroredDeleted {
		request = request.IncludeErroredDeleted(includeErroredDeleted)
	}
	
	sandboxes, response, err := request.Execute()
	
	if err != nil {
		// Check if the error contains authentication-related information
		if response != nil {
			switch response.StatusCode {
			case http.StatusUnauthorized:
				return nil, fmt.Errorf("authentication failed: invalid API key")
			case http.StatusForbidden:
				return nil, fmt.Errorf("access forbidden: insufficient permissions")
			case http.StatusNotFound:
				// For 404, we need to distinguish between "no sandboxes" and "invalid API key"
				// Since the API returns 404 for both cases, we'll check the error message
				errorMsg := err.Error()
				if strings.Contains(strings.ToLower(errorMsg), "unauthorized") || 
				   strings.Contains(strings.ToLower(errorMsg), "invalid") ||
				   strings.Contains(strings.ToLower(errorMsg), "auth") {
					return nil, fmt.Errorf("authentication failed: invalid API key")
				}
				// Note: The API returns 404 for both "no sandboxes" and "invalid API key"
				// If you're sure you have sandboxes, check your API key
				return []apiclient.Sandbox{}, nil // No sandboxes found
			default:
				return nil, fmt.Errorf("API returned status %d: %s", response.StatusCode, response.Status)
			}
		}
		return nil, fmt.Errorf("failed to list sandboxes: %w", err)
	}
	
	// Handle different HTTP status codes appropriately
	switch response.StatusCode {
	case http.StatusOK:
		return sandboxes, nil
	case http.StatusNotFound:
		return []apiclient.Sandbox{}, nil // No sandboxes found
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("authentication failed: invalid API key")
	case http.StatusForbidden:
		return nil, fmt.Errorf("access forbidden: insufficient permissions")
	default:
		return nil, fmt.Errorf("API returned status %d: %s", response.StatusCode, response.Status)
	}
}

// GetSandbox gets a specific sandbox by ID
func (c *Client) GetSandbox(sandboxId string) (*apiclient.Sandbox, error) {
	ctx := c.getAuthenticatedContext()
	
	// Call the GetSandbox API
	request := c.apiClient.SandboxAPI.GetSandbox(ctx, sandboxId)
	sandbox, response, err := request.Execute()
	
	if err != nil {
		if response != nil {
			switch response.StatusCode {
			case http.StatusUnauthorized:
				return nil, fmt.Errorf("authentication failed: invalid API key")
			case http.StatusForbidden:
				return nil, fmt.Errorf("access forbidden: insufficient permissions")
			case http.StatusNotFound:
				return nil, fmt.Errorf("sandbox not found: %s", sandboxId)
			default:
				return nil, fmt.Errorf("API returned status %d: %s", response.StatusCode, response.Status)
			}
		}
		return nil, fmt.Errorf("failed to get sandbox: %w", err)
	}
	
	return sandbox, nil
}

// StartSandbox starts a sandbox
func (c *Client) StartSandbox(sandboxId string) (*apiclient.Sandbox, error) {
	ctx := c.getAuthenticatedContext()
	
	// Call the StartSandbox API
	request := c.apiClient.SandboxAPI.StartSandbox(ctx, sandboxId)
	sandbox, response, err := request.Execute()
	
	if err != nil {
		if response != nil {
			switch response.StatusCode {
			case http.StatusUnauthorized:
				return nil, fmt.Errorf("authentication failed: invalid API key")
			case http.StatusForbidden:
				return nil, fmt.Errorf("access forbidden: insufficient permissions")
			case http.StatusNotFound:
				return nil, fmt.Errorf("sandbox not found: %s", sandboxId)
			case http.StatusBadRequest:
				return nil, fmt.Errorf("cannot start sandbox: %s", err.Error())
			default:
				return nil, fmt.Errorf("API returned status %d: %s", response.StatusCode, response.Status)
			}
		}
		return nil, fmt.Errorf("failed to start sandbox: %w", err)
	}
	
	return sandbox, nil
}

// CreateSshAccess creates SSH access for a sandbox
func (c *Client) CreateSshAccess(sandboxId string, expiresInMinutes float32) (*apiclient.SshAccessDto, error) {
	ctx := c.getAuthenticatedContext()
	
	// Call the CreateSshAccess API
	request := c.apiClient.SandboxAPI.CreateSshAccess(ctx, sandboxId)
	if expiresInMinutes > 0 {
		request = request.ExpiresInMinutes(expiresInMinutes)
	}
	
	sshAccess, response, err := request.Execute()
	
	if err != nil {
		if response != nil {
			switch response.StatusCode {
			case http.StatusUnauthorized:
				return nil, fmt.Errorf("authentication failed: invalid API key")
			case http.StatusForbidden:
				return nil, fmt.Errorf("access forbidden: insufficient permissions")
			case http.StatusNotFound:
				return nil, fmt.Errorf("sandbox not found: %s", sandboxId)
			case http.StatusBadRequest:
				return nil, fmt.Errorf("cannot create SSH access: %s", err.Error())
			default:
				return nil, fmt.Errorf("API returned status %d: %s", response.StatusCode, response.Status)
			}
		}
		return nil, fmt.Errorf("failed to create SSH access: %w", err)
	}
	
	return sshAccess, nil
}

// UploadFile uploads a single file to a sandbox using the API
func (c *Client) UploadFile(sandboxId, localPath, remotePath string) error {
	ctx := c.getAuthenticatedContext()

	// Open the local file
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file %s: %w", localPath, err)
	}
	defer file.Close()

	// Create the upload request
	request := c.apiClient.ToolboxAPI.UploadFile(ctx, sandboxId)
	request = request.Path(remotePath)
	request = request.File(file)

	// Execute the upload
	response, err := request.Execute()
	if err != nil {
		if response != nil {
			switch response.StatusCode {
			case http.StatusUnauthorized:
				return fmt.Errorf("authentication failed: invalid API key")
			case http.StatusForbidden:
				return fmt.Errorf("access forbidden: insufficient permissions")
			case http.StatusNotFound:
				return fmt.Errorf("sandbox not found: %s", sandboxId)
			case http.StatusBadRequest:
				return fmt.Errorf("bad request: %s", err.Error())
			default:
				return fmt.Errorf("API returned status %d: %s", response.StatusCode, response.Status)
			}
		}
		return fmt.Errorf("failed to upload file: %w", err)
	}

	return nil
}

// UploadTarFile uploads a tar file to a sandbox and extracts it
func (c *Client) UploadTarFile(sandboxId, localTarPath, remotePath string) error {
	ctx := c.getAuthenticatedContext()

	// Open the local tar file
	file, err := os.Open(localTarPath)
	if err != nil {
		return fmt.Errorf("failed to open tar file %s: %w", localTarPath, err)
	}
	defer file.Close()

	// Upload the tar file to a temporary location
	tempTarPath := "/tmp/upload.tar.gz"
	request := c.apiClient.ToolboxAPI.UploadFile(ctx, sandboxId)
	request = request.Path(tempTarPath)
	request = request.File(file)

	// Execute the upload
	response, err := request.Execute()
	if err != nil {
		if response != nil {
			switch response.StatusCode {
			case http.StatusUnauthorized:
				return fmt.Errorf("authentication failed: invalid API key")
			case http.StatusForbidden:
				return fmt.Errorf("access forbidden: insufficient permissions")
			case http.StatusNotFound:
				return fmt.Errorf("sandbox not found: %s", sandboxId)
			case http.StatusBadRequest:
				return fmt.Errorf("bad request: %s", err.Error())
			default:
				return fmt.Errorf("API returned status %d: %s", response.StatusCode, response.Status)
			}
		}
		return fmt.Errorf("failed to upload tar file: %w", err)
	}

	// Extract the tar file on the remote system using separate commands for better reliability
	utils.DebugPrintf("Creating target directory: %s\n", remotePath)
	_, err = c.RunCommand(sandboxId, fmt.Sprintf("mkdir -p '%s'", remotePath), "")
	if err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	utils.DebugPrintf("Extracting tar file: %s -> %s\n", tempTarPath, remotePath)
	_, err = c.RunCommand(sandboxId, fmt.Sprintf("tar -xzf '%s'", tempTarPath), remotePath)
	if err != nil {
		return fmt.Errorf("failed to extract tar file: %w", err)
	}

	utils.DebugPrintf("Cleaning up temporary tar file: %s\n", tempTarPath)
	_,err = c.RunCommand(sandboxId, fmt.Sprintf("rm '%s'", tempTarPath), "")
	if err != nil {
		// Don't fail the whole operation if cleanup fails
		utils.DebugPrintf("Warning - failed to clean up temp tar file: %v\n", err)
	}

	return nil
}

// RunCommand executes a command in the sandbox using the Daytona API
func (c *Client) RunCommand(sandboxId, command string, cwd string) (*RunCommandResponse, error) {
	ctx := c.getAuthenticatedContext()

	// Create the execute request
	executeReq := apiclient.NewExecuteRequest(command)
	if cwd != "" {
		executeReq.Cwd = &cwd
	}

	// Create the API request
	request := c.apiClient.ToolboxAPI.ExecuteCommand(ctx, sandboxId)
	request = request.ExecuteRequest(*executeReq)

	// Execute the command
	response, httpResponse, err := request.Execute()
	if err != nil {
		if httpResponse != nil {
			switch httpResponse.StatusCode {
			case http.StatusUnauthorized:
				return nil, fmt.Errorf("authentication failed: invalid API key")
			case http.StatusForbidden:
				return nil, fmt.Errorf("access forbidden: insufficient permissions")
			case http.StatusNotFound:
				return nil, fmt.Errorf("sandbox not found: %s", sandboxId)
			case http.StatusBadRequest:
				return nil, fmt.Errorf("bad request: %s", err.Error())
			default:
				return nil, fmt.Errorf("API returned status %d: %s", httpResponse.StatusCode, httpResponse.Status)
			}
		}
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}

	// Check if the command execution was successful
	if response != nil && response.ExitCode != 0 {
		return nil, fmt.Errorf("command failed with exit code %.0f: %s", response.ExitCode, response.Result)
	}

	return &RunCommandResponse{
		Output: response.Result,
		ExitCode: int(response.ExitCode),
	}, nil
}

// RunAsyncCommand runs a command asynchronously using session execute for long-running operations
func (c *Client) RunAsyncCommand(sandboxId, command string) error {
	ctx := c.getAuthenticatedContext()

	utils.DebugPrintf("Starting async command execution for sandbox %s: %s\n", sandboxId, command)

	// Generate a unique session ID for this command
	sessionId := fmt.Sprintf("async-session-%d", time.Now().UnixNano())
	utils.DebugPrintf("Generated session ID: %s\n", sessionId)

	// Create session request body
	createSessionRequest := apiclient.NewCreateSessionRequest(sessionId)

	// Create a new session for this command
	sessionReq := c.apiClient.ToolboxAPI.CreateSession(ctx, sandboxId).CreateSessionRequest(*createSessionRequest)
	_, err := sessionReq.Execute()
	if err != nil {
		utils.DebugPrintf("Failed to create session: %v\n", err)
		return fmt.Errorf("failed to create session for async command: %w", err)
	}

	utils.DebugPrintf("Session created successfully: %s\n", sessionId)

	// Create session execute request with async flag
	executeReq := apiclient.NewSessionExecuteRequest(command)
	executeReq.SetRunAsync(true)

	// Execute command in session asynchronously
	request := c.apiClient.ToolboxAPI.ExecuteSessionCommand(ctx, sandboxId, sessionId).SessionExecuteRequest(*executeReq)

	utils.DebugPrintf("Executing async command in session %s\n", sessionId)
	_, _, err = request.Execute()
	if err != nil {
		utils.DebugPrintf("Failed to execute async command: %v\n", err)
		return fmt.Errorf("failed to execute async command: %w", err)
	}

	utils.DebugPrintf("Async command started successfully in session %s\n", sessionId)
	return nil
}

// CreateDirectory creates a directory on the remote sandbox (using a placeholder file approach)
func (c *Client) CreateDirectory(sandboxId, remotePath string) error {
	// Since there's no direct API for creating directories, we'll handle this
	// during file uploads by ensuring parent directories exist
	return nil
}

// CreateSandbox creates a new sandbox
func (c *Client) CreateSandbox(createSandbox *apiclient.CreateSandbox) (*apiclient.Sandbox, error) {
	ctx := c.getAuthenticatedContext()
	
	// Call the CreateSandbox API
	request := c.apiClient.SandboxAPI.CreateSandbox(ctx)
	request = request.CreateSandbox(*createSandbox)
	
	sandbox, response, err := request.Execute()
	
	if err != nil {
		if response != nil {
			// Read response body for more detailed error information
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()
			
			switch response.StatusCode {
			case http.StatusUnauthorized:
				return nil, fmt.Errorf("authentication failed: invalid API key")
			case http.StatusForbidden:
				return nil, fmt.Errorf("access forbidden: insufficient permissions")
			case http.StatusBadRequest:
				return nil, fmt.Errorf("cannot create sandbox (400 Bad Request): %s\nResponse body: %s", err.Error(), string(body))
			default:
				return nil, fmt.Errorf("API returned status %d: %s\nResponse body: %s", response.StatusCode, response.Status, string(body))
			}
		}
		return nil, fmt.Errorf("failed to create sandbox: %w", err)
	}

	return sandbox, nil
}

// DeleteSandbox deletes a sandbox by ID
func (c *Client) DeleteSandbox(sandboxId string) error {
	utils.DebugPrintf("Force deleting sandbox via API: %s\n", sandboxId)

	ctx := context.Background()

	// Call the API to delete the sandbox with force=true
	response, err := c.apiClient.SandboxAPI.DeleteSandbox(ctx, sandboxId).Force(true).Execute()

	if err != nil {
		if response != nil {
			// Read response body for more detailed error information
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()

			switch response.StatusCode {
			case http.StatusUnauthorized:
				return fmt.Errorf("authentication failed: invalid API key")
			case http.StatusForbidden:
				return fmt.Errorf("access forbidden: insufficient permissions")
			case http.StatusNotFound:
				return fmt.Errorf("sandbox not found: %s", sandboxId)
			case http.StatusBadRequest:
				return fmt.Errorf("cannot delete sandbox (400 Bad Request): %s\nResponse body: %s", err.Error(), string(body))
			default:
				return fmt.Errorf("API returned status %d: %s\nResponse body: %s", response.StatusCode, response.Status, string(body))
			}
		}
		return fmt.Errorf("failed to delete sandbox: %w", err)
	}

	utils.DebugPrintf("Successfully deleted sandbox: %s\n", sandboxId)
	return nil
}

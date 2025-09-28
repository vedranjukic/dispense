package client

import (
	"context"
	"io"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "dispense/grpc-server/proto"
)

// DispenseClient provides a wrapper for the gRPC client
type DispenseClient struct {
	conn   *grpc.ClientConn
	client pb.DispenseServiceClient
	apiKey string
}

// ClientConfig contains configuration for the client
type ClientConfig struct {
	Address string
	APIKey  string
	Timeout time.Duration
}

// NewDispenseClient creates a new Dispense gRPC client
func NewDispenseClient(config *ClientConfig) (*DispenseClient, error) {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	// Create gRPC connection
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, config.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	client := pb.NewDispenseServiceClient(conn)

	return &DispenseClient{
		conn:   conn,
		client: client,
		apiKey: config.APIKey,
	}, nil
}

// Close closes the gRPC connection
func (c *DispenseClient) Close() error {
	return c.conn.Close()
}

// createContext creates a context with API key metadata
func (c *DispenseClient) createContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	if c.apiKey != "" {
		md := metadata.New(map[string]string{
			"api-key": c.apiKey,
		})
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	return ctx, cancel
}

// Sandbox methods

// CreateSandbox creates a new sandbox
func (c *DispenseClient) CreateSandbox(req *pb.CreateSandboxRequest) (*pb.CreateSandboxResponse, error) {
	ctx, cancel := c.createContext(30 * time.Second)
	defer cancel()

	return c.client.CreateSandbox(ctx, req)
}

// ListSandboxes lists sandboxes
func (c *DispenseClient) ListSandboxes(req *pb.ListSandboxesRequest) (*pb.ListSandboxesResponse, error) {
	ctx, cancel := c.createContext(30 * time.Second)
	defer cancel()

	return c.client.ListSandboxes(ctx, req)
}

// DeleteSandbox deletes a sandbox
func (c *DispenseClient) DeleteSandbox(req *pb.DeleteSandboxRequest) (*pb.DeleteSandboxResponse, error) {
	ctx, cancel := c.createContext(30 * time.Second)
	defer cancel()

	return c.client.DeleteSandbox(ctx, req)
}

// GetSandbox gets a specific sandbox
func (c *DispenseClient) GetSandbox(req *pb.GetSandboxRequest) (*pb.GetSandboxResponse, error) {
	ctx, cancel := c.createContext(10 * time.Second)
	defer cancel()

	return c.client.GetSandbox(ctx, req)
}

// WaitForSandbox waits for sandbox readiness
func (c *DispenseClient) WaitForSandbox(req *pb.WaitForSandboxRequest) (*pb.WaitForSandboxResponse, error) {
	timeout := 60 * time.Second
	if req.TimeoutSeconds > 0 {
		timeout = time.Duration(req.TimeoutSeconds) * time.Second
	}

	ctx, cancel := c.createContext(timeout)
	defer cancel()

	return c.client.WaitForSandbox(ctx, req)
}

// Claude methods

// RunClaudeTask runs a Claude task with streaming response
func (c *DispenseClient) RunClaudeTask(req *pb.RunClaudeTaskRequest, handler func(*pb.RunClaudeTaskResponse) error) error {
	ctx, cancel := c.createContext(30 * time.Minute) // Long timeout for Claude tasks
	defer cancel()

	stream, err := c.client.RunClaudeTask(ctx, req)
	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if err := handler(resp); err != nil {
			return err
		}

		if resp.IsFinished {
			break
		}
	}

	return nil
}

// GetClaudeStatus gets Claude daemon status
func (c *DispenseClient) GetClaudeStatus(req *pb.GetClaudeStatusRequest) (*pb.GetClaudeStatusResponse, error) {
	ctx, cancel := c.createContext(10 * time.Second)
	defer cancel()

	return c.client.GetClaudeStatus(ctx, req)
}

// GetClaudeLogs gets Claude logs
func (c *DispenseClient) GetClaudeLogs(req *pb.GetClaudeLogsRequest) (*pb.GetClaudeLogsResponse, error) {
	ctx, cancel := c.createContext(30 * time.Second)
	defer cancel()

	return c.client.GetClaudeLogs(ctx, req)
}

// Config methods

// GetAPIKey gets the API key
func (c *DispenseClient) GetAPIKey(req *pb.GetAPIKeyRequest) (*pb.GetAPIKeyResponse, error) {
	ctx, cancel := c.createContext(30 * time.Second)
	defer cancel()

	return c.client.GetAPIKey(ctx, req)
}

// SetAPIKey sets the API key
func (c *DispenseClient) SetAPIKey(req *pb.SetAPIKeyRequest) (*pb.SetAPIKeyResponse, error) {
	ctx, cancel := c.createContext(10 * time.Second)
	defer cancel()

	return c.client.SetAPIKey(ctx, req)
}

// ValidateAPIKey validates an API key
func (c *DispenseClient) ValidateAPIKey(req *pb.ValidateAPIKeyRequest) (*pb.ValidateAPIKeyResponse, error) {
	ctx, cancel := c.createContext(30 * time.Second)
	defer cancel()

	return c.client.ValidateAPIKey(ctx, req)
}

// Convenience methods

// CreateLocalSandbox creates a local sandbox with simplified parameters
func (c *DispenseClient) CreateLocalSandbox(name string) (*pb.CreateSandboxResponse, error) {
	return c.CreateSandbox(&pb.CreateSandboxRequest{
		Name:     name,
		IsRemote: false,
	})
}

// CreateRemoteSandbox creates a remote sandbox with simplified parameters
func (c *DispenseClient) CreateRemoteSandbox(name string, resources *pb.ResourceAllocation) (*pb.CreateSandboxResponse, error) {
	return c.CreateSandbox(&pb.CreateSandboxRequest{
		Name:      name,
		IsRemote:  true,
		Resources: resources,
	})
}

// ListAllSandboxes lists both local and remote sandboxes
func (c *DispenseClient) ListAllSandboxes() (*pb.ListSandboxesResponse, error) {
	return c.ListSandboxes(&pb.ListSandboxesRequest{
		ShowLocal:  true,
		ShowRemote: true,
	})
}

// RunTaskInSandbox runs a Claude task in the specified sandbox
func (c *DispenseClient) RunTaskInSandbox(sandboxID, task string, handler func(*pb.RunClaudeTaskResponse) error) error {
	return c.RunClaudeTask(&pb.RunClaudeTaskRequest{
		SandboxIdentifier: sandboxID,
		TaskDescription:   task,
	}, handler)
}
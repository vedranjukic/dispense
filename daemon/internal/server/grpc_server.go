package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"daemon/proto"

	"google.golang.org/grpc"
)

type GRPCServer struct {
	server *grpc.Server
}

type ProjectServiceServer struct {
	proto.UnimplementedProjectServiceServer
}

type AgentServiceServer struct {
	proto.UnimplementedAgentServiceServer
	taskManager *TaskManager
}

// NewGRPCServer creates a new gRPC server instance
func NewGRPCServer() *GRPCServer {
	return &GRPCServer{}
}

// Start starts the gRPC server on the specified port
func (s *GRPCServer) Start(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %v", port, err)
	}

	s.server = grpc.NewServer()
	
	// Create log directory for Claude tasks
	homeDir, _ := os.UserHomeDir()
	logDir := filepath.Join(homeDir, ".dispense", "logs")
	taskManager := NewTaskManager(logDir)

	// Register services
	projectServer := &ProjectServiceServer{}
	agentServer := &AgentServiceServer{
		taskManager: taskManager,
	}
	
	proto.RegisterProjectServiceServer(s.server, projectServer)
	proto.RegisterAgentServiceServer(s.server, agentServer)

	log.Printf("Starting gRPC server on port %s", port)
	
	if err := s.server.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve gRPC server: %v", err)
	}

	return nil
}

// Stop gracefully stops the gRPC server
func (s *GRPCServer) Stop() {
	if s.server != nil {
		log.Println("Stopping gRPC server...")
		s.server.GracefulStop()
		log.Println("gRPC server stopped")
	}
}

// ProjectService implementation

// Init initializes the project
func (s *ProjectServiceServer) Init(ctx context.Context, req *proto.InitRequest) (*proto.InitResponse, error) {
	log.Printf("ProjectService.Init called with project_type: %s", req.ProjectType)
	
	// Add your project initialization logic here
	// You can now use req.ProjectType to determine how to initialize the project
	
	return &proto.InitResponse{
		Success: true,
		Message: fmt.Sprintf("Project initialization started for type: %s", req.ProjectType),
	}, nil
}

// Logs streams project logs
func (s *ProjectServiceServer) Logs(req *proto.LogsRequest, stream proto.ProjectService_LogsServer) error {
	log.Println("ProjectService.Logs called - starting log stream")
	
	// Simulate log streaming
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			log.Println("Log stream context cancelled")
			return nil
		case <-ticker.C:
			logEntry := fmt.Sprintf("Project log entry at %s", time.Now().Format(time.RFC3339))
			
			if err := stream.Send(&proto.LogsResponse{
				LogEntry: logEntry,
				Timestamp: time.Now().Unix(),
			}); err != nil {
				log.Printf("Error sending log: %v", err)
				return err
			}
		}
	}
}

// AgentService implementation

// Init initializes the agent
func (s *AgentServiceServer) Init(ctx context.Context, req *proto.InitRequest) (*proto.InitResponse, error) {
	log.Printf("AgentService.Init called with project_type: %s", req.ProjectType)
	
	// Add your agent initialization logic here
	// You can now use req.ProjectType to determine how to initialize the agent
	
	return &proto.InitResponse{
		Success: true,
		Message: fmt.Sprintf("Agent initialized successfully for type: %s", req.ProjectType),
	}, nil
}

// CreateTask creates a new task based on the prompt
func (s *AgentServiceServer) CreateTask(ctx context.Context, req *proto.CreateTaskRequest) (*proto.CreateTaskResponse, error) {
	log.Printf("AgentService.CreateTask called with prompt: %s", req.Prompt)

	// Start Claude task using the task manager
	taskID, err := s.taskManager.StartClaudeTask(req.Prompt, req.WorkingDirectory, req.AnthropicApiKey, req.Model, req.EnvironmentVars)
	if err != nil {
		log.Printf("Failed to start Claude task: %v", err)
		return &proto.CreateTaskResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to start Claude task: %v", err),
		}, nil
	}

	return &proto.CreateTaskResponse{
		Success: true,
		TaskId:  taskID,
		Message: fmt.Sprintf("Claude task started successfully with ID: %s", taskID),
	}, nil
}

// ExecuteClaude executes Claude with a prompt and streams the output
func (s *AgentServiceServer) ExecuteClaude(req *proto.ExecuteClaudeRequest, stream proto.AgentService_ExecuteClaudeServer) error {
	log.Printf("AgentService.ExecuteClaude called with prompt: %s", req.Prompt)

	// Start Claude task
	taskID, err := s.taskManager.StartClaudeTask(req.Prompt, req.WorkingDirectory, req.AnthropicApiKey, req.Model, req.EnvironmentVars)
	if err != nil {
		log.Printf("Failed to start Claude task: %v", err)
		return stream.Send(&proto.ExecuteClaudeResponse{
			Type:      proto.ExecuteClaudeResponse_ERROR,
			Content:   fmt.Sprintf("Failed to start Claude: %v", err),
			Timestamp: time.Now().Unix(),
		})
	}

	log.Printf("Started Claude task %s, streaming output...", taskID)

	// Stream the task output
	return s.taskManager.StreamTaskOutput(taskID, stream)
}

// GetTaskStatus returns the status of a specific task
func (s *AgentServiceServer) GetTaskStatus(ctx context.Context, req *proto.TaskStatusRequest) (*proto.TaskStatusResponse, error) {
	log.Printf("AgentService.GetTaskStatus called for task: %s", req.TaskId)

	status, err := s.taskManager.GetTaskStatus(req.TaskId)
	if err != nil {
		log.Printf("Failed to get task status: %v", err)
		return &proto.TaskStatusResponse{
			State:   proto.TaskStatusResponse_FAILED,
			Message: fmt.Sprintf("Task not found: %v", err),
		}, nil
	}

	return status, nil
}

// ListTasks returns a list of all tasks
func (s *AgentServiceServer) ListTasks(ctx context.Context, req *proto.ListTasksRequest) (*proto.ListTasksResponse, error) {
	log.Printf("AgentService.ListTasks called")

	var stateFilter *proto.TaskStatusResponse_TaskState
	if req.StateFilter != proto.TaskStatusResponse_PENDING || req.StateFilter != 0 {
		stateFilter = &req.StateFilter
	}

	tasks, err := s.taskManager.ListTasks(stateFilter)
	if err != nil {
		log.Printf("Failed to list tasks: %v", err)
		return &proto.ListTasksResponse{
			Tasks: []*proto.TaskInfo{},
		}, nil
	}

	return &proto.ListTasksResponse{
		Tasks: tasks,
	}, nil
}

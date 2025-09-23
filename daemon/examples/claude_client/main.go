package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"daemon/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connect to the daemon
	conn, err := grpc.Dial("localhost:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to daemon: %v", err)
	}
	defer conn.Close()

	// Create agent service client
	client := proto.NewAgentServiceClient(conn)

	// Test 1: Create a simple task
	fmt.Println("=== Testing CreateTask ===")
	createReq := &proto.CreateTaskRequest{
		Prompt: "Write a simple Hello World function in Python",
	}

	createResp, err := client.CreateTask(context.Background(), createReq)
	if err != nil {
		log.Fatalf("CreateTask failed: %v", err)
	}

	fmt.Printf("Task created: ID=%s, Success=%v, Message=%s\n",
		createResp.TaskId, createResp.Success, createResp.Message)

	// Test 2: Execute Claude with streaming output
	fmt.Println("\n=== Testing ExecuteClaude with streaming ===")
	executeReq := &proto.ExecuteClaudeRequest{
		Prompt:           "List the files in the current directory and explain what each one does",
		WorkingDirectory: "/tmp",
		EnvironmentVars: map[string]string{
			"HOME": "/home/dispense",
		},
	}

	stream, err := client.ExecuteClaude(context.Background(), executeReq)
	if err != nil {
		log.Fatalf("ExecuteClaude failed: %v", err)
	}

	fmt.Println("Streaming Claude output:")
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			fmt.Println("Stream finished")
			break
		}
		if err != nil {
			log.Printf("Stream error: %v", err)
			break
		}

		timestamp := time.Unix(resp.Timestamp, 0).Format(time.RFC3339)
		switch resp.Type {
		case proto.ExecuteClaudeResponse_STDOUT:
			fmt.Printf("[%s] STDOUT: %s\n", timestamp, resp.Content)
		case proto.ExecuteClaudeResponse_STDERR:
			fmt.Printf("[%s] STDERR: %s\n", timestamp, resp.Content)
		case proto.ExecuteClaudeResponse_STATUS:
			fmt.Printf("[%s] STATUS: %s (exit code: %d)\n", timestamp, resp.Content, resp.ExitCode)
			if resp.IsFinished {
				fmt.Println("Claude execution finished")
				break
			}
		case proto.ExecuteClaudeResponse_ERROR:
			fmt.Printf("[%s] ERROR: %s\n", timestamp, resp.Content)
		}
	}

	// Test 3: Check task status
	if createResp.Success {
		fmt.Println("\n=== Testing GetTaskStatus ===")
		statusReq := &proto.TaskStatusRequest{
			TaskId: createResp.TaskId,
		}

		statusResp, err := client.GetTaskStatus(context.Background(), statusReq)
		if err != nil {
			log.Printf("GetTaskStatus failed: %v", err)
		} else {
			startedAt := time.Unix(statusResp.StartedAt, 0).Format(time.RFC3339)
			finishedAt := ""
			if statusResp.FinishedAt > 0 {
				finishedAt = time.Unix(statusResp.FinishedAt, 0).Format(time.RFC3339)
			}

			fmt.Printf("Task Status: State=%s, Message=%s, ExitCode=%d\n",
				statusResp.State, statusResp.Message, statusResp.ExitCode)
			fmt.Printf("Started: %s, Finished: %s\n", startedAt, finishedAt)
		}
	}

	fmt.Println("\nAll tests completed!")
}
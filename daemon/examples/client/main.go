package main

import (
	"context"
	"fmt"
	"log"

	"daemon/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connect to the gRPC server
	conn, err := grpc.Dial("localhost:28080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Create clients
	projectClient := proto.NewProjectServiceClient(conn)
	agentClient := proto.NewAgentServiceClient(conn)

	// Test ProjectService.Init
	fmt.Println("Testing ProjectService.Init...")
	projectInitResp, err := projectClient.Init(context.Background(), &proto.InitRequest{
		ProjectType: "web-application",
	})
	if err != nil {
		log.Printf("ProjectService.Init failed: %v", err)
	} else {
		fmt.Printf("ProjectService.Init response: %+v\n", projectInitResp)
	}

	// Test AgentService.Init
	fmt.Println("\nTesting AgentService.Init...")
	agentInitResp, err := agentClient.Init(context.Background(), &proto.InitRequest{
		ProjectType: "agent-service",
	})
	if err != nil {
		log.Printf("AgentService.Init failed: %v", err)
	} else {
		fmt.Printf("AgentService.Init response: %+v\n", agentInitResp)
	}

	// Test AgentService.CreateTask
	fmt.Println("\nTesting AgentService.CreateTask...")
	createTaskResp, err := agentClient.CreateTask(context.Background(), &proto.CreateTaskRequest{
		Prompt: "Create a new task for testing",
	})
	if err != nil {
		log.Printf("AgentService.CreateTask failed: %v", err)
	} else {
		fmt.Printf("AgentService.CreateTask response: %+v\n", createTaskResp)
	}

	// Test ProjectService.Logs (streaming)
	fmt.Println("\nTesting ProjectService.Logs (streaming)...")
	logsStream, err := projectClient.Logs(context.Background(), &proto.LogsRequest{})
	if err != nil {
		log.Printf("ProjectService.Logs failed: %v", err)
		return
	}

	// Read a few log entries
	for i := 0; i < 3; i++ {
		logResp, err := logsStream.Recv()
		if err != nil {
			log.Printf("Error receiving log: %v", err)
			break
		}
		fmt.Printf("Log entry %d: %s (timestamp: %d)\n", i+1, logResp.LogEntry, logResp.Timestamp)
	}

	fmt.Println("\nClient test completed!")
}

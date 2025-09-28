package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "dispense/grpc-server/proto"
	"dispense/grpc-server/server"
	"dispense/grpc-server/server/middleware"
	"cli/internal/services"
)

func main() {
	// Configuration
	port := getEnv("DISPENSE_GRPC_ADDRESS", ":8080")
	apiKey := getEnv("DISPENSE_API_KEY", "")
	authEnabled := getEnv("DISPENSE_GRPC_AUTH_ENABLED", "true") == "true"

	// Create service container
	serviceContainer := services.NewServiceContainer()

	// Create gRPC server with middleware
	var grpcServer *grpc.Server
	if authEnabled && apiKey != "" {
		// With authentication
		authInterceptor := middleware.NewAuthInterceptor(apiKey)
		loggingInterceptor := middleware.NewLoggingInterceptor()
		validationInterceptor := middleware.NewValidationInterceptor()

		grpcServer = grpc.NewServer(
			grpc.ChainUnaryInterceptor(
				loggingInterceptor.UnaryServerInterceptor(),
				authInterceptor.UnaryServerInterceptor(),
				validationInterceptor.UnaryServerInterceptor(),
			),
			grpc.ChainStreamInterceptor(
				loggingInterceptor.StreamServerInterceptor(),
				authInterceptor.StreamServerInterceptor(),
			),
		)
	} else {
		// Without authentication (development mode)
		loggingInterceptor := middleware.NewLoggingInterceptor()
		validationInterceptor := middleware.NewValidationInterceptor()

		grpcServer = grpc.NewServer(
			grpc.ChainUnaryInterceptor(
				loggingInterceptor.UnaryServerInterceptor(),
				validationInterceptor.UnaryServerInterceptor(),
			),
			grpc.ChainStreamInterceptor(
				loggingInterceptor.StreamServerInterceptor(),
			),
		)
	}

	// Create and register dispense server
	dispenseServer := server.NewDispenseServer(serviceContainer)
	pb.RegisterDispenseServiceServer(grpcServer, dispenseServer)

	// Enable gRPC reflection for development
	if getEnv("DISPENSE_GRPC_REFLECTION", "false") == "true" {
		reflection.Register(grpcServer)
		log.Println("gRPC reflection enabled")
	}

	// Create listener
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", port, err)
	}

	log.Printf("Starting Dispense gRPC server on %s", port)
	if !authEnabled {
		log.Println("WARNING: Authentication is disabled (development mode)")
	}

	// Start server in a goroutine
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gRPC server...")
	grpcServer.GracefulStop()
	log.Println("gRPC server stopped")
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
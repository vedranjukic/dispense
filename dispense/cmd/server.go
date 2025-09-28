package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"cli/internal/grpc/gateway"
	"cli/internal/grpc/middleware"
	pb "cli/internal/grpc/proto"
	"cli/internal/grpc/server"
	"cli/internal/services"
	"cli/pkg/utils"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the gRPC server and REST gateway",
	Long: `Start the Dispense gRPC server and REST gateway.

This command starts both a gRPC server and an HTTP/REST gateway:
- gRPC server: Provides native gRPC API for high-performance clients
- HTTP gateway: Provides REST API for web clients and simple integrations

The server exposes all sandbox management, Claude operations, and configuration endpoints.

Use --grpc-only to start only the gRPC server, or --http-only to start only the HTTP gateway.`,
	Run: runServerCommand,
}

func init() {
	// Server configuration flags
	serverCmd.Flags().String("grpc-port", ":8080", "gRPC server port")
	serverCmd.Flags().String("http-port", ":8081", "HTTP gateway port")
	serverCmd.Flags().String("api-key", "", "API key for authentication (reads from DISPENSE_API_KEY env if not provided)")
	serverCmd.Flags().Bool("no-auth", false, "Disable authentication (development mode)")
	serverCmd.Flags().Bool("grpc-reflection", false, "Enable gRPC reflection for debugging")
	serverCmd.Flags().Bool("grpc-only", false, "Start only gRPC server (no HTTP gateway)")
	serverCmd.Flags().Bool("http-only", false, "Start only HTTP gateway (requires existing gRPC server)")
	serverCmd.Flags().String("grpc-endpoint", "localhost:8080", "gRPC endpoint for HTTP gateway (used with --http-only)")
}

func runServerCommand(cmd *cobra.Command, args []string) {
	utils.DebugPrintf("Server command called with args: %v\n", args)

	// Get flags
	grpcPort, _ := cmd.Flags().GetString("grpc-port")
	httpPort, _ := cmd.Flags().GetString("http-port")
	apiKey, _ := cmd.Flags().GetString("api-key")
	noAuth, _ := cmd.Flags().GetBool("no-auth")
	grpcReflection, _ := cmd.Flags().GetBool("grpc-reflection")
	grpcOnly, _ := cmd.Flags().GetBool("grpc-only")
	httpOnly, _ := cmd.Flags().GetBool("http-only")
	grpcEndpoint, _ := cmd.Flags().GetString("grpc-endpoint")

	// Get API key from environment if not provided
	if apiKey == "" {
		apiKey = os.Getenv("DISPENSE_API_KEY")
	}

	// Disable authentication if no-auth flag is set
	if noAuth {
		apiKey = ""
	}

	// Validate conflicting flags
	if grpcOnly && httpOnly {
		fmt.Fprintf(os.Stderr, "Error: --grpc-only and --http-only cannot be used together\n")
		os.Exit(1)
	}

	fmt.Printf("🚀 Starting Dispense Server\n")
	fmt.Printf("==========================\n")

	// Create service container
	serviceContainer := services.NewServiceContainer()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Start gRPC server if not HTTP-only
	if !httpOnly {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := startGRPCServer(ctx, grpcPort, apiKey, grpcReflection, serviceContainer); err != nil {
				fmt.Fprintf(os.Stderr, "gRPC server error: %v\n", err)
			}
		}()
	}

	// Start HTTP gateway if not gRPC-only
	if !grpcOnly {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := startHTTPGateway(ctx, httpPort, grpcEndpoint, apiKey); err != nil {
				fmt.Fprintf(os.Stderr, "HTTP gateway error: %v\n", err)
			}
		}()
	}

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	fmt.Printf("\n🔄 Shutting down servers...\n")

	cancel() // Signal all goroutines to stop
	wg.Wait() // Wait for all servers to shutdown

	fmt.Printf("✅ Server shutdown complete\n")
}

// startGRPCServer starts the gRPC server
func startGRPCServer(ctx context.Context, grpcPort, apiKey string, enableReflection bool, serviceContainer *services.ServiceContainer) error {
	// Create listener
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", grpcPort, err)
	}

	fmt.Printf("📡 gRPC server listening on %s\n", grpcPort)

	// Create middleware
	authInterceptor := middleware.NewAuthInterceptor(apiKey)
	loggingInterceptor := middleware.NewLoggingInterceptor()
	validationInterceptor := middleware.NewValidationInterceptor()

	// Create gRPC server with middleware
	grpcServer := grpc.NewServer(
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

	// Enable reflection if requested
	if enableReflection {
		reflection.Register(grpcServer)
		fmt.Printf("🔍 gRPC reflection enabled\n")
	}

	// Create and register the dispense server
	dispenseServer := server.NewDispenseServer(serviceContainer)
	pb.RegisterDispenseServiceServer(grpcServer, dispenseServer)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- grpcServer.Serve(lis)
	}()

	// Wait for context cancellation or server error
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		fmt.Printf("🔄 Shutting down gRPC server...\n")

		// Graceful shutdown with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		done := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(done)
		}()

		select {
		case <-done:
			fmt.Printf("✅ gRPC server stopped gracefully\n")
		case <-shutdownCtx.Done():
			fmt.Printf("⚠️  gRPC server force stopped\n")
			grpcServer.Stop()
		}

		return nil
	}
}

// startHTTPGateway starts the HTTP gateway server
func startHTTPGateway(ctx context.Context, httpPort, grpcEndpoint, apiKey string) error {
	fmt.Printf("🌐 HTTP gateway connecting to gRPC at %s\n", grpcEndpoint)
	fmt.Printf("🌐 HTTP gateway listening on %s\n", httpPort)

	// Create gateway
	gatewayConfig := &gateway.GatewayConfig{
		GRPCEndpoint: grpcEndpoint,
		HTTPAddress:  httpPort,
		APIKey:       apiKey,
	}

	gw, err := gateway.NewGateway(gatewayConfig)
	if err != nil {
		return fmt.Errorf("failed to create gateway: %w", err)
	}

	// Start gateway in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- gw.Start()
	}()

	// Wait for context cancellation or server error
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		fmt.Printf("🔄 Shutting down HTTP gateway...\n")

		// Graceful shutdown with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		err := gw.Stop(shutdownCtx)
		if err != nil {
			fmt.Printf("⚠️  HTTP gateway shutdown error: %v\n", err)
		} else {
			fmt.Printf("✅ HTTP gateway stopped gracefully\n")
		}

		return nil
	}
}
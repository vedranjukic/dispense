package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "dispense/grpc-server/proto"
	"dispense/grpc-server/server"
	"dispense/grpc-server/server/middleware"
	"dispense/grpc-server/gateway"
	"cli/internal/services"
)

func main() {
	// Configuration
	grpcPort := getEnv("DISPENSE_GRPC_PORT", ":8080")
	httpPort := getEnv("DISPENSE_HTTP_PORT", ":8081")
	apiKey := getEnv("DISPENSE_API_KEY", "")
	authEnabled := getEnv("DISPENSE_GRPC_AUTH_ENABLED", "true") == "true"
	reflectionEnabled := getEnv("DISPENSE_GRPC_REFLECTION", "false") == "true"

	log.Printf("Starting Dispense combined gRPC/HTTP server")
	log.Printf("gRPC endpoint: %s", grpcPort)
	log.Printf("HTTP/REST endpoint: %s", httpPort)

	// Create service container
	serviceContainer := services.NewServiceContainer()

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	// Start gRPC server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := startGRPCServer(ctx, grpcPort, serviceContainer, apiKey, authEnabled, reflectionEnabled); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	// Start HTTP gateway server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := startHTTPGateway(ctx, httpPort, grpcPort, apiKey); err != nil {
			log.Printf("HTTP gateway error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down servers...")
	cancel()

	// Wait for servers to shutdown
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Servers stopped gracefully")
	case <-time.After(30 * time.Second):
		log.Println("Servers shutdown timeout exceeded")
	}
}

// startGRPCServer starts the gRPC server
func startGRPCServer(ctx context.Context, port string, serviceContainer *services.ServiceContainer, apiKey string, authEnabled, reflectionEnabled bool) error {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", port, err)
	}

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

	// Register service
	dispenseServer := server.NewDispenseServer(serviceContainer)
	pb.RegisterDispenseServiceServer(grpcServer, dispenseServer)

	// Enable reflection if requested
	if reflectionEnabled {
		reflection.Register(grpcServer)
		log.Println("gRPC reflection enabled")
	}

	log.Printf("gRPC server listening on %s", port)
	if !authEnabled {
		log.Println("WARNING: gRPC authentication is disabled (development mode)")
	}

	// Start server in background
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Shutting down gRPC server...")
	grpcServer.GracefulStop()

	return nil
}

// startHTTPGateway starts the HTTP/REST gateway
func startHTTPGateway(ctx context.Context, httpPort, grpcPort, apiKey string) error {
	// Wait a bit for gRPC server to start
	time.Sleep(1 * time.Second)

	// Create gateway
	gatewayConfig := &gateway.GatewayConfig{
		GRPCEndpoint: fmt.Sprintf("localhost%s", grpcPort),
		HTTPAddress:  httpPort,
		APIKey:       apiKey,
	}

	gw, err := gateway.NewGateway(gatewayConfig)
	if err != nil {
		return fmt.Errorf("failed to create gateway: %w", err)
	}

	// Add middleware
	mux := gw.GetMux()

	// Create HTTP server with middleware stack
	loggingMiddleware := gateway.NewLoggingMiddleware()
	healthCheckMiddleware := gateway.NewHealthCheckMiddleware()
	securityMiddleware := gateway.NewSecurityMiddleware()
	rateLimitMiddleware := gateway.NewRateLimitMiddleware(100, time.Minute)

	// Build middleware chain
	handler := loggingMiddleware.Handler(
		healthCheckMiddleware.Handler(
			securityMiddleware.Handler(
				rateLimitMiddleware.Handler(mux),
			),
		),
	)

	httpServer := &http.Server{
		Addr:         httpPort,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("HTTP gateway listening on %s", httpPort)
	if apiKey == "" {
		log.Println("WARNING: HTTP gateway authentication is disabled (development mode)")
	}

	// Start server in background
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Shutting down HTTP gateway...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return httpServer.Shutdown(shutdownCtx)
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dispense/grpc-server/gateway"
)

func main() {
	// Configuration
	grpcEndpoint := getEnv("DISPENSE_GRPC_ENDPOINT", "localhost:8080")
	httpPort := getEnv("DISPENSE_HTTP_PORT", ":8081")
	apiKey := getEnv("DISPENSE_API_KEY", "")

	log.Printf("Starting Dispense HTTP/REST Gateway")
	log.Printf("Connecting to gRPC server: %s", grpcEndpoint)
	log.Printf("HTTP gateway listening on: %s", httpPort)

	// Create gateway
	gatewayConfig := &gateway.GatewayConfig{
		GRPCEndpoint: grpcEndpoint,
		HTTPAddress:  httpPort,
		APIKey:       apiKey,
	}

	gw, err := gateway.NewGateway(gatewayConfig)
	if err != nil {
		log.Fatalf("Failed to create gateway: %v", err)
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

	if apiKey == "" {
		log.Println("WARNING: Authentication is disabled (development mode)")
	}

	// Start server in a goroutine
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	log.Println("HTTP gateway started successfully")

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down HTTP gateway...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("HTTP gateway stopped")
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
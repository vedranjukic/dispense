package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"daemon/internal/server"
)

// Build metadata - set during compilation
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

func main() {
	// Parse command line flags
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Print version information and exit")
	flag.Parse()

	// Handle version flag
	if showVersion {
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		fmt.Printf("Build Time: %s\n", BuildTime)
		return
	}

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the daemon
	log.Println("Starting daemon...")
	
	// Create and start gRPC server
	grpcServer := server.NewGRPCServer()
	
	// Start gRPC server in a goroutine
	go func() {
		if err := grpcServer.Start("28080"); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()
	
	// Run the daemon in a goroutine
	go runDaemon(ctx)

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutdown signal received, stopping daemon...")
	
	// Stop gRPC server
	grpcServer.Stop()
	
	// Cancel the context to stop the daemon
	cancel()
	
	// Give the daemon time to clean up
	time.Sleep(1 * time.Second)
	log.Println("Daemon stopped")
}

func runDaemon(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Daemon context cancelled, stopping...")
			return
		case <-ticker.C:
			log.Println("Daemon is running... (gRPC server available on port 28080)")
			// Add your daemon logic here
		}
	}
}

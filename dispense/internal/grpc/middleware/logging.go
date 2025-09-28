package middleware

import (
	"context"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor provides logging middleware for gRPC
type LoggingInterceptor struct {
	logger *log.Logger
}

// NewLoggingInterceptor creates a new logging interceptor
func NewLoggingInterceptor() *LoggingInterceptor {
	return &LoggingInterceptor{
		logger: log.New(os.Stdout, "[grpc-middleware] ", log.LstdFlags),
	}
}

// UnaryServerInterceptor returns a unary server interceptor for logging
func (l *LoggingInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		l.logger.Printf("Start: %s", info.FullMethod)

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		code := status.Code(err)

		if err != nil {
			l.logger.Printf("End: %s [%v] (%v) - Error: %v", info.FullMethod, code, duration, err)
		} else {
			l.logger.Printf("End: %s [%v] (%v)", info.FullMethod, code, duration)
		}

		return resp, err
	}
}

// StreamServerInterceptor returns a stream server interceptor for logging
func (l *LoggingInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		l.logger.Printf("Start stream: %s", info.FullMethod)

		err := handler(srv, stream)

		duration := time.Since(start)
		code := status.Code(err)

		if err != nil {
			l.logger.Printf("End stream: %s [%v] (%v) - Error: %v", info.FullMethod, code, duration, err)
		} else {
			l.logger.Printf("End stream: %s [%v] (%v)", info.FullMethod, code, duration)
		}

		return err
	}
}
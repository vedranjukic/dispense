package middleware

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthInterceptor provides authentication middleware for gRPC
type AuthInterceptor struct {
	apiKey string
}

// NewAuthInterceptor creates a new auth interceptor
func NewAuthInterceptor(apiKey string) *AuthInterceptor {
	return &AuthInterceptor{
		apiKey: apiKey,
	}
}

// UnaryServerInterceptor returns a unary server interceptor for authentication
func (a *AuthInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip auth for certain methods like GetAPIKey and SetAPIKey
		if a.skipAuth(info.FullMethod) {
			return handler(ctx, req)
		}

		if err := a.authenticate(ctx); err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns a stream server interceptor for authentication
func (a *AuthInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip auth for certain methods
		if a.skipAuth(info.FullMethod) {
			return handler(srv, stream)
		}

		if err := a.authenticate(stream.Context()); err != nil {
			return err
		}

		return handler(srv, stream)
	}
}

// authenticate validates the API key from metadata
func (a *AuthInterceptor) authenticate(ctx context.Context) error {
	// Skip authentication if no API key is configured
	if a.apiKey == "" {
		return nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "metadata not found")
	}

	apiKeys := md.Get("api-key")
	if len(apiKeys) == 0 {
		return status.Error(codes.Unauthenticated, "api key not found")
	}

	if apiKeys[0] != a.apiKey {
		return status.Error(codes.Unauthenticated, "invalid api key")
	}

	return nil
}

// skipAuth determines if authentication should be skipped for certain methods
func (a *AuthInterceptor) skipAuth(method string) bool {
	// Skip auth for config-related methods that might be used to set up auth
	skipMethods := []string{
		"/dispense.DispenseService/GetAPIKey",
		"/dispense.DispenseService/SetAPIKey",
		"/dispense.DispenseService/ValidateAPIKey",
	}

	for _, skipMethod := range skipMethods {
		if strings.Contains(method, skipMethod) {
			return true
		}
	}

	return false
}
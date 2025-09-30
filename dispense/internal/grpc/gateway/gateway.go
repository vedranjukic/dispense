package gateway

import (
	"context"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "cli/internal/grpc/proto"
	"cli/internal/dashboard"
)

// Gateway wraps the gRPC-Gateway server
type Gateway struct {
	mux    *runtime.ServeMux
	server *http.Server
}

// GatewayConfig contains configuration for the gateway
type GatewayConfig struct {
	GRPCEndpoint string
	HTTPAddress  string
	APIKey       string
}

// NewGateway creates a new gRPC-Gateway instance
func NewGateway(config *GatewayConfig) (*Gateway, error) {
	// Create runtime mux with custom options
	mux := runtime.NewServeMux(
		runtime.WithMetadata(annotateContext),
		runtime.WithIncomingHeaderMatcher(customHeaderMatcher),
		runtime.WithOutgoingHeaderMatcher(customHeaderMatcher),
		runtime.WithErrorHandler(customErrorHandler),
	)

	// Register the gRPC service with the gateway
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := pb.RegisterDispenseServiceHandlerFromEndpoint(
		context.Background(),
		mux,
		config.GRPCEndpoint,
		opts,
	)
	if err != nil {
		return nil, err
	}

	// Create main HTTP handler that combines gRPC gateway and dashboard
	mainHandler, err := createMainHandler(mux, config.APIKey)
	if err != nil {
		return nil, err
	}

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    config.HTTPAddress,
		Handler: corsHandler(mainHandler),
	}

	return &Gateway{
		mux:    mux,
		server: httpServer,
	}, nil
}

// Start starts the HTTP gateway server
func (g *Gateway) Start() error {
	return g.server.ListenAndServe()
}

// Stop stops the HTTP gateway server
func (g *Gateway) Stop(ctx context.Context) error {
	return g.server.Shutdown(ctx)
}

// GetMux returns the runtime mux for custom routing
func (g *Gateway) GetMux() *runtime.ServeMux {
	return g.mux
}

// annotateContext adds metadata from HTTP headers to gRPC context
func annotateContext(ctx context.Context, req *http.Request) metadata.MD {
	md := make(metadata.MD)

	// Forward authorization headers
	if auth := req.Header.Get("Authorization"); auth != "" {
		md.Set("authorization", auth)
	}

	// Forward API key headers
	if apiKey := req.Header.Get("X-API-Key"); apiKey != "" {
		md.Set("api-key", apiKey)
	}

	// Forward user agent
	if ua := req.Header.Get("User-Agent"); ua != "" {
		md.Set("user-agent", ua)
	}

	return md
}

// customHeaderMatcher determines which headers to forward
func customHeaderMatcher(key string) (string, bool) {
	switch strings.ToLower(key) {
	case "x-api-key":
		return key, true
	case "authorization":
		return key, true
	case "x-request-id":
		return key, true
	case "user-agent":
		return key, true
	default:
		return key, false
	}
}

// customErrorHandler provides custom error handling
func customErrorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, req *http.Request, err error) {
	// Use default error handler with custom headers
	w.Header().Set("Content-Type", "application/json")

	// Add CORS headers for error responses
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

	// Use the default error handler
	runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, req, err)
}

// authMiddleware provides HTTP-level authentication
func authMiddleware(next http.Handler, apiKey string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for preflight requests
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth for certain endpoints or if no API key configured
		if apiKey == "" || skipAuthForPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Check for API key in various places
		reqAPIKey := getAPIKeyFromRequest(r)
		if reqAPIKey == "" {
			http.Error(w, `{"error": "API key required"}`, http.StatusUnauthorized)
			return
		}

		if reqAPIKey != apiKey {
			http.Error(w, `{"error": "Invalid API key"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// corsHandler adds CORS headers
func corsHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Request-ID")
		w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getAPIKeyFromRequest extracts API key from various sources
func getAPIKeyFromRequest(r *http.Request) string {
	// Check X-API-Key header
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return apiKey
	}

	// Check Authorization header (Bearer token)
	if auth := r.Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			return auth[7:] // Remove "Bearer " prefix
		}
	}

	// Check query parameter
	return r.URL.Query().Get("api_key")
}

// createMainHandler creates the main HTTP handler that routes between dashboard and API
func createMainHandler(mux *runtime.ServeMux, apiKey string) (http.Handler, error) {
	// Get dashboard handler
	dashboardHandler, err := dashboard.GetHandler()
	if err != nil {
		return nil, err
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Route API requests to gRPC gateway with auth
		if strings.HasPrefix(r.URL.Path, "/api/") {
			// Strip /api prefix and route to gRPC gateway
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/api")
			authMiddleware(mux, apiKey).ServeHTTP(w, r)
			return
		}

		// Route all other requests (including root) to dashboard
		dashboardHandler.ServeHTTP(w, r)
	}), nil
}

// skipAuthForPath determines if authentication should be skipped for certain paths
func skipAuthForPath(path string) bool {
	skipPaths := []string{
		"/v1/config/api-key",
		"/v1/config/api-key/validate",
	}

	for _, skipPath := range skipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}

	return false
}
package gateway

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// LoggingMiddleware provides HTTP request logging
type LoggingMiddleware struct {
	logger *log.Logger
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware() *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: log.New(os.Stdout, "[http-gateway] ", log.LstdFlags),
	}
}

// Handler returns the HTTP middleware handler
func (l *LoggingMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a wrapped response writer to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		l.logger.Printf("Start: %s %s %s", r.Method, r.URL.Path, r.RemoteAddr)

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		l.logger.Printf("End: %s %s [%d] (%v)", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// RateLimitMiddleware provides simple rate limiting
type RateLimitMiddleware struct {
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(limit int, window time.Duration) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// Handler returns the HTTP middleware handler
func (rl *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract client IP (simplified)
		clientIP := r.RemoteAddr
		if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			clientIP = strings.Split(forwardedFor, ",")[0]
		}

		now := time.Now()

		// Clean old requests
		if requests, exists := rl.requests[clientIP]; exists {
			var validRequests []time.Time
			for _, reqTime := range requests {
				if now.Sub(reqTime) < rl.window {
					validRequests = append(validRequests, reqTime)
				}
			}
			rl.requests[clientIP] = validRequests
		}

		// Check rate limit
		if len(rl.requests[clientIP]) >= rl.limit {
			http.Error(w, `{"error": "Rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}

		// Add current request
		rl.requests[clientIP] = append(rl.requests[clientIP], now)

		next.ServeHTTP(w, r)
	})
}

// HealthCheckMiddleware provides health check endpoint
type HealthCheckMiddleware struct{}

// NewHealthCheckMiddleware creates a new health check middleware
func NewHealthCheckMiddleware() *HealthCheckMiddleware {
	return &HealthCheckMiddleware{}
}

// Handler returns the HTTP middleware handler
func (h *HealthCheckMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health check endpoint
		if r.URL.Path == "/health" || r.URL.Path == "/healthz" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "healthy", "service": "dispense-gateway"}`))
			return
		}

		// Readiness check
		if r.URL.Path == "/ready" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ready", "service": "dispense-gateway"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SecurityMiddleware adds security headers
type SecurityMiddleware struct{}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware() *SecurityMiddleware {
	return &SecurityMiddleware{}
}

// Handler returns the HTTP middleware handler
func (s *SecurityMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}

// CompressionMiddleware adds gzip compression (simplified)
type CompressionMiddleware struct{}

// NewCompressionMiddleware creates a new compression middleware
func NewCompressionMiddleware() *CompressionMiddleware {
	return &CompressionMiddleware{}
}

// Handler returns the HTTP middleware handler
func (c *CompressionMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if client accepts gzip
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
		}

		next.ServeHTTP(w, r)
	})
}
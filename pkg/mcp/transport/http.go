// Package transport implements MCP transport layers
package transport

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"mcp-memory/pkg/mcp/protocol"
)

// HTTPConfig contains configuration for HTTP transport
type HTTPConfig struct {
	// Address to listen on (e.g., ":8080" or "localhost:8080")
	Address string

	// TLS configuration (optional)
	TLSConfig *tls.Config

	// Read/write timeouts
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	// Maximum request body size (default: 10MB)
	MaxBodySize int64

	// Custom headers to include in responses
	CustomHeaders map[string]string

	// Enable CORS
	EnableCORS bool

	// CORS allowed origins (if EnableCORS is true)
	AllowedOrigins []string

	// Path to handle requests (default: "/")
	Path string
}

// HTTPTransport implements HTTP/HTTPS transport for MCP
type HTTPTransport struct {
	config   *HTTPConfig
	server   *http.Server
	handler  RequestHandler
	mu       sync.RWMutex
	running  bool
	certFile string
	keyFile  string
}

// NewHTTPTransport creates a new HTTP transport
func NewHTTPTransport(config *HTTPConfig) *HTTPTransport {
	if config.MaxBodySize == 0 {
		config.MaxBodySize = 10 * 1024 * 1024 // 10MB default
	}
	if config.Path == "" {
		config.Path = "/"
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 30 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 30 * time.Second
	}
	if config.IdleTimeout == 0 {
		config.IdleTimeout = 120 * time.Second
	}

	return &HTTPTransport{
		config: config,
	}
}

// NewHTTPSTransport creates a new HTTPS transport
func NewHTTPSTransport(config *HTTPConfig, certFile, keyFile string) *HTTPTransport {
	transport := NewHTTPTransport(config)
	transport.certFile = certFile
	transport.keyFile = keyFile
	return transport
}

// Start starts the HTTP transport
func (t *HTTPTransport) Start(ctx context.Context, handler RequestHandler) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return fmt.Errorf("transport already running")
	}

	t.handler = handler

	mux := http.NewServeMux()
	mux.HandleFunc(t.config.Path, t.handleRequest)

	t.server = &http.Server{
		Addr:         t.config.Address,
		Handler:      t.wrapWithMiddleware(mux),
		ReadTimeout:  t.config.ReadTimeout,
		WriteTimeout: t.config.WriteTimeout,
		IdleTimeout:  t.config.IdleTimeout,
		TLSConfig:    t.config.TLSConfig,
	}

	t.running = true

	// Start server in a goroutine
	go func() {
		var err error
		if t.certFile != "" && t.keyFile != "" {
			err = t.server.ListenAndServeTLS(t.certFile, t.keyFile)
		} else {
			err = t.server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			t.mu.Lock()
			t.running = false
			t.mu.Unlock()
		}
	}()

	// Wait for context cancellation
	go func() {
		<-ctx.Done()
		t.Stop()
	}()

	return nil
}

// Stop stops the HTTP transport
func (t *HTTPTransport) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return nil
	}

	t.running = false

	if t.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return t.server.Shutdown(ctx)
	}

	return nil
}

// handleRequest handles incoming HTTP requests
func (t *HTTPTransport) handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, t.config.MaxBodySize)

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.writeError(w, protocol.ParseError, "Failed to read request body", nil)
		return
	}

	// Parse JSON-RPC request
	var req protocol.JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.writeError(w, protocol.ParseError, "Invalid JSON", nil)
		return
	}

	// Handle request
	ctx := r.Context()
	resp := t.handler.HandleRequest(ctx, &req)

	// Write response
	t.writeResponse(w, resp)
}

// writeResponse writes a JSON-RPC response
func (t *HTTPTransport) writeResponse(w http.ResponseWriter, resp *protocol.JSONRPCResponse) {
	w.Header().Set("Content-Type", "application/json")
	
	// Add custom headers
	for k, v := range t.config.CustomHeaders {
		w.Header().Set(k, v)
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		// Log error but can't send another response
		fmt.Printf("Failed to encode response: %v\n", err)
	}
}

// writeError writes a JSON-RPC error response
func (t *HTTPTransport) writeError(w http.ResponseWriter, code int, message string, data interface{}) {
	resp := &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		Error:   protocol.NewJSONRPCError(code, message, data),
	}
	t.writeResponse(w, resp)
}

// wrapWithMiddleware wraps the handler with middleware
func (t *HTTPTransport) wrapWithMiddleware(handler http.Handler) http.Handler {
	// CORS middleware
	if t.config.EnableCORS {
		handler = t.corsMiddleware(handler)
	}

	// Recovery middleware
	handler = t.recoveryMiddleware(handler)

	return handler
}

// corsMiddleware adds CORS headers
func (t *HTTPTransport) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		
		// Check if origin is allowed
		allowed := false
		if len(t.config.AllowedOrigins) == 0 {
			allowed = true // Allow all origins if none specified
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			for _, allowedOrigin := range t.config.AllowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// recoveryMiddleware recovers from panics
func (t *HTTPTransport) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				t.writeError(w, protocol.InternalError, "Internal server error", nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// IsRunning returns whether the transport is running
func (t *HTTPTransport) IsRunning() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.running
}
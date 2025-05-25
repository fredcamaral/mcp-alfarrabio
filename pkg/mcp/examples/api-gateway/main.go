package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"mcp-memory/pkg/mcp/protocol"
	"mcp-memory/pkg/mcp/server"
	"mcp-memory/pkg/mcp/transport"

	"golang.org/x/time/rate"
	"gopkg.in/yaml.v3"
)

// Config represents the API gateway configuration
type Config struct {
	APIs map[string]APIConfig `yaml:"apis"`
}

// APIConfig represents configuration for a single API
type APIConfig struct {
	BaseURL    string            `yaml:"base_url"`
	AuthType   string            `yaml:"auth_type"` // "api_key", "bearer", "oauth2"
	AuthConfig map[string]string `yaml:"auth_config"`
	RateLimit  RateLimitConfig   `yaml:"rate_limit"`
	CacheTTL   time.Duration     `yaml:"cache_ttl"`
	Endpoints  []EndpointConfig  `yaml:"endpoints"`
	Headers    map[string]string `yaml:"headers"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	RequestsPerSecond int `yaml:"requests_per_second"`
	Burst             int `yaml:"burst"`
}

// EndpointConfig represents a single endpoint configuration
type EndpointConfig struct {
	Name        string            `yaml:"name"`
	Path        string            `yaml:"path"`
	Method      string            `yaml:"method"`
	Description string            `yaml:"description"`
	Parameters  []ParameterConfig `yaml:"parameters"`
}

// ParameterConfig represents a parameter configuration
type ParameterConfig struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Required    bool   `yaml:"required"`
	Description string `yaml:"description"`
	In          string `yaml:"in"` // "query", "path", "body"
}

// CacheEntry represents a cached response
type CacheEntry struct {
	Response  interface{}
	ExpiresAt time.Time
}

// APIGatewayServer implements an MCP server that proxies to external APIs
type APIGatewayServer struct {
	*server.MCPServer
	config       Config
	httpClient   *http.Client
	rateLimiters map[string]*rate.Limiter
	cache        map[string]*CacheEntry
	cacheMutex   sync.RWMutex
}

// NewAPIGatewayServer creates a new API gateway server
func NewAPIGatewayServer() *APIGatewayServer {
	s := &APIGatewayServer{
		MCPServer:    server.NewMCPServer("api-gateway", "1.0.0"),
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		rateLimiters: make(map[string]*rate.Limiter),
		cache:        make(map[string]*CacheEntry),
	}

	// Load configuration
	if err := s.loadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize rate limiters
	for apiName, apiConfig := range s.config.APIs {
		if apiConfig.RateLimit.RequestsPerSecond > 0 {
			s.rateLimiters[apiName] = rate.NewLimiter(
				rate.Limit(apiConfig.RateLimit.RequestsPerSecond),
				apiConfig.RateLimit.Burst,
			)
		}
	}

	// Register tools for each API endpoint
	s.registerTools()

	return s
}

func (s *APIGatewayServer) loadConfig() error {
	configPath := os.Getenv("API_GATEWAY_CONFIG")
	if configPath == "" {
		configPath = "config.yaml"
	}

	data, err := os.ReadFile(configPath) // #nosec G304 -- Config file path from environment variable or default
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	return yaml.Unmarshal(data, &s.config)
}

func (s *APIGatewayServer) registerTools() {
	for apiName, apiConfig := range s.config.APIs {
		for _, endpoint := range apiConfig.Endpoints {
			toolName := fmt.Sprintf("%s_%s", apiName, endpoint.Name)
			s.registerTool(toolName, apiName, apiConfig, endpoint)
		}
	}
}

func (s *APIGatewayServer) registerTool(toolName, apiName string, apiConfig APIConfig, endpoint EndpointConfig) {
	// Build parameter schema
	properties := make(map[string]interface{})
	required := []string{}

	for _, param := range endpoint.Parameters {
		paramSchema := map[string]interface{}{
			"type":        param.Type,
			"description": param.Description,
		}
		properties[param.Name] = paramSchema

		if param.Required {
			required = append(required, param.Name)
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	// Create tool handler
	handler := func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		return s.handleAPICall(ctx, apiName, apiConfig, endpoint, params)
	}

	// Register the tool
	s.RegisterTool(protocol.Tool{
		Name:        toolName,
		Description: endpoint.Description,
		InputSchema: schema,
	}, handler)
}

func (s *APIGatewayServer) handleAPICall(ctx context.Context, apiName string, apiConfig APIConfig, endpoint EndpointConfig, params map[string]interface{}) (interface{}, error) {
	// Check rate limit
	if limiter, exists := s.rateLimiters[apiName]; exists {
		if !limiter.Allow() {
			return nil, fmt.Errorf("rate limit exceeded for %s", apiName)
		}
	}

	// Build cache key
	cacheKey := s.buildCacheKey(apiName, endpoint.Name, params)

	// Check cache
	if apiConfig.CacheTTL > 0 {
		if cached, found := s.getCached(cacheKey); found {
			return cached, nil
		}
	}

	// Build request URL
	url := s.buildURL(apiConfig.BaseURL, endpoint.Path, endpoint, params)

	// Create request
	var body io.Reader
	if endpoint.Method == "POST" || endpoint.Method == "PUT" || endpoint.Method == "PATCH" {
		if bodyParams := s.extractBodyParams(endpoint, params); len(bodyParams) > 0 {
			jsonBody, _ := json.Marshal(bodyParams)
			body = strings.NewReader(string(jsonBody))
		}
	}

	req, err := http.NewRequestWithContext(ctx, endpoint.Method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range apiConfig.Headers {
		req.Header.Set(key, value)
	}

	// Add authentication
	if err := s.addAuthentication(req, apiConfig); err != nil {
		return nil, fmt.Errorf("failed to add authentication: %w", err)
	}

	// Set content type for body
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Make request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(responseBody))
	}

	// Parse response
	var result interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		// If JSON parsing fails, return as string
		result = string(responseBody)
	}

	// Cache response
	if apiConfig.CacheTTL > 0 {
		s.cacheResponse(cacheKey, result, apiConfig.CacheTTL)
	}

	return result, nil
}

func (s *APIGatewayServer) buildURL(baseURL, path string, endpoint EndpointConfig, params map[string]interface{}) string {
	url := baseURL + path

	// Replace path parameters
	for _, param := range endpoint.Parameters {
		if param.In == "path" {
			if value, exists := params[param.Name]; exists {
				placeholder := fmt.Sprintf("{%s}", param.Name)
				url = strings.ReplaceAll(url, placeholder, fmt.Sprint(value))
			}
		}
	}

	// Add query parameters
	queryParams := []string{}
	for _, param := range endpoint.Parameters {
		if param.In == "query" {
			if value, exists := params[param.Name]; exists {
				queryParams = append(queryParams, fmt.Sprintf("%s=%v", param.Name, value))
			}
		}
	}

	if len(queryParams) > 0 {
		url += "?" + strings.Join(queryParams, "&")
	}

	return url
}

func (s *APIGatewayServer) extractBodyParams(endpoint EndpointConfig, params map[string]interface{}) map[string]interface{} {
	bodyParams := make(map[string]interface{})
	for _, param := range endpoint.Parameters {
		if param.In == "body" {
			if value, exists := params[param.Name]; exists {
				bodyParams[param.Name] = value
			}
		}
	}
	return bodyParams
}

func (s *APIGatewayServer) addAuthentication(req *http.Request, apiConfig APIConfig) error {
	switch apiConfig.AuthType {
	case "api_key":
		if key, exists := apiConfig.AuthConfig["key"]; exists {
			if header, exists := apiConfig.AuthConfig["header"]; exists {
				req.Header.Set(header, key)
			} else if param, exists := apiConfig.AuthConfig["param"]; exists {
				// Add to URL
				separator := "?"
				if strings.Contains(req.URL.String(), "?") {
					separator = "&"
				}
				req.URL.RawQuery += separator + param + "=" + key
			}
		}
	case "bearer":
		if token, exists := apiConfig.AuthConfig["token"]; exists {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	case "oauth2":
		// For OAuth2, we'd need to implement token refresh logic
		if token, exists := apiConfig.AuthConfig["access_token"]; exists {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	return nil
}

func (s *APIGatewayServer) buildCacheKey(apiName, endpointName string, params map[string]interface{}) string {
	paramsJSON, _ := json.Marshal(params)
	return fmt.Sprintf("%s:%s:%s", apiName, endpointName, string(paramsJSON))
}

func (s *APIGatewayServer) getCached(key string) (interface{}, bool) {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	if entry, exists := s.cache[key]; exists {
		if time.Now().Before(entry.ExpiresAt) {
			return entry.Response, true
		}
		// Remove expired entry
		delete(s.cache, key)
	}
	return nil, false
}

func (s *APIGatewayServer) cacheResponse(key string, response interface{}, ttl time.Duration) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cache[key] = &CacheEntry{
		Response:  response,
		ExpiresAt: time.Now().Add(ttl),
	}
}

func main() {
	// Create and start the server
	server := NewAPIGatewayServer()

	// Create stdio transport
	stdioTransport := transport.NewStdioTransport()

	// Start server
	log.Println("Starting API Gateway MCP Server...")
	if err := server.Serve(stdioTransport); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

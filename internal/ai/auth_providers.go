// Package ai provides authentication providers for different AI services.
package ai

import "net/http"

// BearerTokenAuth implements AuthProvider for Bearer token authentication (OpenAI, Perplexity)
type BearerTokenAuth struct{}

// AddAuth adds Bearer token authentication to the request
func (b *BearerTokenAuth) AddAuth(req *http.Request, apiKey string) {
	req.Header.Set("Authorization", "Bearer "+apiKey)
}

// ClaudeAuth implements AuthProvider for Claude API authentication
type ClaudeAuth struct{}

// AddAuth adds Claude-specific authentication to the request
func (c *ClaudeAuth) AddAuth(req *http.Request, apiKey string) {
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
}

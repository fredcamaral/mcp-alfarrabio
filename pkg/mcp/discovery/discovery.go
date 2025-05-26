package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"mcp-memory/pkg/mcp/protocol"
	"time"
)

// Service provides discovery functionality for MCP
type Service struct {
	registry      *Registry
	pluginWatcher *PluginWatcher
}

// NewService creates a new discovery service
func NewService() *Service {
	return &Service{
		registry: NewRegistry(),
	}
}

// NewServiceWithPluginPath creates a discovery service that watches a plugin directory
func NewServiceWithPluginPath(pluginPath string, scanInterval time.Duration) (*Service, error) {
	s := &Service{
		registry: NewRegistry(),
	}
	
	watcher, err := NewPluginWatcher(pluginPath, scanInterval, s.registry)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin watcher: %w", err)
	}
	
	s.pluginWatcher = watcher
	return s, nil
}

// Start starts the discovery service
func (s *Service) Start(ctx context.Context) error {
	if s.pluginWatcher != nil {
		return s.pluginWatcher.Start(ctx)
	}
	return nil
}

// Stop stops the discovery service
func (s *Service) Stop() error {
	if s.pluginWatcher != nil {
		return s.pluginWatcher.Stop()
	}
	return nil
}

// GetRegistry returns the underlying registry
func (s *Service) GetRegistry() *Registry {
	return s.registry
}

// HandleDiscover handles the discovery request
func (s *Service) HandleDiscover(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var filter DiscoveryFilter
	if params != nil {
		if err := json.Unmarshal(params, &filter); err != nil {
			return nil, fmt.Errorf("invalid filter parameters: %w", err)
		}
	}
	
	return s.registry.Discover(&filter), nil
}

// RegisterTool registers a tool with the discovery service
func (s *Service) RegisterTool(tool protocol.Tool, handler protocol.ToolHandler, source string, tags []string) error {
	return s.registry.RegisterTool(tool, handler, source, tags)
}

// RegisterResource registers a resource with the discovery service
func (s *Service) RegisterResource(resource protocol.Resource, source string, tags []string) error {
	return s.registry.RegisterResource(resource, source, tags)
}

// RegisterPrompt registers a prompt with the discovery service
func (s *Service) RegisterPrompt(prompt protocol.Prompt, source string, tags []string) error {
	return s.registry.RegisterPrompt(prompt, source, tags)
}

// Subscribe returns a channel for receiving registration events
func (s *Service) Subscribe() <-chan RegistrationEvent {
	return s.registry.Subscribe()
}

// Unsubscribe removes a listener channel
func (s *Service) Unsubscribe(ch <-chan RegistrationEvent) {
	s.registry.Unsubscribe(ch)
}

// GetCapabilities returns the discovery capabilities
func (s *Service) GetCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"discovery": map[string]interface{}{
			"enabled":      true,
			"filtering":    true,
			"subscription": true,
			"plugins":      s.pluginWatcher != nil,
		},
	}
}
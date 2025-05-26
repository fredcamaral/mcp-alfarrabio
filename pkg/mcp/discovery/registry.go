package discovery

import (
	"fmt"
	"mcp-memory/pkg/mcp/protocol"
	"strings"
	"sync"
	"time"
)

// Registry manages dynamic registration of tools, resources, and prompts
type Registry struct {
	tools     map[string]*ToolInfo
	resources map[string]*ResourceInfo
	prompts   map[string]*PromptInfo
	mutex     sync.RWMutex
	listeners []chan RegistrationEvent
}

// NewRegistry creates a new discovery registry
func NewRegistry() *Registry {
	return &Registry{
		tools:     make(map[string]*ToolInfo),
		resources: make(map[string]*ResourceInfo),
		prompts:   make(map[string]*PromptInfo),
		listeners: make([]chan RegistrationEvent, 0),
	}
}

// RegisterTool registers a new tool
func (r *Registry) RegisterTool(tool protocol.Tool, handler protocol.ToolHandler, source string, tags []string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("tool %s already registered", tool.Name)
	}

	info := &ToolInfo{
		Tool:      tool,
		Available: true,
		LastSeen:  time.Now(),
		Source:    source,
		Tags:      tags,
	}

	r.tools[tool.Name] = info
	r.notifyListeners(RegistrationEvent{
		Type:      "register",
		Category:  "tool",
		Name:      tool.Name,
		Item:      info,
		Timestamp: time.Now(),
	})

	return nil
}

// UnregisterTool removes a tool from the registry
func (r *Registry) UnregisterTool(name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("tool %s not found", name)
	}

	delete(r.tools, name)
	r.notifyListeners(RegistrationEvent{
		Type:      "unregister",
		Category:  "tool",
		Name:      name,
		Timestamp: time.Now(),
	})

	return nil
}

// RegisterResource registers a new resource
func (r *Registry) RegisterResource(resource protocol.Resource, source string, tags []string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.resources[resource.URI]; exists {
		return fmt.Errorf("resource %s already registered", resource.URI)
	}

	info := &ResourceInfo{
		Resource:  resource,
		Available: true,
		LastSeen:  time.Now(),
		Source:    source,
		Tags:      tags,
	}

	r.resources[resource.URI] = info
	r.notifyListeners(RegistrationEvent{
		Type:      "register",
		Category:  "resource",
		Name:      resource.URI,
		Item:      info,
		Timestamp: time.Now(),
	})

	return nil
}

// UnregisterResource removes a resource from the registry
func (r *Registry) UnregisterResource(uri string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.resources[uri]; !exists {
		return fmt.Errorf("resource %s not found", uri)
	}

	delete(r.resources, uri)
	r.notifyListeners(RegistrationEvent{
		Type:      "unregister",
		Category:  "resource",
		Name:      uri,
		Timestamp: time.Now(),
	})

	return nil
}

// RegisterPrompt registers a new prompt
func (r *Registry) RegisterPrompt(prompt protocol.Prompt, source string, tags []string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.prompts[prompt.Name]; exists {
		return fmt.Errorf("prompt %s already registered", prompt.Name)
	}

	info := &PromptInfo{
		Prompt:    prompt,
		Available: true,
		LastSeen:  time.Now(),
		Source:    source,
		Tags:      tags,
	}

	r.prompts[prompt.Name] = info
	r.notifyListeners(RegistrationEvent{
		Type:      "register",
		Category:  "prompt",
		Name:      prompt.Name,
		Item:      info,
		Timestamp: time.Now(),
	})

	return nil
}

// UnregisterPrompt removes a prompt from the registry
func (r *Registry) UnregisterPrompt(name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.prompts[name]; !exists {
		return fmt.Errorf("prompt %s not found", name)
	}

	delete(r.prompts, name)
	r.notifyListeners(RegistrationEvent{
		Type:      "unregister",
		Category:  "prompt",
		Name:      name,
		Timestamp: time.Now(),
	})

	return nil
}

// Discover returns filtered discovery information
func (r *Registry) Discover(filter *DiscoveryFilter) *DiscoveryInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	info := &DiscoveryInfo{
		Tools:      make([]ToolInfo, 0),
		Resources:  make([]ResourceInfo, 0),
		Prompts:    make([]PromptInfo, 0),
		LastUpdate: time.Now(),
		Version:    "1.0.0",
	}

	// Filter tools
	for _, tool := range r.tools {
		if r.matchesFilter(tool, filter) {
			info.Tools = append(info.Tools, *tool)
		}
	}

	// Filter resources
	for _, resource := range r.resources {
		if r.matchesFilter(resource, filter) {
			info.Resources = append(info.Resources, *resource)
		}
	}

	// Filter prompts
	for _, prompt := range r.prompts {
		if r.matchesFilter(prompt, filter) {
			info.Prompts = append(info.Prompts, *prompt)
		}
	}

	return info
}

// Subscribe returns a channel for receiving registration events
func (r *Registry) Subscribe() <-chan RegistrationEvent {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	ch := make(chan RegistrationEvent, 100)
	r.listeners = append(r.listeners, ch)
	return ch
}

// Unsubscribe removes a listener channel
func (r *Registry) Unsubscribe(ch <-chan RegistrationEvent) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for i, listener := range r.listeners {
		if listener == ch {
			close(listener)
			r.listeners = append(r.listeners[:i], r.listeners[i+1:]...)
			break
		}
	}
}

// matchesFilter checks if an item matches the discovery filter
func (r *Registry) matchesFilter(item interface{}, filter *DiscoveryFilter) bool {
	if filter == nil {
		return true
	}

	// Helper to check tags
	hasTag := func(itemTags []string, filterTags []string) bool {
		if len(filterTags) == 0 {
			return true
		}
		for _, filterTag := range filterTags {
			for _, itemTag := range itemTags {
				if itemTag == filterTag {
					return true
				}
			}
		}
		return false
	}

	// Helper to check search string
	matchesSearch := func(text string) bool {
		if filter.Search == "" {
			return true
		}
		return strings.Contains(strings.ToLower(text), strings.ToLower(filter.Search))
	}

	switch v := item.(type) {
	case *ToolInfo:
		if filter.Available != nil && v.Available != *filter.Available {
			return false
		}
		if filter.Source != "" && v.Source != filter.Source {
			return false
		}
		if !hasTag(v.Tags, filter.Tags) {
			return false
		}
		if !matchesSearch(v.Tool.Name) && !matchesSearch(v.Tool.Description) {
			return false
		}
		return true

	case *ResourceInfo:
		if filter.Available != nil && v.Available != *filter.Available {
			return false
		}
		if filter.Source != "" && v.Source != filter.Source {
			return false
		}
		if !hasTag(v.Tags, filter.Tags) {
			return false
		}
		if !matchesSearch(v.Resource.URI) && !matchesSearch(v.Resource.Name) && !matchesSearch(v.Resource.Description) {
			return false
		}
		return true

	case *PromptInfo:
		if filter.Available != nil && v.Available != *filter.Available {
			return false
		}
		if filter.Source != "" && v.Source != filter.Source {
			return false
		}
		if !hasTag(v.Tags, filter.Tags) {
			return false
		}
		if !matchesSearch(v.Prompt.Name) && !matchesSearch(v.Prompt.Description) {
			return false
		}
		return true

	default:
		return false
	}
}

// notifyListeners sends an event to all registered listeners
func (r *Registry) notifyListeners(event RegistrationEvent) {
	for _, listener := range r.listeners {
		select {
		case listener <- event:
		default:
			// Channel full, skip
		}
	}
}

// UpdateAvailability updates the availability status of an item
func (r *Registry) UpdateAvailability(category, name string, available bool) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	switch category {
	case "tool":
		if info, exists := r.tools[name]; exists {
			info.Available = available
			info.LastSeen = time.Now()
			return nil
		}
		return fmt.Errorf("tool %s not found", name)

	case "resource":
		if info, exists := r.resources[name]; exists {
			info.Available = available
			info.LastSeen = time.Now()
			return nil
		}
		return fmt.Errorf("resource %s not found", name)

	case "prompt":
		if info, exists := r.prompts[name]; exists {
			info.Available = available
			info.LastSeen = time.Now()
			return nil
		}
		return fmt.Errorf("prompt %s not found", name)

	default:
		return fmt.Errorf("unknown category: %s", category)
	}
}
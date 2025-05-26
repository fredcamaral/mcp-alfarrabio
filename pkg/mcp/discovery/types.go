package discovery

import (
	"mcp-memory/pkg/mcp/protocol"
	"time"
)

// ToolInfo provides discoverable information about a tool
type ToolInfo struct {
	Tool         protocol.Tool   `json:"tool"`
	Available    bool            `json:"available"`
	LastSeen     time.Time       `json:"lastSeen"`
	Source       string          `json:"source"`
	Tags         []string        `json:"tags,omitempty"`
	Requirements []string        `json:"requirements,omitempty"`
	Examples     []ToolExample   `json:"examples,omitempty"`
}

// ToolExample provides an example of tool usage
type ToolExample struct {
	Description string                 `json:"description"`
	Arguments   map[string]interface{} `json:"arguments"`
	Result      interface{}            `json:"result,omitempty"`
}

// ResourceInfo provides discoverable information about a resource
type ResourceInfo struct {
	Resource     protocol.Resource `json:"resource"`
	Available    bool              `json:"available"`
	LastSeen     time.Time         `json:"lastSeen"`
	Source       string            `json:"source"`
	Tags         []string          `json:"tags,omitempty"`
	Permissions  []string          `json:"permissions,omitempty"`
}

// PromptInfo provides discoverable information about a prompt
type PromptInfo struct {
	Prompt       protocol.Prompt `json:"prompt"`
	Available    bool            `json:"available"`
	LastSeen     time.Time       `json:"lastSeen"`
	Source       string          `json:"source"`
	Tags         []string        `json:"tags,omitempty"`
	Examples     []PromptExample `json:"examples,omitempty"`
}

// PromptExample provides an example of prompt usage
type PromptExample struct {
	Description string                 `json:"description"`
	Arguments   map[string]interface{} `json:"arguments"`
	Result      string                 `json:"result"`
}

// DiscoveryInfo provides a complete discovery response
type DiscoveryInfo struct {
	Tools      []ToolInfo     `json:"tools"`
	Resources  []ResourceInfo `json:"resources"`
	Prompts    []PromptInfo   `json:"prompts"`
	LastUpdate time.Time      `json:"lastUpdate"`
	Version    string         `json:"version"`
}

// DiscoveryFilter allows filtering discovery results
type DiscoveryFilter struct {
	Tags       []string `json:"tags,omitempty"`
	Source     string   `json:"source,omitempty"`
	Available  *bool    `json:"available,omitempty"`
	Search     string   `json:"search,omitempty"`
}

// RegistrationEvent represents a registration/deregistration event
type RegistrationEvent struct {
	Type      string      `json:"type"` // "register" or "unregister"
	Category  string      `json:"category"` // "tool", "resource", or "prompt"
	Name      string      `json:"name"`
	Item      interface{} `json:"item,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}
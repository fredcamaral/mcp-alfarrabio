package mcp

// APIStability defines the stability guarantees for different API components.
type APIStability string

const (
	// Stable APIs will not have breaking changes within a major version.
	Stable APIStability = "stable"
	
	// Beta APIs may have breaking changes in minor versions.
	Beta APIStability = "beta"
	
	// Experimental APIs may change or be removed at any time.
	Experimental APIStability = "experimental"
	
	// Deprecated APIs are scheduled for removal in a future version.
	Deprecated APIStability = "deprecated"
)

// APIMetadata provides metadata about API components.
type APIMetadata struct {
	// Component name (e.g., "server.Server", "protocol.Tool")
	Component string
	
	// Stability level
	Stability APIStability
	
	// Version when the component was introduced
	Since string
	
	// Version when the component will be removed (for deprecated APIs)
	RemovalVersion string
	
	// Replacement component (for deprecated APIs)
	ReplacedBy string
	
	// Additional notes about the component
	Notes string
}

// APIRegistry tracks the stability of all public APIs.
var APIRegistry = []APIMetadata{
	// Core Protocol Types - Stable
	{Component: "protocol.JSONRPCRequest", Stability: Stable, Since: "v0.1.0"},
	{Component: "protocol.JSONRPCResponse", Stability: Stable, Since: "v0.1.0"},
	{Component: "protocol.JSONRPCError", Stability: Stable, Since: "v0.1.0"},
	{Component: "protocol.Tool", Stability: Stable, Since: "v0.1.0"},
	{Component: "protocol.Resource", Stability: Stable, Since: "v0.1.0"},
	{Component: "protocol.Prompt", Stability: Stable, Since: "v0.1.0"},
	{Component: "protocol.ToolHandler", Stability: Stable, Since: "v0.1.0"},
	
	// Server APIs - Stable
	{Component: "server.Server", Stability: Stable, Since: "v0.1.0"},
	{Component: "server.ResourceHandler", Stability: Stable, Since: "v0.1.0"},
	{Component: "server.PromptHandler", Stability: Stable, Since: "v0.1.0"},
	
	// Transport APIs - Stable
	{Component: "transport.Transport", Stability: Stable, Since: "v0.1.0"},
	{Component: "transport.StdioTransport", Stability: Stable, Since: "v0.1.0"},
	
	// Convenience Functions - Stable
	{Component: "mcp.NewServer", Stability: Stable, Since: "v0.1.0"},
	{Component: "mcp.NewTool", Stability: Stable, Since: "v0.1.0"},
	{Component: "mcp.NewResource", Stability: Stable, Since: "v0.1.0"},
	{Component: "mcp.NewPrompt", Stability: Stable, Since: "v0.1.0"},
	
	// Future HTTP Transport - Beta
	{Component: "transport.HTTPTransport", Stability: Beta, Since: "v0.3.0", Notes: "Coming in Phase 3"},
	{Component: "transport.WebSocketTransport", Stability: Beta, Since: "v0.3.0", Notes: "Coming in Phase 3"},
	
	// Future Middleware - Experimental
	{Component: "middleware.Middleware", Stability: Experimental, Since: "v0.3.0", Notes: "Coming in Phase 3"},
	{Component: "middleware.AuthMiddleware", Stability: Experimental, Since: "v0.3.0", Notes: "Coming in Phase 3"},
	{Component: "middleware.RateLimitMiddleware", Stability: Experimental, Since: "v0.3.0", Notes: "Coming in Phase 3"},
}

// GetAPIStability returns the stability information for a given component.
func GetAPIStability(component string) *APIMetadata {
	for _, api := range APIRegistry {
		if api.Component == component {
			return &api
		}
	}
	return nil
}
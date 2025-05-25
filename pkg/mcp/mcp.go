// Package mcp provides a high-level interface for the Model Context Protocol.
//
// MCP-Go is a production-ready Go implementation of the Model Context Protocol,
// enabling seamless integration between AI models and external tools, resources,
// and services.
//
// # Basic Usage
//
// Create a simple MCP server with a tool:
//
//	server := mcp.NewServer("my-app", "1.0.0")
//	
//	tool := mcp.NewTool(
//	    "greet",
//	    "Greet a user",
//	    mcp.ObjectSchema("Greeting parameters", map[string]interface{}{
//	        "name": mcp.StringParam("Name to greet", true),
//	    }, []string{"name"}),
//	)
//	
//	server.AddTool(tool, mcp.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
//	    name := params["name"].(string)
//	    return map[string]interface{}{
//	        "greeting": fmt.Sprintf("Hello, %s!", name),
//	    }, nil
//	}))
//	
//	server.Start(context.Background(), mcp.NewStdioTransport())
//
// # Architecture
//
// The package is organized into several subpackages:
//   - protocol: Core MCP protocol types and interfaces
//   - server: Server implementation and request handling
//   - transport: Transport layer implementations (stdio, HTTP, etc.)
//
// # Features
//
//   - Full MCP 2024-11-05 specification support
//   - Type-safe tool and resource registration
//   - Built-in JSON Schema validation
//   - Multiple transport options
//   - Comprehensive error handling
//   - Production-ready performance
package mcp

import (
	"context"
	"mcp-memory/pkg/mcp/protocol"
	"mcp-memory/pkg/mcp/server"
	"mcp-memory/pkg/mcp/transport"
)

// NewServer creates a new MCP server with the given name and version.
//
// The server name and version are used during the MCP handshake to identify
// your application to clients.
//
// Example:
//
//	server := mcp.NewServer("file-manager", "1.0.0")
//	server.SetDescription("A file management MCP server")
func NewServer(name, version string) *server.Server {
	return server.NewServer(name, version)
}

// NewTool creates a new MCP tool with the given name, description, and JSON Schema.
//
// Tools are the primary way to extend MCP server functionality. Each tool must have
// a unique name and a JSON Schema that describes its parameters.
//
// Example:
//
//	tool := mcp.NewTool(
//	    "search_files",
//	    "Search for files by name pattern",
//	    mcp.ObjectSchema("Search parameters", map[string]interface{}{
//	        "pattern": mcp.StringParam("File name pattern (e.g., *.go)", true),
//	        "path":    mcp.StringParam("Directory to search in", false),
//	    }, []string{"pattern"}),
//	)
func NewTool(name, description string, inputSchema map[string]interface{}) protocol.Tool {
	return protocol.Tool{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
	}
}

// NewResource creates a new MCP resource with URI pattern.
//
// Resources provide a way to expose data that can be accessed by URI.
// The URI can include placeholders in curly braces that will be extracted
// when the resource is accessed.
//
// Example:
//
//	resource := mcp.NewResource(
//	    "file:///{path}",
//	    "Local Files",
//	    "Access to local file system",
//	    "text/plain",
//	)
func NewResource(uri, name, description, mimeType string) protocol.Resource {
	return protocol.Resource{
		URI:         uri,
		Name:        name,
		Description: description,
		MimeType:    mimeType,
	}
}

// NewPrompt creates a new MCP prompt template.
//
// Prompts are reusable templates that help guide AI interactions.
// They can include arguments that are filled in when the prompt is used.
//
// Example:
//
//	prompt := mcp.NewPrompt(
//	    "code_review",
//	    "Review code for quality and suggest improvements",
//	    []protocol.PromptArgument{
//	        mcp.NewPromptArgument("file_path", "Path to the file to review", true),
//	        mcp.NewPromptArgument("focus_areas", "Specific areas to focus on", false),
//	    },
//	)
func NewPrompt(name, description string, arguments []protocol.PromptArgument) protocol.Prompt {
	return protocol.Prompt{
		Name:        name,
		Description: description,
		Arguments:   arguments,
	}
}

// NewPromptArgument creates a new prompt argument
func NewPromptArgument(name, description string, required bool) protocol.PromptArgument {
	return protocol.PromptArgument{
		Name:        name,
		Description: description,
		Required:    required,
	}
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport() transport.Transport {
	return transport.NewStdioTransport()
}

// ToolHandlerFunc creates a protocol.ToolHandler from a function.
//
// This is a convenience function for creating tool handlers without
// implementing the full ToolHandler interface.
//
// Example:
//
//	handler := mcp.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
//	    name := params["name"].(string)
//	    return map[string]interface{}{
//	        "message": fmt.Sprintf("Hello, %s!", name),
//	    }, nil
//	})
//	
//	server.AddTool(tool, handler)
func ToolHandlerFunc(f func(ctx context.Context, params map[string]interface{}) (interface{}, error)) protocol.ToolHandler {
	return protocol.ToolHandlerFunc(f)
}

// ResourceHandlerFunc is an adapter to allow the use of ordinary functions as resource handlers.
//
// Example:
//
//	handler := mcp.ResourceHandlerFunc(func(ctx context.Context, uri string) ([]protocol.Content, error) {
//	    // Extract path from URI
//	    path := strings.TrimPrefix(uri, "file:///")
//	    
//	    // Read file content
//	    content, err := os.ReadFile(path)
//	    if err != nil {
//	        return nil, err
//	    }
//	    
//	    return []protocol.Content{
//	        {Type: "text", Text: string(content)},
//	    }, nil
//	})
//	
//	server.AddResource(resource, handler)
type ResourceHandlerFunc func(ctx context.Context, uri string) ([]protocol.Content, error)

func (f ResourceHandlerFunc) Handle(ctx context.Context, uri string) ([]protocol.Content, error) {
	return f(ctx, uri)
}

// PromptHandlerFunc creates a server.PromptHandler from a function
type PromptHandlerFunc func(ctx context.Context, args map[string]interface{}) ([]protocol.Content, error)

func (f PromptHandlerFunc) Handle(ctx context.Context, args map[string]interface{}) ([]protocol.Content, error) {
	return f(ctx, args)
}

// JSONSchema helper functions for common input schemas

// StringParam creates a string parameter schema for use in tool input schemas.
//
// The required parameter is currently not used in the schema itself but is
// documented for clarity. Use the required array in ObjectSchema to specify
// which parameters are required.
//
// Example:
//
//	schema := mcp.ObjectSchema("Tool parameters", map[string]interface{}{
//	    "name":  mcp.StringParam("User's name", true),
//	    "email": mcp.StringParam("User's email (optional)", false),
//	}, []string{"name"}) // Only name is required
func StringParam(description string, required bool) map[string]interface{} {
	schema := map[string]interface{}{
		"type":        "string",
		"description": description,
	}
	return schema
}

// NumberParam creates a number parameter schema
func NumberParam(description string, required bool) map[string]interface{} {
	schema := map[string]interface{}{
		"type":        "number",
		"description": description,
	}
	return schema
}

// BooleanParam creates a boolean parameter schema
func BooleanParam(description string, required bool) map[string]interface{} {
	schema := map[string]interface{}{
		"type":        "boolean",
		"description": description,
	}
	return schema
}

// ObjectSchema creates an object schema with properties and required fields.
//
// This is the most common schema type for tool parameters. The properties map
// defines the available parameters, and the required array lists which ones
// must be provided.
//
// Example:
//
//	schema := mcp.ObjectSchema("Search parameters", 
//	    map[string]interface{}{
//	        "query": mcp.StringParam("Search query", true),
//	        "limit": map[string]interface{}{
//	            "type": "integer",
//	            "description": "Maximum results",
//	            "minimum": 1,
//	            "maximum": 100,
//	            "default": 10,
//	        },
//	        "includeHidden": mcp.BooleanParam("Include hidden files", false),
//	    },
//	    []string{"query"}, // Only query is required
//	)
func ObjectSchema(description string, properties map[string]interface{}, required []string) map[string]interface{} {
	schema := map[string]interface{}{
		"type":        "object",
		"description": description,
		"properties":  properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// ArraySchema creates an array schema with item type definition.
//
// Use this for parameters that accept lists of values.
//
// Example:
//
//	// Array of strings
//	tagsSchema := mcp.ArraySchema("List of tags", 
//	    map[string]interface{}{"type": "string"},
//	)
//	
//	// Array of objects
//	filesSchema := mcp.ArraySchema("List of files to process",
//	    mcp.ObjectSchema("File info", map[string]interface{}{
//	        "path": mcp.StringParam("File path", true),
//	        "encoding": mcp.StringParam("File encoding", false),
//	    }, []string{"path"}),
//	)
func ArraySchema(description string, items map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"description": description,
		"items":       items,
	}
}
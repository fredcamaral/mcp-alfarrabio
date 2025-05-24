// Package mcp provides a high-level interface for the Model Context Protocol
package mcp

import (
	"context"
	"mcp-memory/pkg/mcp/protocol"
	"mcp-memory/pkg/mcp/server"
	"mcp-memory/pkg/mcp/transport"
)

// NewServer creates a new MCP server with the given name and version
func NewServer(name, version string) *server.Server {
	return server.NewServer(name, version)
}

// NewTool creates a new MCP tool with the given name and description
func NewTool(name, description string, inputSchema map[string]interface{}) protocol.Tool {
	return protocol.Tool{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
	}
}

// NewResource creates a new MCP resource
func NewResource(uri, name, description, mimeType string) protocol.Resource {
	return protocol.Resource{
		URI:         uri,
		Name:        name,
		Description: description,
		MimeType:    mimeType,
	}
}

// NewPrompt creates a new MCP prompt template
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

// ToolHandlerFunc creates a protocol.ToolHandler from a function
func ToolHandlerFunc(f func(ctx context.Context, params map[string]interface{}) (interface{}, error)) protocol.ToolHandler {
	return protocol.ToolHandlerFunc(f)
}

// ResourceHandlerFunc creates a server.ResourceHandler from a function  
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

// StringParam creates a string parameter schema
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

// ObjectSchema creates an object schema with properties
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

// ArraySchema creates an array schema
func ArraySchema(description string, items map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"description": description,
		"items":       items,
	}
}
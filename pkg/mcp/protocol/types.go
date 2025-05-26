// Package protocol implements the Model Context Protocol types and interfaces
package protocol

import (
	"context"
)

// Version represents the MCP protocol version
const Version = "2024-11-05"

// JSONRPCRequest represents a JSON-RPC request
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error implements the error interface
func (e *JSONRPCError) Error() string {
	return e.Message
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolCallRequest represents a tool call request
type ToolCallRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ToolCallResult represents a tool call result
type ToolCallResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content represents content in MCP responses
type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// Resource represents an MCP resource
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// Prompt represents an MCP prompt template
type Prompt struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Arguments   []PromptArgument       `json:"arguments,omitempty"`
}

// PromptArgument represents a prompt argument
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// ServerCapabilities represents server capabilities
type ServerCapabilities struct {
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	Logging      map[string]interface{} `json:"logging,omitempty"`
	Prompts      *PromptCapability      `json:"prompts,omitempty"`
	Resources    *ResourceCapability    `json:"resources,omitempty"`
	Tools        *ToolCapability        `json:"tools,omitempty"`
	Sampling     *SamplingCapability    `json:"sampling,omitempty"`
	Roots        *RootsCapability       `json:"roots,omitempty"`
}

// PromptCapability represents prompt capabilities
type PromptCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourceCapability represents resource capabilities  
type ResourceCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// ToolCapability represents tool capabilities
type ToolCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// SamplingCapability represents sampling capabilities
type SamplingCapability struct {
	// No specific fields defined in the spec yet
}

// RootsCapability represents roots capabilities
type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// InitializeRequest represents an initialization request
type InitializeRequest struct {
	ProtocolVersion string           `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ClientInfo       `json:"clientInfo"`
}

// ClientCapabilities represents client capabilities
type ClientCapabilities struct {
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	Sampling     map[string]interface{} `json:"sampling,omitempty"`
}

// ClientInfo represents client information
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResult represents initialization result
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

// ServerInfo represents server information
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ToolHandler defines the interface for tool handlers
type ToolHandler interface {
	Handle(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// ToolHandlerFunc is a function adapter for ToolHandler
type ToolHandlerFunc func(ctx context.Context, params map[string]interface{}) (interface{}, error)

// Handle implements the ToolHandler interface
func (f ToolHandlerFunc) Handle(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return f(ctx, params)
}

// Error codes as defined by MCP specification
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// NewJSONRPCError creates a new JSON-RPC error
func NewJSONRPCError(code int, message string, data interface{}) *JSONRPCError {
	return &JSONRPCError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// NewContent creates new text content
func NewContent(text string) Content {
	return Content{
		Type: "text",
		Text: text,
	}
}

// NewToolCallResult creates a new tool call result
func NewToolCallResult(content ...Content) *ToolCallResult {
	return &ToolCallResult{
		Content: content,
		IsError: false,
	}
}

// NewToolCallError creates a new tool call error result
func NewToolCallError(message string) *ToolCallResult {
	return &ToolCallResult{
		Content: []Content{NewContent(message)},
		IsError: true,
	}
}
// Package transport implements MCP transport layers
package transport

import (
	"context"
	"mcp-memory/pkg/mcp/protocol"
)

// Transport defines the interface for MCP transport layers
type Transport interface {
	Start(ctx context.Context, handler RequestHandler) error
	Stop() error
}

// RequestHandler defines the interface for handling MCP requests
type RequestHandler interface {
	HandleRequest(ctx context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse
}
// Package mcp provides MCP server implementation
package mcp

const (
	// GlobalMemoryRepository is a special repository name for global memories
	// that are not tied to a specific project
	GlobalMemoryRepository = "_global"

	// GlobalRepository is a special repository name for global scope operations
	GlobalRepository = "global"

	// GlobalMemoryDescription is used in tool parameter descriptions
	GlobalMemoryDescription = " (use '_global' for global memories)"

	// MCP tool operation names
	OperationStoreChunk    = "store_chunk"
	OperationStoreDecision = "store_decision"
	OperationHealth        = "health"
	OperationStatus        = "status"

	// Common filter values
	FilterValueAll = "all"
)

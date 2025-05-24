package mcp

import (
	"context"
	"fmt"
	"mcp-memory/internal/logging"
	"mcp-memory/pkg/mcp/protocol"
)

// MCPToolExecutor provides a way to execute MCP tools for testing and demonstration
type MCPToolExecutor struct {
	server *MemoryServer
}

// NewMCPToolExecutor creates a new tool executor
func NewMCPToolExecutor(server *MemoryServer) *MCPToolExecutor {
	return &MCPToolExecutor{
		server: server,
	}
}

// ExecuteTool executes a named MCP tool with given parameters
func (executor *MCPToolExecutor) ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP EXECUTOR: Executing tool", "tool_name", toolName, "params", params)
	
	// Create an MCP tools/call request
	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: protocol.ToolCallRequest{
			Name:      toolName,
			Arguments: params,
		},
	}

	logging.Info("MCP EXECUTOR: Sending request to server", "method", req.Method, "tool_name", toolName)
	
	// Handle the request using our MCP server
	response := executor.server.mcpServer.HandleRequest(ctx, req)
	
	logging.Info("MCP EXECUTOR: Received response", "has_error", response.Error != nil, "tool_name", toolName)
	
	if response.Error != nil {
		logging.Error("MCP EXECUTOR: Tool execution failed", "error", response.Error.Message, "code", response.Error.Code, "tool_name", toolName)
		return nil, fmt.Errorf("tool execution failed: %s", response.Error.Message)
	}

	logging.Info("MCP EXECUTOR: Tool execution successful", "tool_name", toolName)
	return response.Result, nil
}

// ListAvailableTools returns a list of all registered tools
func (executor *MCPToolExecutor) ListAvailableTools() []string {
	// Create an MCP tools/list request
	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	// Handle the request using our MCP server
	response := executor.server.mcpServer.HandleRequest(context.Background(), req)
	
	if response.Error != nil {
		return []string{}
	}

	// Extract tool names from response
	if result, ok := response.Result.(map[string]interface{}); ok {
		if toolsArray, ok := result["tools"].([]protocol.Tool); ok {
			toolNames := make([]string, len(toolsArray))
			for i, tool := range toolsArray {
				toolNames[i] = tool.Name
			}
			return toolNames
		}
	}

	return []string{}
}

// GetToolInfo returns information about a specific tool
func (executor *MCPToolExecutor) GetToolInfo(toolName string) map[string]interface{} {
	info := map[string]interface{}{
		"name":        toolName,
		"available":   false,
		"description": "",
	}

	availableTools := executor.ListAvailableTools()
	for _, name := range availableTools {
		if name == toolName {
			info["available"] = true
			info["description"] = getToolDescription(toolName)
			break
		}
	}

	return info
}

// getToolDescription returns a description for each tool
func getToolDescription(toolName string) string {
	descriptions := map[string]string{
		"mcp__memory__memory_store_chunk":      "Store a conversation chunk in memory with automatic analysis and embedding generation",
		"mcp__memory__memory_search":           "Search for similar conversation chunks using semantic similarity",
		"mcp__memory__memory_get_context":      "Get conversation context and recent activity for a repository",
		"mcp__memory__memory_find_similar":     "Find similar past problems and solutions",
		"mcp__memory__memory_store_decision":   "Store an architectural decision with rationale",
		"mcp__memory__memory_get_patterns":     "Identify recurring patterns in project history",
		"mcp__memory__memory_health":           "Check the health status of the memory system",
		"mcp__memory__memory_suggest_related":  "Get AI-powered suggestions for related context based on current work",
		"mcp__memory__memory_export_project":   "Export all memory data for a project in various formats",
		"mcp__memory__memory_import_context":   "Import conversation context from external source",
	}

	if desc, exists := descriptions[toolName]; exists {
		return desc
	}
	return "Tool description not available"
}

// DemoAllTools demonstrates all available tools with sample data
func (executor *MCPToolExecutor) DemoAllTools(ctx context.Context) map[string]interface{} {
	results := make(map[string]interface{})

	// Demo memory_health (simplest tool)
	healthResult, err := executor.ExecuteTool(ctx, "mcp__memory__memory_health", map[string]interface{}{})
	if err != nil {
		results["mcp__memory__memory_health"] = map[string]interface{}{"error": err.Error()}
	} else {
		results["mcp__memory__memory_health"] = healthResult
	}

	// Demo memory_suggest_related
	suggestParams := map[string]interface{}{
		"current_context":   "I'm working on implementing authentication for my web application",
		"repository":        "demo-project",
		"max_suggestions":   float64(3),
		"include_patterns":  true,
		"session_id":        "demo-session-001",
	}
	suggestResult, err := executor.ExecuteTool(ctx, "mcp__memory__memory_suggest_related", suggestParams)
	if err != nil {
		results["mcp__memory__memory_suggest_related"] = map[string]interface{}{"error": err.Error()}
	} else {
		results["mcp__memory__memory_suggest_related"] = suggestResult
	}

	// Demo memory_export_project
	exportParams := map[string]interface{}{
		"repository":       "demo-project",
		"format":           "json",
		"include_vectors":  false,
		"session_id":       "demo-session-001",
	}
	exportResult, err := executor.ExecuteTool(ctx, "mcp__memory__memory_export_project", exportParams)
	if err != nil {
		results["mcp__memory__memory_export_project"] = map[string]interface{}{"error": err.Error()}
	} else {
		results["mcp__memory__memory_export_project"] = exportResult
	}

	// Demo memory_import_context
	importParams := map[string]interface{}{
		"source":            "conversation",
		"data":              "User: How do I set up authentication? Assistant: You can use JWT tokens for stateless authentication...",
		"repository":        "demo-project",
		"chunking_strategy": "auto",
		"metadata": map[string]interface{}{
			"source_system": "claude",
			"tags":          []interface{}{"auth", "jwt", "demo"},
		},
		"session_id": "demo-session-001",
	}
	importResult, err := executor.ExecuteTool(ctx, "mcp__memory__memory_import_context", importParams)
	if err != nil {
		results["mcp__memory__memory_import_context"] = map[string]interface{}{"error": err.Error()}
	} else {
		results["mcp__memory__memory_import_context"] = importResult
	}

	return results
}
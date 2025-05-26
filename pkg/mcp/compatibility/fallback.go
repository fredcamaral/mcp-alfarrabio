package compatibility

import (
	"encoding/json"
	"fmt"
	"mcp-memory/pkg/mcp/protocol"
)

// FallbackHandler provides fallback implementations for unsupported features
type FallbackHandler struct {
	profile  *ClientProfile
	detector *Detector
}

// NewFallbackHandler creates a new fallback handler
func NewFallbackHandler(profile *ClientProfile) *FallbackHandler {
	return &FallbackHandler{
		profile:  profile,
		detector: NewDetector(),
	}
}

// HandleUnsupportedMethod provides a fallback for unsupported methods
func (f *FallbackHandler) HandleUnsupportedMethod(method string) *protocol.JSONRPCResponse {
	feature := f.methodToFeature(method)
	supported, workaround := f.detector.CheckFeatureSupport(f.profile, feature)
	
	if supported {
		// Feature should be supported, likely an implementation issue
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			Error: protocol.NewJSONRPCError(
				protocol.InternalError,
				fmt.Sprintf("Feature %s is supported but not implemented", feature),
				nil,
			),
		}
	}
	
	// Provide helpful error with workaround
	errorData := map[string]interface{}{
		"feature":    feature,
		"client":     f.profile.Name,
		"workaround": workaround,
	}
	
	return &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		Error: protocol.NewJSONRPCError(
			-32002, // Custom error code for unsupported feature
			fmt.Sprintf("Feature '%s' is not supported by %s", feature, f.profile.Name),
			errorData,
		),
	}
}

// ConvertResourceToTool converts a resource read request to a tool call
func (f *FallbackHandler) ConvertResourceToTool(uri string) *protocol.ToolCallRequest {
	return &protocol.ToolCallRequest{
		Name: "read_resource",
		Arguments: map[string]interface{}{
			"uri": uri,
		},
	}
}

// ConvertPromptToTool converts a prompt to a tool
func (f *FallbackHandler) ConvertPromptToTool(prompt protocol.Prompt, args map[string]interface{}) *protocol.ToolCallRequest {
	return &protocol.ToolCallRequest{
		Name: fmt.Sprintf("prompt_%s", prompt.Name),
		Arguments: args,
	}
}

// WrapToolsWithCompatibility wraps tools to ensure compatibility
func (f *FallbackHandler) WrapToolsWithCompatibility(tools []protocol.Tool) []protocol.Tool {
	// For clients with limited support, we might need to simplify tool schemas
	if f.profile == nil || contains(f.profile.SupportedFeatures, FeatureTools) {
		return tools // No wrapping needed
	}
	
	// For clients without tool support, return empty
	return []protocol.Tool{}
}

// SimplifyResponse simplifies responses for clients with limitations
func (f *FallbackHandler) SimplifyResponse(response interface{}) interface{} {
	switch v := response.(type) {
	case *protocol.ToolCallResult:
		// Some clients might not handle complex content types
		if len(v.Content) == 1 && v.Content[0].Type == "text" {
			return v // Already simple
		}
		
		// Convert to simple text
		simplified := &protocol.ToolCallResult{
			Content: []protocol.Content{},
			IsError: v.IsError,
		}
		
		for _, content := range v.Content {
			if content.Type == "text" {
				simplified.Content = append(simplified.Content, content)
			} else {
				// Convert non-text content to JSON string
				if data, err := json.Marshal(content); err == nil {
					simplified.Content = append(simplified.Content, protocol.NewContent(string(data)))
				}
			}
		}
		
		return simplified
		
	default:
		return response
	}
}

// GetFallbackResponse provides a generic fallback response
func (f *FallbackHandler) GetFallbackResponse(method string) interface{} {
	switch method {
	case "tools/list":
		return map[string]interface{}{
			"tools": []protocol.Tool{},
		}
	case "resources/list":
		return map[string]interface{}{
			"resources": []protocol.Resource{},
		}
	case "prompts/list":
		return map[string]interface{}{
			"prompts": []protocol.Prompt{},
		}
	case "roots/list":
		return map[string]interface{}{
			"roots": []interface{}{},
		}
	default:
		return nil
	}
}

// methodToFeature maps RPC methods to feature names
func (f *FallbackHandler) methodToFeature(method string) string {
	switch {
	case hasPrefix(method, "tools/"):
		return FeatureTools
	case hasPrefix(method, "resources/"):
		return FeatureResources
	case hasPrefix(method, "prompts/"):
		return FeaturePrompts
	case hasPrefix(method, "roots/"):
		return FeatureRoots
	case hasPrefix(method, "sampling/"):
		return FeatureSampling
	case hasPrefix(method, "discovery/"):
		return FeatureDiscovery
	default:
		return "unknown"
	}
}

// hasPrefix checks if a string has a prefix
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
package mcp

// Version represents the MCP library version.
const Version = "v0.2.0"

// MCPVersion represents the MCP protocol version supported.
const MCPVersion = "2024-11-05"

// APIVersion represents the library API version for compatibility.
const APIVersion = 2

// VersionInfo contains detailed version information.
type VersionInfo struct {
	// Library version following semantic versioning
	Library string `json:"library"`
	
	// MCP protocol version supported
	Protocol string `json:"protocol"`
	
	// API version for compatibility checking
	API int `json:"api"`
	
	// Build metadata (commit hash, date, etc.)
	BuildMetadata map[string]string `json:"build_metadata,omitempty"`
}

// GetVersionInfo returns the current version information.
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Library:  Version,
		Protocol: MCPVersion,
		API:      APIVersion,
	}
}

// IsCompatibleAPI checks if the given API version is compatible.
func IsCompatibleAPI(version int) bool {
	// API v2 is compatible with v2.x
	return version == APIVersion
}
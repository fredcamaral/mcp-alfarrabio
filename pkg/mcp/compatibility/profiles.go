package compatibility

// ClientProfile defines the capabilities of a known MCP client
type ClientProfile struct {
	Name             string
	Pattern          string   // Pattern to match in user agent or client info
	SupportedFeatures []string
	RequiresFeatures []string // Features that must be present
	Limitations      []string // Known limitations
	Workarounds      map[string]string // Feature -> workaround description
}

// GetKnownProfiles returns all known client profiles
func GetKnownProfiles() []ClientProfile {
	return []ClientProfile{
		{
			Name:    "Claude Desktop",
			Pattern: "claude-desktop",
			SupportedFeatures: []string{
				"tools",
				"resources", 
				"prompts",
			},
			RequiresFeatures: []string{},
			Limitations: []string{
				"No discovery support",
				"No roots support",
				"No sampling support",
			},
			Workarounds: map[string]string{
				"discovery": "All tools must be registered at startup",
			},
		},
		{
			Name:    "Claude.ai",
			Pattern: "claude.ai",
			SupportedFeatures: []string{
				"tools",
				"resources",
				"prompts",
			},
			RequiresFeatures: []string{},
			Limitations: []string{
				"Remote servers only",
				"No local file access",
			},
			Workarounds: map[string]string{
				"local_files": "Use resource URIs with proper access control",
			},
		},
		{
			Name:    "Claude Code",
			Pattern: "claude-code",
			SupportedFeatures: []string{
				"tools",
				"prompts",
			},
			RequiresFeatures: []string{},
			Limitations: []string{
				"No resource support",
			},
			Workarounds: map[string]string{
				"resources": "Implement resources as tools that return content",
			},
		},
		{
			Name:    "VS Code GitHub Copilot",
			Pattern: "vscode.*copilot|copilot.*vscode",
			SupportedFeatures: []string{
				"tools",
				"discovery",
				"roots",
			},
			RequiresFeatures: []string{
				"roots", // VS Code expects roots for workspace access
			},
			Limitations: []string{
				"No resource subscriptions",
				"No prompts support",
			},
			Workarounds: map[string]string{
				"prompts": "Expose prompts as tools with template parameters",
			},
		},
		{
			Name:    "Cursor",
			Pattern: "cursor",
			SupportedFeatures: []string{
				"tools",
			},
			RequiresFeatures: []string{},
			Limitations: []string{
				"Tools only",
				"No advanced features",
			},
			Workarounds: map[string]string{
				"resources": "Wrap resource access in tool handlers",
				"prompts":   "Convert prompts to tools",
			},
		},
		{
			Name:    "Continue",
			Pattern: "continue",
			SupportedFeatures: []string{
				"tools",
				"prompts",
				"resources",
			},
			RequiresFeatures: []string{},
			Limitations: []string{
				"No discovery",
				"No subscriptions",
			},
			Workarounds: map[string]string{
				"discovery": "Static registration only",
			},
		},
		{
			Name:    "Cline",
			Pattern: "cline",
			SupportedFeatures: []string{
				"tools",
				"resources",
			},
			RequiresFeatures: []string{},
			Limitations: []string{
				"No prompts",
				"Discovery support varies by version",
			},
			Workarounds: map[string]string{
				"prompts": "Use tools with pre-defined templates",
			},
		},
		{
			Name:    "Windsurf Editor",
			Pattern: "windsurf",
			SupportedFeatures: []string{
				"tools",
				"discovery",
			},
			RequiresFeatures: []string{},
			Limitations: []string{
				"Limited resource support",
			},
			Workarounds: map[string]string{
				"resources": "Use tool-based file access",
			},
		},
		{
			Name:    "Zed",
			Pattern: "zed",
			SupportedFeatures: []string{
				"prompts",
			},
			RequiresFeatures: []string{},
			Limitations: []string{
				"Prompts only (as slash commands)",
				"No tools or resources",
			},
			Workarounds: map[string]string{
				"tools": "Not supported - design prompts to be self-contained",
			},
		},
		{
			Name:    "Generic/Unknown",
			Pattern: ".*",
			SupportedFeatures: []string{
				"tools", // Assume basic tool support
			},
			RequiresFeatures: []string{},
			Limitations: []string{
				"Unknown capabilities",
				"Conservative feature set",
			},
			Workarounds: map[string]string{
				"advanced": "Stick to basic tool functionality",
			},
		},
	}
}

// Feature constants
const (
	FeatureTools        = "tools"
	FeatureResources    = "resources"
	FeaturePrompts      = "prompts"
	FeatureDiscovery    = "discovery"
	FeatureSampling     = "sampling"
	FeatureRoots        = "roots"
	FeatureSubscriptions = "subscriptions"
)
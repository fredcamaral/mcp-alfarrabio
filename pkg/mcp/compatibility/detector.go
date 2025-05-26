package compatibility

import (
	"mcp-memory/pkg/mcp/protocol"
	"regexp"
	"strings"
)

// Detector identifies client capabilities and limitations
type Detector struct {
	profiles []ClientProfile
}

// NewDetector creates a new compatibility detector
func NewDetector() *Detector {
	return &Detector{
		profiles: GetKnownProfiles(),
	}
}

// DetectClient identifies a client from its info and capabilities
func (d *Detector) DetectClient(clientInfo protocol.ClientInfo, capabilities protocol.ClientCapabilities) *ClientProfile {
	// Try to match based on client name and version
	clientString := strings.ToLower(clientInfo.Name + " " + clientInfo.Version)
	
	for _, profile := range d.profiles {
		if profile.Pattern == ".*" {
			continue // Skip generic profile for now
		}
		
		pattern, err := regexp.Compile("(?i)" + profile.Pattern)
		if err != nil {
			continue
		}
		
		if pattern.MatchString(clientString) {
			return &profile
		}
	}
	
	// Check for specific capability signatures
	if d.hasCapabilitySignature(capabilities, []string{"sampling"}) {
		// Clients with sampling are likely newer/advanced
		for _, profile := range d.profiles {
			if contains(profile.SupportedFeatures, FeatureSampling) {
				return &profile
			}
		}
	}
	
	// Return generic profile as fallback
	for _, profile := range d.profiles {
		if profile.Pattern == ".*" {
			return &profile
		}
	}
	
	return nil
}

// CheckFeatureSupport checks if a feature is supported by the client
func (d *Detector) CheckFeatureSupport(profile *ClientProfile, feature string) (supported bool, workaround string) {
	if profile == nil {
		return false, "Unknown client - conservative feature set recommended"
	}
	
	// Check if feature is supported
	for _, f := range profile.SupportedFeatures {
		if f == feature {
			return true, ""
		}
	}
	
	// Check for workaround
	if workaround, exists := profile.Workarounds[feature]; exists {
		return false, workaround
	}
	
	return false, "Feature not supported by " + profile.Name
}

// GetRequiredFeatures returns features that must be enabled for a client
func (d *Detector) GetRequiredFeatures(profile *ClientProfile) []string {
	if profile == nil {
		return []string{}
	}
	return profile.RequiresFeatures
}

// GetSupportedCapabilities builds server capabilities based on client profile
func (d *Detector) GetSupportedCapabilities(profile *ClientProfile) protocol.ServerCapabilities {
	caps := protocol.ServerCapabilities{
		Experimental: make(map[string]interface{}),
	}
	
	if profile == nil {
		// Conservative default
		caps.Tools = &protocol.ToolCapability{}
		return caps
	}
	
	// Enable capabilities based on client support
	for _, feature := range profile.SupportedFeatures {
		switch feature {
		case FeatureTools:
			caps.Tools = &protocol.ToolCapability{
				ListChanged: contains(profile.SupportedFeatures, FeatureSubscriptions),
			}
		case FeatureResources:
			caps.Resources = &protocol.ResourceCapability{
				Subscribe:   contains(profile.SupportedFeatures, FeatureSubscriptions),
				ListChanged: contains(profile.SupportedFeatures, FeatureSubscriptions),
			}
		case FeaturePrompts:
			caps.Prompts = &protocol.PromptCapability{
				ListChanged: contains(profile.SupportedFeatures, FeatureSubscriptions),
			}
		case FeatureSampling:
			caps.Sampling = &protocol.SamplingCapability{}
		case FeatureRoots:
			caps.Roots = &protocol.RootsCapability{
				ListChanged: contains(profile.SupportedFeatures, FeatureSubscriptions),
			}
		}
	}
	
	// Add experimental features if discovery is supported
	if contains(profile.SupportedFeatures, FeatureDiscovery) {
		caps.Experimental["discovery"] = map[string]interface{}{
			"enabled": true,
		}
	}
	
	return caps
}

// hasCapabilitySignature checks if capabilities match a signature
func (d *Detector) hasCapabilitySignature(caps protocol.ClientCapabilities, features []string) bool {
	for _, feature := range features {
		switch feature {
		case "sampling":
			if caps.Sampling != nil && len(caps.Sampling) > 0 {
				return true
			}
		case "experimental":
			if caps.Experimental != nil && len(caps.Experimental) > 0 {
				return true
			}
		}
	}
	return false
}

// contains checks if a slice contains a value
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
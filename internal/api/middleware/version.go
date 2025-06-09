// Package middleware provides HTTP middleware components for the MCP Memory Server API.
package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"lerian-mcp-memory/internal/api/response"
)

// Version represents a semantic version
type Version struct {
	Major int
	Minor int
	Patch int
}

// VersionChecker middleware validates client version compatibility
type VersionChecker struct {
	supportedVersions map[string]bool
	minVersion        Version
	maxVersion        Version
}

// NewVersionChecker creates a new version checking middleware
func NewVersionChecker() *VersionChecker {
	return &VersionChecker{
		supportedVersions: map[string]bool{
			"1.0.0": true,
			"1.0.1": true,
			"1.1.0": true,
		},
		minVersion: Version{Major: 1, Minor: 0, Patch: 0},
		maxVersion: Version{Major: 1, Minor: 1, Patch: 999},
	}
}

// Handler returns the version checking middleware handler
func (vc *VersionChecker) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// For public endpoints, still set version headers but skip validation
			if vc.isPublicEndpoint(r.URL.Path) {
				// Add version info to response headers even for public endpoints
				w.Header().Set("X-Server-Version", "1.0.0")
				w.Header().Set("X-Compatible-Versions", vc.getSupportedVersionsList())
				next.ServeHTTP(w, r)
				return
			}

			// Extract client version from headers
			clientVersion := vc.extractClientVersion(r)

			// If no version provided, allow for backward compatibility
			if clientVersion == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Validate version compatibility
			if !vc.isVersionSupported(clientVersion) {
				response.WriteVersionMismatch(w,
					"Client version not supported",
					"Supported versions: "+vc.getSupportedVersionsList())
				return
			}

			// Add version info to response headers
			w.Header().Set("X-Server-Version", "1.0.0")
			w.Header().Set("X-Compatible-Versions", vc.getSupportedVersionsList())

			next.ServeHTTP(w, r)
		})
	}
}

// extractClientVersion extracts client version from request headers
func (vc *VersionChecker) extractClientVersion(r *http.Request) string {
	// Try different header names for client version
	headerNames := []string{
		"X-Client-Version",
		"X-CLI-Version",
		"User-Agent",
	}

	for _, headerName := range headerNames {
		if version := r.Header.Get(headerName); version != "" {
			// Extract version from User-Agent if needed
			if headerName == "User-Agent" {
				return vc.extractVersionFromUserAgent(version)
			}
			return version
		}
	}

	return ""
}

// extractVersionFromUserAgent extracts version from User-Agent header
func (vc *VersionChecker) extractVersionFromUserAgent(userAgent string) string {
	// Look for patterns like "lmmc/1.0.0" or "lerian-cli/1.0.0"
	parts := strings.Split(userAgent, "/")
	if len(parts) >= 2 {
		// Check if first part is a known client
		clientNames := []string{"lmmc", "lerian-cli", "lerian-mcp-memory-cli"}
		for _, name := range clientNames {
			if strings.Contains(strings.ToLower(parts[0]), name) {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// isVersionSupported checks if the client version is supported
func (vc *VersionChecker) isVersionSupported(version string) bool {
	// Check exact version match first
	if vc.supportedVersions[version] {
		return true
	}

	// Parse version and check range
	clientVer, err := vc.parseVersion(version)
	if err != nil {
		return false
	}

	return vc.isVersionInRange(clientVer)
}

// parseVersion parses a version string into Version struct
func (vc *VersionChecker) parseVersion(version string) (Version, error) {
	// Simple version parsing - in production, use a proper semver library
	parts := strings.Split(strings.TrimSpace(version), ".")
	if len(parts) != 3 {
		return Version{}, nil
	}

	var major, minor, patch int
	if _, err := fmt.Sscanf(parts[0]+"."+parts[1]+"."+parts[2], "%d.%d.%d", &major, &minor, &patch); err != nil {
		return Version{}, err
	}

	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

// isVersionInRange checks if version is within supported range
func (vc *VersionChecker) isVersionInRange(version Version) bool {
	return vc.compareVersions(version, vc.minVersion) >= 0 &&
		vc.compareVersions(version, vc.maxVersion) <= 0
}

// compareVersions compares two versions (-1: v1 < v2, 0: v1 == v2, 1: v1 > v2)
func (vc *VersionChecker) compareVersions(v1, v2 Version) int {
	if v1.Major != v2.Major {
		if v1.Major < v2.Major {
			return -1
		}
		return 1
	}

	if v1.Minor != v2.Minor {
		if v1.Minor < v2.Minor {
			return -1
		}
		return 1
	}

	if v1.Patch != v2.Patch {
		if v1.Patch < v2.Patch {
			return -1
		}
		return 1
	}

	return 0
}

// getSupportedVersionsList returns a comma-separated list of supported versions
func (vc *VersionChecker) getSupportedVersionsList() string {
	versions := make([]string, 0, len(vc.supportedVersions))
	for version := range vc.supportedVersions {
		versions = append(versions, version)
	}
	return strings.Join(versions, ", ")
}

// isPublicEndpoint checks if the endpoint should skip version checking
func (vc *VersionChecker) isPublicEndpoint(path string) bool {
	publicEndpoints := []string{
		"/health",
		"/api/v1/health",
		"/metrics",
		"/docs",
		"/openapi.json",
	}

	for _, endpoint := range publicEndpoints {
		if path == endpoint {
			return true
		}
	}

	return false
}

// AddSupportedVersion adds a new supported version
func (vc *VersionChecker) AddSupportedVersion(version string) {
	vc.supportedVersions[version] = true
}

// RemoveSupportedVersion removes a supported version
func (vc *VersionChecker) RemoveSupportedVersion(version string) {
	delete(vc.supportedVersions, version)
}

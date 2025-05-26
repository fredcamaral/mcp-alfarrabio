package roots

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
)

// Handler implements the roots functionality for MCP
type Handler struct {
	roots []Root
}

// NewHandler creates a new roots handler with default roots
func NewHandler() *Handler {
	h := &Handler{
		roots: []Root{},
	}
	
	// Add default roots
	h.addDefaultRoots()
	
	return h
}

// NewHandlerWithRoots creates a new roots handler with custom roots
func NewHandlerWithRoots(roots []Root) *Handler {
	return &Handler{
		roots: roots,
	}
}

// ListRoots handles the roots/list request
func (h *Handler) ListRoots(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return ListRootsResponse{
		Roots: h.roots,
	}, nil
}

// AddRoot adds a new root to the handler
func (h *Handler) AddRoot(root Root) {
	h.roots = append(h.roots, root)
}

// RemoveRoot removes a root by URI
func (h *Handler) RemoveRoot(uri string) {
	filtered := make([]Root, 0, len(h.roots))
	for _, root := range h.roots {
		if root.URI != uri {
			filtered = append(filtered, root)
		}
	}
	h.roots = filtered
}

// GetRoots returns all registered roots
func (h *Handler) GetRoots() []Root {
	return h.roots
}

// addDefaultRoots adds commonly used roots
func (h *Handler) addDefaultRoots() {
	// Add home directory
	if home, err := os.UserHomeDir(); err == nil {
		h.roots = append(h.roots, Root{
			URI:         "file://" + filepath.ToSlash(home),
			Name:        "Home",
			Description: "User home directory",
		})
	}
	
	// Add current working directory
	if cwd, err := os.Getwd(); err == nil {
		h.roots = append(h.roots, Root{
			URI:         "file://" + filepath.ToSlash(cwd),
			Name:        "Working Directory",
			Description: "Current working directory",
		})
	}
	
	// Add temp directory
	tempDir := os.TempDir()
	h.roots = append(h.roots, Root{
		URI:         "file://" + filepath.ToSlash(tempDir),
		Name:        "Temporary",
		Description: "System temporary directory",
	})
}
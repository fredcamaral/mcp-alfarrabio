package cli

import (
	"errors"
	"path/filepath"
	"strings"
)

// validateFilePath validates a file path to prevent directory traversal attacks
func validateFilePath(filePath string) error {
	cleanPath := filepath.Clean(filePath)

	// Check for directory traversal patterns
	if strings.Contains(cleanPath, "..") {
		return errors.New("directory traversal not allowed")
	}

	// Ensure path doesn't start with system directories (additional security)
	if strings.HasPrefix(cleanPath, "/etc/") ||
		strings.HasPrefix(cleanPath, "/sys/") ||
		strings.HasPrefix(cleanPath, "/proc/") {
		return errors.New("access to system directories not allowed")
	}

	return nil
}

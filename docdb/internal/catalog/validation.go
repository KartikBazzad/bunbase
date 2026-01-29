package catalog

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	// MaxDBNameLen is the maximum allowed database name length in bytes.
	MaxDBNameLen = 64
)

// ValidateDBName validates a database name to prevent path traversal and invalid characters.
// Rejects: empty, /, \, .., null byte, invalid UTF-8, and names exceeding MaxDBNameLen.
func ValidateDBName(name string) error {
	if name == "" {
		return fmt.Errorf("database name cannot be empty")
	}

	if !utf8.ValidString(name) {
		return fmt.Errorf("database name must be valid UTF-8")
	}

	if len(name) > MaxDBNameLen {
		return fmt.Errorf("database name exceeds maximum length of %d bytes", MaxDBNameLen)
	}

	// Path traversal and path separators
	if strings.Contains(name, "/") {
		return fmt.Errorf("database name cannot contain '/'")
	}
	if strings.Contains(name, "\\") {
		return fmt.Errorf("database name cannot contain '\\'")
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("database name cannot contain '..'")
	}

	// Null byte
	if strings.ContainsRune(name, 0) {
		return fmt.Errorf("database name cannot contain null bytes")
	}

	return nil
}

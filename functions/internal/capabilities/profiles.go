package capabilities

import (
	"time"
)

// StrictProfile creates a strict security profile with minimal permissions
// Suitable for untrusted code execution in multi-tenant environments
func StrictProfile(projectID string) *Capabilities {
	return &Capabilities{
		AllowFilesystem:   false,
		AllowNetwork:      false,
		AllowChildProcess: false,
		AllowEval:         false,
		AllowedPaths:      nil,
		AllowedDomains:    nil,
		MaxMemory:         100 * 1024 * 1024, // 100MB default
		MaxCPU:             30 * time.Second,  // 30 seconds default
		MaxFileDescriptors: 10,                 // 10 file descriptors default
		ProjectID:          projectID,
	}
}

// PermissiveProfile creates a permissive security profile
// Suitable for trusted code execution
func PermissiveProfile(projectID string) *Capabilities {
	return &Capabilities{
		AllowFilesystem:   true,
		AllowNetwork:      true,
		AllowChildProcess: true,
		AllowEval:         true,
		AllowedPaths:      nil, // All paths allowed
		AllowedDomains:    nil, // All domains allowed
		MaxMemory:         512 * 1024 * 1024, // 512MB default
		MaxCPU:             5 * time.Minute,   // 5 minutes default
		MaxFileDescriptors: 100,                // 100 file descriptors default
		ProjectID:          projectID,
	}
}

// CustomProfile creates a custom security profile with specified options
func CustomProfile(projectID string, opts ...CapabilityOption) *Capabilities {
	c := &Capabilities{
		AllowFilesystem:   false,
		AllowNetwork:      false,
		AllowChildProcess: false,
		AllowEval:         false,
		AllowedPaths:      nil,
		AllowedDomains:    nil,
		MaxMemory:         256 * 1024 * 1024, // 256MB default
		MaxCPU:             1 * time.Minute,   // 1 minute default
		MaxFileDescriptors: 50,                // 50 file descriptors default
		ProjectID:          projectID,
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// DefaultProfile returns the default profile (strict for security)
func DefaultProfile(projectID string) *Capabilities {
	return StrictProfile(projectID)
}

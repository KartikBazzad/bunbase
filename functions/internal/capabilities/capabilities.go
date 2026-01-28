package capabilities

import "time"

// Capabilities defines security boundaries for worker execution
type Capabilities struct {
	// Filesystem access
	AllowFilesystem bool
	AllowedPaths    []string // If AllowFilesystem is true, restrict to these paths

	// Network access
	AllowNetwork   bool
	AllowedDomains []string // If AllowNetwork is true, restrict to these domains

	// Process execution
	AllowChildProcess bool

	// Code execution
	AllowEval bool // Allow eval() and Function() constructor

	// Resource limits
	MaxMemory          int64         // Maximum memory in bytes (0 = unlimited)
	MaxCPU             time.Duration // Maximum CPU time (0 = unlimited)
	MaxFileDescriptors int           // Maximum open file descriptors (0 = unlimited)

	// Project/tenant identification
	ProjectID string
}

// CapabilityOption is a function that modifies capabilities
type CapabilityOption func(*Capabilities)

// WithFilesystemAccess enables filesystem access, optionally restricted to paths
func WithFilesystemAccess(allowedPaths ...string) CapabilityOption {
	return func(c *Capabilities) {
		c.AllowFilesystem = true
		c.AllowedPaths = allowedPaths
	}
}

// WithNetworkAccess enables network access, optionally restricted to domains
func WithNetworkAccess(allowedDomains ...string) CapabilityOption {
	return func(c *Capabilities) {
		c.AllowNetwork = true
		c.AllowedDomains = allowedDomains
	}
}

// WithChildProcessAccess enables child process spawning
func WithChildProcessAccess() CapabilityOption {
	return func(c *Capabilities) {
		c.AllowChildProcess = true
	}
}

// WithEvalAccess enables eval() and Function() constructor
func WithEvalAccess() CapabilityOption {
	return func(c *Capabilities) {
		c.AllowEval = true
	}
}

// WithMemoryLimit sets the maximum memory limit in bytes
func WithMemoryLimit(bytes int64) CapabilityOption {
	return func(c *Capabilities) {
		c.MaxMemory = bytes
	}
}

// WithCPULimit sets the maximum CPU time limit
func WithCPULimit(duration time.Duration) CapabilityOption {
	return func(c *Capabilities) {
		c.MaxCPU = duration
	}
}

// WithFileDescriptorLimit sets the maximum number of open file descriptors
func WithFileDescriptorLimit(count int) CapabilityOption {
	return func(c *Capabilities) {
		c.MaxFileDescriptors = count
	}
}

// Validate checks if capabilities are valid
func (c *Capabilities) Validate() error {
	if c.MaxMemory < 0 {
		return ErrInvalidMemoryLimit
	}
	if c.MaxFileDescriptors < 0 {
		return ErrInvalidFileDescriptorLimit
	}
	return nil
}

// IsPathAllowed checks if a filesystem path is allowed
func (c *Capabilities) IsPathAllowed(path string) bool {
	if !c.AllowFilesystem {
		return false
	}
	if len(c.AllowedPaths) == 0 {
		return true // All paths allowed if no restrictions
	}
	// Check if path matches any allowed path (prefix match)
	for _, allowed := range c.AllowedPaths {
		if path == allowed || len(path) > len(allowed) && path[:len(allowed)+1] == allowed+"/" {
			return true
		}
	}
	return false
}

// IsDomainAllowed checks if a network domain is allowed
func (c *Capabilities) IsDomainAllowed(domain string) bool {
	if !c.AllowNetwork {
		return false
	}
	if len(c.AllowedDomains) == 0 {
		return true // All domains allowed if no restrictions
	}
	// Check if domain matches any allowed domain
	for _, allowed := range c.AllowedDomains {
		if domain == allowed {
			return true
		}
		// Support wildcard subdomains: *.example.com matches sub.example.com
		if len(allowed) > 2 && allowed[0:2] == "*." {
			baseDomain := allowed[2:]
			if domain == baseDomain || (len(domain) > len(baseDomain)+1 && domain[len(domain)-len(baseDomain)-1:] == "."+baseDomain) {
				return true
			}
		}
	}
	return false
}

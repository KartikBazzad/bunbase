package capabilities

import (
	"testing"
	"time"
)

func TestStrictProfile(t *testing.T) {
	caps := StrictProfile("test-project")
	
	if caps.AllowFilesystem {
		t.Error("Strict profile should not allow filesystem")
	}
	if caps.AllowNetwork {
		t.Error("Strict profile should not allow network")
	}
	if caps.AllowChildProcess {
		t.Error("Strict profile should not allow child process")
	}
	if caps.AllowEval {
		t.Error("Strict profile should not allow eval")
	}
	if caps.MaxMemory == 0 {
		t.Error("Strict profile should have memory limit")
	}
	if caps.ProjectID != "test-project" {
		t.Errorf("Expected project ID 'test-project', got '%s'", caps.ProjectID)
	}
}

func TestPermissiveProfile(t *testing.T) {
	caps := PermissiveProfile("test-project")
	
	if !caps.AllowFilesystem {
		t.Error("Permissive profile should allow filesystem")
	}
	if !caps.AllowNetwork {
		t.Error("Permissive profile should allow network")
	}
	if !caps.AllowChildProcess {
		t.Error("Permissive profile should allow child process")
	}
	if !caps.AllowEval {
		t.Error("Permissive profile should allow eval")
	}
}

func TestCustomProfile(t *testing.T) {
	caps := CustomProfile("test-project",
		WithFilesystemAccess("/tmp", "/var/tmp"),
		WithMemoryLimit(256*1024*1024),
	)
	
	if !caps.AllowFilesystem {
		t.Error("Custom profile should allow filesystem when specified")
	}
	if len(caps.AllowedPaths) != 2 {
		t.Errorf("Expected 2 allowed paths, got %d", len(caps.AllowedPaths))
	}
	if caps.MaxMemory != 256*1024*1024 {
		t.Errorf("Expected 256MB memory limit, got %d", caps.MaxMemory)
	}
}

func TestIsPathAllowed(t *testing.T) {
	caps := &Capabilities{
		AllowFilesystem: true,
		AllowedPaths:    []string{"/tmp", "/var/tmp"},
	}
	
	if !caps.IsPathAllowed("/tmp") {
		t.Error("/tmp should be allowed")
	}
	if !caps.IsPathAllowed("/tmp/file.txt") {
		t.Error("/tmp/file.txt should be allowed (subpath)")
	}
	if caps.IsPathAllowed("/home") {
		t.Error("/home should not be allowed")
	}
	
	// Test with no restrictions
	caps.AllowedPaths = nil
	if !caps.IsPathAllowed("/any/path") {
		t.Error("All paths should be allowed when no restrictions")
	}
	
	// Test with filesystem disabled
	caps.AllowFilesystem = false
	if caps.IsPathAllowed("/tmp") {
		t.Error("No paths should be allowed when filesystem is disabled")
	}
}

func TestIsDomainAllowed(t *testing.T) {
	caps := &Capabilities{
		AllowNetwork:   true,
		AllowedDomains: []string{"example.com", "*.subdomain.com"},
	}
	
	if !caps.IsDomainAllowed("example.com") {
		t.Error("example.com should be allowed")
	}
	if !caps.IsDomainAllowed("subdomain.com") {
		t.Error("subdomain.com should be allowed (wildcard match)")
	}
	if caps.IsDomainAllowed("other.com") {
		t.Error("other.com should not be allowed")
	}
	
	// Test with no restrictions
	caps.AllowedDomains = nil
	if !caps.IsDomainAllowed("any.domain.com") {
		t.Error("All domains should be allowed when no restrictions")
	}
	
	// Test with network disabled
	caps.AllowNetwork = false
	if caps.IsDomainAllowed("example.com") {
		t.Error("No domains should be allowed when network is disabled")
	}
}

func TestValidate(t *testing.T) {
	caps := &Capabilities{
		MaxMemory:         100 * 1024 * 1024,
		MaxFileDescriptors: 10,
	}
	
	if err := caps.Validate(); err != nil {
		t.Errorf("Valid capabilities should not error: %v", err)
	}
	
	caps.MaxMemory = -1
	if err := caps.Validate(); err != ErrInvalidMemoryLimit {
		t.Errorf("Expected ErrInvalidMemoryLimit, got %v", err)
	}
	
	caps.MaxMemory = 0
	caps.MaxFileDescriptors = -1
	if err := caps.Validate(); err != ErrInvalidFileDescriptorLimit {
		t.Errorf("Expected ErrInvalidFileDescriptorLimit, got %v", err)
	}
}

func TestCapabilityOptions(t *testing.T) {
	caps := CustomProfile("test",
		WithFilesystemAccess("/tmp"),
		WithNetworkAccess("example.com"),
		WithChildProcessAccess(),
		WithEvalAccess(),
		WithMemoryLimit(512*1024*1024),
		WithCPULimit(5*time.Minute),
		WithFileDescriptorLimit(100),
	)
	
	if !caps.AllowFilesystem {
		t.Error("Should allow filesystem")
	}
	if !caps.AllowNetwork {
		t.Error("Should allow network")
	}
	if !caps.AllowChildProcess {
		t.Error("Should allow child process")
	}
	if !caps.AllowEval {
		t.Error("Should allow eval")
	}
	if caps.MaxMemory != 512*1024*1024 {
		t.Error("Memory limit should be set")
	}
	if caps.MaxCPU != 5*time.Minute {
		t.Error("CPU limit should be set")
	}
	if caps.MaxFileDescriptors != 100 {
		t.Error("File descriptor limit should be set")
	}
}

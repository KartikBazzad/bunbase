package config

import (
	"github.com/kartikbazzad/bunbase/functions/internal/capabilities"
)

// RuntimeConfig provides runtime-specific configuration
type RuntimeConfig struct {
	Type                string                      // "bun" | "quickjs" | "quickjs-ng"
	BunPath             string                      // Path to bun executable
	QuickJSPath         string                      // Path to quickjs-worker binary
	DefaultCapabilities *capabilities.Capabilities  // Default capabilities for new functions
}

// GetRuntimeConfig returns runtime configuration from worker config
func (wc *WorkerConfig) GetRuntimeConfig() *RuntimeConfig {
	return &RuntimeConfig{
		Type:                wc.Runtime,
		BunPath:             wc.BunPath,
		QuickJSPath:         wc.QuickJSPath,
		DefaultCapabilities: wc.Capabilities,
	}
}

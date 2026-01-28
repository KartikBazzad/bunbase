package config

import (
	"time"

	"github.com/kartikbazzad/bunbase/functions/internal/capabilities"
)

type Config struct {
	DataDir    string
	SocketPath string
	Worker     WorkerConfig
	Gateway    GatewayConfig
	Metadata   MetadataConfig
	Logs       LogsConfig
}

type WorkerConfig struct {
	MaxWorkersPerFunction  int
	WarmWorkersPerFunction int
	IdleTimeout            time.Duration
	StartupTimeout         time.Duration
	ExecutionTimeout       time.Duration
	MemoryLimitMB          int
	BunPath                string
	Runtime                string                      // "bun" or "quickjs" or "quickjs-ng"
	QuickJSPath            string                      // Path to quickjs-worker binary
	Capabilities           *capabilities.Capabilities  // Security capabilities
}

type GatewayConfig struct {
	HTTPPort   int
	EnableHTTP bool
}

type MetadataConfig struct {
	DBPath string
}

type LogsConfig struct {
	DBPath    string
	JSONLPath string
	Retention time.Duration
}

func DefaultConfig() *Config {
	return &Config{
		DataDir:    "./data",
		SocketPath: "/tmp/functions.sock",
		Worker: WorkerConfig{
			MaxWorkersPerFunction:  10,
			WarmWorkersPerFunction: 2,
			IdleTimeout:            5 * time.Minute,
			StartupTimeout:          10 * time.Second,
			ExecutionTimeout:        30 * time.Second,
			MemoryLimitMB:           256,
			BunPath:                 "bun",
			Runtime:                 "bun", // Default to bun for backward compatibility
			QuickJSPath:             "./cmd/quickjs-worker/quickjs-worker",
			Capabilities:             nil,  // Will be set per-function
		},
		Gateway: GatewayConfig{
			HTTPPort:   8080,
			EnableHTTP: true,
		},
		Metadata: MetadataConfig{
			DBPath: "./data/metadata.db",
		},
		Logs: LogsConfig{
			DBPath:    "./data/logs.db",
			JSONLPath: "./data/logs",
			Retention: 30 * 24 * time.Hour, // 30 days
		},
	}
}

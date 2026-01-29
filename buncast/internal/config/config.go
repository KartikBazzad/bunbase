package config

import (
	"time"
)

// Config holds Buncast server configuration.
type Config struct {
	IPC  IPCConfig
	HTTP HTTPConfig
}

// IPCConfig configures the Unix socket IPC server.
type IPCConfig struct {
	SocketPath     string
	MaxConnections int // 0 = unlimited
	DebugMode      bool
}

// HTTPConfig configures the HTTP server (health, admin, SSE).
type HTTPConfig struct {
	ListenAddr   string
	Enabled      bool
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DefaultConfig returns default Buncast configuration.
func DefaultConfig() *Config {
	return &Config{
		IPC: IPCConfig{
			SocketPath:     "/tmp/buncast.sock",
			MaxConnections: 0,
			DebugMode:      false,
		},
		HTTP: HTTPConfig{
			ListenAddr:   ":8081",
			Enabled:      true,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
	}
}

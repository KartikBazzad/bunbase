// Package config provides Bunder server and storage configuration and flag parsing.
package config

import (
	"flag"
	"time"
)

// Config holds Bunder server and storage configuration.
// Paths under DataPath (WALPath, SnapshotPath, AOFPath) are derived if empty.
type Config struct {
	DataPath         string        // Database directory; contains data.db, wal/, etc.
	PageSize         int           // Page size in bytes (default 4096)
	BufferPoolSize   int           // Number of pages in buffer pool (default 10000 = 40MB)
	WALPath          string        // Write-ahead log directory (default DataPath/wal)
	SnapshotPath     string        // Snapshot directory (default DataPath/snapshots)
	SnapshotInterval time.Duration // Interval for background snapshots
	AOFEnabled       bool          // Enable append-only file
	AOFPath          string        // AOF file path (default DataPath/appendonly.aof)
	Shards           int           // Number of shards for the KV map (default 256)
	ListenAddr       string        // TCP listen address for RESP (default :6379)
	HTTPAddr         string        // HTTP API listen address (default :8080)
	MaxClients       int           // Maximum concurrent connections (default 10000)
	TTLCheckInterval time.Duration // How often to sweep expired keys (default 1s)
	BuncastAddr      string        // Buncast Unix socket path for pub/sub
	BuncastEnabled   bool          // Enable Buncast pub/sub integration
}

// Default returns default configuration.
func Default() *Config {
	return &Config{
		DataPath:         "./data",
		PageSize:         4096,
		BufferPoolSize:   10000,
		WALPath:          "",
		SnapshotPath:     "",
		SnapshotInterval: 5 * time.Minute,
		AOFEnabled:       false,
		AOFPath:          "",
		Shards:           256,
		ListenAddr:       ":6379",
		HTTPAddr:         ":8080",
		MaxClients:       10000,
		TTLCheckInterval: time.Second,
		BuncastAddr:      "/tmp/buncast.sock",
		BuncastEnabled:   false,
	}
}

// ParseFlags parses command-line flags into the config.
func (c *Config) ParseFlags() error {
	flag.StringVar(&c.DataPath, "data", c.DataPath, "database directory")
	flag.StringVar(&c.ListenAddr, "addr", c.ListenAddr, "TCP listen address")
	flag.StringVar(&c.HTTPAddr, "http", c.HTTPAddr, "HTTP API listen address")
	flag.IntVar(&c.BufferPoolSize, "buffer-pool", c.BufferPoolSize, "buffer pool size (pages)")
	flag.IntVar(&c.Shards, "shards", c.Shards, "number of shards")
	flag.BoolVar(&c.BuncastEnabled, "buncast", c.BuncastEnabled, "enable Buncast pub/sub")
	flag.Parse()

	if c.WALPath == "" {
		c.WALPath = c.DataPath + "/wal"
	}
	if c.SnapshotPath == "" {
		c.SnapshotPath = c.DataPath + "/snapshots"
	}
	if c.AOFPath == "" {
		c.AOFPath = c.DataPath + "/appendonly.aof"
	}
	return nil
}

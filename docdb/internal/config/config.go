package config

import (
	"runtime"
	"time"
)

type Config struct {
	DataDir string

	Memory  MemoryConfig
	WAL     WALConfig
	Sched   SchedulerConfig
	DB      DBConfig
	IPC     IPCConfig
	Healing HealingConfig
	Query   QueryConfig // Phase D.7/D.8: Query limits and defaults
}

type MemoryConfig struct {
	GlobalCapacityMB uint64
	PerDBLimitMB     uint64
	ReplayBudgetMB   uint64 // Memory budget for WAL replay (0 = use PerDBLimitMB)
	BufferSizes      []uint64
}

type FsyncMode int

const (
	FsyncAlways   FsyncMode = iota // Sync on every write (safest, slowest)
	FsyncGroup                     // Batch syncs with group commit (recommended)
	FsyncInterval                  // Sync at fixed intervals
	FsyncNone                      // Never sync (for benchmarks only, unsafe)
)

type FsyncConfig struct {
	Mode         FsyncMode // Sync strategy: always | group | interval | none
	IntervalMS   int       // Milliseconds for interval mode (default: 1ms)
	MaxBatchSize int       // Max records per group commit batch (default: 100)
}

type WALConfig struct {
	Dir                 string
	MaxFileSizeMB       uint64
	FsyncOnCommit       bool // Deprecated: Use FsyncConfig instead
	Checkpoint          CheckpointConfig
	TrimAfterCheckpoint bool        // Automatically trim WAL segments after checkpoint
	KeepSegments        int         // Number of segments to keep before checkpoint
	Fsync               FsyncConfig // New: Fsync configuration
}

type CheckpointConfig struct {
	IntervalMB     uint64 // Create checkpoint every X MB
	AutoCreate     bool   // Automatically create checkpoints
	MaxCheckpoints int    // Maximum checkpoints to keep (0 = unlimited)
}

type SchedulerConfig struct {
	QueueDepth        int           // Per-DB queue depth (backpressure)
	RoundRobinDBs     bool          // Whether to use round-robin across DBs (may be ignored in executor mode)
	WorkerCount       int           // Number of scheduler workers (0 = auto-scale; v0.4: recommended = 1)
	MaxWorkers        int           // Maximum workers for auto-tuning (default: 256)
	WorkerExpiry      time.Duration // Goroutine expiry for ants pool (default: 1s)
	PreAlloc          bool          // Pre-allocate goroutine queue (default: false)
	UnsafeMultiWriter bool          // Allow more than one scheduler worker (can increase contention)
}

type DBConfig struct {
	CompactionSizeThresholdMB uint64
	CompactionTombstoneRatio  float64
	MaxOpenDBs                int
	IdleTimeout               time.Duration
}

// LogicalDBConfig configures partitioning and execution for a LogicalDB (v0.4).
type LogicalDBConfig struct {
	PartitionCount      int           // Number of partitions (default: 16)
	WorkerCount         int           // Number of workers per LogicalDB (default: NumCPU)
	QueueSize           int           // Task queue size per partition (default: 1024)
	GroupCommitInterval time.Duration // WAL group commit interval per partition (default: 1-5ms)
	MaxSegmentSize      int64         // Max WAL segment size per partition (default: from WALConfig)
}

type IPCConfig struct {
	SocketPath     string
	EnableTCP      bool
	TCPPort        int
	MaxConnections int  // Max concurrent connections (0 = unlimited, used with ants)
	DebugMode      bool // Phase E.9: Enable request flow logging
}

type HealingConfig struct {
	Enabled          bool          // Enable automatic healing
	Interval         time.Duration // Periodic health scan interval
	OnReadCorruption bool          // Trigger healing on corruption detection during read
	MaxBatchSize     int           // Maximum documents to heal in one batch
}

// QueryConfig configures query execution limits (Phase D.7/D.8).
type QueryConfig struct {
	MaxPartitionsPerDB   int           // Maximum partitions per LogicalDB (default: 256)
	MaxConcurrentQueries int           // Maximum concurrent queries per LogicalDB (default: 100)
	QueryTimeout         time.Duration // Query execution timeout (default: 30s)
	MaxQueryMemoryMB     uint64        // Maximum memory per query in MB (default: 100MB)
	MaxQueryLimit        int           // Maximum query result limit; client limit is clamped to this (default: 10000)
	MaxWALSizePerDB      uint64        // WAL disk cap per DB in bytes (default: 10GB)
}

func DefaultConfig() *Config {
	return &Config{
		DataDir: "./data",
		Memory: MemoryConfig{
			GlobalCapacityMB: 1024,
			PerDBLimitMB:     256,
			BufferSizes:      []uint64{1024, 4096, 16384, 65536, 262144},
		},
		WAL: WALConfig{
			Dir:                 "./data/wal",
			MaxFileSizeMB:       64,
			FsyncOnCommit:       true, // Deprecated: kept for backward compatibility
			TrimAfterCheckpoint: true,
			KeepSegments:        2,
			Checkpoint: CheckpointConfig{
				IntervalMB:     64,
				AutoCreate:     true,
				MaxCheckpoints: 0, // Unlimited for v0.1
			},
			Fsync: FsyncConfig{
				Mode:         FsyncGroup, // Conservative default: 1ms group commit
				IntervalMS:   1,          // Default batch interval
				MaxBatchSize: 100,        // Max records per batch
			},
		},
		Sched: SchedulerConfig{
			QueueDepth:        100,
			RoundRobinDBs:     true,
			WorkerCount:       1,           // v0.4: single worker by default (single-writer per DB)
			MaxWorkers:        1,           // cap at 1 unless UnsafeMultiWriter is set
			WorkerExpiry:      time.Second, // Idle goroutine expiry for ants
			PreAlloc:          false,       // Pre-allocate ring buffer
			UnsafeMultiWriter: false,
		},
		DB: DBConfig{
			CompactionSizeThresholdMB: 100,
			CompactionTombstoneRatio:  0.3,
			MaxOpenDBs:                100,
			IdleTimeout:               5 * time.Minute,
		},
		IPC: IPCConfig{
			SocketPath: "/tmp/docdb.sock",
			EnableTCP:  false,
			TCPPort:    0,
			DebugMode:  false, // Phase E.9: Debug mode disabled by default
		},
		Healing: HealingConfig{
			Enabled:          true,
			Interval:         1 * time.Hour,
			OnReadCorruption: true,
			MaxBatchSize:     100,
		},
		Query: QueryConfig{
			MaxPartitionsPerDB:   256,                     // Phase D.7: Sensible default
			MaxConcurrentQueries: 100,                     // Phase D.7: Sensible default
			QueryTimeout:         30 * time.Second,        // Phase D.7: Sensible default
			MaxQueryMemoryMB:     100,                     // Phase D.7: Sensible default (100MB)
			MaxQueryLimit:        10000,                   // Max rows per query; client limit clamped
			MaxWALSizePerDB:      10 * 1024 * 1024 * 1024, // Phase D.8: 10GB WAL cap per DB
		},
	}
}

// DefaultLogicalDBConfig returns default configuration for a LogicalDB (v0.4).
// Phase D.7: Ship-ready defaults (PartitionCount = 2×CPU, WorkerCount = 1×CPU, QueueSize = 1024).
func DefaultLogicalDBConfig() *LogicalDBConfig {
	return &LogicalDBConfig{
		PartitionCount:      2 * runtime.NumCPU(), // Write parallelism
		WorkerCount:         runtime.NumCPU(),     // Execution concurrency
		QueueSize:           1024,                 // Backpressure buffer
		GroupCommitInterval: 1 * time.Millisecond,
		MaxSegmentSize:      64 * 1024 * 1024, // 64MB default
	}
}

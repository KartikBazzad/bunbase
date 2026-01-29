package config

import "time"

type Config struct {
	DataDir string

	Memory  MemoryConfig
	WAL     WALConfig
	Sched   SchedulerConfig
	DB      DBConfig
	IPC     IPCConfig
	Healing HealingConfig
}

type MemoryConfig struct {
	GlobalCapacityMB uint64
	PerDBLimitMB     uint64
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
	QueueDepth    int
	RoundRobinDBs bool
	WorkerCount   int // Number of scheduler workers (0 = auto-scale)
	MaxWorkers    int // Maximum workers for auto-tuning (default: 256)
}

type DBConfig struct {
	CompactionSizeThresholdMB uint64
	CompactionTombstoneRatio  float64
	MaxOpenDBs                int
	IdleTimeout               time.Duration
}

type IPCConfig struct {
	SocketPath string
	EnableTCP  bool
	TCPPort    int
}

type HealingConfig struct {
	Enabled          bool          // Enable automatic healing
	Interval         time.Duration // Periodic health scan interval
	OnReadCorruption bool          // Trigger healing on corruption detection during read
	MaxBatchSize     int           // Maximum documents to heal in one batch
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
			QueueDepth:    100,
			RoundRobinDBs: true,
			WorkerCount:   0,   // 0 = dynamic auto-scaling
			MaxWorkers:    256, // Cap auto-tuning
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
		},
		Healing: HealingConfig{
			Enabled:          true,
			Interval:         1 * time.Hour,
			OnReadCorruption: true,
			MaxBatchSize:     100,
		},
	}
}

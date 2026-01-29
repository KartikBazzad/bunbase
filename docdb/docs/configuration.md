# DocDB Configuration Guide

This guide describes all configuration options, their defaults, and recommended settings for different use cases.

## Table of Contents

1. [Configuration File](#configuration-file)
2. [Memory Configuration](#memory-configuration)
3. [WAL Configuration](#wal-configuration)
4. [Scheduler Configuration](#scheduler-configuration)
5. [Database Configuration](#database-configuration)
6. [IPC Configuration](#ipc-configuration)
7. [Command-Line Flags](#command-line-flags)
8. [Recommended Configurations](#recommended-configurations)
9. [Performance Tuning](#performance-tuning)

---

## Configuration File

DocDB uses a configuration struct with sensible defaults. You can customize it programmatically or via command-line flags.

### Default Configuration

```go
cfg := config.DefaultConfig()
```

This provides:

```go
Config{
    DataDir: "./data",
    Memory: MemoryConfig{
        GlobalCapacityMB: 1024,      // 1 GB total
        PerDBLimitMB: 256,           // 256 MB per database
        BufferSizes: []uint64{1024, 4096, 16384, 65536, 262144},
    },
    WAL: WALConfig{
        Dir: "./data/wal",
        MaxFileSizeMB: 100,            // 100 MB before warning
        FsyncOnCommit: true,           // Durability vs performance tradeoff
    },
    Sched: SchedulerConfig{
        QueueDepth: 100,              // 100 pending requests per DB
        RoundRobinDBs: true,
    },
    DB: DBConfig{
        CompactionSizeThresholdMB: 100,  // Trigger at 100 MB data file
        CompactionTombstoneRatio: 0.3,  // Trigger at 30% tombstones
        MaxOpenDBs: 100,             // Maximum concurrent databases
        IdleTimeout: 5 * time.Minute,  // Close after 5 min idle
    },
    IPC: IPCConfig{
        SocketPath: "/tmp/docdb.sock", // Unix socket path
        EnableTCP: false,              // TCP not supported in v0
        TCPPort: 0,
    },
    Healing: HealingConfig{
        Enabled:          true,              // Enable automatic healing
        Interval:         1 * time.Hour,    // Periodic health scan interval
        OnReadCorruption: true,              // Trigger healing on corruption detection
        MaxBatchSize:     100,               // Maximum documents to heal per batch
    },
    WAL: WALConfig{
        // ... existing WAL config ...
        Checkpoint: CheckpointConfig{
            IntervalMB:     64,    // Create checkpoint every 64 MB
            AutoCreate:     true,  // Automatically create checkpoints
            MaxCheckpoints: 0,     // Unlimited checkpoints
        },
        TrimAfterCheckpoint: true,  // Automatically trim old segments
        KeepSegments:        2,     // Keep 2 segments before checkpoint
    },
}
```

---

## Memory Configuration

### GlobalCapacityMB

**Description:** Total memory limit across all databases.

**Default:** `1024` (1 GB)

**Range:** 1 - 16384 MB (1 MB - 16 GB)

**Recommended Settings:**

- **Development:** 256 MB (enough for testing)
- **Production Small:** 1 GB (1000 small documents or 100 medium)
- **Production Large:** 4 GB (10000 small documents or 1000 medium)
- **Stress Testing:** 8 GB+ (push memory limits)

**Trade-offs:**

- **Lower:** Less memory usage, faster recovery, but more OOM errors
- **Higher:** More documents in memory, but slower recovery and higher OS pressure

**Example:**

```go
cfg.Memory.GlobalCapacityMB = 2048 // 2 GB total
```

### PerDBLimitMB

**Description:** Memory limit per individual database.

**Default:** `256` (256 MB)

**Range:** 1 - GlobalCapacityMB

**Recommended Settings:**

- **Single Database:** Set equal to GlobalCapacityMB
- **Few Databases (< 10):** 1/4 of GlobalCapacityMB
- **Many Databases (10-100):** 1/20 of GlobalCapacityMB

**Trade-offs:**

- **Lower:** Prevents one DB from using all memory, fair sharing
- **Higher:** Allows large documents in single DB, but risk OOM

**Example:**

```go
// Allow each database up to 512 MB
cfg.Memory.PerDBLimitMB = 512
```

### BufferSizes

**Description:** Buffer pool sizes for efficient memory allocation.

**Default:** `[1024, 4096, 16384, 65536, 262144]` (1KB, 4KB, 16KB, 64KB, 256KB)

**Recommended Settings:**

- **Default:** Works well for most use cases
- **Small Documents:** Focus on smaller sizes `[512, 1024, 2048, 4096]`
- **Large Documents:** Add larger sizes `[..., 524288, 1048576]`

**Trade-offs:**

- **Smaller:** Less memory waste, more allocations
- **Larger:** Fewer allocations, but more memory overhead

**Example:**

```go
// Optimize for medium-sized documents (4-32 KB)
cfg.Memory.BufferSizes = []uint64{4096, 8192, 16384, 32768}
```

---

## WAL Configuration

### Dir

**Description:** Directory for WAL files.

**Default:** `"./data/wal"`

**Recommended Settings:**

- **Same Volume:** Keep WAL and data files on same storage volume
- **Fast Storage:** Use SSD if available
- **Separate Partition:** Optional - can improve performance

**Example:**

```go
cfg.WAL.Dir = "/fastssd/docdb/wal"
```

### MaxFileSizeMB

**Description:** Maximum WAL file size before logging warning.

**Default:** `100` (100 MB)

**Range:** 10 - 10240 MB (10 MB - 10 GB)

**Recommended Settings:**

- **Development:** 10 MB (fast rotation)
- **Production:** 100 MB (balance rotation speed vs overhead)
- **High Write Volume:** 500 MB (reduce rotation frequency)

**Trade-offs:**

- **Lower:** More frequent rotation, faster recovery
- **Higher:** Less rotation overhead, but slower recovery

**Note:** v0 doesn't automatically rotate WAL, this is just a warning.

**Example:**

```go
cfg.WAL.MaxFileSizeMB = 500 // 500 MB before warning
```

### FsyncOnCommit

**Description:** Whether to fsync WAL file on every commit.

**Default:** `true`

**Options:**

- `true`: Maximum durability, slower writes (fsync on every write)
- `false`: Faster writes, risk of losing last ~100ms data on crash

**Recommended Settings:**

- **Production:** `true` (durability is critical)
- **Testing/Benchmarks:** `false` (performance is critical)
- **Caching Layer:** `false` if you have external caching

**Trade-offs:**

- **true:** Durable but slow (100-1000x slower than buffered writes)
- **false:** Fast but less durable (risk losing last ~100ms data)

**Example:**

```go
// Disable fsync for maximum performance (not recommended for production)
cfg.WAL.FsyncOnCommit = false
```

### Checkpoint Configuration

**Description:** Controls checkpoint creation for bounded recovery time.

**Default Configuration:**

```go
Checkpoint: CheckpointConfig{
    IntervalMB:     64,    // Create checkpoint every 64 MB
    AutoCreate:     true,  // Automatically create checkpoints
    MaxCheckpoints: 0,     // Unlimited checkpoints (0 = unlimited)
}
```

#### IntervalMB

**Description:** WAL size threshold (in MB) that triggers checkpoint creation.

**Default:** `64` MB

**Range:** 1 - 10240 MB

**Recommended Settings:**

- **Development:** 1-10 MB (frequent checkpoints, faster recovery)
- **Production:** 64 MB (balanced recovery time vs overhead)
- **High Write Volume:** 128-256 MB (less frequent checkpoints)

**Trade-offs:**

- **Lower:** Faster recovery (less WAL to replay), more checkpoint overhead
- **Higher:** Slower recovery (more WAL to replay), less checkpoint overhead

**Example:**

```go
cfg.WAL.Checkpoint.IntervalMB = 128 // Checkpoint every 128 MB
```

#### AutoCreate

**Description:** Whether to automatically create checkpoints at the configured interval.

**Default:** `true`

**Options:**

- `true`: Checkpoints created automatically during normal operation
- `false`: Checkpoints must be created manually (not recommended)

**Example:**

```go
cfg.WAL.Checkpoint.AutoCreate = true
```

#### MaxCheckpoints

**Description:** Maximum number of checkpoints to keep (0 = unlimited).

**Default:** `0` (unlimited)

**Range:** 0 - 1000

**Recommended Settings:**

- **Development:** 0 (unlimited, simpler)
- **Production:** 0-10 (keep recent checkpoints for recovery)

**Note:** Old checkpoints are automatically cleaned up when limit is reached.

**Example:**

```go
cfg.WAL.Checkpoint.MaxCheckpoints = 5 // Keep last 5 checkpoints
```

### WAL Trimming Configuration

**Description:** Controls automatic cleanup of old WAL segments after checkpoints.

#### TrimAfterCheckpoint

**Description:** Whether to automatically trim WAL segments that are before the last checkpoint.

**Default:** `true`

**Options:**

- `true`: Old segments trimmed automatically after checkpoint
- `false`: WAL segments persist until manual cleanup

**Recommended Settings:**

- **Production:** `true` (prevents unbounded disk usage)
- **Debugging:** `false` (preserve WAL history for analysis)

**Trade-offs:**

- **true:** Reduces disk usage, but requires checkpoint coordination
- **false:** Preserves full WAL history, but disk usage grows unbounded

**Example:**

```go
cfg.WAL.TrimAfterCheckpoint = true
```

#### KeepSegments

**Description:** Number of WAL segments to keep before the checkpoint (safety margin).

**Default:** `2`

**Range:** 0 - 100

**Recommended Settings:**

- **Production:** 2-5 (safety margin for recovery)
- **Development:** 1 (minimal safety margin)

**Trade-offs:**

- **Lower:** Less disk usage, but smaller safety margin
- **Higher:** More disk usage, but larger safety margin for recovery

**Note:** The active WAL segment is always kept, regardless of this setting.

**Example:**

```go
cfg.WAL.KeepSegments = 3 // Keep 3 segments before checkpoint
```

---

## Scheduler Configuration

### QueueDepth

**Description:** Maximum pending requests per database queue.

**Default:** `100`

**Range:** 10 - 10000

**Recommended Settings:**

- **Low Latency:** 10 (small queues, fast backpressure)
- **Balanced:** 100 (default)
- **High Throughput:** 1000 (large queues, accept bursts)

**Trade-offs:**

- **Lower:** Faster backpressure signaling, but more queue-full errors
- **Higher:** Fewer queue-full errors, but higher tail latency

**Example:**

```go
cfg.Sched.QueueDepth = 1000 // Accept bursts of 1000 requests
```

### RoundRobinDBs

**Description:** Whether to use round-robin scheduling across databases.

**Default:** `true`

**Options:**

- `true`: Fair round-robin across all databases
- `false`: Process databases in order they're created (not recommended)

**Recommended Settings:**

- **Production:** `true` (fairness is important)
- **Testing:** `false` (deterministic ordering for reproducibility)

**Example:**

```go
cfg.Sched.RoundRobinDBs = false // Deterministic ordering (testing only)
```

---

## Database Configuration

### CompactionSizeThresholdMB

**Description:** Data file size triggering automatic compaction.

**Default:** `100` (100 MB)

**Range:** 10 - 10240 MB

**Recommended Settings:**

- **Small Databases:** 10 MB (frequent compaction, smaller files)
- **Balanced:** 100 MB (default)
- **Large Databases:** 500 MB (reduce compaction overhead)

**Trade-offs:**

- **Lower:** More frequent compaction, smaller data files
- **Higher:** Less compaction overhead, but larger data files

**Example:**

```go
cfg.DB.CompactionSizeThresholdMB = 500 // Compact at 500 MB
```

### CompactionTombstoneRatio

**Description:** Ratio of tombstones triggering automatic compaction.

**Default:** `0.3` (30%)

**Range:** 0.1 - 0.9 (10% - 90%)

**Recommended Settings:**

- **High Delete Volume:** 0.2 (compact when 20% are tombstones)
- **Balanced:** 0.3 (default)
- **Low Delete Volume:** 0.7 (compact only when many tombstones)

**Trade-offs:**

- **Lower:** Frequent compaction, less dead data on disk
- **Higher:** Less compaction overhead, but more wasted space

**Example:**

```go
cfg.DB.CompactionTombstoneRatio = 0.2 // Compact at 20% tombstones
```

### MaxOpenDBs

**Description:** Maximum number of concurrently open databases.

**Default:** `100`

**Range:** 1 - 10000

**Recommended Settings:**

- **Small Applications:** 10 (few databases)
- **Balanced:** 100 (default)
- **Large Applications:** 1000 (many databases)

**Trade-offs:**

- **Lower:** Fewer file descriptors, faster startup
- **Higher:** More concurrent databases, more resource usage

**Example:**

```go
cfg.DB.MaxOpenDBs = 1000 // Allow up to 1000 databases
```

### IdleTimeout

**Description:** Duration before closing idle databases.

**Default:** `5 * time.Minute` (5 minutes)

**Range:** 1 _ time.Second - 1 _ time.Hour

**Recommended Settings:**

- **Frequently Accessed Databases:** 1 \* time.Hour (keep open)
- **Infrequently Accessed:** 1 \* time.Minute (close quickly)
- **Balanced:** 5 \* time.Minute (default)

**Trade-offs:**

- **Lower:** Faster resource cleanup, more reopens
- **Higher:** Fewer reopens, more memory usage

**Example:**

```go
cfg.DB.IdleTimeout = 30 * time.Minute // Keep open for 30 minutes
```

---

## LogicalDB (v0.4) Defaults and Limits

When using partitioned mode (`PartitionCount > 1`), the following defaults and limits apply.

### DefaultLogicalDBConfig

**Defaults (Phase D.7):**

| Option         | Default                | Rationale                                            |
| -------------- | ---------------------- | ---------------------------------------------------- |
| PartitionCount | `2 × runtime.NumCPU()` | Write parallelism; balances throughput vs contention |
| WorkerCount    | `runtime.NumCPU()`     | Execution concurrency; matches scaling matrix        |
| QueueSize      | `1024`                 | Backpressure buffer; tested value                    |

**Tuning:**

- **Higher write throughput:** Increase `PartitionCount` (e.g. 4× CPU); stay ≤ `Query.MaxPartitionsPerDB`.
- **Lower latency:** Keep `WorkerCount` at NumCPU; increase `QueueSize` for bursts.
- **Many small DBs:** Use lower `PartitionCount` (e.g. 2) to reduce WAL files.

### Query and WAL Limits (Phase D.8)

| Limit                | Default | Description                                           |
| -------------------- | ------- | ----------------------------------------------------- |
| MaxPartitionsPerDB   | 256     | Maximum partitions per LogicalDB                      |
| MaxConcurrentQueries | 100     | Concurrent query semaphore per DB                     |
| QueryTimeout         | 30s     | Query execution timeout (context)                     |
| MaxQueryMemoryMB     | 100     | Max bytes scanned per query (MB)                      |
| MaxQueryLimit        | 10000   | Max rows per query; client `limit` is clamped to this |
| MaxWALSizePerDB      | 10 GB   | Total WAL size cap per DB (bytes)                     |

**Errors:** Exceeding limits returns: `ErrTooManyPartitions`, `ErrTooManyConcurrentQueries`, `ErrQueryTimeout`, `ErrQueryMemoryLimit`, `ErrWALSizeLimit`.

**IPC protocol:** Maximum operations per request frame is capped at `MaxOpsPerFrame` (65535) to prevent DoS/OOM.

---

## IPC Configuration

**Database name validation:** Database names (used when creating or opening a DB) are validated to prevent path traversal. Allowed: non-empty, valid UTF-8, at most 64 bytes. Rejected: names containing `/`, `\`, `..`, or null. Invalid names are rejected at catalog create and at IPC OpenDB.

### SocketPath

**Description:** Unix domain socket path for client connections.

**Default:** `"/tmp/docdb.sock"`

**Recommended Settings:**

- **Linux/Mac:** `/tmp/docdb.sock` (default)
- **Production:** `/var/run/docdb.sock` (persistent across reboots)
- **Development:** `/tmp/docdb-dev.sock` (isolated)

**Example:**

```go
cfg.IPC.SocketPath = "/var/run/docdb.sock"
```

### EnableTCP

**Description:** Whether to enable TCP/IP support (not implemented in v0).

**Default:** `false`

**Note:** TCP support is explicitly out of scope for v0.

### DebugMode

**Description:** When enabled, the IPC handler logs request/response payload previews (e.g. first 100 bytes of payloads, full query payload for CmdQuery) for debugging.

**Default:** `false`

**Security:** Debug mode must **not** be used in production when handling sensitive data. When enabled, request and response payloads (or previews) may be written to logs and could include PII or secrets. Log aggregation or shared log access increases exposure. Keep DebugMode off in production.

---

## Healing Configuration

**Description:** Controls automatic document healing and corruption detection.

**Default Configuration:**

```go
Healing: HealingConfig{
    Enabled:          true,              // Enable automatic healing
    Interval:         1 * time.Hour,    // Periodic health scan interval
    OnReadCorruption: true,              // Trigger healing on corruption detection during read
    MaxBatchSize:     100,               // Maximum documents to heal in one batch
}
```

### Enabled

**Description:** Whether to enable the automatic healing service.

**Default:** `true`

**Options:**

- `true`: Background healing service runs periodic health scans
- `false`: Healing service disabled (manual healing still available)

**Recommended Settings:**

- **Production:** `true` (automatic corruption recovery)
- **Development:** `false` (manual control, less overhead)

**Example:**

```go
cfg.Healing.Enabled = true
```

### Interval

**Description:** Time interval between periodic health scans.

**Default:** `1 * time.Hour`

**Range:** 1 _ time.Minute - 24 _ time.Hour

**Recommended Settings:**

- **Production:** 1 hour (balanced detection vs overhead)
- **High-Value Data:** 15-30 minutes (faster detection)
- **Low-Priority:** 6-24 hours (less overhead)

**Trade-offs:**

- **Lower:** Faster corruption detection, more CPU overhead
- **Higher:** Slower detection, less CPU overhead

**Example:**

```go
cfg.Healing.Interval = 30 * time.Minute // Scan every 30 minutes
```

### OnReadCorruption

**Description:** Whether to trigger healing immediately when corruption is detected during read operations.

**Default:** `true`

**Options:**

- `true`: Heal document immediately on corruption detection
- `false`: Only heal during periodic scans

**Recommended Settings:**

- **Production:** `true` (immediate recovery)
- **High-Load:** `false` (defer to periodic scans to avoid read latency)

**Trade-offs:**

- **true:** Immediate recovery, but may add latency to read operations
- **false:** No read latency impact, but corruption persists until next scan

**Example:**

```go
cfg.Healing.OnReadCorruption = true
```

### MaxBatchSize

**Description:** Maximum number of documents to heal in a single batch operation.

**Default:** `100`

**Range:** 1 - 10000

**Recommended Settings:**

- **Small Databases:** 10-50 (smaller batches)
- **Large Databases:** 100-500 (larger batches for efficiency)
- **Memory-Constrained:** 10-50 (lower memory usage)

**Trade-offs:**

- **Lower:** Less memory usage, more operations
- **Higher:** More efficient, but higher memory usage

**Example:**

```go
cfg.Healing.MaxBatchSize = 200 // Heal up to 200 documents per batch
```

---

## Command-Line Flags

### Available Flags

```bash
./docdb --help
```

| Flag              | Description                     | Default         |
| ----------------- | ------------------------------- | --------------- |
| `--data-dir`      | Directory for data files        | ./data          |
| `--socket`        | Unix socket path                | /tmp/docdb.sock |
| `--memory-global` | Global memory limit in MB       | 1024            |
| `--memory-per-db` | Per-database memory limit in MB | 256             |
| `--wal-dir`       | Directory for WAL files         | ./data/wal      |
| `--wal-fsync`     | Enable fsync on WAL writes      | true            |
| `--wal-max-size`  | WAL max file size in MB         | 100             |
| `--queue-depth`   | Request queue depth             | 100             |
| `--compact-size`  | Compaction size threshold in MB | 100             |
| `--compact-ratio` | Compaction tombstone ratio      | 0.3             |

### Examples

**Development:**

```bash
./docdb \
  --data-dir ./dev-data \
  --socket /tmp/docdb-dev.sock \
  --memory-global 256 \
  --memory-per-db 64
```

**Production:**

```bash
./docdb \
  --data-dir /var/lib/docdb \
  --socket /var/run/docdb.sock \
  --memory-global 4096 \
  --memory-per-db 512 \
  --wal-fsync true \
  --queue-depth 1000
```

**Benchmarks:**

```bash
./docdb \
  --data-dir /tmp/docdb-bench \
  --memory-global 8192 \
  --memory-per-db 2048 \
  --wal-fsync false \
  --compact-size 1000
```

---

## Recommended Configurations

### Development Configuration

```go
cfg := config.DefaultConfig()

// Smaller memory for development
cfg.Memory.GlobalCapacityMB = 256
cfg.Memory.PerDBLimitMB = 64

// Faster WAL for quick restarts
cfg.WAL.MaxFileSizeMB = 10

// Disable fsync for speed (acceptable for dev)
cfg.WAL.FsyncOnCommit = false

// Smaller queues for faster backpressure
cfg.Sched.QueueDepth = 10

// Frequent compaction
cfg.DB.CompactionSizeThresholdMB = 10
```

**Use Case:** Local development, testing, debugging

**Pros:** Fast startup, low memory usage, quick feedback loops
**Cons:** Not production-ready, data loss on crash (no fsync)

---

### Small Production Configuration

```go
cfg := config.DefaultConfig()

// Moderate memory limits
cfg.Memory.GlobalCapacityMB = 1024 // 1 GB
cfg.Memory.PerDBLimitMB = 256    // 256 MB per DB

// Standard WAL settings
cfg.WAL.MaxFileSizeMB = 100
cfg.WAL.FsyncOnCommit = true       // Durability required

// Balanced queue depth
cfg.Sched.QueueDepth = 100

// Standard compaction
cfg.DB.CompactionSizeThresholdMB = 100
cfg.DB.CompactionTombstoneRatio = 0.3
```

**Use Case:** Small production deployment, single server

**Pros:** Durable, balanced performance, reasonable memory usage
**Cons:** May need tuning for specific workload

---

### Large Production Configuration

```go
cfg := config.DefaultConfig()

// Large memory limits
cfg.Memory.GlobalCapacityMB = 4096 // 4 GB
cfg.Memory.PerDBLimitMB = 1024    // 1 GB per DB

// Large WAL for high write volume
cfg.WAL.MaxFileSizeMB = 500
cfg.WAL.FsyncOnCommit = true

// Large queues for high concurrency
cfg.Sched.QueueDepth = 1000

// Less frequent compaction
cfg.DB.CompactionSizeThresholdMB = 500
cfg.DB.CompactionTombstoneRatio = 0.5
```

**Use Case:** Large production deployment, high write volume

**Pros:** Handles high throughput, good resource utilization
**Cons:** Higher memory usage, slower recovery

---

## Performance Tuning

### 1. Memory Tuning

**Monitor Usage:**

```go
stats, _ := cli.Stats()
usedPercent := float64(stats.MemoryUsed) / float64(stats.MemoryCapacity) * 100

if usedPercent > 80 {
    log.Warn("Memory at %.0f%% capacity", usedPercent)
}
```

**Tune for Workload:**

- **Many Small Documents:** Increase buffer pool small sizes
- **Few Large Documents:** Increase PerDBLimitMB
- **High Concurrency:** Increase GlobalCapacityMB

### 2. WAL Tuning

**Durability vs Performance:**

```go
// Maximum durability (slowest)
cfg.WAL.FsyncOnCommit = true

// Maximum performance (least durable)
cfg.WAL.FsyncOnCommit = false

// Balanced (if OS supports)
cfg.WAL.FsyncOnCommit = true  // But mount with noatime
```

**Rotation Frequency:**

```go
// Frequent rotation (fast recovery)
cfg.WAL.MaxFileSizeMB = 10

// Infrequent rotation (less overhead)
cfg.WAL.MaxFileSizeMB = 1000
```

### 3. Scheduler Tuning

**Latency vs Throughput:**

```go
// Low latency (small queues)
cfg.Sched.QueueDepth = 10

// High throughput (large queues)
cfg.Sched.QueueDepth = 1000

// Balanced
cfg.Sched.QueueDepth = 100
```

### 4. Database Tuning

**Compaction Strategy:**

```go
// Aggressive compaction (more CPU, less disk)
cfg.DB.CompactionSizeThresholdMB = 10
cfg.DB.CompactionTombstoneRatio = 0.2

// Conservative compaction (less CPU, more disk)
cfg.DB.CompactionSizeThresholdMB = 500
cfg.DB.CompactionTombstoneRatio = 0.7
```

**Idle Timeout:**

```go
// Close quickly (save resources)
cfg.DB.IdleTimeout = 1 * time.Minute

// Keep open (avoid reopens)
cfg.DB.IdleTimeout = 30 * time.Minute
```

### 5. Storage Tuning

**File System:**

- Use SSD for data directory if possible
- Mount with `noatime` (no access time updates)
- Use ext4, XFS, or ZFS (journaling file systems)

**Directory Layout:**

```
/fastssd/docdb/
├── data/           # Data files
└── wal/            # WAL files (same volume as data)
```

---

## Troubleshooting

### Memory Issues

**Symptom:** "memory limit exceeded" errors

**Solutions:**

1. Increase `Memory.GlobalCapacityMB`
2. Increase `Memory.PerDBLimitMB`
3. Reduce document size
4. Implement client-side caching

### WAL Issues

**Symptom:** Slow writes

**Solutions:**

1. Set `WAL.FsyncOnCommit = false` (testing only)
2. Move WAL to faster storage
3. Increase `WAL.MaxFileSizeMB` (less rotation)

**Symptom:** Slow recovery

**Solutions:**

1. Decrease `WAL.MaxFileSizeMB` (more frequent rotation)
2. Enable `WAL.FsyncOnCommit` (faster recovery with less corruption)
3. Reduce number of WAL records (batch operations)

### Performance Issues

**Symptom:** High latency

**Solutions:**

1. Reduce `Sched.QueueDepth` (faster backpressure)
2. Increase `Memory.GlobalCapacityMB` (more cache)
3. Use batch operations
4. Enable connection pooling in client

**Symptom:** Low throughput

**Solutions:**

1. Increase `Sched.QueueDepth` (accept bursts)
2. Disable `WAL.FsyncOnCommit` (testing only)
3. Increase buffer pool sizes
4. Move to faster storage

---

## Next Steps

- Read [usage.md](usage.md) for code examples
- See [architecture.md](architecture.md) for system design
- Review [transactions.md](transactions.md) for transaction behavior
- Check [performance_tuning.md](performance_tuning.md) for more optimization tips

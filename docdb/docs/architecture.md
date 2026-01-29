# DocDB Architecture

This document describes DocDB's system architecture, component interactions, and design decisions.

## Table of Contents

1. [System Overview](#system-overview)
2. [Component Architecture](#component-architecture)
3. [Data Flow](#data-flow)
4. [Concurrency Model](#concurrency-model)
5. [Storage Architecture](#storage-architecture)
6. [Transaction Model](#transaction-model)
7. [Collections](#collections-v02) (v0.2)
8. [Path-Based Updates](#path-based-updates-v02) (v0.2)
9. [Healing Service Architecture](#healing-service-architecture-v02) (v0.2)
10. [WAL Lifecycle](#wal-lifecycle-v02) (v0.2)
11. [Error Classification and Retry System](#error-classification-and-retry-system-v02) (v0.2)
12. [Observability and Metrics](#observability-and-metrics-v02) (v0.2)
13. [Design Decisions](#design-decisions)
14. [Alternatives Considered](#alternatives-considered)

---

## System Overview

DocDB is a file-based, ACID document database written in Go. It supports multiple logical databases in a single runtime, avoiding SQLite's single-writer collapse through sharded indexing and per-database WALs.

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Client Applications                 │
│         (Go, TypeScript, Bun, Other)             │
└──────────────────────┬──────────────────────────────┘
                       │ IPC (Unix Socket)
                       ▼
┌─────────────────────────────────────────────────────────┐
│                 DocDB Pool (Go)                    │
│  ┌──────────────────────────────────────────────┐ │
│  │         Scheduler (Round-Robin)            │ │
│  │   ┌──────┐  ┌──────┐  ┌──────┐      │ │
│  │   │ DB 1 │  │ DB 2 │  │ DB N │      │ │
│  │   └──┬───┘  └──┬───┘  └──┬───┘      │ │
│  └───────┼───────────┼───────────┼────────────┘ │
│          │           │           │               │
│          ▼           ▼           ▼               │
│  ┌──────────────────────────────────────┐        │
│  │       Worker Pool (4 workers)       │        │
│  └──────────────────────────────────────┘        │
└────────────────────────────────────────────────────┘
```

### Key Characteristics

- **Single Process**: All databases run in one Go process
- **Multi-Database**: Support for multiple isolated logical databases
- **Concurrent**: Sharded index allows concurrent reads
- **Durable**: WAL ensures data survives crashes
- **Bounded Memory**: Configurable memory limits per database and globally

---

## Component Architecture

### 1. DocDB Pool

**Purpose:** Manages multiple logical databases in a single runtime.

**Responsibilities:**

- Database lifecycle (create, open, close, delete)
- Request scheduling across databases
- Memory management (global + per-DB)
- Catalog persistence (metadata)

**Key Components:**

- `Scheduler`: Round-robin request distribution
- `Catalog`: Database metadata storage
- `MemoryCaps`: Memory limit tracking
- `BufferPool`: Efficient buffer allocation

**Thread Safety:** Pool operations are thread-safe.

---

### 2. LogicalDB

**Purpose:** Manages a single logical database instance.

**Responsibilities:**

- Document CRUD operations
- Collection management (v0.2)
- Path-based updates (patch operations) (v0.2)
- Transaction management
- Index management
- WAL writing
- Data file operations
- Crash recovery (WAL replay)

**Key Components:**

- `Index`: Sharded in-memory index
- `CollectionRegistry`: Collection management (v0.2)
- `MVCC`: Multi-version concurrency control
- `TransactionManager`: Transaction lifecycle
- `DataFile`: Append-only file operations
- `WALWriter`: WAL record writing
- Path utilities: JSON path parsing and manipulation (v0.2)

**Thread Safety:** All public methods are thread-safe (RWMutex).

---

### 3. Sharded Index

**Purpose:** Fast document lookups with reduced lock contention.

**Design:**

```
┌───────────────────────────────────────────────────┐
│              Sharded Index (256 shards)        │
│  ┌──────┐  ┌──────┐  ┌──────┐      │
│  │Shard 0│  │Shard 1│  │Shard N│      │
│  │Hash   │  │Hash   │  │Hash   │      │
│  │mod 256│  │mod 256│  │mod 256│      │
│  └───┬──┘  └───┬──┘  └───┬───┘      │
└───────┼───────────┼───────────┼───────────┘
        │           │           │
        ▼           ▼           ▼
   Document Versions (per shard)
```

**Sharding Strategy:**

- Formula: `shard_id = doc_id % num_shards`
- Default: 256 shards
- Each shard: Protected by own RWMutex
- Benefit: Concurrent reads across different shards

**Visibility Rules:**

- Version is visible if: `created_tx_id <= snapshot_tx_id`
- Version is hidden if: `deleted_tx_id <= snapshot_tx_id`

---

### 4. WAL (Write-Ahead Log)

**Purpose:** Durability and crash recovery.

**Design:**

```
┌───────────────────────────────────────────────────┐
│                 WAL Writer                    │
│  ┌──────────────────────────────────────┐     │
│  │  Append-Only Log (Sequential)     │     │
│  │  ┌─────┬─────┬─────┬─────┐     │
│  │  │Rec1 │Rec2 │Rec3 │RecN │     │
│  │  └─────┴─────┴─────┴─────┘     │
│  │  CRC32 Checksum per Record        │     │
│  └──────────────────────────────────────┘     │
│         Optional fsync per Record            │
└───────────────────────────────────────────────┘
```

**Record Format:**

```
[8 bytes: record_len]
[8 bytes: tx_id]
[8 bytes: db_id]
[1 byte: op_type]
[8 bytes: doc_id]
[4 bytes: payload_len]
[N bytes: payload]
[4 bytes: crc32]
```

**Recovery Process:**

1. Open WAL file
2. Read records sequentially
3. Validate CRC32 for each record
4. Stop at first corrupt record
5. Truncate WAL to last valid offset
6. Write payloads to data file
7. Rebuild in-memory index

**Scope:** Per-database WAL (not global).

---

### 5. Data File

**Purpose:** Append-only storage for document payloads.

**Design:**

```
┌───────────────────────────────────────────────────┐
│              Data File (<db>.data)           │
│  ┌──────┬──────┬──────┬──────┐         │
│  │Doc1 │Doc2 │Doc3 │DocN │         │
│  │Header│Header│Header│Header│         │
│  │Payload│Payload│Payload│Payload│        │
│  └──────┴──────┴──────┴──────┘         │
│        Append-Only (No Overwrites)          │
└───────────────────────────────────────────────┘
```

**Record Format:**

```
[4 bytes: payload_len]
[N bytes: payload]
```

**Characteristics:**

- Append-only: Never overwrite existing data
- Offset-addressed: Documents referenced by byte offset
- No in-place mutations: Updates append new versions

---

### 6. Transaction Manager

**Purpose:** Manage short-lived atomic transactions.

**Design:**

```
┌───────────────────────────────────────────────────┐
│         Transaction Manager                    │
│  ┌──────────────────────────────────────┐     │
│  │  Transaction Lifecycle:              │     │
│  │  ┌────────────────────────────┐      │     │
│  │  │ Begin → Buffer → Commit/┃│      │     │
│  │  │        Rollback          │      │     │
│  │  └────────────────────────────┘      │     │
│  │  WAL Records Generated          │     │     │
│  └──────────────────────────────────────┘     │
└───────────────────────────────────────────────┘
```

**ACID Properties:**

- **Atomicity:** All operations succeed or all fail (via rollback)
- **Consistency:** Validated before commit
- **Isolation:** Snapshot reads (MVCC-lite)
- **Durability:** WAL writes (with optional fsync)

---

### 7. IPC Server

**Purpose:** Client communication over Unix domain socket.

**Design:**

```
┌───────────────────────────────────────────────────┐
│              IPC Server                       │
│  ┌──────────────────────────────────────┐     │
│  │  Unix Domain Socket Listener        │     │
│  │  ┌────────────────────────────┐    │     │
│  │  │  Request Frame Parser      │    │     │
│  │  │  Command Handler          │    │     │
│  │  │  Response Serializer       │    │     │
│  │  │  Error Propagation        │    │     │
│  │  └────────────────────────────┘    │     │
│  └──────────────────────────────────────┘     │
└───────────────────────────────────────────────┘
```

**Protocol:**

- Binary frames over Unix socket
- Length-prefixed messages
- Batch operation support
- Request/response pattern

---

## Data Flow

### 1. Document Creation Flow

```
Client Request
    │
    ▼
IPC Server (Receive Frame)
    │
    ▼
Scheduler (Queue Request)
    │
    ▼
Worker (Execute Request)
    │
    ├─► Check for duplicate
    ├─► Allocate memory
    ├─► Write payload to data file (append)
    ├─► Write WAL record (with fsync)
    └─► Update index (make visible)
    │
    ▼
Response (Success/Error)
    │
    ▼
Client (Receive Response)
```

### 2. Document Read Flow

```
Client Request
    │
    ▼
IPC Server (Receive Frame)
    │
    ▼
Scheduler (Queue Request)
    │
    ▼
Worker (Execute Request)
    │
    ├─► Lookup in index (RWMutex read lock)
    ├─► Get document version
    ├─► Check visibility (snapshot)
    └─► Read payload from data file (offset)
    │
    ▼
Response (Document Data)
    │
    ▼
Client (Receive Data)
```

### 3. Crash Recovery Flow

```
Server Startup
    │
    ▼
Load Catalog (Database Metadata)
    │
    ▼
For Each Database:
    │
    ├─► Open Data File
    ├─► Open WAL File
    ├─► Read WAL Records (Sequential)
    ├─► Validate CRC32
    ├─► Stop at First Error
    ├─► Write Payloads to Data File
    ├─► Rebuild Index (with real offsets)
    └─► Set MVCC TxID (max + 1)
    │
    ▼
Ready to Serve Requests
```

---

## Concurrency Model

### Locking Strategy

```
┌───────────────────────────────────────────────────┐
│         LogicalDB (RWMutex)               │
│  ┌──────────────────────────────────────┐     │
│  │  Write Operations (Exclusive)       │     │
│  │  Create, Update, Delete, Commit    │     │
│  │  Acquires Write Lock             │     │
│  │  ┌───────────────────────────┐     │     │
│  │  │  WAL + Index Atomic      │     │     │
│  │  └───────────────────────────┘     │     │
│  └──────────────────────────────────────┘     │
│  ┌──────────────────────────────────────┐     │
│  │  Read Operations (Shared)          │     │
│  │  Read, Stats                      │     │
│  │  Acquires Read Lock              │     │     │
│  │  Multiple Readers Allowed         │     │     │
│  └──────────────────────────────────────┘     │
└───────────────────────────────────────────────┘
```

### Index-Level Concurrency

```
┌───────────────────────────────────────────────────┐
│         Sharded Index                      │
│  ┌──────────────────────────────────────┐     │
│  │  Shard 0 (RWMutex)              │     │
│  │  ┌──────┐  ┌──────┐            │     │
│  │  │Reads  │  │Writes│            │     │
│  │  │Share  │  │Exclusive│          │     │
│  │  └──────┘  └──────┘            │     │
│  └──────────────────────────────────────┘     │
│  Shard 1 (RWMutex)              │     │
│  Shard N (RWMutex)              │     │
└───────────────────────────────────────────────┘
```

**Benefits:**

- Concurrent reads across different shards (no blocking)
- Concurrent reads within shard (shared lock)
- Serialized writes per shard (exclusive lock)
- Reduced contention vs single global lock

---

## Storage Architecture

### File Layout

```
<data_dir>/
├── data/
│   ├── <dbname1>.data      # Data file (append-only)
│   ├── <dbname2>.data
│   └── <dbnameN>.data
├── wal/
│   ├── <dbname1>.wal       # WAL file (append-only)
│   ├── <dbname2>.wal
│   └── <dbnameN>.wal
└── .catalog                # Database metadata
```

### Append-Only Model

**Why Append-Only:**

- **Simplicity:** No complex page management
- **Durability:** Crash recovery is straightforward
- **Concurrency:** No need for locking during writes
- **Atomicity:** Append is atomic at filesystem level

**Trade-offs:**

- **Space:** Updates create new versions, old data remains until compaction
- **Reads:** Require offset tracking and potential multiple disk seeks
- **Writes:** May need to skip over deleted data during compaction

**Compaction Strategy:**

1. Create `.compact` file
2. Copy only live documents
3. Update index with new offsets
4. Atomic rename (`.compact` → `.data`)
5. Cleanup old data file

---

## Transaction Model

### MVCC-Lite

**Key Concepts:**

- **Transaction IDs:** Monotonically increasing integers
- **Snapshots:** Point-in-time view of database
- **Versions:** Each document has multiple versions over time

**Version Structure:**

```
Document Version:
  ├─ Document ID
  ├─ Created Transaction ID
  ├─ Deleted Transaction ID (optional)
  ├─ Data File Offset
  └─ Payload Length
```

**Visibility Rules:**

```
Version is visible if:
  created_tx_id <= snapshot_tx_id
  AND (deleted_tx_id == nil OR deleted_tx_id > snapshot_tx_id)
```

### Transaction Lifecycle

```
Begin:
  1. Assign Transaction ID (monotonically increasing)
  2. Capture Snapshot (current_tx_id - 1)
  3. Create Transaction object
  4. Buffer operations in Transaction

Execute (within Transaction):
  1. Perform operations in memory
  2. Generate WAL records
  3. Do NOT update index yet

Commit:
  1. Write all WAL records to WAL file
  2. Fsync WAL (if enabled)
  3. Update index (make transactions visible)
  4. Release transaction locks

Rollback:
  1. Discard buffered operations
  2. Do NOT write to WAL
  3. Do NOT update index
  4. Release transaction locks
```

### Concurrency Behavior

**"Last Commit Wins" Model:**

- No conflict detection on concurrent updates
- Both versions are written to WAL
- Index shows last committed version
- Non-deterministic across restarts (acceptable for v0)

**Safety:**

- ACID properties maintained
- No data loss (both versions in WAL)
- Reads see consistent snapshot
- No silent failures

---

## Design Decisions

### 1. Per-Database WAL (vs Global)

**Decision:** One WAL file per database

**Rationale:**

- **Simpler Isolation:** Each database has independent recovery
- **Easier Cleanup:** Can delete WAL without affecting other DBs
- **Parallel Recovery:** Can recover databases concurrently

**Trade-offs:**

- **Harder Scheduling:** No global ordering across databases
- **Fairness:** Requires pool-level coordination
- **Cross-DB Operations:** Not supported (intentional for v0)

---

### 2. Sharded Index (vs Global Lock)

**Decision:** 256-shard hash-based index

**Rationale:**

- **Reduced Contention:** Locks at shard level, not global
- **Scalability:** Can handle concurrent reads/writes better
- **Simple Sharding:** Hash modulo is O(1)

**Trade-offs:**

- **Memory Overhead:** 256 RWMutex instances
- **Cache Locality:** Related documents may be in different shards
- **Rebalancing:** Not supported (fixed shard count)

---

### 3. Snapshot Isolation (vs Serializable)

**Decision:** MVCC-lite with snapshot reads

**Rationale:**

- **Simplicity:** No locking during reads
- **Performance:** Readers never block writers
- **Correctness:** Sufficient for v0 scope

**Trade-offs:**

- **No Repeatable Reads:** Different reads may see different states
- **No Serialization:** Concurrent updates may interleave
- **Phantom Reads:** Possible if scans were supported

---

### 4. Append-Only Storage (vs In-Place Updates)

**Decision:** Append-only data file with versioned documents

**Rationale:**

- **Simplicity:** No complex page management
- **Durability:** Crash recovery is straightforward
- **Concurrency:** No need for locking during writes

**Trade-offs:**

- **Space:** Old versions remain until compaction
- **Reads:** Need offset tracking, multiple versions
- **Writes:** May need to skip deleted data during compaction

---

### 5. Round-Robin Scheduling (vs Priority Queue)

**Decision:** Fair round-robin across databases

**Rationale:**

- **Fairness:** All databases get equal service
- **Simplicity:** No priority configuration needed
- **Starvation Prevention:** No database is starved

**Trade-offs:**

- **No Priority:** Critical requests not prioritized
- **No Weighting:** Important databases not favored
- **Simple:** Easier to reason about, less configurable

---

## Alternatives Considered

### 1. Global WAL vs Per-Database WAL

**Global WAL (Rejected):**

- **Pros:** Global ordering, cross-DB operations easier
- **Cons:** Complex recovery, single point of failure, harder cleanup

**Per-Database WAL (Chosen):**

- **Pros:** Simple recovery, independent databases, parallel recovery possible
- **Cons:** Harder scheduling, no global ordering

---

### 2. B+ Tree Index vs Sharded Hash Map

**B+ Tree (Rejected):**

- **Pros:** Ordered scans, range queries
- **Cons:** Complex implementation, lock contention on hot spots

**Sharded Hash Map (Chosen):**

- **Pros:** Simple, fast lookups, concurrent reads, reduced contention
- **Cons:** No ordered scans, limited to point lookups

---

### 3. LSM Tree vs Append-Only with Compaction

**LSM Tree (Rejected):**

- **Pros:** Efficient writes, built-in compaction
- **Cons:** Complex implementation, read amplification

**Append-Only + Compaction (Chosen):**

- **Pros:** Simple implementation, no read amplification, easy recovery
- **Cons:** Write amplification, periodic compaction needed

---

### 4. Single-Threaded vs Multi-Threaded WAL

**Single-Threaded (Chosen):**

- **Pros:** Simpler implementation, no race conditions
- **Cons:** Write serialization across all operations

**Multi-Threaded (Rejected):**

- **Pros:** Parallel writes, higher throughput
- **Cons:** Complex locking, race conditions, harder to reason about

---

### 5. TCP vs Unix Sockets

**TCP (Rejected for v0):**

- **Pros:** Network access, multi-machine deployment
- **Cons:** More complex, slower (TCP overhead), security concerns

**Unix Sockets (Chosen):**

- **Pros:** Faster, simpler, file-system based security
- **Cons:** Local-only access

---

## Performance Characteristics

### Latency

| Operation | Expected Latency | Factors                  |
| --------- | ---------------- | ------------------------ |
| Create    | 1-10 ms          | Disk I/O, WAL fsync      |
| Read      | 0.1-1 ms         | Memory lookup, disk seek |
| Update    | 1-10 ms          | Disk I/O, WAL fsync      |
| Delete    | 1-10 ms          | Disk I/O, WAL fsync      |
| Batch     | O(n)             | n = operation count      |

### Throughput

| Metric             | Target | Notes                 |
| ------------------ | ------ | --------------------- |
| Concurrent Writers | 10-100 | Limited by disk I/O   |
| Concurrent Readers | 100+   | Limited by memory     |
| Batch Operations   | 1000/s | Depends on batch size |

### Resource Usage

| Resource     | Typical Usage    | Limits               |
| ------------ | ---------------- | -------------------- |
| Memory       | 10-100 MB per DB | Configurable         |
| Disk I/O     | Write-bound      | Depends on workload  |
| CPU          | Low (mostly I/O) | Bursty on compaction |
| File Handles | 1 per open DB    | OS limits apply      |

---

## WAL Lifecycle (v0.2)

DocDB v0.2 implements a complete WAL lifecycle with rotation, checkpoints, and trimming.

### WAL Lifecycle Stages

```
┌─────────────┐
│  Active WAL │
│  (writing)  │
└──────┬──────┘
       │
       │ Size >= MaxSizeMB
       ▼
┌─────────────┐
│   Rotation  │
│  (atomic)   │
└──────┬──────┘
       │
       ▼
┌─────────────┐      ┌─────────────┐
│ WAL Segment │      │ Active WAL  │
│   .wal.1    │      │  (new)      │
└──────┬──────┘      └─────────────┘
       │
       │ Size >= IntervalMB
       ▼
┌─────────────┐
│ Checkpoint  │
│  Created    │
└──────┬──────┘
       │
       │ TrimAfterCheckpoint = true
       ▼
┌─────────────┐
│   Trimming  │
│  (delete    │
│   old segs) │
└─────────────┘
```

### Rotation

**Trigger:** WAL size reaches `MaxFileSizeMB` (default: 64MB)

**Process:**

1. Current WAL file renamed to `dbname.wal.1`
2. New active WAL file created as `dbname.wal`
3. Atomic rename ensures no data loss
4. Old segments numbered sequentially (.wal.1, .wal.2, ...)

**Recovery:** All segments replayed in order during recovery

### Checkpoints

**Purpose:** Bound recovery time by marking safe replay points

**Trigger:** WAL size reaches `Checkpoint.IntervalMB` (default: 64MB)

**Process:**

1. Checkpoint record written to WAL with transaction ID
2. CheckpointManager tracks last checkpoint
3. Recovery starts from last checkpoint (not from beginning)

**Benefits:**

- Bounded recovery time (only replay since last checkpoint)
- Faster startup after crashes
- Enables WAL trimming

### Trimming

**Purpose:** Prevent unbounded disk usage by cleaning old segments

**Trigger:** After checkpoint creation (if `TrimAfterCheckpoint` enabled)

**Process:**

1. Identify segments before last checkpoint
2. Keep `KeepSegments` segments as safety margin (default: 2)
3. Delete older segments atomically
4. Active segment always preserved

**Safety:**

- Only trims segments before checkpoint
- Safety margin prevents accidental data loss
- Atomic deletion ensures consistency

### Recovery with Checkpoints

```
Recovery Process:
1. Find last checkpoint (highest tx_id)
2. Start replay from checkpoint (not from beginning)
3. Replay all segments after checkpoint
4. Apply committed transactions only
5. Rebuild index from replayed records
```

**Recovery Time:** Bounded to size since last checkpoint (not full WAL history)

---

## Error Classification and Retry System (v0.2)

DocDB v0.2 includes intelligent error handling with automatic retry for transient errors.

### Error Flow

```
┌──────────────┐
│  Operation   │
│  (WAL/Data)  │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│   Error      │
│   Occurs     │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Classifier  │
│  Categorizes │
└──────┬───────┘
       │
       ├──> Transient ──> RetryController ──> Retry with backoff
       ├──> Permanent ──> Return Error (no retry)
       ├──> Critical ────> Alert + Return Error
       └──> Validation ──> Return Error (no retry)
```

### Retry Strategy

**Exponential Backoff with Jitter:**

- Initial delay: 10ms
- Max delay: 1s
- Max retries: 5
- Jitter: ±25% random variation

**Formula:**

```
delay = min(initialDelay * 2^attempt, maxDelay) + jitter
```

**Example Retry Sequence:**

- Attempt 1: Immediate
- Attempt 2: ~10ms ± 2.5ms
- Attempt 3: ~20ms ± 5ms
- Attempt 4: ~40ms ± 10ms
- Attempt 5: ~80ms ± 20ms
- Attempt 6: ~160ms ± 40ms (capped at 1s)

### Error Tracking

**Metrics Collected:**

- Error counts by category
- Error rates (errors per second)
- Last occurrence time
- Critical error alerts

**Integration:**

- Errors tracked in `ErrorTracker`
- Metrics exposed via Prometheus exporter
- Critical errors logged immediately

---

## Observability and Metrics (v0.2)

DocDB v0.2 provides comprehensive observability through Prometheus metrics.

### Metrics Architecture

```
┌──────────────┐
│  Operations  │
│  (WAL, Pool) │
└──────┬───────┘
       │ Record
       ▼
┌──────────────┐
│  Exporter    │
│  (Metrics)   │
└──────┬───────┘
       │ Export
       ▼
┌──────────────┐
│  IPC Handler │
│  (CmdMetrics)│
└──────┬───────┘
       │ Prometheus Format
       ▼
┌──────────────┐
│   Client     │
│  (Monitoring)│
└──────────────┘
```

### Available Metrics

**Operation Metrics:**

- `docdb_operations_total{operation, status}` - Total operations
- `docdb_operation_duration_seconds{operation}` - Duration histogram

**System Metrics:**

- `docdb_documents_total` - Live document count
- `docdb_memory_bytes` - Memory usage
- `docdb_wal_size_bytes` - WAL size

**Error Metrics:**

- `docdb_errors_total{category}` - Error counts by category

**Healing Metrics:**

- `docdb_healing_operations_total` - Total healing operations
- `docdb_documents_healed_total` - Total documents healed

### Metrics Collection

**Automatic Collection:**

- Operations tracked during IPC handling
- Errors tracked via ErrorTracker
- Healing tracked during healing operations

**Access:**

- Via IPC `CmdMetrics` command
- Returns Prometheus text format
- Can be scraped by Prometheus server

---

## Evolution Path

### v0 (Current)

- Basic CRUD operations
- Per-database WAL
- Sharded index
- MVCC-lite
- Unix socket IPC

### v0.1 (Completed)

- WAL rotation
- Data file checksums (CRC32)
- Checkpoint-based recovery
- Graceful shutdown
- Document corruption detection

### v0.2 (Completed)

- **Collections** - Collection management and collection-aware operations
- **Path-Based Updates** - JSON patch operations (set, delete, insert)
- Automatic document healing
- WAL trimming after checkpoints
- Error classification and retry
- Prometheus/OpenMetrics metrics
- Enhanced observability

### v0.3 (Current)

- **Ants goroutine pool** - Scheduler workers use ants (recycling, panic recovery, expiry)
- **WAL group commit** - Batched fsync via FsyncConfig (group/interval modes)
- **Fast WAL replay** - Single-pass replay, WriteNoSync during apply, one Sync at end
- **Scheduler fairness** - Queue-depth–aware scheduling (PickNextQueue)
- **Parallel healing** - Ants PoolWithFunc for batch healing
- **Optional IPC pool** - Bounded connection handlers via MaxConnections
- **Metrics** - Ants pool (running/waiting/free) and WAL group-commit stats

### v1 (Future)

- Global WAL
- Full MVCC (serializable)
- Secondary indexes
- Scan operations
- Query language (simple)

---

## References

- [Usage Guide](usage.md) - How to use DocDB
- [Configuration Guide](configuration.md) - Configuration options
- [Failure Modes](failure_modes.md) - Error handling
- [Progress](../PROGRESS.md) - Implementation status
- [On-Disk Format](ondisk_format.md) - Binary formats

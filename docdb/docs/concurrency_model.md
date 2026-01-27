# Concurrency Model

This document describes DocDB's concurrency model, locking strategy, and concurrent operation guarantees.

## Table of Contents

1. [Overview](#overview)
2. [Locking Strategy](#locking-strategy)
3. [Sharded Index Concurrency](#sharded-index-concurrency)
4. [Read-Write Patterns](#read-write-patterns)
5. [Scheduler Concurrency](#scheduler-concurrency)
6. [Memory Safety](#memory-safety)
7. [Performance Characteristics](#performance-characteristics)
8. [Limitations](#limitations)

---

## Overview

DocDB uses a **multi-level locking strategy** to enable concurrent operations while maintaining data integrity:

1. **Database-Level Locking**: RWMutex on LogicalDB
2. **Index-Level Locking**: Per-shard RWMutex (256 shards)
3. **Transaction-Level Locking**: Transaction manager mutex

### Concurrency Goals

- **Concurrent Reads**: Multiple readers can read simultaneously
- **Serialized Writes**: Writes are serialized per database
- **Shard-Level Parallelism**: Reads from different shards don't block
- **No Reader-Writer Starvation**: Readers don't block writers indefinitely

---

## Locking Strategy

### Database-Level Locking

**Purpose:** Protect database state during operations.

**Lock Type:** `sync.RWMutex`

**Lock Acquisition:**

| Operation | Lock Type | Scope |
|-----------|-----------|-------|
| Create    | Write     | Exclusive |
| Update    | Write     | Exclusive |
| Delete    | Write     | Exclusive |
| Commit    | Write     | Exclusive |
| Read      | Read      | Shared |
| Stats     | Read      | Shared |

**Implementation:**
```go
type LogicalDB struct {
    mu sync.RWMutex // Protects all internal state
    // ...
}

// Write operations
func (db *LogicalDB) Create(docID uint64, payload []byte) error {
    db.mu.Lock()         // Exclusive lock
    defer db.mu.Unlock()
    // ... create logic ...
}

// Read operations
func (db *LogicalDB) Read(docID uint64) ([]byte, error) {
    db.mu.RLock()        // Shared lock
    defer db.mu.RUnlock()
    // ... read logic ...
}
```

**Behavior:**
- **Writes**: Exclusive lock, blocks all other operations
- **Reads**: Shared lock, allows concurrent reads
- **Write-Write**: Serialized (one at a time)
- **Read-Read**: Concurrent (multiple readers)
- **Read-Write**: Mutually exclusive (readers wait for writers, writers wait for readers)

### Index-Level Locking

**Purpose:** Enable concurrent reads across different shards.

**Lock Type:** Per-shard `sync.RWMutex` (256 shards)

**Sharding Strategy:**
```go
shardID = docID % numShards  // Default: 256 shards
```

**Implementation:**
```go
type Index struct {
    shards [256]*shard
}

type shard struct {
    mu   sync.RWMutex
    data map[uint64]*DocumentVersion
}

func (idx *Index) Get(docID uint64, snapshotTxID uint64) (*DocumentVersion, bool) {
    shardID := docID % 256
    shard := idx.shards[shardID]
    
    shard.mu.RLock()        // Shared lock on shard
    defer shard.mu.RUnlock()
    
    // ... lookup logic ...
}
```

**Benefits:**
- **Reduced Contention**: Locks at shard level, not global
- **Concurrent Reads**: Different shards don't block each other
- **Scalability**: Can handle concurrent reads better

**Example:**
```
Document 1 (shard 1) ← Read lock on shard 1
Document 2 (shard 2) ← Read lock on shard 2
Document 3 (shard 1) ← Waits for shard 1 lock
Document 4 (shard 3) ← Read lock on shard 3 (concurrent with 1 and 2)
```

### Transaction-Level Locking

**Purpose:** Protect transaction manager state.

**Lock Type:** `sync.RWMutex` on TransactionManager

**Usage:**
- **Begin**: Write lock (assigns transaction ID)
- **Get**: Read lock (lookup transaction)
- **Commit/Rollback**: Write lock (updates transaction state)

---

## Sharded Index Concurrency

### Shard Distribution

**Default Configuration:**
- 256 shards (fixed)
- Hash-based distribution: `shardID = docID % 256`

**Shard Locking:**
```
┌─────────────────────────────────────────┐
│         Sharded Index (256 shards)     │
│  ┌──────────┐  ┌──────────┐           │
│  │ Shard 0  │  │ Shard 1  │  ...       │
│  │ RWMutex  │  │ RWMutex  │           │
│  └────┬─────┘  └────┬─────┘           │
│       │             │                  │
│  Read/Write    Read/Write             │
│  (per shard)   (per shard)            │
└─────────────────────────────────────────┘
```

### Concurrent Read Patterns

**Pattern 1: Different Shards (No Contention)**
```
Thread 1: Read docID = 1  (shard 1) → RLock shard 1
Thread 2: Read docID = 2  (shard 2) → RLock shard 2
Thread 3: Read docID = 3  (shard 3) → RLock shard 3

Result: All three reads proceed concurrently
```

**Pattern 2: Same Shard (Contention)**
```
Thread 1: Read docID = 1  (shard 1) → RLock shard 1
Thread 2: Read docID = 257 (shard 1) → Waits for RLock shard 1
Thread 3: Read docID = 513 (shard 1) → Waits for RLock shard 1

Result: Thread 2 and 3 wait, but can proceed concurrently after Thread 1
```

**Pattern 3: Mixed Read-Write**
```
Thread 1: Read docID = 1   (shard 1) → RLock shard 1
Thread 2: Write docID = 1  (shard 1) → Waits for WLock shard 1
Thread 3: Read docID = 2   (shard 2) → RLock shard 2 (proceeds)

Result: Thread 3 proceeds, Thread 2 waits for Thread 1
```

### Write Serialization

**Database-Level:**
- All writes acquire exclusive database lock
- Writes are serialized globally (one at a time per database)

**Shard-Level:**
- Within write, shard locks acquired per operation
- Multiple shards can be updated in single write (batch)

**Example:**
```
Write 1: Update docID = 1 (shard 1), docID = 2 (shard 2)
  → Database lock (exclusive)
  → Shard 1 lock (exclusive)
  → Shard 2 lock (exclusive)
  → Update both
  → Release locks

Write 2: Update docID = 3 (shard 3)
  → Waits for database lock
  → Acquires database lock
  → Shard 3 lock (exclusive)
  → Update
  → Release locks
```

---

## Read-Write Patterns

### Pattern 1: Multiple Concurrent Readers

**Scenario:** Multiple threads reading different documents.

**Locking:**
```
Thread 1: RLock database → RLock shard 1 → Read → Unlock
Thread 2: RLock database → RLock shard 2 → Read → Unlock
Thread 3: RLock database → RLock shard 3 → Read → Unlock
```

**Result:** All reads proceed concurrently (no blocking).

### Pattern 2: Reader During Write

**Scenario:** Reader attempts to read while write is in progress.

**Locking:**
```
Writer:   Lock database (exclusive) → Write → Unlock
Reader:   Waits for RLock database → RLock → Read → Unlock
```

**Result:** Reader waits for writer to complete, then proceeds.

### Pattern 3: Writer During Read

**Scenario:** Writer attempts to write while readers are active.

**Locking:**
```
Reader 1: RLock database → Read → Unlock
Reader 2: RLock database → Read → Unlock
Writer:   Waits for Lock database → Lock → Write → Unlock
```

**Result:** Writer waits for all readers to complete, then proceeds.

### Pattern 4: Concurrent Writes

**Scenario:** Multiple threads attempting to write simultaneously.

**Locking:**
```
Writer 1: Lock database → Write → Unlock
Writer 2: Waits for Lock database → Lock → Write → Unlock
Writer 3: Waits for Lock database → Lock → Write → Unlock
```

**Result:** Writes are serialized (one at a time).

---

## Scheduler Concurrency

### Request Scheduling

**Architecture:**
```
┌─────────────────────────────────────────┐
│           Scheduler                     │
│  ┌──────────────────────────────────┐   │
│  │  Per-DB Queues (Round-Robin)     │   │
│  │  ┌──────┐  ┌──────┐  ┌──────┐   │   │
│  │  │ DB 1 │  │ DB 2 │  │ DB N │   │   │
│  │  └──┬───┘  └──┬───┘  └──┬───┘   │   │
│  └─────┼──────────┼──────────┼──────┘   │
│        │          │          │          │
│        ▼          ▼          ▼          │
│  ┌──────────────────────────────┐      │
│  │   Worker Pool (4 workers)    │      │
│  └──────────────────────────────┘      │
└─────────────────────────────────────────┘
```

### Round-Robin Scheduling

**Algorithm:**
```go
func (s *Scheduler) worker() {
    for {
        // Round-robin: pick next database
        s.currentDB = (s.currentDB + 1) % len(s.dbIDs)
        dbID := s.dbIDs[s.currentDB]
        queue := s.queues[dbID]
        
        // Process one request from queue
        req := <-queue
        s.pool.handleRequest(req)
    }
}
```

**Fairness:**
- Each database gets equal service time
- No database can monopolize workers
- Starvation prevention built-in

### Worker Pool

**Configuration:**
- Default: 4 workers
- Fixed size (not configurable in v0)
- Each worker processes one request at a time

**Concurrency:**
- Up to 4 requests processed concurrently
- Each request may target different database
- Database-level locking serializes per-database operations

---

## Memory Safety

### Memory Allocation Tracking

**Per-Database Limits:**
- Each database has configurable memory limit
- Memory usage tracked per database
- Allocation failures return `ErrMemoryLimit`

**Thread Safety:**
```go
type Caps struct {
    mu      sync.RWMutex
    dbUsage map[uint64]uint64
    dbLimit map[uint64]uint64
}

func (c *Caps) TryAllocate(dbID uint64, size uint64) bool {
    c.mu.Lock()
    defer c.mu.Unlock()
    // ... allocation logic ...
}
```

### Buffer Pool

**Purpose:** Efficient buffer allocation and reuse.

**Thread Safety:**
- Buffer pool is thread-safe
- Buffers can be allocated concurrently
- Buffers returned to pool safely

---

## Performance Characteristics

### Read Performance

**Best Case:**
- Different shards: Fully concurrent
- No contention: O(1) lookup
- Memory-only: Sub-millisecond latency

**Worst Case:**
- Same shard: Serialized reads
- High contention: Queue behind writes
- Disk I/O: Millisecond latency

**Typical:**
- 100+ concurrent readers (different shards)
- 0.1-1ms latency per read
- Limited by memory bandwidth

### Write Performance

**Characteristics:**
- Serialized per database (one at a time)
- WAL write overhead (disk I/O)
- Index update overhead
- 1-10ms latency per write

**Bottlenecks:**
- Disk I/O (WAL writes)
- Fsync (if enabled)
- Index updates

### Concurrent Operations

**Throughput:**
- Reads: 1000+ ops/sec (concurrent)
- Writes: 100-1000 ops/sec (serialized)
- Mixed: Depends on read/write ratio

**Scalability:**
- Reads scale with number of shards (256)
- Writes scale with number of databases
- Limited by disk I/O and memory

---

## Limitations

### v0 Limitations

1. **Single Writer Per Database:**
   - Writes are serialized globally per database
   - No concurrent writes to same database
   - Acceptable for v0 scope

2. **Fixed Shard Count:**
   - 256 shards (not configurable)
   - No dynamic rebalancing
   - Hash distribution may be uneven

3. **No Lock Timeout:**
   - Locks can block indefinitely
   - No deadlock detection
   - No lock escalation

4. **No Priority Scheduling:**
   - Round-robin only
   - No priority queues
   - No request prioritization

5. **Limited Worker Pool:**
   - Fixed 4 workers
   - Not configurable
   - May be bottleneck for many databases

### Future Enhancements (v0.1+)

- Configurable shard count
- Dynamic rebalancing
- Lock timeouts
- Deadlock detection
- Priority scheduling
- Configurable worker pool size
- Read replicas
- Write batching optimization

---

## Best Practices

### 1. Distribute Document IDs

**Good:**
```go
// Document IDs distributed across shards
docID = hash(userID) % maxDocID
```

**Bad:**
```go
// Sequential IDs → all in same shard
docID = counter++
```

### 2. Batch Operations

**Good:**
```go
// Single transaction with multiple operations
tx := db.Begin()
for i := 0; i < 100; i++ {
    db.CreateInTx(tx, uint64(i), data[i])
}
db.Commit(tx)
```

**Bad:**
```go
// Multiple transactions (more locking overhead)
for i := 0; i < 100; i++ {
    db.Create(uint64(i), data[i])
}
```

### 3. Read-Heavy Workloads

**Good:**
- Use different document IDs (different shards)
- Leverage concurrent reads
- Minimize writes

**Bad:**
- All reads from same shard
- Frequent writes blocking reads
- Hot shards

### 4. Write-Heavy Workloads

**Good:**
- Use multiple databases (parallel writes)
- Batch operations
- Disable fsync if durability not critical

**Bad:**
- Single database (serialized writes)
- Many small transactions
- Fsync on every commit

---

## References

- [Architecture Guide](architecture.md) - System architecture
- [Transactions Guide](transactions.md) - Transaction model
- [Usage Guide](usage.md) - How to use DocDB
- [Configuration Guide](configuration.md) - Configuration options

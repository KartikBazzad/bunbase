# Bundoc Architecture

**Version:** 1.0  
**Last Updated:** February 1, 2026

---

## Overview

Bundoc is a high-performance, ACID-compliant document database written in Go from scratch. It's designed as an embedded database with a focus on:

- **Concurrency**: Non-blocking reads via MVCC
- **Durability**: Write-Ahead Logging with group commits
- **Performance**: Lock-free hot paths, shared global flusher
- **Simplicity**: Embedded library (no network layer)

---

## System Architecture

```
┌──────────────────────────────────────────────────────────┐
│                     Application Layer                     │
├──────────────────────────────────────────────────────────┤
│                    bundoc.Database                        │
│  ┌────────────────┐  ┌─────────────────────────────────┐ │
│  │  Collections   │  │    Transaction Manager          │ │
│  │  - users       │  │  - BeginTransaction()           │ │
│  │  - posts       │  │  - Commit() / Rollback()        │ │
│  │  - ...         │  │  - Isolation Levels (4 types)   │ │
│  └────────────────┘  └─────────────────────────────────┘ │
├──────────────────────────────────────────────────────────┤
│                     MVCC Layer                             │
│  ┌─────────────────────┐  ┌──────────────────────────┐   │
│  │  Version Manager    │  │   Snapshot Manager       │   │
│  │  - Version Chains   │  │   - Snapshot Isolation   │   │
│  │  - Garbage Collect  │  │   - Visibility Rules     │   │
│  └─────────────────────┘  └──────────────────────────┘   │
├──────────────────────────────────────────────────────────┤
│                   Durability Layer                         │
│  ┌──────────────────────────────────────────────────────┐│
│  │              Write-Ahead Log (WAL)                    ││
│  │  - Buffered writes (4KB)                              ││
│  │  - Group commits (100+ txns/fsync)                    ││
│  │  - Shared global flusher (cross-database batching)    ││
│  │  - 64MB segments with auto-rotation                   ││
│  │  - CRC32 checksums                                    ││
│  └──────────────────────────────────────────────────────┘│
├──────────────────────────────────────────────────────────┤
│                    Storage Layer                           │
│  ┌────────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │  Buffer Pool   │  │   B+ Tree    │  │    Pager     │  │
│  │  - LRU Cache   │  │  - Order 64  │  │  - 8KB pages │  │
│  │  - Pin/Unpin   │  │  - Auto Split│  │  - Disk I/O  │  │
│  └────────────────┘  └──────────────┘  └──────────────┘  │
└──────────────────────────────────────────────────────────┘
                            ↓
                    ┌──────────────┐
                    │  Disk Files  │
                    │  - data.db   │
                    │  - wal-*.log │
                    └──────────────┘
```

---

## Core Components

### 1. Database

**File:** `database.go`

The main entry point that coordinates all subsystems.

**Responsibilities:**

- Database lifecycle (Open/Close)
- Collection management (Create/Drop/List)
- Transaction coordination
- Subsystem initialization

**Key Methods:**

```go
func Open(opts *Options) (*Database, error)
func (db *Database) CreateCollection(name string) (*Collection, error)
func (db *Database) BeginTransaction(level IsolationLevel) (*Transaction, error)
func (db *Database) Close() error
```

---

### 2. Collection

**File:** `collection.go`

Represents a logical grouping of documents (like a table).

**Responsibilities:**

- Document CRUD operations
- Index management
- Transaction integration
- Cross-collection reference validation (schema extension `x-bundoc-ref`) and delete-policy execution (restrict / set_null / cascade)

**Key Methods:**

```go
func (c *Collection) Insert(txn *Transaction, doc Document) error
func (c *Collection) FindByID(txn *Transaction, id string) (Document, error)
func (c *Collection) Update(txn *Transaction, id string, doc Document) error
func (c *Collection) Delete(txn *Transaction, id string) error
```

**Cross-Collection References:** The database maintains in-memory registries of reference rules (outbound by source collection, inbound by target collection). On write (Insert/Update/Patch), reference fields are validated against the target collection. On Delete, inbound rules are applied (restrict blocks, set_null patches dependents, cascade deletes dependents) with a visited set to prevent infinite cascade cycles.

---

### 2a. Metadata Manager

**File:** `metadata.go`

Manages the persistence of database schema and index locations.

**Responsibilities:**

- Stores mapping of `Collection Name` -> `Index Field` -> `Root PageID`
- Persists to `system_catalog.json`
- Updates atomically when B-Tree roots split

**Key Methods:**

```go
func (m *MetadataManager) Load() error
func (m *MetadataManager) UpdateCollection(name string, indexes map[string]PageID) error
func (m *MetadataManager) GetCollection(name string) (CollectionMeta, bool)
```

---

### 3. Storage Layer

#### Buffer Pool

**File:** `internal/storage/buffer_pool.go`

LRU cache for 8KB pages to minimize disk I/O.

**Features:**

- Pin/unpin mechanism prevents eviction of in-use pages
- Thread-safe with RWMutex
- Configurable capacity (default: 256 pages = 2MB)

#### B+ Tree Index

**File:** `internal/storage/index.go`

Ordered index for fast lookups and range scans.

**Features:**

- Order 64 (up to 63 keys per node)
- Automatic node splitting
- Efficient point lookups and range queries

#### Pager

**File:** `internal/storage/pager.go`

Manages disk I/O for fixed-size 8KB pages.

**Features:**

- Page allocation/deallocation
- Read/write operations
- Page header management

---

### 4. MVCC (Multi-Version Concurrency Control)

#### Version Manager

**File:** `internal/mvcc/version.go`

Manages version chains for documents.

**Features:**

- Each update creates a new version
- Old versions retained for concurrent readers
- Background garbage collection

#### Snapshot Manager

**File:** `internal/mvcc/snapshot.go`

Provides snapshot isolation for reads.

**Features:**

- Atomic timestamp generation
- Snapshot creation per transaction
- Visibility rules enforcement

**How it works:**

```
Document ID: user-1

Version Chain:
┌─────────────────┐
│ v3 (ts=300)     │ ← Latest (visible to ts≥300)
├─────────────────┤
│ v2 (ts=200)     │ ← Old (visible to 200≤ts<300)
├─────────────────┤
│ v1 (ts=100)     │ ← Oldest (GC candidate if min_ts>200)
└─────────────────┘
```

---

### 5. Transaction Manager

**File:** `internal/transaction/manager.go`

Coordinates ACID transactions.

**Isolation Levels:**

1. **ReadUncommitted**: Dirty reads allowed
2. **ReadCommitted**: Read only committed data
3. **RepeatableRead**: Consistent snapshot for all reads
4. **Serializable**: Full serializability (current: same as RepeatableRead)

**Transaction Lifecycle:**

```
BeginTransaction(level)
    ↓
[Operations: Insert/Update/Delete/Find]
    ↓
Commit() or Rollback()
```

---

### 6. Write-Ahead Log (WAL)

#### WAL Writer

**File:** `internal/wal/wal.go`

Ensures durability by logging changes before applying them.

**Record Types:**

- `RecordTypeInsert`: New document
- `RecordTypeUpdate`: Modified document
- `RecordTypeDelete`: Deleted document (tombstone)
- `RecordTypeCommit`: Transaction committed

#### Group Commit

**File:** `internal/wal/group_commit.go`

Batches multiple transactions into a single `fsync()` call.

**How it works:**

```
Thread 1: Commit() ─┐
Thread 2: Commit() ─┤
Thread 3: Commit() ─┼─→ [Batch together] → fsync() (once!)
Thread 4: Commit() ─┤
Thread 5: Commit() ─┘
```

**Performance Impact:**

- Without grouping: 100 commits = 100 fsync calls (~1-5ms each) = **100-500ms**
- With grouping (100 txns): 1 fsync call = **1-5ms total** 🚀

#### Shared Global Flusher

**File:** `internal/wal/flusher.go`

Singleton that batches `fsync()` requests across **all databases** in the process.

**Why:**
Even with group commits, multiple bundoc instances would each call `fsync()` separately. The shared flusher batches ALL instances together.

**Performance Impact:**

- 10 databases × 10 fsync/sec = 100 fsync calls
- Shared flusher: ~10 fsync calls (10x reduction!)

---

## Data Flow

### Write Path

```
1. Application calls col.Insert(txn, doc)
                ↓
2. Serialize document to JSON bytes
                ↓
3. Add to transaction's write set
                ↓
4. Insert into B+ tree index
                ↓
5.  Application calls txn.Commit()
                ↓
6. Transaction Manager:
   - Write to WAL (buffered)
   - Mark as committed
                ↓
7. Group Commit:
   - Wait for batch (max 5ms)
   - Flush buffer → WAL segment
                ↓
8. Shared Flusher:
   - Batch requests from all databases
   - Single fsync() call
                ↓
9. Update MVCC version chain
                ↓
10. Release locks, return success
```

### Read Path

```
1. Application calls col.FindByID(txn, "user-1")
                ↓
2. Check transaction's write set (read-your-own-writes)
                ↓
3. If not found, search B+ tree index
                ↓
4. Get version chain for "user-1"
                ↓
5. Apply visibility rules based on txn snapshot
                ↓
6. Return visible version
                ↓
7. Deserialize JSON → Document
                ↓
8. Return to application
```

**Key Point:** Readers NEVER block writers (and vice versa) thanks to MVCC!

---

## Concurrency Model

### Lock-Free Hot Paths

**Atomic Operations:**

- LSN generation (`atomic.Uint64`)
- Timestamp generation (`atomic.Uint64`)
- Reference counting

**Mutexes Only Where Needed:**

- Collection operations: `sync.RWMutex`
- Buffer pool: `sync.RWMutex`
- WAL writes: `sync.Mutex`

### MVCC Benefits

**Problem (Traditional Locking):**

```
Writer holds lock →  Reader blocks waiting → Slow!
```

**Solution (MVCC):**

```
Writer creates new version → Reader sees old version → Both proceed!
```

**Result:**

- Readers never wait for writers
- Writers never wait for readers
- Only writer-writer conflicts require coordination

---

## Garbage Collection

Old MVCC versions must be cleaned up to avoid unbounded growth.

**Strategy:**

1. Track minimum active snapshot timestamp
2. Versions older than `min_snapshot_ts` are safe to delete
3. Background GC scans version chains and prunes old versions

**Example:**

```
Active snapshots: [ts=500, ts=600, ts=700]
min_snapshot_ts = 500

Version Chain:
- v4 (ts=650) ← Keep (might be needed by ts=500 snapshot)
- v3 (ts=550) ← Keep
- v2 (ts=400) ← DELETE (no snapshot needs this)
- v1 (ts=300) ← DELETE
```

---

## Recovery

On database open, WAL replay ensures durability.

**Process:**

1. Scan all WAL segments (oldest to newest)
2. Read records and rebuild in-memory state
3. Filter: only replay records from **committed** transactions
4. **Restore Indices**: Load B+ Trees using Root PageIDs from `system_catalog.json`
5. Apply inserts/updates/deletes to B+ tree
6. Rebuild MVCC version chains
7. Resume normal operations

**Integrity Checks:**

- CRC32 checksum validation for every record
- Corrupted records abort recovery with error

---

## Performance Characteristics

### Write Performance

**Single-threaded:**

- ~70 inserts/sec (with full durability)

**Bottlenecks:**

- `fsync()` latency (~5-15ms per call)
- Mitigated by group commits and shared flusher

**Concurrent writes (50 workers):**

- Expected: ~3,000-5,000 inserts/sec

### Read Performance

**Point lookups:**

- B+ tree search: O(log n)
- Buffer pool hit: <1µs
- Disk read: ~100µs

**Expected throughput:**

- Single-threaded: ~10,000-50,000 reads/sec
- Concurrent (50 readers): ~50,000-100,000 reads/sec

---

## Configuration Options

**File:** `options.go`

```go
type Options struct {
    Path           string // Database directory
    BufferPoolSize int    // Number of 8KB pages to cache (default: 256)
    WALSegmentSize int64  // WAL segment size (default: 64MB)
}
```

**Tuning Recommendations:**

| Workload         | BufferPoolSize     | WALSegmentSize |
| ---------------- | ------------------ | -------------- |
| Low memory       | 128 (1MB)          | 32MB           |
| Default          | 256 (2MB)          | 64MB           |
| High performance | 1024-4096 (8-32MB) | 128MB          |

---

## File Structure

```
/path/to/database/
├── data.db              # B+ tree data file
├── wal-000001.log       # WAL segment 1
├── wal-000002.log       # WAL segment 2
└── ...
```

**WAL Segment Rotation:**

- When segment reaches 64MB, create new segment
- Old segments can be deleted after checkpoint (future feature)

---

## Security Considerations

**Current State (v1.0):**

- ❌ No authentication (embedded library)
- ❌ No encryption at rest
- ❌ No network security (no network layer)

**Future:**

- File-level encryption
- Application-level access control (when wrapped in server)

---

## Limitations

**Current Version:**

1. **No advanced queries**: Only point lookups by ID
   - No filters, sorts, aggregations
   - Coming in future versions

2. **Single-process only**: No multi-process file locking
   - Only one bundoc instance per database directory

3. **No replication**: Single-node only
   - No master-slave, no sharding

4. **Recovery only**: No Point-In-Time Recovery (PITR)

---

## Future Roadmap

### Phase 7: Advanced Queries

- Query parser for filters: `{age: {$gt: 18}}`
- Range queries
- Sort, limit, skip
- Basic aggregation pipeline

### Phase 8: Optimization

- Checkpointing to truncate WAL
- Bloom filters for faster negative lookups
- Compression (BSON encoding)

### Phase 9: Replication

- Master-slave replication
- Read replicas
- Automatic failover

### Phase 10: Sharding

- Horizontal partitioning
- Query routing
- Rebalancing

---

## References

- **MVCC**: PostgreSQL's MVCC implementation
- **WAL**: SQLite's rollback journal
- **B+ Tree**: Classic database systems textbook
- **Group Commits**: MySQL's group commit optimization

---

**For API documentation**: See [API.md](./API.md)
**For performance tuning**: See [PERFORMANCE.md](./PERFORMANCE.md)
**For configuration**: See [CONFIGURATION.md](./CONFIGURATION.md)

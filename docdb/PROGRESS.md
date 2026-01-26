# DocDB Progress Documentation

> **Status**: Active Development (v0)  
> **Last Updated**: January 2026

This document provides a comprehensive overview of implemented features, architecture components, client libraries, testing coverage, and known limitations in DocDB.

## Table of Contents

1. [Project Status](#project-status)
2. [Core Features Implemented](#core-features-implemented)
3. [Architecture Components](#architecture-components)
4. [Client Libraries](#client-libraries)
5. [Testing Coverage](#testing-coverage)
6. [Documentation](#documentation)
7. [Known Limitations](#known-limitations)
8. [Performance Characteristics](#performance-characteristics)

---

## Project Status

DocDB is a **toy/learning system** - a file-based, ACID document database written in Go. The project prioritizes **correctness over features** and **simplicity over generality**.

### Current State

- ✅ **Core Engine**: Fully functional with ACID guarantees
- ✅ **IPC Server**: Unix domain socket server operational
- ✅ **Go Client**: Complete implementation
- ✅ **TypeScript Client**: Complete implementation with JSON API
- ✅ **Crash Recovery**: WAL replay with CRC32 validation
- ✅ **Memory Management**: Global and per-DB capacity limits
- ⚠️ **Concurrency**: Basic support, some advanced scenarios not yet implemented
- ❌ **Persistence**: WAL replay partially working (test skipped)
- ❌ **Secondary Indexes**: Explicitly out of scope for v0
- ❌ **Query Operations**: Limited to primary key lookups

---

## Core Features Implemented

### Storage Layer

#### ✅ File-Based Storage
- **Location**: `internal/docdb/datafile.go`
- **Format**: Append-only data files (`.data` extension)
- **Structure**: Length-prefixed payloads (4-byte length + N-byte payload)
- **Max Payload**: 16 MB per document
- **Features**:
  - Append-only writes (no in-place mutations)
  - Offset-based addressing
  - Atomic file operations

#### ✅ Write-Ahead Log (WAL)
- **Location**: `internal/wal/`
- **Format**: Binary format with CRC32 checksums
- **Structure**: See [On-Disk Format](docs/ondisk_format.md)
- **Features**:
  - CRC32 validation (IEEE 802.3)
  - Per-database WAL files
  - Append-only writes
  - Crash recovery support
  - Configurable fsync on commit
  - Max file size limits (with warning when approaching)
- **Limitations**:
  - No automatic rotation (v0)
  - No WAL trimming (v0)
  - Single WAL file per database

#### ✅ Catalog System
- **Location**: `internal/catalog/catalog.go`
- **Format**: Binary catalog file (`.catalog`)
- **Features**:
  - Database metadata storage
  - Database ID assignment
  - Database name mapping
  - Status tracking (Active/Deleted)
  - Persistent across restarts

### Transaction System

#### ✅ ACID Transactions
- **Location**: `internal/docdb/transaction.go`, `internal/docdb/core.go`
- **Atomicity**: All-or-nothing transaction commits
- **Consistency**: Transaction validation before commit
- **Isolation**: Snapshot reads via MVCC-lite
- **Durability**: WAL writes with optional fsync
- **Features**:
  - Short-lived transactions
  - Transaction ID generation
  - Rollback support
  - WAL record generation

#### ✅ MVCC-Lite
- **Location**: `internal/docdb/mvcc.go`
- **Features**:
  - Versioned documents
  - Snapshot isolation for reads
  - Transaction ID-based visibility
  - Deleted document tracking (tombstones)
- **Implementation**:
  - Each document version tracks `CreatedTxID` and optional `DeletedTxID`
  - Reads use current snapshot transaction ID
  - Visibility determined by transaction ID comparison

### Index System

#### ✅ Sharded In-Memory Index
- **Location**: `internal/docdb/index.go`
- **Architecture**:
  - 256 shards by default (configurable)
  - Per-shard locking (RWMutex)
  - Document ID-based sharding (hash modulo)
- **Features**:
  - O(1) lookup time
  - Concurrent read access across shards
  - Per-shard write locking
  - Snapshot-based visibility checks
  - Size tracking
  - Iteration support (`ForEach`)

### Database Management

#### ✅ Multiple Logical Databases
- **Location**: `internal/pool/pool.go`
- **Features**:
  - Multiple isolated databases in one runtime
  - Per-database memory limits
  - Database creation/deletion
  - Database catalog management
  - Lazy database opening (on first access)

#### ✅ Memory Management
- **Location**: `internal/memory/`
- **Features**:
  - Global memory capacity limits
  - Per-database memory limits
  - Memory allocation tracking
  - Automatic memory freeing on delete
  - Buffer pool for efficient allocations
- **Implementation**:
  - Pre-allocation check before writes
  - Memory freed on document deletion
  - Delta tracking for updates

### Operations

#### ✅ Document CRUD Operations
- **Location**: `internal/docdb/core.go`
- **Create**: Insert new document with duplicate detection
- **Read**: Fetch document by ID with snapshot isolation
- **Update**: Replace full document (append-only)
- **Delete**: Mark document as deleted (tombstone)
- **Error Handling**:
  - `ErrDocNotFound`: Document doesn't exist
  - `ErrDocAlreadyExists`: Duplicate document ID
  - `ErrMemoryLimit`: Memory capacity exceeded
  - `ErrDBNotOpen`: Database not initialized

#### ✅ Batch Operations
- **Location**: `internal/ipc/handler.go`, `internal/pool/pool.go`
- **Features**:
  - Multiple operations in single request
  - Sequential execution
  - Atomic batch responses
  - Error propagation

#### ✅ Statistics
- **Location**: `internal/pool/pool.go`
- **Metrics**:
  - Total databases
  - Active databases
  - Total transactions (not yet tracked)
  - WAL size (not yet tracked)
  - Memory usage (global)
  - Memory capacity (global)

### Crash Recovery

#### ✅ WAL Replay
- **Location**: `internal/wal/recovery.go`, `internal/docdb/core.go`
- **Features**:
  - Sequential WAL record replay
  - CRC32 validation
  - Corruption detection
  - Automatic truncation at corruption point
  - Index rebuilding from WAL
- **Recovery Process**:
  1. Load catalog
  2. Open WAL file
  3. Read records sequentially
  4. Validate CRC32
  5. Rebuild in-memory index
  6. Truncate at first error
- **Status**: ⚠️ Partially working (persistence test skipped)

### Compaction

#### ✅ Data File Compaction
- **Location**: `internal/docdb/compaction.go`
- **Features**:
  - Size-based compaction triggers
  - Tombstone ratio-based triggers
  - Atomic compaction (`.compact` file + rename)
  - Dead document removal
  - Offset remapping
  - Periodic compaction support
- **Triggers**:
  - Data file exceeds size threshold
  - Tombstone ratio exceeds threshold
- **Process**:
  1. Create `.compact` file
  2. Write only live documents
  3. Update index offsets
  4. Atomic rename
  5. Cleanup old file

---

## Architecture Components

### Core Components

| Component | Location | Status | Description |
|-----------|----------|--------|-------------|
| **LogicalDB** | `internal/docdb/core.go` | ✅ | Main database instance |
| **Index** | `internal/docdb/index.go` | ✅ | Sharded in-memory index |
| **MVCC** | `internal/docdb/mvcc.go` | ✅ | Multi-version concurrency control |
| **TransactionManager** | `internal/docdb/transaction.go` | ✅ | Transaction lifecycle management |
| **DataFile** | `internal/docdb/datafile.go` | ✅ | Append-only data file operations |
| **Compactor** | `internal/docdb/compaction.go` | ✅ | Data file compaction |

### Infrastructure Components

| Component | Location | Status | Description |
|-----------|----------|--------|-------------|
| **Pool** | `internal/pool/pool.go` | ✅ | Database pool management |
| **Scheduler** | `internal/pool/scheduler.go` | ✅ | Request scheduling (round-robin) |
| **Catalog** | `internal/catalog/catalog.go` | ✅ | Database metadata storage |
| **WAL Writer** | `internal/wal/writer.go` | ✅ | WAL record writing |
| **WAL Reader** | `internal/wal/reader.go` | ✅ | WAL record reading |
| **WAL Recovery** | `internal/wal/recovery.go` | ⚠️ | WAL replay (partially working) |
| **IPC Server** | `internal/ipc/server.go` | ✅ | Unix socket server |
| **IPC Handler** | `internal/ipc/handler.go` | ✅ | Request handling |
| **IPC Protocol** | `internal/ipc/protocol.go` | ✅ | Binary protocol encoding/decoding |
| **Memory Caps** | `internal/memory/caps.go` | ✅ | Memory limit management |
| **Buffer Pool** | `internal/memory/pool.go` | ✅ | Buffer allocation pool |
| **Logger** | `internal/logger/logger.go` | ✅ | Structured logging |

### Configuration

| Component | Location | Status | Description |
|-----------|----------|--------|-------------|
| **Config** | `internal/config/config.go` | ✅ | Configuration management |
| **Default Config** | `internal/config/config.go` | ✅ | Sensible defaults |

### Entry Point

| Component | Location | Status | Description |
|-----------|----------|--------|-------------|
| **Main** | `cmd/docdb/main.go` | ✅ | Server entry point with CLI flags |

---

## Client Libraries

### Go Client

#### ✅ Implementation Status
- **Location**: `pkg/client/client.go`
- **Status**: Complete
- **Features**:
  - Unix socket connection
  - Connection management (auto-connect)
  - Database operations (OpenDB, CloseDB)
  - Document operations (Create, Read, Update, Delete)
  - Batch operations (BatchExecute)
  - Statistics (Stats)
  - Error handling
  - Request ID management
  - Frame encoding/decoding

#### Example Usage
```go
cli := client.New("/tmp/docdb.sock")
dbID, err := cli.OpenDB("mydb")
err = cli.Create(dbID, 1, []byte("hello"))
data, err := cli.Read(dbID, 1)
err = cli.Update(dbID, 1, []byte("updated"))
err = cli.Delete(dbID, 1)
```

### TypeScript/Bun Client

#### ✅ Implementation Status
- **Location**: `tsclient/`
- **Status**: Complete
- **Package**: `@docdb/client`
- **Features**:
  - Native Bun Unix socket support
  - Binary protocol implementation
  - Type-safe API
  - JSON convenience API (`DocDBJSONClient`)
  - Generic type parameters
  - Error handling (custom error classes)
  - Timeout support
  - Dual build (ESM + CJS)
  - Full protocol compatibility with Go server

#### Core Client (`DocDBClient`)
- Connection management
- Database operations
- Document operations (binary payloads)
- Batch operations
- Statistics

#### JSON Client (`DocDBJSONClient`)
- Type-safe JSON serialization
- Generic type parameters
- Convenience methods (`createJSON`, `readJSON`, `updateJSON`)
- Inherits all `DocDBClient` methods

#### Implementation Details
- **Files**: 13 TypeScript source files
- **Tests**: 6 unit tests (all passing)
- **Examples**: 2 working examples
- **Build**: ESM + CJS outputs
- **Protocol**: Fully compatible with Go server

#### Example Usage
```typescript
// Binary API
const client = new DocDBClient({ socketPath: '/tmp/docdb.sock' });
const dbID = await client.openDB('mydb');
await client.create(dbID, 1n, new TextEncoder().encode('Hello'));

// JSON API
const jsonClient = new DocDBJSONClient();
const dbID = await jsonClient.openDB('users');
await jsonClient.createJSON(dbID, 1n, { id: 1, name: 'John' });
const user = await jsonClient.readJSON<User>(dbID, 1n);
```

See [TypeScript Client Implementation Summary](tsclient/IMPLEMENTATION_SUMMARY.md) for complete details.

---

## Testing Coverage

### Integration Tests

#### ✅ Status: Implemented
- **Location**: `tests/integration/integration_test.go`
- **Coverage**:
  - Database creation and duplicate detection
  - Document CRUD operations
  - Multiple documents (100+ documents)
  - MVCC versioning
  - Memory limit enforcement
  - Index size tracking
- **Skipped Tests**:
  - `TestPersistence`: WAL replay not fully working

### Concurrency Tests

#### ⚠️ Status: Partial
- **Location**: `tests/concurrency/concurrency_test.go`
- **Coverage**:
  - Concurrent writes to different documents (✅ working)
  - 10 concurrent writers, 100 documents each
- **Skipped Tests**:
  - `TestConcurrentReadsWrites`: Requires more sophisticated locking
  - `TestMultipleDBs`: Requires pool-level coordination
  - `TestStarvationPrevention`: Requires pool-level coordination

### Failure Mode Tests

#### ✅ Status: Implemented
- **Location**: `tests/failure/failure_test.go`
- **Coverage**:
  - Corrupted WAL records
  - Truncated WAL files
  - Missing WAL files
  - Partial writes
  - CRC32 validation
  - Error recovery

### Benchmarks

#### ✅ Status: Implemented
- **Location**: `tests/benchmarks/bench_test.go`
- **Coverage**:
  - `BenchmarkCreateDocument`: Parallel document creation
  - `BenchmarkReadDocument`: Parallel document reads
  - `BenchmarkUpdateDocument`: Document updates
- **Features**:
  - Parallel execution support
  - Realistic payload sizes
  - Performance measurement

### Test Execution

```bash
# Run all tests
go test ./...

# Run specific suites
go test ./tests/integration
go test ./tests/failure
go test ./tests/concurrency

# Run benchmarks
go test -bench=. ./tests/benchmarks
```

---

## Documentation

### ✅ Existing Documentation

| Document | Location | Description |
|----------|----------|-------------|
| **README** | `README.md` | Project overview, features, quick start |
| **On-Disk Format** | `docs/ondisk_format.md` | Binary format specifications |
| **Failure Modes** | `docs/failure_modes.md` | Failure handling and recovery |
| **TypeScript Client** | `tsclient/README.md` | TypeScript client documentation |
| **TS Implementation** | `tsclient/IMPLEMENTATION_SUMMARY.md` | Detailed TS client status |

### Documentation Coverage

- ✅ Architecture overview
- ✅ Storage format specifications
- ✅ Failure mode handling
- ✅ Client library documentation
- ✅ API examples
- ✅ Error codes
- ⚠️ Performance tuning guide (missing)
- ⚠️ Deployment guide (missing)
- ⚠️ Configuration reference (partial)

---

## Known Limitations

### v0 Limitations (Explicitly Out of Scope)

These features are **intentionally not implemented** in v0:

- ❌ **SQL**: No SQL query language
- ❌ **Joins**: No join operations
- ❌ **Arbitrary Queries**: No query planner
- ❌ **Secondary Indexes**: Primary key only
- ❌ **Cross-Database Transactions**: Isolated databases
- ❌ **Distributed Replication**: Single-node only
- ❌ **Long-Running Transactions**: Short-lived only
- ❌ **Scan Operation**: Limited support (v0)

### Implementation Limitations

#### ⚠️ WAL Replay
- **Status**: Partially working
- **Issue**: Persistence test is skipped
- **Location**: `tests/integration/integration_test.go:274`
- **Impact**: Data may not persist across restarts reliably
- **Note**: WAL replay code exists but may have bugs

#### ⚠️ Concurrency
- **Status**: Basic support only
- **Limitations**:
  - Concurrent writes to same document not fully tested
  - Multiple concurrent databases require pool-level coordination
  - Starvation prevention not implemented
- **Location**: `tests/concurrency/concurrency_test.go`
- **Skipped Tests**: 3 tests skipped due to limitations

#### ❌ WAL Rotation
- **Status**: Not implemented
- **Issue**: Single WAL file grows indefinitely
- **Location**: `internal/wal/writer.go:61`
- **Impact**: Large WAL files, slower recovery
- **Mitigation**: Manual WAL deletion after backup

#### ❌ WAL Trimming
- **Status**: Not implemented
- **Issue**: WAL contains all historical records
- **Impact**: Larger WAL than necessary
- **Mitigation**: Manual WAL deletion after verified backup

#### ❌ Data File Corruption Detection
- **Status**: Not implemented
- **Issue**: No checksums on data file records
- **Location**: `docs/failure_modes.md:147`
- **Impact**: Silent data corruption possible
- **Mitigation**: Use filesystem with journaling, regular backups

#### ❌ TCP Support
- **Status**: Unix sockets only
- **Issue**: No TCP/IP support
- **Location**: `tsclient/IMPLEMENTATION_SUMMARY.md:262`
- **Impact**: Local-only access
- **Note**: TypeScript client uses Bun-specific Unix socket API

#### ⚠️ Statistics
- **Status**: Partial
- **Missing Metrics**:
  - Total transactions count
  - WAL size
- **Location**: `internal/pool/pool.go:215`

### Test Coverage Gaps

- Persistence across restarts (test skipped)
- Concurrent read/write to same document
- Multiple concurrent databases
- Starvation prevention
- Comprehensive error recovery scenarios

---

## Performance Characteristics

### Design Targets

| Metric | Target | Status |
|--------|--------|--------|
| **Concurrent Writers** | 10-100 | ✅ Supported |
| **P95 Latency** | < 100ms | ⚠️ Disk-dependent |
| **P99 Latency** | Bounded | ⚠️ Not measured |
| **Throughput** | Disk-bound | ✅ Expected |
| **Startup Time** | < 500ms (small DBs) | ⚠️ Not measured |

### Benchmarks

Benchmark tests are available in `tests/benchmarks/bench_test.go`:

- `BenchmarkCreateDocument`: Parallel document creation
- `BenchmarkReadDocument`: Parallel document reads (1000 docs)
- `BenchmarkUpdateDocument`: Document updates

### Performance Notes

- **Index**: O(1) lookup time with sharded hash map
- **Writes**: Append-only, disk-bound
- **Reads**: Single disk seek per read
- **Concurrency**: Per-shard locking reduces contention
- **Memory**: Bounded by configuration limits
- **WAL**: Append-only writes, fast

### Known Performance Issues

- No connection pooling in clients
- No read caching
- No write batching optimization
- Single-threaded WAL writes (per database)
- No compression

---

## Summary

### ✅ Fully Implemented

- Core storage engine (append-only files)
- WAL with CRC32 validation
- ACID transactions
- MVCC-lite snapshot isolation
- Sharded in-memory index
- Multiple logical databases
- Memory management (global + per-DB)
- Compaction system
- IPC server (Unix socket)
- Go client library
- TypeScript client library
- Basic crash recovery
- Integration tests
- Failure mode tests
- Benchmarks

### ⚠️ Partially Implemented

- WAL replay (code exists, test skipped)
- Concurrency (basic support, advanced scenarios missing)
- Statistics (some metrics missing)

### ❌ Not Implemented (v0)

- WAL rotation
- WAL trimming
- Data file corruption detection
- TCP/IP support
- Secondary indexes
- Query operations
- Cross-database transactions
- Distributed replication

---

## Next Steps (Potential v0.1 Enhancements)

1. Fix WAL replay for full persistence
2. Implement WAL rotation
3. Add data file CRC32 checksums
4. Improve concurrency (pool-level coordination)
5. Add TCP/IP support
6. Complete statistics tracking
7. Add connection pooling to clients
8. Implement read caching
9. Add comprehensive integration tests
10. Performance profiling and optimization

---

**Note**: This is a learning/toy project. Features are implemented for educational purposes, not production use. See [README.md](README.md) for project goals and non-goals.

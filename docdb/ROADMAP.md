# DocDB Roadmap

This document outlines the planned evolution of DocDB from v0.1 to v1.0 and beyond.

## Philosophy

From v0.1 onward, every step is about **polish, safety, and leverage** — not redesign. The core contract is locked.

---

## Phase 1 — Lock the Core ✅

### 1️⃣ Enforce JSON-Only Everywhere (Final Sweep)

**Status**: In Progress

**Goal**: Make it impossible to corrupt the DB via API misuse.

**Implementation Checklist**:
- [ ] Shell rejects `raw:` / `hex:` prefixes explicitly
- [ ] IPC validates JSON even if shell bypassed
- [ ] Engine validates JSON before writing to WAL
- [ ] WAL contains **only valid JSON payloads**
- [ ] Tests assert WAL is unchanged on invalid input

**Files**:
- `cmd/docdbsh/parser/payload.go` (shell validation)
- `internal/ipc/handler.go` (IPC validation)
- `internal/docdb/core.go` (engine validation)
- `tests/integration/wal_integrity_test.go` (new - WAL tests)

**Rationale**: This is the last correctness gate. Invalid JSON should never reach storage.

---

### 2️⃣ Freeze Error Surface

**Status**: In Progress

**Goal**: Create stable, comparable error values for predictable client behavior.

**Implementation**:
- Create `internal/errors/errors.go` with all public errors
- No new error strings outside this file
- No dynamic error text for core errors
- Errors compared by value, not string
- Group errors by category (Core, WAL, Pool, IPC)

**Error Categories**:
- **Core**: `ErrInvalidJSON`, `ErrDocExists`, `ErrDocNotFound`, `ErrDBNotOpen`, `ErrMemoryLimit`
- **WAL**: `ErrPayloadTooLarge`, `ErrCorruptRecord`, `ErrCRCMismatch`, `ErrFileOpen`, `ErrFileWrite`, `ErrFileSync`, `ErrFileRead`
- **Pool**: `ErrPoolStopped`, `ErrQueueFull`
- **IPC**: `ErrInvalidRequestID`, `ErrFrameTooLarge`

**Files**:
- `internal/errors/errors.go` (new - centralized errors)
- Update all packages to import from `internal/errors`

**Rationale**: Stable errors enable clean error handling in clients and better shell UX.

---

## Phase 2 — Shell Becomes a Real Tool

### 3️⃣ Shell Quality-of-Life

**Status**: Planned

**Goal**: Make shell useful for admin/debug (not queries or scripting).

**New Commands**:
- `.ls` — List all databases with status
- `.use <db>` — Alias for `.open`
- `.pwd` — Show current database name + ID
- `.pretty on|off` — Toggle JSON formatting
- `.history` — Show command history (in-memory)

**Files**:
- `cmd/docdbsh/shell/shell.go` (add state: pretty bool, history slice)
- `cmd/docdbsh/parser/parser.go` (add command parsing)
- `cmd/docdbsh/commands/commands.go` (implement handlers)

**Rationale**: Low effort, high payoff for debugging and admin tasks.

---

### 4️⃣ Shell Transcript Tests

**Status**: Planned

**Goal**: Lock UX, prevent regressions, document behavior better than README.

**Implementation**:
- Create test harness for scripted shell execution
- Golden test files for basic operations and errors
- Diff actual output vs expected output

**Files**:
- `tests/shell/harness.go` (new - test utilities)
- `tests/shell/basic.txt` (golden - normal operations)
- `tests/shell/errors.txt` (golden - error cases)
- `tests/shell/shell_test.go` (new - test runner)

**Rationale**: Most DBs skip this. DocDB will have rock-solid shell UX.

---

## Phase 3 — Durability Hardening

### 5️⃣ WAL Rotation

**Status**: Planned

**Goal**: Prevent unbounded restart time and support long-term operation.

**Implementation**:
- Rotate WAL at size limit (default 64MB)
- Naming: `dbname.wal` (active), `dbname.wal.1`, `dbname.wal.2`, ... (rotated)
- Old WAL segments marked immutable
- Recovery replays **all WAL segments in order**
- Crash-safe rotation (atomic rename)

**Configuration**:
```go
WAL:
  MaxSizeMB: 64      // Rotation threshold
  MaxSegments: 0    // Unlimited (v0.1), for trimming in v1.1
```

**Files**:
- `internal/wal/rotator.go` (new - rotation logic)
- `internal/wal/segment.go` (new - segment management)
- `internal/wal/recovery.go` (update - multi-segment support)
- `internal/config/config.go` (add rotation config)
- `tests/integration/wal_rotation_test.go` (new)

**Rationale**: Without rotation, restart time grows unbounded, compaction becomes scary, and testing slows down.

---

### 6️⃣ Data File Checksums

**Status**: Planned

**Goal**: Detect silent corruption in data files.

**New Data File Format**:
```
[4 bytes: payload_len]   Length of payload
[N bytes: payload]       Document payload (valid UTF-8 JSON)
[4 bytes: crc32]        CRC32 checksum
```

**Implementation**:
- Calculate CRC32 on write (after payload)
- Validate CRC32 on read (compare after payload)
- Return error on CRC mismatch
- Backward compatibility support (detect old format)

**Files**:
- `internal/docdb/datafile.go` (add CRC support)
- `internal/docdb/compaction.go` (upgrade old format)
- `tests/integration/crc_test.go` (new)

**Rationale**: WAL protects history, but `.data` files need payload validation. Silent corruption is unacceptable once shell exists.

---

## Phase 4 — Observability & Trust

### 7️⃣ Make Stats Real

**Status**: Planned

**Goal**: Provide meaningful statistics to trust the system.

**New Stats Fields**:
```go
type Stats struct {
    TotalDBs       int
    ActiveDBs      int
    TotalTxns      uint64
    TxnsCommitted  uint64      // New
    WALSize        uint64
    DocsLive       uint64      // New
    DocsTombstoned uint64      // New
    MemoryUsed     uint64
    MemoryCapacity uint64
    LastCompaction time.Time   // New
}
```

**Implementation**:
- Track TxnsCommitted on each commit
- Count live/tombstoned docs from index
- Update LastCompaction timestamp
- Aggregate stats across databases

**Files**:
- `internal/types/types.go` (expand Stats struct)
- `internal/docdb/core.go` (track stats)
- `internal/docdb/index.go` (add doc count methods)
- `internal/pool/pool.go` (aggregate stats)
- `cmd/docdbsh/commands/commands.go` (display new fields)

**Rationale**: How do you trust the system if you can't see what it's doing?

---

### 8️⃣ Failure-Mode Drills

**Status**: Planned

**Goal**: Last correctness milestone — verify crash safety.

**Test Scenarios**:
1. Kill -9 during WAL write
2. Kill -9 during compaction
3. Partial WAL segment recovery
4. Corrupt record detection
5. Truncated WAL file handling
6. Crash during data file write

**Implementation**:
- Create test utilities for process management
- File corruption helpers
- WAL truncation helpers
- Automated crash/recovery tests

**Files**:
- `tests/failure/crash_test.go` (new - shared utilities)
- `tests/failure/wal_write_crash_test.go` (new)
- `tests/failure/compaction_crash_test.go` (new)
- `tests/failure/partial_wal_test.go` (new)
- `tests/failure/corrupt_record_test.go` (new)

**Rationale**: If these tests pass, **the DB is real**.

---

## Phase 5 — Future Direction (v1.1)

Choose **one** direction:

### Option A: WAL Trimming + Compaction Coordination
- Trim old WAL segments after checkpoint
- Coordinate with compaction to avoid data loss
- Reduces storage overhead
- **Complexity**: Medium

### Option B: Schema-on-Read
- Extract schema from documents on read
- Allow schema queries
- Lightweight indexing
- **Complexity**: Medium-High

### Option C: JSON Path Indexing
- Index specific JSON paths
- Support simple queries: `$.name == "Alice"`
- Very limited scope
- **Complexity**: High

### Option D: TCP Support
- Replace Unix sockets with TCP
- Support remote clients
- Add authentication
- **Complexity**: Medium

**Recommendation**: **Option A** — WAL trimming is natural next step after rotation.

---

## Version Planning

### v0.1 (Current Focus)
- ✅ JSON-only enforcement
- ✅ Frozen error surface
- ✅ Shell QoL commands
- ✅ Shell transcript tests
- ✅ WAL rotation
- ✅ Data file CRCs

### v0.2
- ✅ Real stats
- ✅ Failure-mode drills

### v1.0 (Future)
- Chosen v1.1 feature (see above)
- Production-grade testing
- Performance optimization
- Documentation polish

---

## Completion Criteria

Each phase is complete when:
- All checklist items are implemented
- Tests pass
- Documentation is updated
- Examples work as documented

**Final Criteria**: DocDB is "boringly reliable" — predictable, well-tested, safe.

---

## References

- [README](../README.md) - Project overview
- [Architecture](../docs/architecture.md) - System design
- [Testing Guide](../docs/testing_guide.md) - Testing philosophy
- [On-Disk Format](../docs/ondisk_format.md) - Binary formats

## Phase 5 — Database Resilience & Crash Safety (v0.1)

### Goal Zero: Ensure Database Integrity Under All Failure Scenarios

### Phase 5.1 — Write Ordering Fix (HIGH PRIORITY)

**Problem:** Crash between WAL write and index update creates inconsistency

**Solution:**
- Add transaction completion marker to WAL
- Index only references WAL records with completion marker
- Two-phase commit protocol: WAL durable → then update index

**Implementation:**
- Create `internal/docdb/transaction_buffer.go`
- Add `OpCommit` operation type
- Update `Index.Set()` to verify completion marker
- Update `TransactionManager.Commit()` for two-phase protocol

**Files:**
- `internal/types/types.go` (OpCommit)
- `internal/docdb/transaction_buffer.go` (NEW)
- `internal/docdb/core.go` (two-phase commit)
- `internal/wal/writer.go` (commit markers)
- `tests/integration/write_ordering_test.go` (NEW)

**Completion Criteria:**
- ✅ Transaction completion markers written to WAL
- ✅ Index only uses committed records
- ✅ Crash-before-commit leaves index unchanged
- ✅ Tests pass for all scenarios

---

### Phase 5.2 — Partial Write Protection (HIGH PRIORITY)

**Problem:** Crash during payload write leaves corrupt record in data file

**Solution:**
- Add 1-byte verification flag after CRC32 in data file
- Recovery skips unverified records
- Prevents corrupt data from being indexed

**Implementation:**
- Update data file format: `[4: len] [N: payload] [4: crc32] [1: verified]`
- Write verification flag LAST (after fsync)
- On read: only process records with verified flag

**Files:**
- `internal/docdb/datafile.go` (verification flag)
- `internal/docdb/core.go` (use verification)
- `internal/wal/recovery.go` (skip logic)
- `tests/integration/partial_write_test.go` (NEW)

**Completion Criteria:**
- ✅ Verification flag implemented in data file format
- ✅ Recovery skips unverified records
- ✅ Tests for incomplete writes pass
- ✅ No corrupt data in index

---

### Phase 5.3 — Error Classification & Smart Retry (MEDIUM PRIORITY)

**Problem:** No intelligent error handling, no retry logic, no observability

**Solution:**
- Classify errors by category (Transient, Permanent, Critical, Validation)
- Smart retry with exponential backoff + jitter
- Error tracking (counts, rates, last occurrence)
- Critical error alerts

**Error Categories:**
- `ErrorTransient`: Temporary errors (EAGAIN, ENOMEM, ETIMEDOUT) - retry with backoff
- `ErrorPermanent`: Permanent errors (ENOENT, EINVAL, EEXIST) - no retry
- `ErrorCritical`: System-level errors (EIO, ENOSPC) - alert immediately
- `ErrorValidation`: Data validation errors (CRC mismatch, parse errors) - no retry
- `ErrorNetwork`: Network-related - retry with backoff

**Implementation:**
- Create `internal/errors/classifier.go`
- Create `internal/errors/tracker.go`
- Create `internal/errors/retry.go`
- Add `RetryController` with exponential backoff + jitter
- Initial delay: 10ms, max: 1s, max retries: 5
- Retry ONLY at subsystem boundaries (WAL write, fsync, socket write)

**Files:**
- `internal/errors/classifier.go` (NEW)
- `internal/errors/tracker.go` (NEW)
- `internal/errors/retry.go` (NEW)
- `internal/wal/writer.go` (use tracker)
- `internal/docdb/datafile.go` (use tracker)
- `internal/docdb/core.go` (use tracker)
- `tests/integration/error_handling_test.go` (NEW)

**Completion Criteria:**
- ✅ Error classification implemented
- ✅ Smart retry with backoff
- ✅ Error metrics collected (counts, rates)
- ✅ Critical error alerts triggered
- ✅ All error handling tests pass

---

### Phase 5.4 — Checkpoint-Based Recovery (HIGH PRIORITY)

**Problem:** Unbounded WAL replay time, corruption loses all subsequent records

**Solution:**
- Add checkpoint records to WAL (every 64MB)
- Checkpoint-aware recovery (replay from last checkpoint)
- Bounds recovery time to size since last checkpoint
- Periodic checkpoints reduce recovery time

**Checkpoint Record Format:**
```
{
  "tx_id": 9999,
  "op_type": 255,
  "snapshot_offset": 1048576,
  "index_consistent": true
}
```

**Implementation:**
- Add `internal/wal/checkpoint.go` (checkpoint management)
- Add `OpCheckpoint` operation type
- Update recovery to support checkpoint-aware replay
- Add `CheckpointManager` to track last checkpoint
- Trigger checkpoints on commit (every 64MB)

**Files:**
- `internal/wal/checkpoint.go` (NEW)
- `internal/wal/writer.go` (checkpoints)
- `internal/wal/recovery.go` (checkpoint-aware)
- `internal/docdb/recovery.go` (NEW, coordination)
- `internal/config/config.go` (checkpoint settings)
- `tests/integration/checkpoint_test.go` (NEW)

**Configuration:**
```go
CheckpointConfig struct {
    IntervalMB    uint64 // Create checkpoint every X MB
    AutoCreate    bool   // Automatically create checkpoints
    MaxCheckpoints int    // Maximum checkpoints to keep
}
```

**Completion Criteria:**
- ✅ Checkpoint records created every 64MB
- ✅ Checkpoint manager tracks state
- ✅ Recovery uses checkpoints (bounded time)
- ✅ Old WAL segments can be trimmed after checkpoint
- ✅ Tests for checkpoint-aware recovery pass

---

### Phase 5.5 — Graceful Shutdown (MEDIUM PRIORITY)

**Problem:** SIGKILL causes data loss, in-flight transactions incomplete, files in inconsistent state

**Solution:**
- Signal handling (SIGTERM, SIGINT)
- Graceful shutdown: refuse new connections, drain queues
- Wait for in-flight transactions (with timeout)
- Sync WAL and data files before closing
- Force exit after timeout (with warning)

**Timeout Breakdown:**
- 0-5s: drain queues
- 5-25s: wait for in-flight transactions
- 25-30s: final sync and close all files
- 30s: force exit (log warning)

**Implementation:**
- Add `internal/pool/shutdown.go` (signal handling)
- Add `internal/pool/drain.go` (queue draining)
- Implement `Shutdown()` method in pool
- Add `waitForInFlight(timeout)` with ticker
- Add `syncAndCloseAll()` for databases
- Add `ShutdownTimeout` config (default: 30s)

**Files:**
- `internal/pool/shutdown.go` (NEW)
- `internal/pool/drain.go` (NEW)
- `internal/pool/pool.go` (shutdown coordination)
- `internal/docdb/core.go` (Sync/Close methods)
- `tests/integration/shutdown_test.go` (NEW)

**Completion Criteria:**
- ✅ SIGTERM/SIGINT handled gracefully
- ✅ No new connections accepted after shutdown signal
- ✅ Request queues drained
- ✅ In-flight transactions complete or timeout
- ✅ All files synced and closed
- ✅ No data loss on normal termination
- ✅ Shutdown tests pass

---

### Phase 5.6 — Document-Level Corruption Detection (MEDIUM PRIORITY)

**Problem:** Data file CRC mismatches detected but not isolated to document level

**Solution:**
- Add document health tracking per WAL record
- Fetch all versions of a document from WAL
- Find latest valid version (CRC32 match)
- Rebuild index entry for this document
- Mark corrupted versions as deleted
- Log warning for each healed document

**Implementation:**
- Add `internal/docdb/validator.go` (document validation)
- Add `internal/docdb/healer.go` (self-healing logic)
- Add `CorruptionStatus` to `types.DocumentVersion`
- Add WAL filtering by document ID
- Manual healing commands: `.heal <doc_id>`, `.validate <doc_id>`
- Automatic healing (configurable, v0.2+)

**Files:**
- `internal/docdb/validator.go` (NEW)
- `internal/docdb/healer.go` (NEW)
- `internal/wal/recovery.go` (document filtering)
- `internal/docdb/index.go` (corruption flag)
- `internal/types/types.go` (corruption status)
- `internal/docdb/core.go` (integrate validator)
- `cmd/docdbsh/commands/commands.go` (heal/validate commands)
- `tests/integration/healing_test.go` (NEW)

**Healing Algorithm:**
1. Get all WAL records for `doc_id`
2. Sort by `tx_id` (descending)
3. Find latest version with valid CRC32
4. Update index to use that version
5. Delete old (corrupt) versions

**Completion Criteria:**
- ✅ Document health status tracked
- ✅ All WAL records for a document retrievable
- ✅ Healing finds latest valid version
- ✅ Corrupted versions identified and removed
- ✅ Manual healing commands work
- ✅ Tests for document-level corruption pass

---

## Completion Criteria

Phase 5 is complete when:

1. ✅ Write ordering ensures index consistency
   - Index never references incomplete WAL records
   - Crash before commit is safe

2. ✅ Partial writes detectable
   - Corrupt records skipped during recovery
   - No corrupt data in index

3. ✅ Error classification implemented
   - Smart retry for transient errors
   - No retry for permanent errors
   - Metrics available (error counts, rates)

4. ✅ Recovery is robust
   - Checkpoints bound recovery time
   - Corruption isolated to minimum impact
   - Graceful shutdown works cleanly

5. ✅ Tests pass
   - Crash simulation tests
   - Partial write tests
   - Recovery tests
   - Error handling tests
   - Shutdown tests

---

## Version Planning

### v0.2 (Future)

- Automatic document healing (background corruption scans)
- WAL trimming (automatic cleanup after checkpoint)
- Better error metrics (histograms, percentiles)
- Health check endpoints (Prometheus, OpenMetrics)


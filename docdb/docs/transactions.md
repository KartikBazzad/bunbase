# Transactions in DocDB

This document describes DocDB's transaction model, ACID properties, and transaction lifecycle.

**Note:** In the partitioned implementation (v0.4+), **multi-doc transactions** (`Begin()`, `Commit()`, `Rollback()`) are **supported**. Single-partition transactions use a fast path (no coordinator). Multi-partition transactions use an **in-process coordinator** with a persistent decision log and **two-phase commit (2PC)** so that commits are atomic across partitions. See [Future Multi-Partition Transactions](future_multi_partition_transactions.md) for the coordinator-based design and recovery.

## Table of Contents

1. [Overview](#overview)
2. [ACID Properties](#acid-properties)
3. [Transaction Lifecycle](#transaction-lifecycle)
4. [Multi-Partition Recovery](#multi-partition-recovery)
5. [MVCC-Lite Model](#mvcc-lite-model)
6. [Visibility Rules](#visibility-rules)
7. [Concurrency Behavior](#concurrency-behavior)
8. [Error Handling](#error-handling)
9. [Limitations](#limitations)

---

## Overview

DocDB provides **short-lived, atomic transactions** for grouping multiple operations. Each transaction ensures that all operations within it either succeed together or fail together.

### Key Characteristics

- **Short-Lived**: Transactions are designed for quick operations (milliseconds, not seconds)
- **Atomic**: All operations in a transaction commit together or rollback together
- **Isolated**: Snapshot-based reads provide isolation
- **Durable**: WAL ensures committed transactions survive crashes

### Transaction States

```
┌─────────┐
│  Open   │  ← Transaction created, operations can be added
└────┬────┘
     │
     ├─► Commit ──► ┌────────────┐
     │              │ Committed  │  ← All operations persisted
     │              └────────────┘
     │
     └─► Rollback ─► ┌─────────────┐
                     │ Rolled Back │  ← All operations discarded
                     └─────────────┘
```

---

## ACID Properties

### Atomicity

**Guarantee:** All operations in a transaction succeed together or fail together.

**Implementation:**

- Operations are buffered in memory during transaction
- WAL records are written atomically on commit
- Index updates happen after WAL is durable
- Rollback discards all buffered operations

**Example:**

```go
tx := db.Begin()
db.CreateInTx(tx, 1, []byte("doc1"))
db.CreateInTx(tx, 2, []byte("doc2"))
db.UpdateInTx(tx, 1, []byte("updated"))

// If commit fails, none of the operations are visible
if err := db.Commit(tx); err != nil {
    // All three operations are rolled back
    // Database state unchanged
}
```

### Consistency

**Guarantee:** Database remains in a valid state after each transaction.

**Implementation:**

- Duplicate document checks before commit
- Memory limit validation
- WAL record validation
- Index integrity maintained

**Example:**

```go
tx := db.Begin()
db.CreateInTx(tx, 1, []byte("doc1"))
db.CreateInTx(tx, 1, []byte("doc2")) // Duplicate!

// Commit fails due to duplicate
err := db.Commit(tx) // Returns error, transaction rolled back
```

### Isolation

**Guarantee:** Concurrent transactions do not interfere with each other.

**Implementation:**

- **Snapshot Isolation**: Each transaction sees a consistent snapshot
- **MVCC-Lite**: Multiple versions of documents tracked
- **Read-Your-Writes**: Not guaranteed (single writer per database)

**Isolation Level:**

- **Read Committed**: Readers see committed data
- **Snapshot Isolation**: Readers see point-in-time snapshot
- **Not Serializable**: Concurrent updates may interleave

**Example:**

```go
// Transaction 1 (txID = 100)
tx1 := db.Begin() // Snapshot = 99
db.Read(1) // Sees version created at txID <= 99

// Transaction 2 (txID = 101) commits
db.Create(2, []byte("new"))

// Transaction 1 still sees snapshot 99
db.Read(2) // Returns error (not found in snapshot 99)
```

### Durability

**Guarantee:** Committed transactions survive crashes.

**Implementation:**

- WAL records written before index update
- Optional fsync on commit (configurable)
- WAL replay on recovery
- Commit ordering invariant enforced
- **Multi-partition:** Coordinator log is fsync'd **before** writing `OpCommit` or `OpAbort` to partition WALs, so recovery can resolve in-doubt transactions from the coordinator

**Commit Ordering (single-partition):**

```
1. Write WAL record (with optional fsync)
2. Update index (make transaction visible)
```

**Commit Ordering (multi-partition 2PC):**

```
1. Phase 1: Write op records to each partition WAL (no commit/abort yet)
2. Phase 2: Append decision to coordinator log and fsync
3. For each partition: write OpCommit to partition WAL, then update index
```

This ensures:

- If crash occurs before index update: WAL (and coordinator log for multi-partition) persists, data recovered on restart
- If crash occurs after index update: WAL already persisted, consistent state
- Transaction only visible after WAL (and coordinator decision for multi-partition) is durable

---

## Transaction Lifecycle

### 1. Begin Transaction

**Purpose:** Create a new transaction and assign transaction ID.

**Steps:**

1. Acquire transaction ID (monotonically increasing)
2. Capture snapshot (current_tx_id - 1)
3. Create transaction object
4. Initialize operation buffer

**Code:**

```go
func (tm *TransactionManager) Begin() *Tx {
    txID := tm.mvcc.NextTxID()
    snapshotTxID := tm.mvcc.CurrentSnapshot()

    tx := &Tx{
        ID:           txID,
        SnapshotTxID: snapshotTxID,
        Operations:   make([]*types.WALRecord, 0),
        state:        TxOpen,
    }

    tm.txs[txID] = tx
    return tx
}
```

**Example:**

```go
tx := db.Begin()
// tx.ID = 100
// tx.SnapshotTxID = 99
// tx.state = TxOpen
```

### 2. Execute Operations

**Purpose:** Add operations to transaction buffer.

**Steps:**

1. Validate transaction state (must be Open)
2. Create WAL record
3. Add to operation buffer
4. Do NOT update index yet

**Code:**

```go
func (tm *TransactionManager) AddOp(tx *Tx, dbID uint64, opType types.OperationType, docID uint64, payload []byte) error {
    if tx.state != TxOpen {
        return ErrTxAlreadyCommitted
    }

    record := &types.WALRecord{
        TxID:       tx.ID,
        DBID:       dbID,
        OpType:     opType,
        DocID:      docID,
        PayloadLen: uint32(len(payload)),
        Payload:    make([]byte, len(payload)),
    }
    copy(record.Payload, payload)

    tx.Operations = append(tx.Operations, record)
    return nil
}
```

**Example:**

```go
tx := db.Begin()
db.CreateInTx(tx, 1, []byte("doc1"))
db.CreateInTx(tx, 2, []byte("doc2"))
// Operations buffered, not yet visible
```

### 3. Commit Transaction

**Purpose:** Persist all operations atomically. In partitioned mode, behavior depends on how many partitions the transaction touches.

**Single-partition (fast path):** When all operations in the transaction target the same partition, `Commit(tx)` writes WAL records and a commit marker to that partition's WAL, applies index updates, and returns. No coordinator log is written.

**Multi-partition (2PC):** When operations span more than one partition, `Commit(tx)` uses two-phase commit:

1. **Phase 1 – Prepare:** For each involved partition (in deterministic order by partition ID), acquire the partition lock, validate and perform data-file writes and memory allocation, and write **op records only** to that partition's WAL (no commit marker, no index updates). If any partition fails, the transaction aborts: if any partition WAL was written, the coordinator logs an abort decision and each prepared partition gets an `OpAbort` record; then memory is freed and an error is returned.
2. **Phase 2 – Commit:** If Phase 1 succeeded, the coordinator appends a **commit** decision to its persistent log and fsyncs. Then, for each partition in the same order: write `OpCommit` to that partition's WAL, apply index updates (make the transaction visible), and release the lock. The transaction is then marked committed and `nil` is returned.

**Invariant:** Any transaction that writes at least one WAL record to any partition must eventually produce exactly one coordinator decision (commit or abort). Fast abort without coordinator is allowed only when no partition WAL was touched (e.g. validation failed before any WAL write).

**Example:**

```go
tx := db.Begin()
db.CreateInTx(tx, 1, []byte("doc1"))
db.CreateInTx(tx, 2, []byte("doc2"))

if err := db.Commit(tx); err != nil {
    // All operations rolled back (single- or multi-partition)
    return err
}
// Both documents now visible
```

### 4. Rollback Transaction

**Purpose:** Discard all operations in transaction.

**Steps:**

1. Validate transaction state (must be Open)
2. Discard buffered operations
3. Do NOT write to WAL
4. Do NOT update index
5. Mark transaction as rolled back
6. Remove from transaction map

**Code:**

```go
func (tm *TransactionManager) Rollback(tx *Tx) error {
    if tx.state != TxOpen {
        return ErrTxAlreadyRolledBack
    }

    tx.state = TxRolledBack
    delete(tm.txs, tx.ID)
    return nil
}
```

**Example:**

```go
tx := db.Begin()
db.CreateInTx(tx, 1, []byte("doc1"))
db.CreateInTx(tx, 2, []byte("doc2"))

if err := db.Rollback(tx); err != nil {
    return err
}
// No documents created, database unchanged
```

---

## Multi-Partition Recovery

When the database opens (partitioned mode), recovery order is deterministic:

1. **Replay coordinator log** → build `txID → decision` (commit | abort). The coordinator log lives at `walDir/dbName/coordinator.log` and is the source of truth for multi-partition transactions.
2. **Replay all partition WALs** → per partition, build transaction records and mark which txIDs are committed (saw `OpCommit`) or aborted (saw `OpAbort`). Transactions that have records but neither commit nor abort on that partition are **in-doubt**.
3. **Resolve in-doubt txIDs** using the coordinator decision: for each in-doubt txID, if the coordinator says commit, apply that transaction's records (same as normal apply); if the coordinator says abort or the txID is missing from the coordinator (e.g. crash before decision), treat as abort and do not apply.
4. **Apply** all committed and resolved-commit transactions; discard aborted and resolved-abort. Then proceed with normal open (worker pool, healing, etc.).

**WAL record types:** In addition to data operations and `OpCommit`, partition WALs can contain `OpAbort` for a txID. On replay, `OpAbort` marks the transaction as aborted on that partition so it is not considered in-doubt. Old WALs without `OpAbort` remain valid: in-doubt transactions are resolved solely from the coordinator log (missing decision ⇒ abort).

---

## MVCC-Lite Model

### What is MVCC-Lite?

DocDB implements a simplified version of Multi-Version Concurrency Control (MVCC) optimized for short-lived transactions.

**"Lite" because:**

- No long-running transactions (all transactions are short-lived)
- No read-your-writes detection (single writer per database)
- Simple snapshot semantics (point-in-time view)
- No conflict detection (last commit wins)

### Transaction IDs

**Purpose:** Order all operations and determine visibility.

**Characteristics:**

- Monotonically increasing integers
- Assigned sequentially (no gaps)
- Start at 1 (0 reserved for uninitialized)
- Never reused

**Example:**

```
Transaction 1: txID = 1
Transaction 2: txID = 2
Transaction 3: txID = 3
...
```

### Snapshots

**Purpose:** Provide point-in-time view of database.

**Calculation:**

```go
snapshotTxID = currentTxID - 1
```

**Meaning:**

- Transaction with txID = N sees all transactions with txID <= N-1
- Transaction with txID = N does NOT see transactions with txID >= N

**Example:**

```go
// Current state: txID = 100
tx := db.Begin() // tx.ID = 100, snapshot = 99

// Transaction 100 sees all data committed in transactions 1-99
// Transaction 100 does NOT see transactions 100+ (including itself until commit)
```

### Document Versions

**Structure:**

```go
type DocumentVersion struct {
    ID          uint64  // Document ID
    CreatedTxID uint64  // Transaction that created this version
    DeletedTxID *uint64 // Transaction that deleted (nil if alive)
    Offset      uint64  // Offset in data file
    Length      uint32  // Payload length
}
```

**Version History:**

```
Document 1:
  Version 1: Created at txID = 10, Offset = 0x1000
  Version 2: Created at txID = 50, Offset = 0x2000 (update)
  Version 3: Deleted at txID = 75
```

---

## Visibility Rules

### Rule: Version is Visible If

```
created_tx_id <= snapshot_tx_id
AND (deleted_tx_id == nil OR deleted_tx_id > snapshot_tx_id)
```

### Examples

**Example 1: Document Created Before Snapshot**

```
Document: Created at txID = 10, Not deleted
Snapshot: txID = 50

10 <= 50 AND (nil == nil OR ...) → TRUE
Result: Visible
```

**Example 2: Document Created After Snapshot**

```
Document: Created at txID = 100, Not deleted
Snapshot: txID = 50

100 <= 50 → FALSE
Result: Not visible
```

**Example 3: Document Deleted Before Snapshot**

```
Document: Created at txID = 10, Deleted at txID = 30
Snapshot: txID = 50

10 <= 50 AND (30 <= 50) → FALSE
Result: Not visible (deleted)
```

**Example 4: Document Deleted After Snapshot**

```
Document: Created at txID = 10, Deleted at txID = 100
Snapshot: txID = 50

10 <= 50 AND (100 > 50) → TRUE
Result: Visible (not yet deleted in snapshot)
```

### Implementation

```go
func (m *MVCC) IsVisible(version *types.DocumentVersion, snapshotTxID uint64) bool {
    // Created after snapshot?
    if version.CreatedTxID > snapshotTxID {
        return false
    }

    // Deleted before or at snapshot?
    if version.DeletedTxID != nil && *version.DeletedTxID <= snapshotTxID {
        return false
    }

    return true
}
```

### Read-your-writes within a transaction

Within the same transaction, a read can see that transaction's own pending writes by using **`ReadInTx(tx, collection, docID)`**. It returns the document as visible to that transaction: pending create/update/patch for the doc return the pending payload; pending delete returns "not found"; if the transaction has no pending op for that doc, the read uses the transaction's snapshot (`tx.SnapshotTxID`) so the result is consistent with the snapshot plus the transaction's own writes.

**Example:**

```go
tx := db.Begin()
db.AddOpToTx(tx, "_default", types.OpCreate, 1, []byte(`{"a":1}`))
data, err := db.ReadInTx(tx, "_default", 1) // sees pending create: data == []byte(`{"a":1}`)
db.AddOpToTx(tx, "_default", types.OpUpdate, 1, []byte(`{"a":2}`))
data, _ = db.ReadInTx(tx, "_default", 1)    // sees pending update: data == []byte(`{"a":2}`)
db.Commit(tx)
```

Use `ReadInTx` when you need to read a document inside a transaction and have already written to it in that transaction. Normal `Read(collection, docID)` uses the current snapshot and does not see uncommitted writes.

### Serializable isolation (SSI-lite)

DocDB provides **best-effort serializable isolation** via conflict detection at commit time (SSI-lite):

- **Read set:** When you call `ReadInTx(tx, collection, docID)`, the `(collection, docID)` pair is added to the transaction's read set. Only reads performed via `ReadInTx` are tracked.
- **Write set:** At commit, the write set is derived from `tx.Operations` (all create/update/delete/patch ops).
- **Conflict check:** Before writing to the WAL, we consider every transaction that **committed after this transaction's snapshot** (i.e. after `tx.SnapshotTxID`). If any such transaction wrote a document this transaction read, or read a document this transaction wrote, we abort this transaction and return **`ErrSerializationFailure`**.
- **Commit history:** Commit read/write sets are stored in a bounded in-memory history (e.g. last 100k commits). If the committing transaction's snapshot is older than the oldest record in the history, we cannot check all conflicts and we **allow** the commit (falling back to snapshot isolation). So serializability is guaranteed only when the commit history window covers all concurrent commits.

**Example:** Two transactions each `ReadInTx` the same document, then one updates it and the other updates a different document. When they commit, one will succeed and the other will get `ErrSerializationFailure` (read-write conflict), avoiding write skew.

---

## Concurrency Behavior

### "Last Commit Wins" Model

**Behavior:**

- No conflict detection on concurrent updates
- Both versions are written to WAL
- Index shows last committed version
- Non-deterministic across restarts (acceptable for v0)

**Example:**

```go
// Thread 1
tx1 := db.Begin()
db.UpdateInTx(tx1, 1, []byte("version1"))
// ... processing ...

// Thread 2 (concurrent)
tx2 := db.Begin()
db.UpdateInTx(tx2, 1, []byte("version2"))
// ... processing ...

// Thread 2 commits first (txID = 100)
db.Commit(tx2) // Document 1 = "version2"

// Thread 1 commits second (txID = 101)
db.Commit(tx1) // Document 1 = "version1" (overwrites)
```

**Result:**

- Both versions in WAL (txID 100 and 101)
- Index shows "version1" (last commit)
- "version2" is lost (but recoverable from WAL)

### Safety Guarantees

**What is Safe:**

- ACID properties maintained
- No data loss (both versions in WAL)
- Reads see consistent snapshot
- No silent failures

**What is Not Safe:**

- Concurrent updates to same document (last commit wins)
- Read-your-writes (not guaranteed)
- Repeatable reads (different reads may see different states)
- Serializable isolation (not provided)

---

## Error Handling

### Transaction Errors

**ErrTxNotFound:**

- Transaction ID does not exist
- Transaction was already rolled back
- Transaction was never created

**ErrTxAlreadyCommitted:**

- Attempting to modify committed transaction
- Attempting to commit already-committed transaction

**ErrTxAlreadyRolledBack:**

- Attempting to modify rolled-back transaction
- Attempting to rollback already-rolled-back transaction

### Commit Errors

**WAL Write Failure:**

- Disk full
- Permission denied
- I/O error

**Behavior:**

- Transaction remains in Open state
- Can retry commit
- Can rollback transaction

**Example:**

```go
tx := db.Begin()
db.CreateInTx(tx, 1, []byte("doc1"))

if err := db.Commit(tx); err != nil {
    // Transaction still open, can retry or rollback
    if retryable(err) {
        // Retry commit
        db.Commit(tx)
    } else {
        // Rollback
        db.Rollback(tx)
    }
}
```

---

## Limitations

### v0 Limitations

1. **No Long-Running Transactions:**
   - Transactions should complete in milliseconds
   - No timeout mechanism
   - No automatic rollback

2. **No Conflict Detection:**
   - Concurrent updates use "last commit wins"
   - No optimistic locking
   - No pessimistic locking

3. **No Read-Your-Writes:**
   - Writes in transaction not visible until commit
   - Must commit before reading own writes

4. **Single-Node Only:**
   - DocDB **does** provide two-phase commit (2PC) for multi-partition transactions, but the coordinator runs **in-process** in the same binary as the database. There is no separate coordinator process, no RPC, and no cross-machine coordination.
   - "Single-node" means: one process, one `LogicalDB`; all partitions and the coordinator live in that process. Commits are atomic across partitions within that process only. There are no distributed commits (multi-process or multi-node).

5. **No Nested Transactions:**
   - Cannot begin transaction within transaction
   - No savepoints
   - No partial rollback

### By-design limitations (multi-partition)

These are intentional scope limits of the current multi-partition design; overcoming them would require new protocols or topology:

| Limitation                               | What it means                                                                                                                                                                                                             | How it could be overcome                                                                                                                                                                                                                |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Serializable isolation**               | Snapshot isolation alone allows write skew. | **Best-effort (SSI-lite):** Transactions that use `ReadInTx` have their read set recorded; at commit we check for conflicts with transactions that committed after our snapshot. On conflict we return `ErrSerializationFailure` and abort. Serializable within the bounded commit-history window; if the window is exceeded we allow commit (snapshot isolation).                                                                                            |
| **Read-your-writes within a transaction** | Within the same transaction, reads can see that transaction's own pending writes. | **Satisfied:** Use `ReadInTx(tx, collection, docID)` to read a doc and see pending create/update/patch/delete for that doc in the same tx. Normal `Read()` uses the current snapshot and does not see uncommitted writes.                                                                                                                                   |
| **Distributed commits**                  | All partitions and the coordinator run in one process; no multi-node or multi-process 2PC.                                                                                                                                | Distributed coordinator (separate process or RPC), participant protocol (prepare/commit/abort) over the network, and failure handling for node/network partitions.                                                                      |
| **Deadlock-free cross-partition writes** | Cross-partition writes can deadlock if different transactions lock partitions in different orders. | **Satisfied:** Partition locks are always taken in deterministic order (ascending partition ID) in both Phase 1 and Phase 2, so concurrent multi-partition commits do not deadlock. To relax this (e.g. app-defined order) would require deadlock detection (timeouts, wait-for graph) or a global lock. |

### Future Enhancements (v0.1+)

- Conflict detection
- Optimistic locking
- Read-your-writes guarantee
- Transaction timeouts
- Nested transactions (savepoints)

---

## Best Practices

### 1. Keep Transactions Short

```go
// Good: Short transaction
tx := db.Begin()
db.CreateInTx(tx, 1, []byte("doc1"))
db.CreateInTx(tx, 2, []byte("doc2"))
db.Commit(tx)

// Bad: Long transaction (don't do this)
tx := db.Begin()
db.CreateInTx(tx, 1, []byte("doc1"))
// ... network call ...
// ... file I/O ...
// ... long computation ...
db.Commit(tx)
```

### 2. Always Handle Errors

```go
tx := db.Begin()
db.CreateInTx(tx, 1, []byte("doc1"))

if err := db.Commit(tx); err != nil {
    // Always rollback on error
    db.Rollback(tx)
    return err
}
```

### 3. Use Batch Operations for Multiple Documents

```go
// Good: Single transaction
tx := db.Begin()
for i := 1; i <= 100; i++ {
    db.CreateInTx(tx, uint64(i), []byte(fmt.Sprintf("doc%d", i)))
}
db.Commit(tx)

// Bad: Multiple transactions
for i := 1; i <= 100; i++ {
    tx := db.Begin()
    db.CreateInTx(tx, uint64(i), []byte(fmt.Sprintf("doc%d", i)))
    db.Commit(tx)
}
```

### 4. Validate Before Commit

```go
tx := db.Begin()
db.CreateInTx(tx, 1, []byte("doc1"))

// Validate before commit
if !isValid(tx) {
    db.Rollback(tx)
    return ErrInvalidTransaction
}

db.Commit(tx)
```

---

## References

- [Architecture Guide](architecture.md) - System architecture
- [Concurrency Model](concurrency_model.md) - Concurrency patterns
- [Usage Guide](usage.md) - How to use transactions
- [Failure Modes](failure_modes.md) - Error handling

## Related documents

Supporting specs and design notes for the DocDB v0.4 transaction model:

- [Transaction Correctness](transaction_correctness.md) — Invariants and correctness arguments (no dirty reads, durability, deterministic recovery).
- [Compaction Semantics](compaction_semantics.md) — How partition compaction fits into the transaction model (snapshot, atomicity, recovery).
- [Future Multi-Partition Transactions](future_multi_partition_transactions.md) — Implemented coordinator-based 2PC, single-partition fast path, and recovery; optional best-effort (saga-like) path documented.

# Transactions in DocDB

This document describes DocDB's transaction model, ACID properties, and transaction lifecycle.

**Note:** In the current partitioned implementation (v0.4), **multi-doc transactions** (`Begin()`, `Commit()`, `Rollback()`) are **temporarily unsupported**. `Commit(tx)` returns an error: "multi-doc transactions not supported in partitioned mode". Use single-document operations (Create, Read, Update, Delete, Patch) or pool requests instead. Partition-aware multi-doc transactions may be re-added in a future release.

## Table of Contents

1. [Overview](#overview)
2. [ACID Properties](#acid-properties)
3. [Transaction Lifecycle](#transaction-lifecycle)
4. [MVCC-Lite Model](#mvcc-lite-model)
5. [Visibility Rules](#visibility-rules)
6. [Concurrency Behavior](#concurrency-behavior)
7. [Error Handling](#error-handling)
8. [Limitations](#limitations)

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

**Commit Ordering:**

```
1. Write WAL record (with optional fsync)
2. Update index (make transaction visible)
```

This ensures:

- If crash occurs before index update: WAL persists, data recovered on restart
- If crash occurs after index update: WAL already persisted, consistent state
- Transaction only visible after WAL is durable

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

**Purpose:** Persist all operations atomically.

**Steps:**

1. Validate transaction state (must be Open)
2. Write all WAL records to WAL file
3. Fsync WAL (if enabled)
4. Update index (make transactions visible)
5. Mark transaction as committed
6. Release transaction locks

**Code:**

```go
func (db *LogicalDB) Commit(tx *Tx) error {
    db.mu.Lock()
    defer db.mu.Unlock()

    records, err := db.txManager.Commit(tx)
    if err != nil {
        return err
    }

    // Write all WAL records
    for _, record := range records {
        if err := db.wal.Write(record.TxID, record.DBID, record.DocID, record.OpType, record.Payload); err != nil {
            return fmt.Errorf("failed to write WAL: %w", err)
        }
    }

    // Update index (make visible)
    txID := db.mvcc.NextTxID()
    for _, record := range records {
        switch record.OpType {
        case types.OpCreate:
            version := db.mvcc.CreateVersion(record.DocID, txID, 0, record.PayloadLen)
            db.index.Set(version)
        case types.OpUpdate:
            existing, exists := db.index.Get(record.DocID, db.mvcc.CurrentSnapshot())
            if exists {
                version := db.mvcc.UpdateVersion(existing, txID, 0, record.PayloadLen)
                db.index.Set(version)
            }
        case types.OpDelete:
            version := db.mvcc.DeleteVersion(record.DocID, txID)
            db.index.Set(version)
        }
    }

    return nil
}
```

**Example:**

```go
tx := db.Begin()
db.CreateInTx(tx, 1, []byte("doc1"))
db.CreateInTx(tx, 2, []byte("doc2"))

if err := db.Commit(tx); err != nil {
    // All operations rolled back
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

4. **No Cross-Database Transactions:**
   - Transactions scoped to single database
   - No two-phase commit
   - No distributed transactions

5. **No Nested Transactions:**
   - Cannot begin transaction within transaction
   - No savepoints
   - No partial rollback

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

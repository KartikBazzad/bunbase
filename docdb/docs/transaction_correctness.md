# Transaction Correctness (DocDB v0.4)

This document states the **system contract** for DocDB v0.4 transactions as **invariants** and gives short **proof sketches** that the implementation satisfies the [DocDB v0.4 Transaction Specification](transactions.md) (single-partition, snapshot-isolated, partition-aware). It does not constitute a full formal proof.

---

## 1. Invariants

The following are the core invariants that the implementation must maintain. References point to the relevant code.

### I1. Single writer per partition

**Statement:** For each partition `p`, at most one writer holds `partition.mu` at any time.

**Enforcement:** The worker pool acquires the partition lock before executing a write task and releases it after the task completes. All writes (Create, Update, Delete, Patch) for a partition go through `executeOnPartition`, which is invoked only while holding that partition's mutex.

**Code:** `docdb/internal/docdb/worker_pool.go` (worker locks partition before execution); partition execution in `core.go` (`executeCreateOnPartition`, etc.).

---

### I2. Global TxID monotonicity

**Statement:** Transaction IDs are allocated globally and are strictly increasing. `NextTxID()` is called before any WAL write for that transaction.

**Enforcement:** `MVCC.NextTxID()` returns the current value and increments. TxID is allocated at transaction begin (for multi-op transactions) or at the start of each single-doc operation before WAL write.

**Code:** `docdb/internal/docdb/mvcc.go` (`NextTxID`, `currentTxID`). Single-doc path: TxID is allocated in `executeCreateOnPartition` (and similar) before `wal.Write(...)`.

---

### I3. Visibility

**Statement:** A version `v` is visible to snapshot `S` if and only if:

- `v.CreatedTxID <= S`, and
- `v.DeletedTxID == nil` or `*v.DeletedTxID > S`.

**Enforcement:** Index lookups and scans use `snapshotTxID`; visibility is implemented as `CreatedTxID <= snapshotTxID` and (if deleted) `DeletedTxID > snapshotTxID`. Same rule appears in MVCC and in the index shard.

**Code:** `docdb/internal/docdb/mvcc.go` (`IsVisible`); `docdb/internal/docdb/index.go` (`isVisible`, `Get`, `ScanVisible`, `ScanCollection`).

---

### I4. Commit-before-visible

**Statement:** A transaction's writes become visible to readers only after its commit record is durable in that partition's WAL. The index is updated only after the commit record has been written (and, when fsync is enabled, synced).

**Enforcement:** In the single-doc path: (1) append WAL record for the operation, (2) append WAL commit record, (3) then update the partition index. Recovery applies only records for TxIDs that have a commit record (I5). So no reader (including after recovery) can see a write whose commit record is not yet durable.

**Code:** `docdb/internal/docdb/core.go` (e.g. `executeCreateOnPartition`: `wal.Write(...)`, then `wal.Write(..., OpCommit, nil)`, then `partition.index.Set(...)`).

---

### I5. Recovery committed-only

**Statement:** During replay, a WAL record is applied to the partition's index and datafile only if the partition WAL contains a commit record for that record's TxID.

**Enforcement:** Replay first passes over the WAL to build a set `committed[TxID]`. When applying records, only records whose `txID` is in `committed` are applied. Uncommitted transactions are skipped.

**Code:** `docdb/internal/docdb/core.go` (`replayPartitionWALForPartition`: `committed` map; apply loop skips `if !committed[txID]`).

---

## 2. Proof sketches

The following arguments show that the implementation satisfies the specification's guarantees. They are proof sketches, not full formal derivations.

### No dirty reads

**Claim:** No read observes data written by a transaction that has not yet committed.

**Argument:** Reads use a snapshot value `SnapshotTxID` (either from the transaction's begin or from the current snapshot for single-doc reads). Visibility (I3) requires `v.CreatedTxID <= SnapshotTxID`. A writer makes its version visible only after writing the commit record to the WAL and then updating the index (I4). Until the commit record is durable and the index is updated, the new version is not visible. Therefore no reader with snapshot `S` can see a version created by a transaction `T` unless `T` has already committed and its commit record is durable; at that point `T <= S` for any snapshot `S` taken after the commit. So no read sees uncommitted data.

---

### No lost updates (within a partition)

**Claim:** Within a single partition, the last committed write to a document is the one that persists; concurrent writers do not lose updates in an undefined way.

**Argument:** By I1, at most one writer holds the partition lock at a time. So all writes to the same partition are serialized. Each write: (1) appends to WAL, (2) writes commit record, (3) updates index. The order of commits is the order of lock holders. Recovery (I5) reapplies only committed transactions in TxID order. Thus the last committed write to a document in that partition is the one that remains visible; there is no lost update within the partition.

---

### Durability

**Claim:** Once a transaction is committed (commit record written and, when configured, fsync completed), its effects persist across crashes.

**Argument:** Commit path: append operation record(s) and commit record to the partition WAL; group commit / fsync ensures the tail of the WAL is durable. Then the index is updated. After a crash, recovery (I5) replays the partition WAL, identifies committed TxIDs, and reapplies their records to the datafile and index. The order of application is deterministic (TxID order). So the state after recovery includes all committed transactions. Compaction does not remove WAL records needed for recovery until after checkpoint/trim; the recovery procedure reads WAL segments in LSN order. Thus committed transactions are durable.

---

### Deterministic recovery

**Claim:** Recovery produces a unique, deterministic state given the same WAL and datafile contents.

**Argument:** Per partition: (1) Replay reads WAL in LSN order and builds `committed` and `txRecords`. (2) The set of applied TxIDs is exactly `{ txID : committed[txID] }`. (3) Applied records are processed in sorted TxID order. (4) For each TxID, records are applied in the order they appeared in the WAL. There is no nondeterminism in this process. Globally, `CurrentTxID` is set to `max(txID over all partitions) + 1`. So recovery is deterministic.

---

## 3. Explicit non-goals

The following are **out of scope** for this document and for v0.x (see specification §12):

- **Multi-partition atomicity:** No claim that a transaction spanning multiple partitions is atomic.
- **Serializable isolation:** Write skew across partitions is possible; no proof of serializability.
- **Cross-partition constraints:** No uniqueness or consistency guarantees across partitions.

No correctness claim is made for these; the implementation may return `ErrCrossPartitionTransaction` or equivalent when multi-partition transactions are attempted.

---

## 4. Implementation mapping

| Spec concept          | Implementation                                                                                                   |
| --------------------- | ---------------------------------------------------------------------------------------------------------------- |
| Partition routing     | `RouteToPartition(docID, partitionCount)` in `docdb/internal/docdb/routing.go`                                   |
| Write serialization   | Partition mutex; `executeOnPartition` in `core.go`                                                               |
| TxID allocation       | `MVCC.NextTxID()` in `docdb/internal/docdb/mvcc.go`                                                              |
| Snapshot / visibility | `MVCC.CurrentSnapshot()`, `IsVisible`; index `Get`/`ScanCollection` with `snapshotTxID` in `index.go`, `core.go` |
| Commit durability     | WAL append + commit record, then index update in `executeCreateOnPartition` (and similar)                        |
| Recovery              | `replayPartitionWALForPartition` in `core.go`; `committed` map and apply loop                                    |

These mappings are sufficient to trace the invariants and proof sketches to the codebase.

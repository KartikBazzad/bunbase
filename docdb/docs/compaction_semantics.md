# Partition Compaction Transaction Semantics

This document defines how **partition compaction** fits into the DocDB v0.4 transaction model: what consistency view compaction observes, what guarantees hold for readers and recovery, and how compaction interacts with WAL and the partition datafile.

---

## 1. Scope

- Compaction is a **background, partition-local** operation. It is **not** a user transaction; it does not allocate a TxID.
- Each partition is compacted independently. The implementation compacts one partition at a time under a global lock, then aggregates collection doc counts.
- **Code:** `docdb/internal/docdb/compaction.go` (`Compact`, `compactPartition`).

---

## 2. Semantic model

### 2.1 Consistency view

Compaction runs while holding `db.mu.Lock()` (current implementation). Therefore it sees a **quiescent** view of the partition: no concurrent user writes run during compaction for that database.

**Definition:** The **compaction snapshot** is the state of the partition index and datafile at the moment compaction starts for that partition. All live (non-tombstone) versions in the index at that time are copied to the new datafile with new offsets; the index is then updated to point to the new offsets.

Compaction does **not** take a formal snapshot TxID. It iterates the current index and copies live versions. The lock ensures that "current" is consistent and no writer modifies the partition during the copy.

### 2.2 No WAL records

Compaction does **not** write WAL records. It only:

1. Reads the partition index and datafile (live versions).
2. Writes a new datafile (e.g. `path.compact`).
3. Updates in-memory index offsets to point into the new file.
4. Closes the old datafile, renames the new file to the main datafile path, and sets the partition's datafile to the new file.

So compaction is invisible to the WAL and to recovery in terms of new log records.

### 2.3 Atomic swap

The transition from the old datafile to the new one is done by:

1. Updating index offsets in memory (all live versions now point to offsets in the new file).
2. Closing the old datafile.
3. Renaming the compact file to the main datafile path.
4. Opening the new datafile and calling `partition.SetDataFile(...)`.

**Guarantee:** A reader that starts **after** the rename and open sees the new datafile. A reader that had the old file open may still read from it until that read path completes or the handle is closed. There is no cross-reader atomicity claim beyond "each read uses one datafile or the other." For v0.x, compaction holds the database lock, so no concurrent user reads run during compaction; the main concern is recovery (see below).

---

## 3. Recovery

### 3.1 Compaction is idempotent with respect to WAL

Recovery replays the partition WAL into the **current** datafile (whatever file is at the partition's datafile path after open). Recovery does **not** "replay" compaction. So:

- If compaction completed before the crash: the datafile on disk is the compacted one; recovery appends replayed payloads to it and rebuilds the index from WAL. The compacted file may have free space at the end for new appends.
- If compaction never ran or was interrupted before rename: the datafile is the pre-compaction file; recovery proceeds as usual.

Compaction does not remove or alter WAL. So recovery semantics are unchanged: apply all committed records in TxID order. Compaction is **idempotent** with respect to the WAL in the sense that recovery does not need to know whether compaction ran; it always replays WAL into the current datafile.

### 3.2 Crash during compaction

If the process crashes **during** compaction (e.g. after rename but before `SetDataFile`, or during rename), the following holds:

- **After rename, before new open:** On next open, the partition opens whatever file is at the main datafile path. If rename succeeded, that is the new compacted file; the index in memory is lost, so recovery must replay WAL to rebuild the index. Replay will append to the new file (which may already contain compacted data). Correctness is preserved because WAL replay reapplies all committed operations; the compacted file content is a subset of what WAL would produce (same live versions), and replay may overwrite or append. To avoid inconsistency, the current implementation runs compaction under `db.mu.Lock()` and does not allow concurrent writers; so the only risk is crash during the brief rename/open window.
- **Best-effort:** For v0.x, compaction is not guaranteed crash-safe mid-swap. Document: avoid killing the process during the rename/open window if possible. A future improvement could add a small WAL record or marker to indicate "compaction started" / "compaction completed" so that recovery can detect and repair a partially swapped state if needed.

---

## 4. Concurrency (future)

If compaction is later allowed to run **without** holding `db.mu` for the whole duration (e.g. only the partition lock, or a snapshot-based copy):

- Compaction must take a **snapshot TxID** and copy only versions visible at that snapshot. The index and datafile would be read consistently at that snapshot.
- Index updates (offset-only) must be applied in a way that does not break in-flight reads: e.g. only update offsets for versions that are still the same; or use a double-buffer / indirection so that readers see a consistent view.

**Current plan:** Keep the global lock and document that compaction blocks user writes (and thus reads that need the partition) for the duration of compaction for that database.

---

## 5. Summary table

| Aspect        | Description                                                                                |
| ------------- | ------------------------------------------------------------------------------------------ |
| **Operation** | Partition compaction (per-partition copy of live versions to new datafile; atomic rename). |
| **TxID**      | None. Compaction does not allocate a transaction ID.                                       |
| **Snapshot**  | Quiescent (lock). Optional future: snapshot TxID if compaction runs without full lock.     |
| **WAL**       | Compaction does not write WAL records.                                                     |
| **Atomicity** | Rename-swap per partition; readers after swap see new file.                                |
| **Recovery**  | WAL replay unchanged. Compaction is not replayed; recovery replays into current datafile.  |

---

## 6. Reference

- Implementation: `docdb/internal/docdb/compaction.go` (`ShouldCompact`, `Compact`, `compactPartition`).
- Transaction model: [DocDB v0.4 Transaction Specification](transactions.md) and [Transaction Correctness](transaction_correctness.md).

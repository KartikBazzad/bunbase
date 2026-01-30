# Multi-Partition Transactions (Coordinator-Based 2PC)

This document describes the **implemented** multi-partition transaction design in DocDB: an in-process coordinator with a persistent decision log and two-phase commit (2PC). It is consistent with the [Transaction Specification](transactions.md). Option A (single-partition) and Option C (atomic multi-partition via 2PC) are **implemented**; Option B (best-effort saga-like) remains documented as an optional evolution.

---

## 1. Current boundary (implemented)

**Supported today:**

- **Single-partition transactions:** All operations in a transaction target the same partition. Commit uses a **fast path**: partition-local WAL write + commit record + index update; no coordinator log write.
- **Multi-partition atomic transactions:** Operations can span multiple partitions. Commit uses **two-phase commit (2PC)** with an in-process coordinator that persists its commit/abort decision. All operations commit atomically or the transaction aborts (with coordinator abort + `OpAbort` on any prepared partition).
- Snapshot isolation; partition-local WAL and recovery; global TxID allocation; deterministic routing (`RouteToPartition(docID, N)`).
- **Recovery:** Coordinator log is replayed first; partition WALs are replayed; in-doubt transactions are resolved using the coordinator decision (no decision ⇒ abort).

**Not supported (out of scope):**

- Distributed coordinator (separate process or RPC).
- Network partitions or multi-node deployments.
- Read-your-writes or serializable isolation across partitions.

---

## 2. Option A: Single-partition Commit (implemented)

**Status: Implemented.** When all operations in `tx.Operations` map to the **same** partition, `Commit(tx)` uses a **fast path**: no coordinator log is written.

**Behavior:**

- On `Commit(tx)`: if the unique set of partition IDs has size 1, acquire that partition's lock; write WAL records and one `OpCommit` for the transaction to that partition's WAL; apply index updates; release lock; mark transaction committed. No coordinator append or fsync.
- Recovery: unchanged for single-partition; each partition recovers independently using existing WAL replay (commit ⇒ apply).

---

## 3. Option B: Best-effort multi-partition (saga-like)

**Goal:** Allow a "transaction" to contain operations that target **multiple** partitions, but **do not** guarantee atomicity. Useful for batch convenience; the application is responsible for compensating on failure.

**Behavior:**

- On `Commit(tx)`:
  1. Group operations by partition ID.
  2. Sort partitions (e.g. by partition ID) to get a deterministic commit order.
  3. For each partition in order: acquire partition lock; write WAL records for that partition's operations; write commit record for this transaction on that partition; update index; release lock.
  4. If one partition fails (e.g. WAL write error), previous partitions are already committed. The transaction is **not** rolled back; the API returns an error and the application must compensate (e.g. issue compensating updates or document the partial commit).

**Recovery:** Unchanged. Each partition recovers independently. Some partitions may have the transaction committed, others not; recovery does not coordinate.

**Documentation:** Clearly document as "multi-partition best-effort; no atomicity; use for batch convenience only; application must handle partial failure."

**Recommendation:** Document Option B as an optional, best-effort API for batches. Implementation can follow after Option A if desired.

---

## 4. Option C: Two-phase commit (implemented)

**Status: Implemented.** Atomic commits across multiple partitions use an **in-process coordinator** with a persistent decision log and classic 2PC.

### Coordinator log

- **Location:** `walDir/dbName/coordinator.log` (append-only).
- **Format:** Each record = `(txID uint64, decision byte, checksum uint32)`. Checksum (e.g. CRC32) over txID+decision to detect torn writes.
- **Durability:** Coordinator **fsync** after appending the decision **before** writing `OpCommit` or `OpAbort` to partition WALs.
- **API:** `AppendDecision(txID uint64, commit bool) error`, `Replay() (map[uint64]bool, error)` (txID → true=commit, false=abort).

### WAL record type: OpAbort

- **Partition WAL:** On abort (e.g. Phase 1 failure after some partition WAL writes), each prepared partition writes one `OpAbort` record for the txID so replay does not treat the transaction as in-doubt on that partition.

### Two-phase Commit(tx) flow

- **Phase 1 – Prepare:** Group ops by partition; sort partition IDs; for each partition in order: acquire lock, validate and write data + op records to partition WAL only (no `OpCommit`), release lock. If any step fails: if no partition WAL was touched, fast abort (no coordinator). If any partition was written, write `(txID, abort)` to coordinator and fsync; write `OpAbort` to every prepared partition; free memory; return error.
- **Phase 2 – Commit:** Append `(tx.ID, commit=true)` to coordinator log and fsync. For each partition in order: write `OpCommit` to partition WAL, apply index updates, release lock. Mark transaction committed, return nil.

### Recovery order

1. Replay coordinator log → `txID → decision`.
2. Replay all partition WALs → `txRecords`, `committed`, `aborted` per partition; in-doubt = has records but neither commit nor abort on that partition.
3. Resolve in-doubt: apply iff coordinator says commit; missing decision ⇒ abort.
4. Proceed with normal open.

### Invariant

Any transaction that writes at least one WAL record to any partition must eventually produce exactly one coordinator decision (commit or abort). Fast abort without coordinator is allowed only when no partition WAL was touched.

---

## 5. What stays unchanged

- Global TxID allocation and snapshot semantics remain as today.
- Partition-local write serialization (one writer per partition at a time) and WAL semantics (commit/abort in partition WAL) remain.
- Routing formula `RouteToPartition(docID, partitionCount)`; recovery extends to coordinator replay and in-doubt resolution.
- Snapshot isolation guarantees within a partition remain.

---

## 6. Summary

| Option | Atomicity                    | Status          | Use case                                        |
| ------ | ---------------------------- | --------------- | ----------------------------------------------- |
| **A**  | Single partition only        | Implemented     | Fast path; no coordinator log.                  |
| **B**  | Best-effort; no atomicity    | Not implemented | Batch convenience; app compensates (optional).  |
| **C**  | Multi-partition atomic (2PC) | Implemented     | In-process coordinator; atomic cross-partition. |

Option A and C are implemented. Single-partition commits bypass the coordinator; multi-partition commits use 2PC with coordinator log and recovery as described above. Option B remains an optional, documented evolution path.

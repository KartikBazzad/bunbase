# DocDB Codebase Review

**Date:** 2026-01-30  
**Scope:** docdb implementation, docs, document flow, and areas for improvement.  
**Status:** In development (v0.4).

---

## 1. Implementation Status Summary

### 1.1 From Docs

- **V0.4 (V0.4_IMPLEMENTATION_STATUS.md):** Partitioned LogicalDBs, per-partition WAL, worker pool, lock-free reads, parallel recovery, server-side query engine. Core invariant: *"Exactly one writer per partition at a time. Unlimited readers. Workers are not bound to partitions."*
- **docs/implementation-status:** README_STATUS describes Phase 5 (resilience) mostly complete: write ordering, partial write protection, checkpoint recovery, graceful shutdown, healing, error classification. ROADMAP still lists Phase 1–3 items (JSON enforcement, shell, WAL rotation) as in progress or planned, which is partly legacy—many are done.
- **Gaps:** V0.4 explicitly marks testing as ⏳ (unit/integration/benchmarks for partitions, recovery, query). Query projection, nested filters, multi-field sort, and full multi-doc transaction story are noted as limitations.

### 1.2 Version Alignment

- **architecture.md** mentions v0.3 (ants pool, group commit, etc.) and v0.2 (collections, patch, healing). **V0.4_IMPLEMENTATION_STATUS.md** describes v0.4 (partitions, PartitionWAL, worker pool). The main codebase is v0.4; architecture doc should be updated to describe partitioned layout and worker-pool execution as the current model.

---

## 2. Document Flow (End-to-End)

### 2.1 High-Level Path

```
Client (Go/TS/Bun)
    → IPC (Unix socket, binary frames)
    → Handler (ipc/handler.go): validate JSON, build Request
    → Pool.Execute(req)
    → Scheduler.Enqueue(req)  [per-DB queue, round-robin]
    → Worker (ants pool) → handleRequest
    → OpenDB(dbID) [lazy open, PartitionCount=1 by default from pool]
    → RouteToPartition(docID, partitionCount)  → partitionID
    → NewTask(partitionID, Op*, collection, docID, payload)
    → db.SubmitTaskAndWait(task)
    → WorkerPool: dequeue task, partition.mu.Lock() for writes
    → executeOnPartition(partition, task)
    → executeCreateOnPartition / executeReadOnPartition / ...
```

### 2.2 Write Path (Create) – Single Operation

**Current order in `executeCreateOnPartition` (core.go:253–300):**

1. `validateJSONPayload(payload)`
2. `collections.EnsureCollection(collection)`
3. `partition.index.Get(..., CurrentSnapshot())` → conflict if exists
4. `memory.TryAllocate(dbID, len(payload))`
5. **DataFile.Write(payload)** → get `offset`
6. `mvcc.NextTxID()`, `mvcc.CreateVersion(docID, txID, offset, len)`
7. **WAL.Write(txID, ..., OpCreate, payload)**
8. **WAL.Write(txID, ..., OpCommit, nil)**
9. `txnsCommitted++`, **partition.index.Set(collection, version)**, `collections.IncrementDocCount`

**Read path:** Snapshot via `mvcc.CurrentSnapshot()` → `partition.index.Get(collection, docID, snapshot)` → `DataFile.Read(version.Offset, version.Length)` (lock-free; no partition.mu).

**Recovery path (`replayPartitionWALForPartition`):** Replay WAL → buffer by txID, determine committed/in-doubt → for each committed op: **DataFile.WriteNoSync(payload)** → **index.Set(collection, version)** → one `DataFile.Sync()` at end. So recovery correctly does WAL → datafile → index.

### 2.3 Multi-Partition Transaction Path (2PC)

- **Begin()** → Tx with SnapshotTxID, readSet for SSI-lite.
- **CreateInTx / UpdateInTx / etc.** → buffer in Tx.Operations (WALRecord); route docID to partition.
- **Commit(tx):**  
  - SSI-lite conflict check via `commitHistory`.  
  - For each partition touched: **preparePartitionForCommit** (datafile.Write, then wal.Write for each op, then wal.Write OpCommit).  
  - Coordinator log decision (commit/abort).  
  - Then **applyPartitionResults** (index.Set for each op).  
- Same **datafile-before-WAL** order appears in `preparePartitionForCommit` (e.g. core.go:1580–1586, 1605–1611).

### 2.4 Document Flow Diagram

```
                    ┌─────────────────────────────────────────────────────────┐
                    │                     Document Write (Create)             │
                    └─────────────────────────────────────────────────────────┘
                                                     │
     ┌───────────────────────────────────────────────┼───────────────────────────────────────────────┐
     │                                               ▼                                               │
     │  IPC Handler  →  Pool.handleRequest  →  RouteToPartition(docID)  →  Task  →  SubmitTaskAndWait
     │                                               │
     │                                               ▼
     │                              WorkerPool: partition.mu.Lock()
     │                                               │
     │                                               ▼
     │  executeCreateOnPartition:  [JSON check] → [Index conflict] → [Memory alloc]
     │                                               │
     │     CURRENT ORDER (BUG):     DataFile.Write(payload) → WAL.Write(OpCreate) → WAL.Write(OpCommit)
     │                              → index.Set(version)
     │
     │     INTENDED ORDER:          WAL.Write(OpCreate) → WAL.Write(OpCommit) → DataFile.Write(payload)
     │                              → index.Set(version)
     └───────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Critical Issue: Write Ordering (WAL vs Datafile)

### 3.1 Problem

The **write-ahead** contract is violated in the single-operation and multi-partition transaction paths:

- **Single-op:** `executeCreateOnPartition`, `executeUpdateOnPartition`, `executeDeleteOnPartition` all write to the **datafile before** the WAL.
- **Multi-partition Commit:** `preparePartitionForCommit` does `dataFile.Write(rec.Payload)` then `wal.Write(...)`.

**core.go** states the intended invariant:

```text
// Commit ordering invariant:
// 1. Write WAL record (includes fsync if enabled)
// 2. Update index (making transaction visible)
```

So the intended order is: **WAL first**, then datafile, then index. Recovery already assumes WAL is the source of truth and applies WAL → datafile → index.

### 3.2 Risk

If the process crashes **after** `DataFile.Write` but **before** `WAL.Write`:

- Payload is on disk in the datafile.
- No WAL record exists.
- Index is not updated.
- On recovery, only WAL is replayed; this write is never applied.
- Result: **Orphan data** in the datafile (space leak, and any future reuse of offsets could cause corruption if not carefully managed).

So durability and “write-ahead” are violated.

### 3.3 Recommended Fix

- **Single-operation path:** For Create/Update, do **WAL.Write(OpCreate/OpUpdate, payload)** and **WAL.Write(OpCommit)** first (and ensure WAL is synced per config). Then **DataFile.Write(payload)** to get offset, then **index.Set(version)**. For Delete, WAL then index (no datafile write).
- **Multi-partition path:** In `preparePartitionForCommit`, write all WAL records (including OpCommit) for that partition before any `dataFile.Write`. Then perform datafile writes and store offsets in `meta`, then in the apply phase set index as today.
- **Recovery:** Already correct (replay WAL → WriteNoSync to datafile → index.Set); no change needed.

After the fix, the documented invariant “WAL then index” should be updated to “WAL then datafile then index” wherever the flow is described.

---

## 4. Areas for Improvement

### 4.1 Codebase and Structure

| Area | Observation | Recommendation |
|------|-------------|----------------|
| **core.go size** | Very large (~2000+ lines); CRUD, recovery, 2PC, query, healing, collections all in one file. | Split into focused files: e.g. `crud_partition.go`, `recovery_partition.go`, `commit_2pc.go`, `query_exec.go`, and keep core.go for LogicalDB struct, Open/Close, and delegation. |
| **Duplicate validation** | JSON validation in IPC handler and again in `validateJSONPayload` in core; collection existence checked in multiple places. | Keep defense-in-depth but document “IPC first line, engine last line” and consider a small shared validation helper to avoid drift. |
| **Error re-exports** | `docdb/errors.go` and `types/errors.go` re-export from `internal/errors`; pool and IPC also re-export. | Plan a single canonical place (e.g. `internal/errors`) and migrate callers; deprecate re-exports to avoid duplicate symbols (e.g. `ErrDocExists` vs `ErrDocAlreadyExists`). |
| **Partition access** | `getPartition(id)` under RLock; workers already serialize per partition via mu. | Fine as-is; document that getPartition is for routing and that workers hold partition.mu for writes. |

### 4.2 Testing and Observability

| Area | Observation | Recommendation |
|------|-------------|----------------|
| **V0.4 tests** | V0.4 status marks unit/integration/benchmarks for partitions, recovery, and query as ⏳. | Add: (1) Test that recovery restores state after crash (single + multi-partition). (2) Test write ordering: crash after WAL, after datafile, after index (once order is fixed). (3) Query fan-out and filter/limit tests. |
| **Failure tests** | `tests/failure/` has compaction, corrupt record, partial WAL, etc. | Ensure at least one test that kills process during a write and verifies recovery and no orphan data (especially after fixing write order). |
| **Metrics** | Prometheus exporter and partition WAL fsync/replay metrics exist. | Add simple metrics for: per-partition task queue depth, commit latency for 2PC, and (if applicable) healing runs. |

### 4.3 Query Engine

| Area | Observation | Recommendation |
|------|-------------|----------------|
| **Projection** | Not implemented; always full payload. | Add optional field list in Query; filter payload (or a future columnar path) before return. |
| **Filter** | In-memory JSON field comparison only. | Document limitation; later consider nested paths and composite conditions. |
| **OrderBy** | Single field only. | Document; multi-field sort is a natural extension. |
| **Fan-out** | ExecuteQuery fans out to all partitions, then merge + filter + sort + limit in memory. | For large datasets, consider streaming or capped per-partition limit to avoid O(all docs) memory. |

### 4.4 Transactions and Consistency

| Area | Observation | Recommendation |
|------|-------------|----------------|
| **Multi-partition Commit** | 2PC with coordinator log and in-doubt resolution on recovery. | Add integration test: kill during prepare/commit and assert no partial visibility and correct coordinator replay. |
| **SSI-lite** | commitHistory and readSet used for conflict detection. | Add test that triggers serialization failure when two transactions conflict on read/write sets. |
| **Single-partition fast path** | Single-partition transactions avoid coordinator. | Document and add test that single-partition Commit does not write coordinator log. |

### 4.5 Documentation

| Area | Observation | Recommendation |
|------|-------------|----------------|
| **architecture.md** | Describes legacy single-WAL/single-datafile and v0.2/v0.3 features; does not fully reflect v0.4 (partitions, PartitionWAL, worker pool). | Update “Data Flow” and “Storage Architecture” to partition layout (walDir/dbName/p{n}.wal, dbname_p{n}.data), and “Concurrency Model” to one writer per partition and lock-free reads. |
| **ondisk_format.md** | Documents older WAL record layout. | Add v0.4 WAL format (LSN, PayloadCRC, collection length + name) and reference partition WAL and checkpoint files. |
| **transactions.md** | Good; references future_multi_partition_transactions. | Add one subsection on “Single-operation vs multi-doc Commit” and point to commit ordering (WAL → datafile → index) once fixed. |
| **Implementation status** | Several docs (README_STATUS, ROADMAP, IMPLEMENTATION_STATUS) overlap and some phases are outdated. | Consolidate into a single “DocDB status” doc (or clearly layered: one “current release” + one “roadmap”) and trim duplicate checklists. |

### 4.6 Configuration and Deployment

| Area | Observation | Recommendation |
|------|-------------|----------------|
| **Pool OpenDB** | Always opens with `PartitionCount = 1` (pool.go:266). | If multi-partition is desired from pool, add config or catalog field for partition count per DB; otherwise document that pool-opened DBs are single-partition. |
| **Defaults** | LogicalDBConfig defaults (e.g. PartitionCount=16 in config) vs NewLogicalDB() forcing 1. | Make default partition count explicit in one place (e.g. config.DefaultLogicalDBConfig()) and document “server vs pool” defaults. |

---

## 5. Document Flow Checklist (Correctness)

| Step | Status | Notes |
|------|--------|--------|
| JSON validated at IPC | ✅ | handler.go validates Create/Update payloads |
| JSON validated in engine | ✅ | validateJSONPayload in core |
| Collection ensured before write | ✅ | EnsureCollection in execute*OnPartition |
| Routing by docID deterministic | ✅ | RouteToPartition = docID % partitionCount |
| Write serialization per partition | ✅ | partition.mu in worker pool |
| Read path lock-free | ✅ | executeReadOnPartitionLockFree, snapshot |
| WAL before index (intended) | ❌ | Currently datafile → WAL → index; should be WAL → datafile → index |
| Recovery replays WAL only | ✅ | replayPartitionWALForPartition applies WAL → datafile → index |
| Checkpoint per partition | ✅ | PartitionCheckpointManager, p{n}.chk |
| Coordinator log for 2PC | ✅ | Replay on open; in-doubt resolved from decision log |

---

## 6. Recommendations Summary

1. **Fix write ordering (high):** Implement WAL-before-datafile in all write paths (single-op and preparePartitionForCommit); then add a test that simulates crash and verifies no orphan data and correct recovery.
2. **Refactor core.go (medium):** Split by concern (CRUD, recovery, 2PC, query, healing) for readability and testing.
3. **Test coverage (medium):** Add partition recovery, query fan-out, and 2PC crash tests; align with V0.4 testing section.
4. **Docs (medium):** Update architecture and on-disk format for v0.4; consolidate implementation status; document commit order (WAL → datafile → index).
5. **Error surface (low):** Unify on internal/errors and reduce re-exports/duplicate error names.
6. **Query and config (low):** Document limitations; consider projection and pool partition count configuration when scaling.

---

*This review is based on the codebase and docs as of the review date. Implementation details (e.g. line numbers) refer to the then-current state.*

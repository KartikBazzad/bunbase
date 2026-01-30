# DocDB Bottleneck Analysis

## Overview

Under high connection load (e.g., 20 DBs × 50 conn/DB = 1000 concurrent connections), latency increases (P95 ~146 ms, P99 ~225 ms) while throughput remains ~750 ops/sec. This analysis identifies the key bottlenecks causing latency growth and proposes fixes.

---

## Bottleneck #1: Commit Mutex (commitMu) – Highest Impact

**Location:** `docdb/internal/docdb/core.go` line 90

```go
commitMu sync.Mutex // Serializes commit + conflict check + append to history
```

**Problem:** In the `Commit()` function (line 1984), ALL commits for a single LogicalDB are serialized under `commitMu`:

```go
// SSI-lite: serialize commit, check conflicts, then append to history on success
t0 := time.Now()
db.commitMu.Lock()
metrics.RecordCommitMuWait(db.dbName, time.Since(t0))
holdStart := time.Now()
defer func() {
    metrics.RecordCommitMuHold(db.dbName, time.Since(holdStart))
    db.commitMu.Unlock()
}()
writeSet := computeWriteSet(tx.Operations)
if db.commitHistory != nil {
    recs := db.commitHistory.CommitsAfter(tx.SnapshotTxID)
    for _, rec := range recs {
        if hasConflict(tx.readSet, writeSet, rec.readSet, rec.writeSet) {
            return ErrSerializationFailure
        }
    }
}
```

**Why it bottlenecks:**
- Under 20 DBs × 50 conn/DB (1000 concurrent connections), commits serialize per DB.
- The work under `commitMu` includes:
  1. Computing write set
  2. Fetching ALL commits after snapshot (O(n) via `CommitsAfter`)
  3. Checking conflicts against ALL recent commits (O(n) via nested loops)
  4. Grouping ops by partition
  5. Writing WAL and updating index
- Even with multiple workers, the per-DB commit serialization prevents full parallelism.

**Metrics to confirm:** Capture `/metrics` during a 20db_50conn_2w run and look at:
- `docdb_commit_mu_wait_seconds` – time waiting to acquire `commitMu`
- `docdb_commit_mu_hold_seconds` – time holding `commitMu`

If wait times dominate, `commitMu` is the bottleneck. If hold times dominate, the work under it (conflict check) is the bottleneck.

---

## Bottleneck #2: CommitHistory.CommitsAfter() – O(n) Scan

**Location:** `docdb/internal/docdb/commit_history.go` lines 47–56

```go
func (h *CommitHistory) CommitsAfter(snapshotTxID uint64) []commitRecord {
    h.mu.Lock()
    defer h.mu.Unlock()
    var out []commitRecord
    for _, rec := range h.records {  // O(n) scan
        if rec.txID > snapshotTxID {
            out = append(out, rec)
        }
    }
    return out
}
```

**Problem:** This function is called on every commit (under `commitMu`), and it:
- Locks the commit history `mu`
- Scans ALL records (up to 100,000 by default)
- Copies all qualifying records into a new slice
- Then the caller does nested O(n) conflict checks via `hasConflict()`

**Why it bottlenecks:**
- Records are stored in order by `txID` (monotonically increasing), but the implementation does a linear scan.
- With high commit throughput (e.g., 750 ops/sec × 20 DBs = 15,000 commits/sec), this lock+scan becomes expensive.
- Each call holds `commitMu` longer, increasing wait times for other commits.

**Fix:** Binary search to find first `txID > snapshotTxID`, then iterate only from there. Since `records` are ordered by txID:

```go
func (h *CommitHistory) CommitsAfter(snapshotTxID uint64) []commitRecord {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    // Binary search for first txID > snapshotTxID
    lo, hi := 0, len(h.records)
    for lo < hi {
        mid := (lo + hi) / 2
        if h.records[mid].txID > snapshotTxID {
            hi = mid
        } else {
            lo = mid + 1
        }
    }
    
    // Copy from lo to end (only relevant records)
    out := make([]commitRecord, len(h.records)-lo)
    copy(out, h.records[lo:])
    return out
}
```

**Impact:** Reduces conflict check from O(n) to O(log n + k) where k = records after snapshot. For 100,000 records, worst case goes from ~100,000 comparisons to ~17 (log₂ 100k) + number of relevant records.

---

## Bottleneck #3: Partition Mutex (partition.mu) – Write Serialization

**Location:** `docdb/internal/docdb/partition.go` line 27

```go
mu sync.Mutex // Write serialization (exactly one writer at a time)
```

**Problem:** All writes to a partition lock `partition.mu` (line 230 in `worker_pool.go`):

```go
lockStart := time.Now()
partition.mu.Lock()
defer partition.mu.Unlock() // Ensure unlock on panic
lockWait := time.Since(lockStart)
metrics.RecordPartitionLockWait(w.db.Name(), strconv.Itoa(partition.ID()), lockWait)
```

**Why it bottlenecks:**
- Workers wait on `partition.mu` before they can execute any write.
- While partition lock is held, other workers targeting the same partition are blocked.
- If worker count is high and many requests target the same partition (uneven hash distribution), this becomes a hotspot.

**Metrics to confirm:**
- `docdb_partition_lock_wait_seconds` – time workers spend waiting to acquire `partition.mu`

**Potential fixes (lower priority):**
- Increase partition count per DB (`-partition-count` CLI flag or `DefaultPartitionCount` config).
- More partitions spread writes across more locks, reducing per-lock contention.
- Current default is 1 partition per DB; increasing to 2–4 can improve write concurrency.

---

## Bottleneck #4: Index Shard Locks – Read Contention

**Location:** `docdb/internal/docdb/index.go` lines 33–35

```go
// Each shard is protected by its own RWMutex to enable:
// - Concurrent reads from different goroutines
// - Serialized writes within a shard
mu sync.RWMutex
```

**Problem:** Each `IndexShard` has its own `RWMutex`. Reads use `RLock()`:

```go
func (s *IndexShard) Get(docID uint64, snapshotTxID uint64) (*types.DocumentVersion, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    // ...
}
```

**Why it bottlenecks:**
- Many workers reading from the same shard concurrently contend on the same `RLock()`.
- Sharding uses `doc_id % num_shards`, so distribution is uniform in expectation but can be skewed for specific access patterns.
- While RWMutex allows multiple readers, high contention still causes latency.

**Impact:** Lower than commit mutex or partition lock because reads don't wait for writers, but P95/P99 can be affected under heavy read-heavy workloads.

**Potential fix (lower priority):**
- Increase shard count (currently 512).
- Or optimize read path to use lock-free techniques (e.g., atomic pointers + version checks).

---

## Bottleneck #5: IPC Handler – One Op at a Time

**Location:** `docdb/internal/ipc/handler.go` lines 164–206

```go
case CmdExecute:
    // ...
    var wg sync.WaitGroup
    for i := range frame.Ops {
        top := &frame.Ops[i]
        // ...
        wg.Add(1)
        go func(idx int, o *Operation) {
            defer wg.Done()
            req := &pool.Request{
                DBID:       frame.DBID,
                Collection: o.Collection,
                DocID:      o.DocID,
                OpType:     o.OpType,
                Payload:    o.Payload,
                PatchOps:   o.PatchOps,
                Response:   make(chan pool.Response, 1),
            }
            h.pool.Execute(req)
            resp := <-req.Response
            statuses[idx] = resp.Status
            if resp.Error != nil {
                responses[idx] = []byte(resp.Error.Error())
            } else if resp.Data != nil {
                responses[idx] = resp.Data
            }
        }(i, op)
    }
    wg.Wait()
```

**Problem:** Each operation in a batch is sent to the pool sequentially as a separate request. Even with the "fast path" for 1 scheduler worker, each operation goes through:
1. `pool.Execute()` → `pool.sched.Enqueue()`
2. Scheduler worker picks from queue → `pool.handleRequest()`
3. `db.SubmitTaskAndWait(task)` → worker pool → partition → execute

**Why it bottlenecks:**
- The IPC handler launches N goroutines but each operation still goes through the full request pipeline.
- No true pipelining: the client waits for the entire batch to complete before sending the next.
- With 20 DBs × 50 conn/DB, this pipeline depth adds up.

**Impact:** Contributes to latency but not the primary bottleneck (since scheduler and worker pools are already parallel).

**Potential fix (lower priority):**
- True pipelining: allow multiple in-flight batches per connection.
- Or batch at a higher level (e.g., all ops targeting the same DB are grouped).

---

## Bottleneck #6: CommitHistory.Append() – Lock Contention

**Location:** `docdb/internal/docdb/commit_history.go` lines 59–77

```go
func (h *CommitHistory) Append(txID uint64, readSet, writeSet map[string]struct{}) {
    r := commitRecord{
        txID:     txID,
        readSet:  make(map[string]struct{}),
        writeSet: make(map[string]struct{}),
    }
    for k := range readSet {
        r.readSet[k] = struct{}{}
    }
    for k := range writeSet {
        r.writeSet[k] = struct{}{}
    }
    h.mu.Lock()
    defer h.mu.Unlock()
    h.records = append(h.records, r)
    for len(h.records) > h.maxSize {
        h.records = h.records[1:]
    }
}
```

**Problem:** `Append()` is called on every commit (after work under `commitMu` completes), so it holds `commitHistory.mu` in addition to `commitMu`.

**Why it bottlenecks:**
- Serializing maps (copying keys) under the lock adds overhead.
- The circular buffer drop (`h.records = h.records[1:]`) copies all remaining records.
- With high commit rate, `commitHistory.mu` becomes a contention point.

**Impact:** Secondary to `CommitsAfter()` since it happens after the commit work, but can add to overall commit latency.

**Potential fix (lower priority):**
- Use ring buffer with head/tail indices to avoid copying.
- Or reduce max size (100,000) if conflict detection window doesn't need that many records.

---

## Priority Ranking

| Rank | Bottleneck | Impact | Effort | Fix |
|-------|-------------|--------|--------|
| 1 | `CommitsAfter()` O(n) scan | **High** – under `commitMu`, called every commit | Low | Binary search implementation |
| 2 | `commitMu` serialization | **High** – per-DB lock for all commits | Medium | Partition commits by time window; reduce conflict window size |
| 3 | `partition.mu` write contention | Medium | Medium | Increase partition count per DB; optimize work under lock |
| 4 | `CommitHistory.Append()` lock + map copy | Medium | Low | Ring buffer; smaller history window |
| 5 | Index shard read locks | Low–Medium | High | Lock-free reads; more shards |
| 6 | IPC handler sequential op dispatch | Low | High | True pipelining; batch grouping |

---

## Recommended Actions

### Immediate (implement and profile)

1. **Optimize `CommitsAfter()` with binary search**
   - File: `docdb/internal/docdb/commit_history.go`
   - Change the linear scan to binary search since records are ordered by txID.
   - Profile: Compare `docdb_commit_mu_hold_seconds` before/after.
   - Expected: 10–20× reduction in conflict check time for 100k records.

2. **Profile under 20db_50conn_2w**
   - Run server with `-debug-addr localhost:6060`
   - Capture `/metrics` endpoint every few seconds during the run
   - Capture CPU + mutex profiles for 30 seconds
   - Compare:
     - `docdb_commit_mu_wait_seconds` vs `docdb_commit_mu_hold_seconds`
     - `docdb_partition_lock_wait_seconds` (per partition)
     - `docdb_partition_wal_fsync_seconds` (WAL I/O)
     - `docdb_partition_datafile_sync_seconds` (datafile I/O)

### After profiling (choose based on data)

3. **If commit mutex hold dominates:**
   - Consider reducing conflict window (e.g., only last 1,000 or 5,000 commits)
   - Add config option `ConflictCheckWindow` and cap `CommitHistory.maxSize` for that.
   - This reduces time spent in `CommitsAfter()` and `hasConflict()`.

4. **If partition lock wait dominates:**
   - Increase partition count per DB: `DefaultPartitionCount = 2` or `4`.
   - More partitions → more `partition.mu` locks → better parallelism.
   - Trade-off: More WAL files, higher recovery work, but higher write throughput.

5. **If WAL/datafile fsync dominates:**
   - Tune group commit: `WAL.Fsync.IntervalMS` and `MaxBatchSize`.
   - Current defaults: 1 ms interval, 100 batch size.
   - Try 2–5 ms interval or 200–500 batch size to reduce fsync frequency.

6. **If index shard locks dominate:**
   - Consider lock-free read path (e.g., atomic pointer with version check).
   - Or increase shard count beyond 512 (e.g., 1024 or 2048).

### Longer-term (larger effort)

7. **Batch IPC requests better**
   - Group ops by DB + partition before dispatching, not just sequential per-op.
   - Reduce scheduler enqueue overhead per operation in a large batch.

8. **Multi-commit coordination**
   - Allow multiple transactions to commit concurrently (e.g., optimistic conflict detection).
   - Replace or augment `commitMu` with finer-grained locking (per doc key ranges).

---

## Summary

The highest-impact bottleneck is **`CommitsAfter()` O(n) scan under `commitMu`**, which makes conflict checking expensive at high commit rates. Fixing it with binary search is straightforward and should yield significant latency reduction (10–20× for 100k records).

Secondary bottlenecks:
- `commitMu` serialization (mitigated by reducing work under it via optimized `CommitsAfter()` and/or smaller conflict window)
- `partition.mu` contention (mitigated by more partitions per DB)
- `CommitHistory.Append()` map copy (mitigated by ring buffer or smaller window)
- Index shard locks (mitigated by lock-free techniques or more shards)

Profile first to confirm, then implement the binary search fix to `CommitsAfter()` as the first optimization.

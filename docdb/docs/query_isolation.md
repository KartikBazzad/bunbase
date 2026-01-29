# Query Isolation Guarantees

This document defines the query snapshot isolation contract for DocDB v0.4.

## Snapshot Acquisition

- A query acquires a **single snapshot** at the start of execution via one `mvcc.CurrentSnapshot()` call.
- The snapshot is a transaction ID (txID) that defines the visibility cutoff: only changes committed at or before this txID are visible.

## Visibility

- **Writers after snapshot are invisible**: Any write (create, update, delete) committed after the query’s snapshot txID is not visible to that query.
- A query sees a **consistent point-in-time view** of the database as of its snapshot.

## Partition Consistency

- In partitioned mode, **all partitions use the same snapshot txID** for a given query.
- The coordinator acquires the snapshot once and passes it to all partition scans, so the merged result is consistent across partitions.

## Read-Only

- Queries are **read-only**: they never block writers.
- Reads use lock-free index snapshots; writers hold partition locks only for the duration of their write.
- A long-running query does not prevent concurrent creates, updates, or deletes on any partition.

## Limits

- **Memory cap**: Total bytes scanned (row payloads) is capped (e.g. 100MB default) in both **partitioned and legacy** query paths. Exceeding the cap returns `ErrQueryMemoryLimit`.
- **Time limit**: Query execution is bound by a context timeout (e.g. 30s). Cancellation or timeout returns the context error.
- **Row limit**: The query spec’s `limit` caps the number of rows returned; streaming merge can stop early when the limit is reached. Client-supplied `limit` is **clamped** to the configured `MaxQueryLimit` (e.g. 10000); values above the cap are reduced to the cap rather than rejected.

## Summary

| Property              | Guarantee                                                             |
| --------------------- | --------------------------------------------------------------------- |
| Snapshot              | Single txID at query start                                            |
| Visibility            | Only commits ≤ snapshot txID visible                                  |
| Partition consistency | Same snapshot txID on all partitions                                  |
| Read-only             | Queries do not block writers                                          |
| Memory                | Capped bytes scanned; over cap → error                                |
| Time                  | Context timeout; over → cancellation error                            |
| Row limit             | Spec limit (clamped to MaxQueryLimit); early termination when reached |

# Horizontal Scaling and Sharding

DocDB supports two levels of distribution:

1. **Within a LogicalDB (v0.4):** Partitioned execution — documents are routed to partitions by `docID`; workers are not bound to partitions; exactly one writer per partition at a time.
2. **Across LogicalDBs:** Multiple databases in one process; each LogicalDB is a single-writer (or partitioned) executor.

This document describes the scaling model, partition routing, and recommended configuration.

## Partition routing (v0.4, within one LogicalDB)

When `PartitionCount > 1`, document IDs are mapped to partitions deterministically:

- **Routing:** `partitionID = docID % PartitionCount` (see `RouteToPartition` in `docdb/routing.go`).
- **Workers** pull tasks from a shared queue and lock the target partition for writes; reads are lock-free (snapshot).
- **Query** fans out to all partitions and merges results (filter/order/limit applied after merge).

No application change is required: the client sends operations by `docID`; the server routes to the correct partition.

## Execution model (multiple LogicalDBs)

- Each **LogicalDB** is either legacy single-writer or v0.4 partitioned (many partitions, one writer per partition).
- Concurrency and throughput come from **many LogicalDBs in parallel** and/or more partitions per DB.
- **Cross-DB sharding:** We do not provide a built-in sharding API across LogicalDBs. Applications that partition data across multiple DBs are responsible for routing (e.g., which DB to open and use). A simple pattern: `dbIndex := key % len(dbNames)` when opening or choosing a DB by key.

## Scaling Guidelines

Based on the scaling matrix results (see `matrix_results/reports/analysis.md`):

- **Workers per DB:** Keep **1** (enforced by default; see configuration).
- **Connections per DB:** Use **8–16 client connections per LogicalDB** to keep the single worker fed.
- **Throughput:** Increase the number of LogicalDBs (more DBs in the pool), not workers per DB.

Example shape:

```text
12 LogicalDBs × 1 worker each × 16 connections per DB
```

## Summary

- **v0.4:** One LogicalDB can use multiple partitions; routing is by `docID % PartitionCount`; query fans out and merges.
- **Multi-DB:** Scale by adding more LogicalDBs; use multiple connections per DB to keep workers fed.
- **Cross-DB routing** is application-defined (e.g. `key % N` to choose a DB).

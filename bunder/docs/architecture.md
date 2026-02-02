# Bunder Architecture

## Overview

Bunder is a Redis-like KV database with the following layers:

1. **Client layer**: RESP (TCP), HTTP API, CLI REPL
2. **Handler layer**: Command dispatch to KV, Lists, Sets, Hashes, TTL
3. **Data layer**: Sharded in-memory map + optional persistent B+Tree
4. **Storage layer**: Pager (4KB pages), Buffer pool (SLRU), Free list
5. **Persistence**: WAL, Snapshots (RDB-like), AOF
6. **Pub/Sub**: Buncast integration for DML events

## Data Flow

- **GET**: Sharded map lookup (O(1) in-memory); optional B+Tree if cold
- **SET**: Update sharded map, append to WAL, insert/update B+Tree
- **DEL**: Remove from sharded map, WAL, B+Tree
- **Lists/Sets/Hashes**: In-memory structures keyed by name; stored as blobs in KV if persisted

## Storage

- **Pager**: Single file `data.db`, 4KB pages, sequential allocation or freelist reuse
- **Buffer pool**: SLRU (80% protected, 20% probation), configurable capacity
- **B+Tree**: Order 32, leaf linking for range scans, root persisted in meta

## Concurrency

- **Sharded map**: 256 shards by FNV hash of key; reduces lock contention
- **B+Tree**: Single mutex per tree (simplified); page latches available for future use
- **TTL**: Timing wheel with per-slot locks; sweeper goroutine

## Persistence

- **WAL**: Segment files, length-prefixed records (SET/DEL/EXPIRE), CRC32, rotation at 64MB
- **Snapshot**: Full dump of KV pairs to `.rdb` file; for backup/restore
- **AOF**: Append-only log of commands; replay on startup (optional)

## Configuration

- `DataPath`: Directory for `data.db`, `wal/`, `snapshots/`, `appendonly.aof`
- `BufferPoolSize`: Number of 4KB pages in buffer pool (default 10000 = 40MB)
- `Shards`: Number of shards (default 256)
- `ListenAddr`: TCP address (default :6379)
- `HTTPAddr`: HTTP API address (default :8080)
- `TTLCheckInterval`: How often to sweep expired keys (default 1s)
- `BuncastEnabled` / `BuncastAddr`: Pub/sub via Buncast Unix socket

## Implementation

For file layout, data formats, and extension points, see [implementation.md](implementation.md).

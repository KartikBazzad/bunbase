# Bunder Implementation Guide

This document describes the implementation details of Bunder: file layout, data structures, formats, and design choices.

## Directory Layout

```
bunder/
├── cmd/
│   ├── server/main.go       # TCP + HTTP server entrypoint; parses flags, starts Server
│   └── cli/main.go          # Interactive REPL; sends RESP arrays, prints responses
├── pkg/
│   └── client/client.go     # Go client: Connect, Get, Set, Delete, Keys, Do; RESP send/read
├── internal/
│   ├── loadtest/            # Load testing: Run(ctx, cfg), Stats/Report, workload set/get/mixed
│   │   ├── loadtest.go      # Run(), runWorker(); Config (Addr, Duration, NumClients, KeySpace, ValueSize, Workload)
│   │   ├── stats.go         # Stats (Record, Ops, Errors, Report); Report (TotalOps, OpsPerSec, P50/P95/P99)
│   │   └── loadtest_test.go # TestLoadTest_InProcess, SetOnly, GetOnly; BenchmarkLoadTest_Mixed
│   ├── config/config.go     # Config struct and ParseFlags(); defaults for data path, ports, shards
│   ├── storage/             # Low-level disk and cache
│   │   ├── page.go          # PageID, PageSize (4KB), Page struct and header accessors
│   │   ├── pager.go         # File I/O for fixed-size pages; AllocatePage, ReadPage, WritePage
│   │   ├── buffer_pool.go   # SLRU cache over Pager; FetchPage, NewPage, UnpinPage, FlushAllPages
│   │   ├── freelist.go      # Free page IDs; Allocate/Free; persisted on page 1
│   │   ├── btree.go         # B+Tree (order 32); Put, Get, RangeScan; leaf/branch splits
│   │   └── errors.go        # ErrInvalidPageID, ErrPageNotFound, ErrPageFull, etc.
│   ├── concurrency/
│   │   ├── sharded_map.go   # 256-shard map; FNV hash; Get, Set, Delete, Exists, Keys
│   │   └── latch.go         # Per-page RWMutex pool for B+Tree (for future fine-grained locking)
│   ├── data_structures/
│   │   ├── kv_store.go      # OpenKVStore; Get, Set, Delete, Exists, Keys(pattern), Close
│   │   ├── list.go          # List: LPush, RPush, LPop, RPop, LRange, LLen, LIndex, LSet, LTrim
│   │   ├── set.go           # Set: SAdd, SRem, SMembers, SIsMember, SCard, SInter, SUnion, SDiff
│   │   └── hash.go          # Hash: HSet, HGet, HGetAll, HDel, HExists, HLen, HKeys, HVals
│   ├── wal/
│   │   ├── record.go        # RecordType (Set/Del/Expire), LSN, Encode/Decode with CRC32
│   │   ├── segment.go       # Single WAL file; write (length+record), readRecords
│   │   ├── wal.go           # WAL: Append, Sync, ReadAllRecords; segment rotation at 64MB
│   │   └── recovery.go      # Recover(w) returns all records for replay
│   ├── ttl/
│   │   ├── wheel.go         # TimingWheel: slots by time; Add, Remove, Expired(now)
│   │   └── manager.go       # TTL Manager: Set/Get/Remove/TTLSeconds; sweeper goroutine
│   ├── persistence/
│   │   ├── snapshot.go      # Write(entries) to .rdb; ReadSnapshot(path) for restore
│   │   └── aof.go           # Append(cmd, args); ReadAOF(path) for replay
│   ├── pubsub/
│   │   └── integration.go   # PubSubManager: PublishOperation to Buncast topic bunder.operations
│   ├── server/
│   │   ├── resp.go          # ReadRESP, ParseCommand, WriteRESP (RESP2 types)
│   │   ├── handler.go       # Handler: Exec(cmd, args) dispatches to KV/lists/sets/hashes/TTL
│   │   ├── server.go        # Server: NewServer (opens KV, TTL, pubsub), Start (TCP + HTTP), Close
│   │   └── http.go          # HTTPHandler: /health, /metrics, /kv/:key, /keys, /subscribe
│   └── metrics/
│       └── metrics.go       # Counters/gauges; PrometheusFormat() for /metrics
├── docs/
│   ├── architecture.md     # Layers, data flow, storage, concurrency, persistence
│   ├── api-reference.md     # RESP commands, HTTP endpoints, Go client
│   └── implementation.md   # This file
├── go.mod
└── README.md
```

## Data Formats

### Page (4KB)

- **Header** (30 bytes): Type(1), Flags(1), KeyCount(2), FreeSpace(2), LSN(8), NextPage(8), PrevPage(8).
- **Leaf**: entries as KeyLen(2)+Key+ValueLen(2)+Value packed from PageHeaderSize.
- **Branch**: LeftPtr(8) then entries KeyLen(2)+Key+ValLen(2)+ChildPageID(8).

### WAL Record

- **Header** (21 bytes): CRC32(4), LSN(8), Type(1), KeyLen(4), ValueLen(4).
- **Body**: Key + Value.
- **Segment**: 4-byte little-endian record length + record bytes; files `wal-0000000000000000.log`, etc.

### List / Set / Hash (in-memory serialization)

- **List**: count(4) + for each element len(4)+data.
- **Set**: count(4) + sorted keys, each len(4)+data (deterministic for replay).
- **Hash**: count(4) + sorted field names, each keyLen(4)+key+valLen(4)+val.

### Snapshot (RDB-like)

- **File**: count(8) + for each key-value keyLen(4)+key+valLen(4)+value.

### AOF

- **Record**: recLen(4) + cmdLen(4)+cmd + argc(4) + for each arg len(4)+data.

## Key Design Choices

- **Sharded map**: Primary KV store is in-memory with 256 shards to reduce lock contention; B+Tree provides persistent index and optional cold-path read.
- **B+Tree**: Single mutex per tree; page latches (latch.go) are in place for future fine-grained concurrency.
- **TTL**: Timing wheel groups keys by expiry slot; sweeper runs at TTLCheckInterval and calls onExpire (e.g. delete from KV).
- **Lists/Sets/Hashes**: Stored in server handler as per-key in-memory structs; not yet persisted to KV/B+Tree in this implementation (can be added by serializing to blob and SET).
- **WAL**: No per-command fsync by default; Sync() is explicit or can be tied to batch/interval for durability vs throughput.

## Load Tests

- **internal/loadtest**: Run(ctx, Config) spawns NumClients workers; each runs GET/SET (or set-only/get-only) for Duration against Addr. Stats collect latencies; Report() returns TotalOps, Errors, OpsPerSec, P50/P95/P99 latency.
- **cmd/loadtest**: CLI that runs a load test against -addr with -duration -clients -workload (set|get|mixed), -keys, -value-size. Prints ops/sec and latency percentiles.
- **Tests**: TestLoadTest_InProcess starts a server in-process (ListenAddr "127.0.0.1:0"), runs Run(), checks TotalOps > 0. TestLoadTest_SetOnly and TestLoadTest_GetOnly do the same for set-only and get-only (get-only pre-populates 100 keys). BenchmarkLoadTest_Mixed runs a 1s mixed workload and reports ops/sec, P50_us, P99_us.

## Extending the Implementation

- **Persistence of lists/sets/hashes**: On SET of a special key prefix (e.g. `list:`, `set:`, `hash:`), serialize the structure with Bytes() and store via KV.Set. On GET, LoadFromBytes into the in-memory struct.
- **WAL integration in KV path**: In KVStore.Set/Delete, append a WAL record and optionally Sync(); on startup, Recover() and replay into KV (and optionally B+Tree).
- **Metrics in handler**: Call metrics.Default().IncGet() etc. in Handler.Exec for each command type.
- **Health checks**: Extend /health to verify pager, WAL, or TTL manager reachability if needed.

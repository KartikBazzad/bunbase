# DocDB

> ⚠️ **Disclaimer:** This is a **toy / learning system**, not a production database. Correctness > features. Simplicity > generality.

DocDB is a file-based, ACID document database written in Go. It supports multiple logical databases in one runtime, avoids SQLite's single-writer collapse, and enables concurrent writes through scheduler + MVCC-lite.

## Features

- **File-based storage**: Append-only data files with WAL (Write-Ahead Log)
- **ACID transactions**: Atomic, Consistent, Isolated (snapshot reads), Durable
- **Concurrent writes**: Sharded in-memory index with per-shard locking
- **Multiple databases**: Manage isolated logical databases in one runtime
- **MVCC-lite**: Versioned documents with snapshot reads
- **IPC interface**: Unix domain socket (default) for client communication
- **Bounded memory**: Memory caps per-DB and globally
- **Crash recovery**: Survives `kill -9` via WAL replay

## Non-Goals (Explicitly Out of Scope)

This project will NOT support:

- SQL
- Joins
- Arbitrary queries
- Secondary indexes (v0)
- Cross-database transactions
- Distributed replication
- Query planner / optimizer
- Long-running transactions

If any of these appear, **the project scope has failed**.

## Current Status

**Version**: v0.3

**Status**: v0.3 features complete. Production-ready with ants goroutine pools, WAL group commit, faster replay, and enhanced observability.

**What Works**:

- ✅ ACID transactions with WAL
- ✅ Sharded in-memory index
- ✅ MVCC-lite snapshot reads
- ✅ Multiple isolated databases
- ✅ Crash recovery via WAL replay
- ✅ Bounded memory management
- ✅ Unix socket IPC
- ✅ Interactive shell
- ✅ Go and TypeScript clients
- ✅ **v0.2:** Collections support (create, delete, list)
- ✅ **v0.2:** Path-based updates (patch operations)
- ✅ **v0.2:** Automatic document healing
- ✅ **v0.2:** WAL rotation and trimming
- ✅ **v0.2:** Checkpoint-based recovery
- ✅ **v0.2:** Error classification and retry
- ✅ **v0.2:** Prometheus/OpenMetrics metrics
- ✅ **v0.3:** Ants goroutine pool (scheduler, healing, optional IPC)
- ✅ **v0.3:** WAL group commit (batched fsync)
- ✅ **v0.3:** Fast WAL replay (single-pass, deferred datafile sync)
- ✅ **v0.3:** Scheduler fairness under skewed workloads
- ✅ **v0.3:** WAL/group-commit and ants pool metrics

**v0.3 Features**:

- ✅ Ants library integration for scheduler workers (recycling, panic recovery, expiry)
- ✅ WAL group commit wired (NewWriterFromConfig, FsyncConfig)
- ✅ Single-pass WAL replay and WriteNoSync during replay
- ✅ Queue-depth–aware scheduling (PickNextQueue)
- ✅ Parallel healing via ants PoolWithFunc
- ✅ Optional IPC connection pool (MaxConnections)
- ✅ Ants pool and WAL group-commit metrics in GetSchedulerStats()

**See [V0.3_IMPLEMENTATION_PLAN.md](docs/implementation-status/docdb/V0.3_IMPLEMENTATION_PLAN.md) and [V0.3_STATUS.md](docs/implementation-status/docdb/V0.3_STATUS.md).**

## Document Contract

**DocDB stores JSON documents.**
Every document is a valid UTF-8 encoded JSON value.
Binary data is supported only as explicitly encoded JSON values.

### Allowed Document Types

- JSON object: `{"key":"value"}`
- JSON array: `[1,2,3]`
- JSON string: `"hello"`
- JSON number: `42`
- JSON boolean: `true`
- JSON null: `null`

### Binary Data Encoding

Binary data MUST be encoded within JSON. Recommended format:

```json
{
  "_type": "bytes",
  "encoding": "base64",
  "data": "SGVsbG8gd29ybGQ="
}
```

### Client Helper Utilities

Client libraries MAY provide helper utilities for encoding/decoding binary data
using canonical JSON wrapper format. The engine and IPC layer remain agnostic.

## Quick Start

### Build

```bash
go build -o docdb ./cmd/docdb
```

### Run Server

```bash
./docdb --data-dir ./data --socket /tmp/docdb.sock
```

### Use Go Client

```go
package main

import (
    "fmt"
    "github.com/kartikbazzad/docdb/pkg/client"
)

func main() {
    cli := client.New("/tmp/docdb.sock")

    // Open database
    dbID, err := cli.OpenDB("mydb")
    if err != nil {
        panic(err)
    }

    // Create document (must be valid JSON)
    err = cli.Create(dbID, 1, []byte(`{"name":"Alice","age":30}`))
    if err != nil {
        panic(err)
    }

    // Read document
    data, err := cli.Read(dbID, 1)
    if err != nil {
        panic(err)
    }

    fmt.Println(string(data)) // Output: {"name":"Alice","age":30}

    // Update document
    err = cli.Update(dbID, 1, []byte("updated"))
    if err != nil {
        panic(err)
    }

    // Delete document
    err = cli.Delete(dbID, 1)
    if err != nil {
        panic(err)
    }

    // Close database
    err = cli.CloseDB(dbID)
    if err != nil {
        panic(err)
    }
}
```

## Architecture

```
┌──────────────┐
│ JS / Bun App │
└──────┬───────┘
       │ IPC (Unix Socket)
┌──────▼────────┐
│  DocDB Pool   │   ← single process (Go)
│  (Runtime)    │
├────────────────┤
│ Scheduler      │
│ WAL Manager    │
│ Memory Manager │
│ Catalog (Meta) │
├────────────────┤
│ Logical DBs    │
│  ├─ db_1       │
│  ├─ db_2       │
│  └─ db_n       │
└────────────────┘
```

## Core Concepts

| Concept         | Meaning                                   |
| --------------- | ----------------------------------------- |
| **DocDB Pool**  | One runtime managing many logical DBs     |
| **Logical DB**  | Isolated document namespace (per project) |
| **Document**    | Valid UTF-8 encoded JSON value            |
| **Transaction** | Short-lived atomic write group            |
| **WAL**         | Global append-only write-ahead log        |
| **MVCC-lite**   | Versioned documents, snapshot reads       |

## Operations

| Operation | Description                           |
| --------- | ------------------------------------- |
| `create`  | Insert new document                   |
| `read`    | Fetch document by id                  |
| `update`  | Replace full document                 |
| `delete`  | Tombstone document                    |
| `scan`    | Optional sequential scan (v0 limited) |

## Testing

Run tests:

```bash
# Run all tests
go test ./...

# Run specific test suites
go test ./tests/integration
go test ./tests/failure
go test ./tests/concurrency

# Run benchmarks
go test -bench=. ./tests/benchmarks
```

## Performance Targets (Toy-Realistic)

| Metric             | Target                    |
| ------------------ | ------------------------- |
| Concurrent writers | 10–100                    |
| P95 latency        | < 100ms (disk permitting) |
| P99 latency        | Bounded                   |
| Throughput         | Disk-bound                |
| Startup time       | < 500ms (small DBs)       |

## Documentation

- [Usage Guide](docs/usage.md) - Comprehensive usage examples for Go and TypeScript
- [Configuration Guide](docs/configuration.md) - All configuration options and recommended settings
- [On-Disk Format](docs/ondisk_format.md) - Binary format specifications
- [Failure Modes](docs/failure_modes.md) - Failure handling and recovery
- [Architecture](docs/architecture.md) - System design and component interactions
- [TypeScript Client](tsclient/README.md) - TypeScript client documentation
- [TS Implementation](tsclient/IMPLEMENTATION_SUMMARY.md) - Detailed TS client status
- [Roadmap](ROADMAP.md) - Planned features and evolution path
- [Progress](PROGRESS.md) - Implementation status and recent fixes

## License

MIT License - see LICENSE file for details.

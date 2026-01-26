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
    
    // Create document
    err = cli.Create(dbID, 1, []byte("hello world"))
    if err != nil {
        panic(err)
    }
    
    // Read document
    data, err := cli.Read(dbID, 1)
    if err != nil {
        panic(err)
    }
    
    fmt.Println(string(data)) // Output: hello world
    
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
| **Document**    | Opaque binary blob identified by `doc_id` |
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

- [On-Disk Format](docs/ondisk_format.md)
- [Failure Modes](docs/failure_modes.md)

## License

MIT License - see LICENSE file for details.

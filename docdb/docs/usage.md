# DocDB Usage Guide

This guide covers common usage patterns for DocDB, including Go and TypeScript clients.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Go Client](#go-client)
3. [TypeScript Client](#typescript-client)
4. [Common Patterns](#common-patterns)
5. [Error Handling](#error-handling)
6. [Performance Tips](#performance-tips)
7. [Document Healing](#document-healing)
8. [Monitoring and Observability](#monitoring-and-observability)
9. [Error Handling Best Practices](#error-handling-best-practices)
10. [WAL Trimming Behavior](#wal-trimming-behavior)
11. [Troubleshooting](#troubleshooting)

---

## Quick Start

### 1. Build and Run Server

```bash
# Build server
go build -o docdb ./cmd/docdb

# Run server (defaults: data dir = ./data, socket = /tmp/docdb.sock)
./docdb

# Or with custom configuration
./docdb --data-dir ./mydata --socket /tmp/mydb.sock
```

### 2. Connect and Use

**Go Client:**

```go
package main

import (
    "fmt"
    "github.com/kartikbazzad/docdb/pkg/client"
)

func main() {
    // Connect to server
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
}
```

**TypeScript Client:**

```typescript
import { DocDBJSONClient } from "@docdb/client";

const client = new DocDBJSONClient({ socketPath: "/tmp/docdb.sock" });

// Open database
const dbID = await client.openDB("mydb");

// Create document
await client.createJSON(dbID, 1n, { id: 1, name: "John" });

// Read document
const user = await client.readJSON<{ id: number; name: string }>(dbID, 1n);
console.log(user); // Output: { id: 1, name: 'John' }
```

---

## Go Client

### Connecting

```go
import "github.com/kartikbazzad/docdb/pkg/client"

// Create client
cli := client.New("/tmp/docdb.sock")

// Client maintains connection internally
// No explicit connect/disconnect needed
```

### Database Operations

```go
// Create/open database
dbID, err := cli.OpenDB("mydb")
if err != nil {
    panic(err)
}

// Close database (frees resources but keeps database metadata)
err = cli.CloseDB(dbID)
if err != nil {
    panic(err)
}

// Delete database (removes all data and metadata)
err = cli.DeleteDB(dbID)
if err != nil {
    panic(err)
}
```

### Collections (v0.2)

```go
// Create a collection
err := cli.CreateCollection(dbID, "users")
if err != nil {
    // Handle errors:
    // - CollectionExistsError: Collection already exists
    // - InvalidNameError: Invalid collection name
}

// List all collections
collections, err := cli.ListCollections(dbID)
if err != nil {
    panic(err)
}
for _, name := range collections {
    fmt.Println(name)
}

// Delete an empty collection
err = cli.DeleteCollection(dbID, "users")
if err != nil {
    // Handle errors:
    // - CollectionNotFoundError: Collection doesn't exist
    // - CollectionNotEmptyError: Collection contains documents
}
```

### Document CRUD

```go
// Create document (in default collection)
err := cli.Create(dbID, 1, []byte("document data"))
if err != nil {
    // Handle errors:
    // - ConflictError: Document already exists
    // - MemoryLimitError: Memory capacity exceeded
    // - ConnectionError: Cannot communicate with server
}

// Create document in specific collection
err = cli.CreateInCollection(dbID, "users", 1, []byte(`{"name":"Alice"}`))
if err != nil {
    panic(err)
}

// Read document (from default collection)
data, err := cli.Read(dbID, 1)
if err != nil {
    // Handle errors:
    // - NotFoundError: Document doesn't exist
    // - ConnectionError: Cannot communicate with server
}
fmt.Println(string(data))

// Read document from specific collection
data, err = cli.ReadFromCollection(dbID, "users", 1)
if err != nil {
    panic(err)
}

// Update document (full replacement)
err = cli.Update(dbID, 1, []byte("new data"))
if err != nil {
    // Handle errors:
    // - NotFoundError: Document doesn't exist
    // - MemoryLimitError: Memory capacity exceeded
}

// Delete document (tombstone)
err = cli.Delete(dbID, 1)
if err != nil {
    // Handle errors:
    // - NotFoundError: Document doesn't exist
}
```

### Path-Based Updates (v0.2)

```go
import "github.com/kartikbazzad/docdb/pkg/client"

// Patch operations allow updating specific fields without reading entire document
patchOps := []client.PatchOperation{
    {
        Op:    "set",
        Path:  "/name",
        Value: "Alice",
    },
    {
        Op:    "set",
        Path:  "/age",
        Value: 30,
    },
    {
        Op:    "set",
        Path:  "/address/city",
        Value: "San Francisco",
    },
    {
        Op:   "delete",
        Path: "/oldField",
    },
    {
        Op:    "insert",
        Path:  "/items/0",
        Value: "newItem",
    },
}

// Apply patch operations
err := cli.Patch(dbID, 1, patchOps)
if err != nil {
    // Handle errors:
    // - NotFoundError: Document doesn't exist
    // - InvalidPatchError: Invalid patch operation or path
    // - MemoryLimitError: Memory capacity exceeded
}

// Patch in specific collection
err = cli.PatchInCollection(dbID, "users", 1, patchOps)
if err != nil {
    panic(err)
}
```

**Patch Operation Types:**

- **set** - Set a value at a path (creates intermediate objects if needed)
- **delete** - Delete a value at a path
- **insert** - Insert a value into an array at an index

**Path Syntax:**

- JSON Pointer-like paths: `/name`, `/address/city`, `/items/0`
- Supports nested objects and arrays
- Array indices are numeric strings
- All operations in a patch are atomic (all succeed or all fail)

### Batch Operations

```go
import "github.com/kartikbazzad/docdb/pkg/client"

ops := []client.Operation{
    { DocID: 1, OpType: client.OpCreate, Payload: []byte("doc1") },
    { DocID: 2, OpType: client.OpCreate, Payload: []byte("doc2") },
    { DocID: 3, OpType: client.OpCreate, Payload: []byte("doc3") },
}

results, err := cli.BatchExecute(dbID, ops)
if err != nil {
    panic(err)
}

// Check individual results
for i, result := range results {
    if result.Error != nil {
        fmt.Printf("Op %d failed: %v\n", i, result.Error)
    } else {
        fmt.Printf("Op %d succeeded\n", i)
    }
}
```

### Statistics

```go
stats, err := cli.Stats()
if err != nil {
    panic(err)
}

fmt.Printf("Total DBs: %d\n", stats.TotalDBs)
fmt.Printf("Active DBs: %d\n", stats.ActiveDBs)
fmt.Printf("Total Transactions: %d\n", stats.TotalTxns)
fmt.Printf("WAL Size: %d bytes\n", stats.WALSize)
fmt.Printf("Memory Used: %d / %d MB\n",
    stats.MemoryUsed/1024/1024,
    stats.MemoryCapacity/1024/1024)
```

---

## TypeScript Client

### Connecting

```typescript
import { DocDBClient, DocDBJSONClient } from "@docdb/client";

// Binary client (for raw bytes)
const client = new DocDBClient({
  socketPath: "/tmp/docdb.sock",
  timeout: 5000, // 5 second timeout
});

// JSON client (for type-safe JSON operations)
const jsonClient = new DocDBJSONClient({
  socketPath: "/tmp/docdb.sock",
  timeout: 5000,
});
```

### Database Operations

```typescript
// Open database
const dbID = await client.openDB("mydb");

// Close database
await client.closeDB(dbID);
```

### Document CRUD (Binary API)

```typescript
// Create document
await client.create(dbID, 1n, new TextEncoder().encode("hello"));

// Read document
const data = await client.read(dbID, 1n);
const text = new TextDecoder().decode(data);

// Update document
await client.update(dbID, 1n, new TextEncoder().encode("updated"));

// Delete document
await client.delete(dbID, 1n);
```

### Document CRUD (JSON API)

```typescript
interface User {
  id: number;
  name: string;
  email: string;
}

// Create document (type-safe)
await jsonClient.createJSON<User>(dbID, 1n, {
  id: 1,
  name: "John Doe",
  email: "john@example.com",
});

// Read document (type-safe)
const user = await jsonClient.readJSON<User>(dbID, 1n);
console.log(user.name); // Type-safe access

// Update document (type-safe)
await jsonClient.updateJSON<User>(dbID, 1n, {
  id: 1,
  name: "Jane Doe",
  email: "jane@example.com",
});

// Delete document
await jsonClient.delete(dbID, 1n);
```

### Batch Operations

```typescript
import { DocDBClient, OperationType } from "@docdb/client";

const ops = [
  {
    docID: 1n,
    opType: OperationType.Create,
    payload: new Uint8Array([1, 2, 3]),
  },
  {
    docID: 2n,
    opType: OperationType.Create,
    payload: new Uint8Array([4, 5, 6]),
  },
  {
    docID: 3n,
    opType: OperationType.Create,
    payload: new Uint8Array([7, 8, 9]),
  },
];

const results = await client.batchExecute(dbID, ops);

// Check individual results
results.forEach((result, i) => {
  if (result.error) {
    console.error(`Op ${i} failed:`, result.error);
  } else {
    console.log(`Op ${i} succeeded`);
  }
});
```

### Statistics

```typescript
const stats = await client.stats();

console.log(`Total DBs: ${stats.totalDBs}`);
console.log(`Active DBs: ${stats.activeDBs}`);
console.log(`Total Transactions: ${stats.totalTxns}`);
console.log(`WAL Size: ${stats.walSize} bytes`);
console.log(`Memory Used: ${stats.memoryUsed} / ${stats.memoryCapacity} bytes`);
```

### Error Handling

```typescript
try {
  await client.create(dbID, 1n, new Uint8Array([1, 2, 3]));
} catch (err) {
  if (err instanceof DocDBError) {
    switch (err.code) {
      case ErrorCode.NotFound:
        console.error("Document not found");
        break;
      case ErrorCode.Conflict:
        console.error("Document already exists");
        break;
      case ErrorCode.MemoryLimit:
        console.error("Memory limit exceeded");
        break;
      default:
        console.error("Unknown error:", err.message);
    }
  } else {
    console.error("Unexpected error:", err);
  }
}
```

---

## Common Patterns

### 1. Auto-Generate Document IDs

```go
// Use timestamp or counter for auto-generated IDs
docID := uint64(time.Now().UnixNano())

// Or use atomic counter
var nextDocID uint64 = 1
docID = atomic.AddUint64(&nextDocID, 1)
```

### 2. Transaction-Like Operations

```go
// Although DocDB has no cross-document transactions,
// you can achieve similar semantics with retry logic

maxRetries := 3
for i := 0; i < maxRetries; i++ {
    err := cli.Create(dbID, docID, payload)
    if err == nil {
        break // Success
    }

    if errors.Is(err, client.ConflictError) {
        // Document exists, retry with new ID
        docID++
        continue
    }

    // Other error, don't retry
    panic(err)
}
```

### 3. Large Payloads

```go
// For payloads > 10KB, consider:
// 1. Breaking into chunks
// 2. Using compression
// 3. Storing reference instead of full data

// Example: Store reference
reference := []byte(fmt.Sprintf("s3://bucket/object%d", docID))
cli.Create(dbID, docID, reference)
```

### 4. Bulk Import

```typescript
import { DocDBJSONClient } from "@docdb/client";

const users: Array<{ id: number; name: string }> = [
  { id: 1, name: "Alice" },
  { id: 2, name: "Bob" },
  { id: 3, name: "Charlie" },
  // ... many more users
];

// Use batch operations for efficiency
const ops = users.map((user) => ({
  docID: BigInt(user.id),
  opType: OperationType.Create,
  payload: new TextEncoder().encode(JSON.stringify(user)),
}));

const results = await client.batchExecute(dbID, ops);

const failures = results.filter((r) => r.error !== null);
if (failures.length > 0) {
  console.error(`${failures.length} operations failed`);
}
```

---

## Error Handling

### Error Codes

| Code | Go Error         | TS Error                      | Description                            |
| ---- | ---------------- | ----------------------------- | -------------------------------------- |
| 0    | -                | OK                            | Operation succeeded                    |
| 1    | ConnectionError  | DocDBError                    | General error (network, timeout, etc.) |
| 2    | NotFoundError    | DocDBError (code=NotFound)    | Document/database not found            |
| 3    | ConflictError    | DocDBError (code=Conflict)    | Document already exists                |
| 4    | MemoryLimitError | DocDBError (code=MemoryLimit) | Memory capacity exceeded               |
| 5    | -                | -                             | Corrupt WAL record (internal)          |
| 6    | -                | -                             | CRC32 mismatch (internal)              |

### Go Error Handling

```go
import (
    "errors"
    "github.com/kartikbazzad/docdb/pkg/client"
)

data, err := cli.Read(dbID, docID)
if err != nil {
    if errors.Is(err, client.NotFoundError) {
        // Handle not found
    } else if errors.Is(err, client.MemoryLimitError) {
        // Handle memory limit
    } else {
        // Handle other errors
        panic(err)
    }
}
```

### TypeScript Error Handling

```typescript
import { DocDBError, ErrorCode } from "@docdb/client";

try {
  await client.read(dbID, docID);
} catch (err) {
  if (err instanceof DocDBError) {
    switch (err.code) {
      case ErrorCode.NotFound:
        console.error("Document not found");
        break;
      case ErrorCode.MemoryLimit:
        console.error("Memory limit exceeded");
        break;
      case ErrorCode.Conflict:
        console.error("Document already exists");
        break;
      default:
        console.error("Unknown error:", err.message);
    }
  } else {
    console.error("Unexpected error:", err);
  }
}
```

---

## Performance Tips

### 1. Use Batch Operations

```go
// Instead of:
for _, docID := range docIDs {
    cli.Create(dbID, docID, payload)
}

// Use batch operations:
ops := make([]client.Operation, len(docIDs))
for i, docID := range docIDs {
    ops[i] = client.Operation{DocID: docID, OpType: client.OpCreate, Payload: payload}
}
cli.BatchExecute(dbID, ops)
```

### 2. Reuse Client Connections

```go
// Create one client and reuse
cli := client.New("/tmp/docdb.sock")
defer cli.Close() // Close when done

// Don't create new client for each operation
```

### 3. Limit Payload Size

```go
// Keep payloads small (< 10KB ideal, < 16MB max)
// Large payloads slow down:
//   - Writes (disk I/O)
//   - Reads (network latency)
//   - Recovery (WAL replay)
```

### 4. Use Appropriate Document IDs

```go
// Use sequential or hash-based IDs
// Avoid random IDs that cause cache misses

// Good:
docID := 1, 2, 3, 4, 5 // Sequential

// Bad:
docID := 123456789, 987654321 // Random, sparse
```

### 5. Monitor Memory Usage

```go
stats, _ := cli.Stats()
usedMB := float64(stats.MemoryUsed) / 1024 / 1024
capMB := float64(stats.MemoryCapacity) / 1024 / 1024
usagePercent := (usedMB / capMB) * 100

if usagePercent > 80 {
    fmt.Printf("Warning: Memory at %.0f%% capacity\n", usagePercent)
}
```

---

## Document Healing

DocDB v0.2 includes automatic document healing to recover from corruption. Healing finds the latest valid version of a document from the WAL and restores it.

### Manual Healing

**Heal a specific document:**

```go
// Using shell command
// In docdbsh:
.heal 123

// Using Go client (if IPC support added)
// Note: This requires IPC protocol support
```

**Heal all corrupted documents:**

```go
// Using shell command
// In docdbsh:
.heal-all

// This returns the count of healed documents
```

**Check healing statistics:**

```go
// Using shell command
// In docdbsh:
.heal-stats

// Shows:
// - Total scans performed
// - Documents healed
// - Documents currently corrupted
// - Last scan time
// - Last healing time
```

### Automatic Healing

Automatic healing runs in the background and scans all documents periodically.

**Configuration:**

```go
cfg := config.DefaultConfig()
cfg.Healing.Enabled = true                    // Enable automatic healing
cfg.Healing.Interval = 1 * time.Hour          // Scan every hour
cfg.Healing.OnReadCorruption = true           // Heal immediately on read corruption
cfg.Healing.MaxBatchSize = 100               // Heal up to 100 docs per batch
```

**How it works:**

1. **Periodic Scans:** Background service scans all documents at configured interval
2. **On-Read Detection:** If corruption detected during read, healing triggered immediately
3. **WAL Recovery:** Healing finds latest valid version from WAL records
4. **Index Update:** Index updated to point to healed version

**Monitoring:**

```go
// Check healing stats via shell
.heal-stats

// Output example:
// OK
// total_scans=42
// documents_healed=3
// documents_corrupted=0
// last_scan_time=2026-01-28T10:30:00Z
// last_healing_time=2026-01-28T09:15:00Z
```

### When Healing Occurs

- **Automatic:** Periodic background scans (configurable interval)
- **On-Read:** Corruption detected during document read (if `OnReadCorruption` enabled)
- **Manual:** Via `.heal` or `.heal-all` shell commands

### Healing Limitations

- Healing only works if valid version exists in WAL
- If all WAL records are corrupted, document cannot be healed
- Healing requires WAL segments to be available (not trimmed)

---

## Monitoring and Observability

DocDB v0.2 provides Prometheus/OpenMetrics metrics for monitoring system health and performance.

### Prometheus Metrics

**Access metrics:**

```bash
# Via IPC client (if metrics command implemented)
# Metrics are returned in Prometheus text format
```

**Available Metrics:**

- `docdb_operations_total{operation, status}` - Total operations by type and status
- `docdb_operation_duration_seconds{operation}` - Operation duration histogram
- `docdb_documents_total` - Total number of live documents
- `docdb_memory_bytes` - Memory usage in bytes
- `docdb_wal_size_bytes` - WAL size in bytes
- `docdb_errors_total{category}` - Error counts by category (transient, permanent, critical, validation, network)
- `docdb_healing_operations_total` - Total healing operations performed
- `docdb_documents_healed_total` - Total documents healed

**Example Prometheus Query:**

```promql
# Operation rate
rate(docdb_operations_total[5m])

# Error rate
rate(docdb_errors_total[5m])

# Memory usage percentage
(docdb_memory_bytes / docdb_memory_capacity_bytes) * 100

# Healing operations per hour
rate(docdb_healing_operations_total[1h]) * 3600
```

### Statistics API

**Get current statistics:**

```go
stats, err := cli.Stats()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Total DBs: %d\n", stats.TotalDBs)
fmt.Printf("Active DBs: %d\n", stats.ActiveDBs)
fmt.Printf("Transactions: %d\n", stats.TotalTxns)
fmt.Printf("Committed: %d\n", stats.TxnsCommitted)
fmt.Printf("WAL Size: %d bytes\n", stats.WALSize)
fmt.Printf("Memory Used: %d bytes\n", stats.MemoryUsed)
fmt.Printf("Memory Capacity: %d bytes\n", stats.MemoryCapacity)
fmt.Printf("Live Documents: %d\n", stats.DocsLive)
fmt.Printf("Tombstoned Documents: %d\n", stats.DocsTombstoned)
fmt.Printf("Last Compaction: %v\n", stats.LastCompaction)
```

### Health Checks

**Basic health check:**

```bash
# Check if server is running
test -S /tmp/docdb.sock && echo "Server running" || echo "Server not running"
```

**Check database status:**

```bash
# Using shell
./docdbsh
> .stats
# Shows pool statistics
```

---

## Error Handling Best Practices

DocDB v0.2 includes error classification and retry logic for improved reliability.

### Error Categories

Errors are automatically classified into categories:

- **Transient:** Temporary errors (EAGAIN, ENOMEM) - automatically retried
- **Permanent:** Permanent errors (ENOENT, EINVAL) - no retry
- **Critical:** System-level errors (EIO, ENOSPC) - requires immediate attention
- **Validation:** Data validation errors (CRC mismatch, invalid JSON) - no retry
- **Network:** Network-related errors - retried with backoff

### Retry Logic

**Automatic Retries:**

WAL and datafile operations automatically retry transient errors with exponential backoff:

- Initial delay: 10ms
- Max delay: 1s
- Max retries: 5
- Jitter: ±25% random variation

**Error Handling Pattern:**

```go
err := cli.Create(dbID, docID, payload)
if err != nil {
    // Check error type
    if errors.Is(err, docdb.ErrDocExists) {
        // Permanent error - document already exists
        // Handle appropriately (update instead?)
    } else if errors.Is(err, docdb.ErrMemoryLimit) {
        // Permanent error - memory limit reached
        // Reduce memory usage or increase limit
    } else {
        // Transient or unknown error
        // May have been retried automatically
        // Log and handle appropriately
        log.Printf("Operation failed: %v", err)
    }
}
```

### Error Tracking

Errors are automatically tracked and categorized:

```go
// Errors are tracked internally
// Access via metrics endpoint:
// docdb_errors_total{category="transient"}
// docdb_errors_total{category="permanent"}
// docdb_errors_total{category="critical"}
```

### Common Error Scenarios

**Memory Limit Exceeded:**

```go
// Error: ErrMemoryLimit
// Solution:
// 1. Increase memory limits in config
// 2. Reduce document size
// 3. Compact database to free memory
```

**Document Not Found:**

```go
// Error: ErrDocNotFound
// Solution:
// 1. Check document exists before operation
// 2. Handle gracefully in application logic
```

**Invalid JSON:**

```go
// Error: ErrInvalidJSON
// Solution:
// 1. Validate JSON before sending
// 2. Ensure UTF-8 encoding
// 3. Check payload is valid JSON structure
```

**File System Errors:**

```go
// Errors: ErrFileOpen, ErrFileWrite, ErrFileSync
// These are automatically retried if transient
// If persistent, check:
// 1. Disk space available
// 2. File permissions
// 3. Disk health
```

---

## WAL Trimming Behavior

WAL trimming automatically cleans up old WAL segments after checkpoints to prevent unbounded disk usage.

### How Trimming Works

1. **Checkpoint Created:** After WAL reaches configured size (default: 64MB)
2. **Segment Identification:** Segments before checkpoint identified
3. **Safety Margin:** Keeps configured number of segments (default: 2)
4. **Deletion:** Old segments deleted atomically

### Configuration

```go
cfg.WAL.TrimAfterCheckpoint = true  // Enable automatic trimming
cfg.WAL.KeepSegments = 2            // Keep 2 segments before checkpoint
cfg.WAL.Checkpoint.IntervalMB = 64  // Create checkpoint every 64MB
```

### When Trimming Occurs

- After each checkpoint creation
- Only if `TrimAfterCheckpoint` is enabled
- Active WAL segment is never trimmed
- At least `KeepSegments` segments are preserved

### Monitoring Trimming

```bash
# Check WAL directory
ls -lh /path/to/wal/

# Should see:
# - Active segment: dbname.wal
# - Rotated segments: dbname.wal.1, dbname.wal.2, ...
# - Old segments automatically removed after checkpoint
```

### Trimming Safety

- Trimming only occurs after checkpoint (data is safe)
- Active segment always preserved
- Safety margin prevents accidental data loss
- Atomic deletion ensures consistency

---

## Troubleshooting

### Common Issues

**Issue: "connection refused"**

- Cause: Server not running or wrong socket path
- Solution: Check server is running and socket path is correct

**Issue: "memory limit exceeded"**

- Cause: Too much data in memory
- Solution: Increase memory limits in config or compact data

**Issue: "document not found"**

- Cause: Document doesn't exist or was deleted
- Solution: Check document exists before reading/updating

**Issue: Slow reads/writes**

- Cause: Large payloads or disk I/O bottleneck
- Solution: Reduce payload size, use faster storage (SSD)

**Issue: Data not persisting after restart**

- Cause: WAL replay issue (should be fixed in v0)
- Solution: Check logs for WAL errors, ensure fsync enabled

### Debugging

**Enable verbose logging:**

```bash
# Server logs include INFO level by default
# Check logs for detailed operation traces
./docdb 2>&1 | tee server.log
```

**Test with small datasets:**

```go
// Start with 10-100 documents
// Verify operations work correctly
// Then scale up to production size
```

**Monitor system resources:**

```bash
# Check memory usage
ps aux | grep docdb

# Check disk I/O
iostat -x 1

# Check CPU usage
top -p $(pgrep docdb)
```

### Getting Help

- Check logs for error messages
- Review [failure_modes.md](failure_modes.md) for recovery scenarios
- See [configuration.md](configuration.md) for tuning options
- Open issue on GitHub if problem persists

---

## Next Steps

- Read [architecture.md](architecture.md) for system design
- See [transactions.md](transactions.md) for transaction details
- Check [configuration.md](configuration.md) for all options
- Review [troubleshooting.md](troubleshooting.md) for more debugging tips

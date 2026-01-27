# DocDB Usage Guide

This guide covers common usage patterns for DocDB, including Go and TypeScript clients.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Go Client](#go-client)
3. [TypeScript Client](#typescript-client)
4. [Common Patterns](#common-patterns)
5. [Error Handling](#error-handling)
6. [Performance Tips](#performance-tips)
7. [Troubleshooting](#troubleshooting)

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
import { DocDBJSONClient } from '@docdb/client';

const client = new DocDBJSONClient({ socketPath: '/tmp/docdb.sock' });

// Open database
const dbID = await client.openDB('mydb');

// Create document
await client.createJSON(dbID, 1n, { id: 1, name: 'John' });

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

### Document CRUD

```go
// Create document
err := cli.Create(dbID, 1, []byte("document data"))
if err != nil {
    // Handle errors:
    // - ConflictError: Document already exists
    // - MemoryLimitError: Memory capacity exceeded
    // - ConnectionError: Cannot communicate with server
}

// Read document
data, err := cli.Read(dbID, 1)
if err != nil {
    // Handle errors:
    // - NotFoundError: Document doesn't exist
    // - ConnectionError: Cannot communicate with server
}
fmt.Println(string(data))

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
import { DocDBClient, DocDBJSONClient } from '@docdb/client';

// Binary client (for raw bytes)
const client = new DocDBClient({
    socketPath: '/tmp/docdb.sock',
    timeout: 5000 // 5 second timeout
});

// JSON client (for type-safe JSON operations)
const jsonClient = new DocDBJSONClient({
    socketPath: '/tmp/docdb.sock',
    timeout: 5000
});
```

### Database Operations

```typescript
// Open database
const dbID = await client.openDB('mydb');

// Close database
await client.closeDB(dbID);
```

### Document CRUD (Binary API)

```typescript
// Create document
await client.create(dbID, 1n, new TextEncoder().encode('hello'));

// Read document
const data = await client.read(dbID, 1n);
const text = new TextDecoder().decode(data);

// Update document
await client.update(dbID, 1n, new TextEncoder().encode('updated'));

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
    name: 'John Doe',
    email: 'john@example.com'
});

// Read document (type-safe)
const user = await jsonClient.readJSON<User>(dbID, 1n);
console.log(user.name); // Type-safe access

// Update document (type-safe)
await jsonClient.updateJSON<User>(dbID, 1n, {
    id: 1,
    name: 'Jane Doe',
    email: 'jane@example.com'
});

// Delete document
await jsonClient.delete(dbID, 1n);
```

### Batch Operations

```typescript
import { DocDBClient, OperationType } from '@docdb/client';

const ops = [
    { docID: 1n, opType: OperationType.Create, payload: new Uint8Array([1, 2, 3]) },
    { docID: 2n, opType: OperationType.Create, payload: new Uint8Array([4, 5, 6]) },
    { docID: 3n, opType: OperationType.Create, payload: new Uint8Array([7, 8, 9]) }
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
                console.error('Document not found');
                break;
            case ErrorCode.Conflict:
                console.error('Document already exists');
                break;
            case ErrorCode.MemoryLimit:
                console.error('Memory limit exceeded');
                break;
            default:
                console.error('Unknown error:', err.message);
        }
    } else {
        console.error('Unexpected error:', err);
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
import { DocDBJSONClient } from '@docdb/client';

const users: Array<{ id: number; name: string }> = [
    { id: 1, name: 'Alice' },
    { id: 2, name: 'Bob' },
    { id: 3, name: 'Charlie' }
    // ... many more users
];

// Use batch operations for efficiency
const ops = users.map(user => ({
    docID: BigInt(user.id),
    opType: OperationType.Create,
    payload: new TextEncoder().encode(JSON.stringify(user))
}));

const results = await client.batchExecute(dbID, ops);

const failures = results.filter(r => r.error !== null);
if (failures.length > 0) {
    console.error(`${failures.length} operations failed`);
}
```

---

## Error Handling

### Error Codes

| Code | Go Error | TS Error | Description |
|------|------------|-----------|-------------|
| 0 | - | OK | Operation succeeded |
| 1 | ConnectionError | DocDBError | General error (network, timeout, etc.) |
| 2 | NotFoundError | DocDBError (code=NotFound) | Document/database not found |
| 3 | ConflictError | DocDBError (code=Conflict) | Document already exists |
| 4 | MemoryLimitError | DocDBError (code=MemoryLimit) | Memory capacity exceeded |
| 5 | - | - | Corrupt WAL record (internal) |
| 6 | - | - | CRC32 mismatch (internal) |

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
import { DocDBError, ErrorCode } from '@docdb/client';

try {
    await client.read(dbID, docID);
} catch (err) {
    if (err instanceof DocDBError) {
        switch (err.code) {
            case ErrorCode.NotFound:
                console.error('Document not found');
                break;
            case ErrorCode.MemoryLimit:
                console.error('Memory limit exceeded');
                break;
            case ErrorCode.Conflict:
                console.error('Document already exists');
                break;
            default:
                console.error('Unknown error:', err.message);
        }
    } else {
        console.error('Unexpected error:', err);
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

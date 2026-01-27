# Testing Guide

This document describes testing strategies, patterns, and best practices for DocDB.

## Table of Contents

1. [Overview](#overview)
2. [Test Structure](#test-structure)
3. [Integration Tests](#integration-tests)
4. [Concurrency Tests](#concurrency-tests)
5. [Failure Tests](#failure-tests)
6. [Benchmarks](#benchmarks)
7. [Test Utilities](#test-utilities)
8. [Best Practices](#best-practices)

---

## Overview

DocDB uses Go's standard `testing` package with the following test categories:

- **Integration Tests**: End-to-end database operations
- **Concurrency Tests**: Concurrent read/write scenarios
- **Failure Tests**: Crash recovery and error handling
- **Benchmarks**: Performance measurements

### Test Organization

```
tests/
├── integration/
│   └── integration_test.go    # End-to-end tests
├── concurrency/
│   └── concurrency_test.go   # Concurrent operations
├── failure/
│   └── failure_test.go       # Failure scenarios
└── benchmarks/
    └── bench_test.go         # Performance benchmarks
```

---

## Test Structure

### Setup and Teardown Pattern

**Pool Setup:**
```go
func setupTestPool(t *testing.T) (*pool.Pool, string, func()) {
    t.Helper()
    
    tmpDir, err := os.MkdirTemp("", "docdb-test-*")
    if err != nil {
        t.Fatalf("Failed to create temp dir: %v", err)
    }
    
    cfg := config.DefaultConfig()
    cfg.DataDir = tmpDir
    cfg.WAL.Dir = filepath.Join(tmpDir, "wal")
    
    log := logger.Default()
    p := pool.NewPool(cfg, log)
    if err := p.Start(); err != nil {
        t.Fatalf("Failed to start pool: %v", err)
    }
    
    cleanup := func() {
        p.Stop()
        os.RemoveAll(tmpDir)
    }
    
    return p, tmpDir, cleanup
}
```

**Usage:**
```go
func TestDatabaseOperations(t *testing.T) {
    p, _, cleanup := setupTestPool(t)
    defer cleanup()
    
    // Test code here
}
```

### Database Setup

**Direct Database Setup:**
```go
func setupTestDB(t *testing.T) (*docdb.LogicalDB, string, func()) {
    t.Helper()
    
    tmpDir, err := os.MkdirTemp("", "docdb-test-*")
    if err != nil {
        t.Fatalf("Failed to create temp dir: %v", err)
    }
    
    cfg := config.DefaultConfig()
    cfg.DataDir = tmpDir
    cfg.WAL.Dir = filepath.Join(tmpDir, "wal")
    
    log := logger.Default()
    memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
    memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
    pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
    
    db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, pool, log)
    if err := db.Open(tmpDir, cfg.WAL.Dir); err != nil {
        t.Fatalf("Failed to open database: %v", err)
    }
    
    cleanup := func() {
        db.Close()
        os.RemoveAll(tmpDir)
    }
    
    return db, tmpDir, cleanup
}
```

---

## Integration Tests

### Purpose

Integration tests verify end-to-end functionality including:
- Database lifecycle (create, open, close, delete)
- Document operations (create, read, update, delete)
- MVCC behavior
- Memory limits
- Persistence across restarts

### Example: Database Operations

```go
func TestDatabaseOperations(t *testing.T) {
    p, _, cleanup := setupTestPool(t)
    defer cleanup()
    
    t.Run("CreateDatabase", func(t *testing.T) {
        dbID, err := p.CreateDB("testdb")
        if err != nil {
            t.Fatalf("Failed to create database: %v", err)
        }
        
        if dbID == 0 {
            t.Fatal("Expected non-zero DB ID")
        }
        
        // Test duplicate creation
        _, err = p.CreateDB("testdb")
        if err == nil {
            t.Fatal("Expected error when creating duplicate database")
        }
    })
    
    t.Run("CreateDocument", func(t *testing.T) {
        dbID, err := p.CreateDB("createdb")
        if err != nil {
            t.Fatalf("Failed to create database: %v", err)
        }
        
        db, err := p.OpenDB(dbID)
        if err != nil {
            t.Fatalf("Failed to open database: %v", err)
        }
        
        payload := []byte("test payload")
        docID := uint64(1)
        
        err = db.Create(docID, payload)
        if err != nil {
            t.Fatalf("Failed to create document: %v", err)
        }
        
        retrieved, err := db.Read(docID)
        if err != nil {
            t.Fatalf("Failed to read document: %v", err)
        }
        
        if string(retrieved) != string(payload) {
            t.Fatalf("Payload mismatch: got %s, want %s", retrieved, payload)
        }
        
        // Test duplicate creation
        err = db.Create(docID, payload)
        if err == nil {
            t.Fatal("Expected error when creating duplicate document")
        }
    })
}
```

### Example: MVCC Testing

```go
func TestMVCC(t *testing.T) {
    db, _, cleanup := setupTestDB(t)
    defer cleanup()
    
    payload1 := []byte("version 1")
    docID := uint64(1)
    
    err := db.Create(docID, payload1)
    if err != nil {
        t.Fatalf("Failed to create document: %v", err)
    }
    
    payload2 := []byte("version 2")
    err = db.Update(docID, payload2)
    if err != nil {
        t.Fatalf("Failed to update document: %v", err)
    }
    
    retrieved, err := db.Read(docID)
    if err != nil {
        t.Fatalf("Failed to read document: %v", err)
    }
    
    if string(retrieved) != string(payload2) {
        t.Fatalf("Payload mismatch: got %s, want %s", retrieved, payload2)
    }
}
```

### Example: Persistence Testing

```go
func TestPersistence(t *testing.T) {
    db, tmpDir, cleanup := setupTestDB(t)
    defer cleanup()
    
    payload := []byte("persistent payload")
    docID := uint64(1)
    
    err := db.Create(docID, payload)
    if err != nil {
        t.Fatalf("Failed to create document: %v", err)
    }
    
    // Close and reopen
    if err := db.Close(); err != nil {
        t.Fatalf("Failed to close database: %v", err)
    }
    
    // Reopen database
    cfg := config.DefaultConfig()
    cfg.DataDir = tmpDir
    cfg.WAL.Dir = filepath.Join(tmpDir, "wal")
    
    log := logger.Default()
    memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
    memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
    pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
    
    db2 := docdb.NewLogicalDB(1, "persistdb", cfg, memCaps, pool, log)
    if err := db2.Open(tmpDir, cfg.WAL.Dir); err != nil {
        t.Fatalf("Failed to reopen database: %v", err)
    }
    defer db2.Close()
    
    // Verify data persisted
    retrieved, err := db2.Read(docID)
    if err != nil {
        t.Fatalf("Failed to read document after restart: %v", err)
    }
    
    if string(retrieved) != string(payload) {
        t.Fatalf("Payload mismatch after restart: got %s, want %s", retrieved, payload)
    }
}
```

---

## Concurrency Tests

### Purpose

Concurrency tests verify:
- Concurrent writes to different documents
- Concurrent reads
- Lock contention behavior
- Thread safety

### Example: Concurrent Writes

```go
func TestConcurrentWrites(t *testing.T) {
    p, _, cleanup := setupTestPool(t)
    defer cleanup()
    
    dbID, err := p.CreateDB("concurrentwrites")
    if err != nil {
        t.Fatalf("Failed to create database: %v", err)
    }
    
    db, err := p.OpenDB(dbID)
    if err != nil {
        t.Fatalf("Failed to open database: %v", err)
    }
    
    numWriters := 10
    numDocs := 100
    
    var wg sync.WaitGroup
    wg.Add(numWriters)
    
    writer := func(workerID int) {
        defer wg.Done()
        
        for i := 1; i <= numDocs; i++ {
            docID := uint64(workerID*1000 + i)
            payload := []byte("payload from worker")
            
            err := db.Create(docID, payload)
            if err != nil {
                t.Logf("Worker %d: Failed to create doc %d: %v", workerID, docID, err)
            }
        }
    }
    
    for i := 0; i < numWriters; i++ {
        go writer(i)
    }
    
    wg.Wait()
    
    expectedDocs := numWriters * numDocs
    if db.IndexSize() != expectedDocs {
        t.Fatalf("Expected %d documents, got %d", expectedDocs, db.IndexSize())
    }
}
```

### Example: Concurrent Reads

```go
func TestConcurrentReads(t *testing.T) {
    db, _, cleanup := setupTestDB(t)
    defer cleanup()
    
    // Create test documents
    numDocs := 100
    for i := 1; i <= numDocs; i++ {
        err := db.Create(uint64(i), []byte(fmt.Sprintf("doc%d", i)))
        if err != nil {
            t.Fatalf("Failed to create document %d: %v", i, err)
        }
    }
    
    numReaders := 20
    var wg sync.WaitGroup
    wg.Add(numReaders)
    
    reader := func(workerID int) {
        defer wg.Done()
        
        for i := 1; i <= numDocs; i++ {
            _, err := db.Read(uint64(i))
            if err != nil {
                t.Logf("Worker %d: Failed to read doc %d: %v", workerID, i, err)
            }
        }
    }
    
    for i := 0; i < numReaders; i++ {
        go reader(i)
    }
    
    wg.Wait()
}
```

---

## Failure Tests

### Purpose

Failure tests verify:
- WAL corruption handling
- Crash recovery
- Disk full scenarios
- Invalid operations

### Example: Corrupted WAL

```go
func TestCorruptWAL(t *testing.T) {
    db, tmpDir, cleanup := setupTestDB(t)
    defer cleanup()
    
    payload := []byte("test payload")
    docID := uint64(1)
    
    err := db.Create(docID, payload)
    if err != nil {
        t.Fatalf("Failed to create document: %v", err)
    }
    
    if err := db.Close(); err != nil {
        t.Fatalf("Failed to close database: %v", err)
    }
    
    // Corrupt WAL file
    walPath := filepath.Join(tmpDir, "wal", "faildb.wal")
    walData, err := os.ReadFile(walPath)
    if err != nil {
        t.Fatalf("Failed to read WAL: %v", err)
    }
    
    if len(walData) > 10 {
        corruptData := make([]byte, len(walData)-5)
        copy(corruptData, walData[:5])
        for i := 5; i < len(corruptData); i++ {
            corruptData[i] = 0xFF
        }
        
        if err := os.WriteFile(walPath, corruptData, 0644); err != nil {
            t.Fatalf("Failed to corrupt WAL: %v", err)
        }
    }
    
    // Reopen and verify recovery
    cfg := config.DefaultConfig()
    cfg.DataDir = tmpDir
    cfg.WAL.Dir = filepath.Join(tmpDir, "wal")
    
    log := logger.Default()
    memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
    memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
    pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
    
    db2 := docdb.NewLogicalDB(1, "faildb", cfg, memCaps, pool, log)
    if err := db2.Open(tmpDir, cfg.WAL.Dir); err != nil {
        t.Fatalf("Failed to reopen database after corrupted WAL: %v", err)
    }
    defer db2.Close()
    
    // Should handle corruption gracefully
    _, err = db2.Read(docID)
    if err == nil {
        t.Fatal("Expected error when reading after corrupted WAL")
    }
}
```

### Example: Memory Limits

```go
func TestMemoryLimits(t *testing.T) {
    tmpDir, err := os.MkdirTemp("", "docdb-memory-test-*")
    if err != nil {
        t.Fatalf("Failed to create temp dir: %v", err)
    }
    defer os.RemoveAll(tmpDir)
    
    cfg := config.DefaultConfig()
    cfg.DataDir = tmpDir
    cfg.WAL.Dir = filepath.Join(tmpDir, "wal")
    cfg.Memory.GlobalCapacityMB = 1
    cfg.Memory.PerDBLimitMB = 1
    
    log := logger.Default()
    memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
    memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
    pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
    
    db := docdb.NewLogicalDB(1, "memorydb", cfg, memCaps, pool, log)
    if err := db.Open(tmpDir, cfg.WAL.Dir); err != nil {
        t.Fatalf("Failed to open database: %v", err)
    }
    defer db.Close()
    
    // Try to create document larger than limit
    largePayload := make([]byte, 2*1024*1024)
    docID := uint64(1)
    
    err = db.Create(docID, largePayload)
    if err != docdb.ErrMemoryLimit {
        t.Fatalf("Expected ErrMemoryLimit, got: %v", err)
    }
}
```

---

## Benchmarks

### Purpose

Benchmarks measure performance characteristics:
- Operation latency
- Throughput
- Memory usage
- Scalability

### Example: Create Benchmark

```go
func BenchmarkCreateDocument(b *testing.B) {
    tmpDir, err := os.MkdirTemp("", "docdb-bench-create-*")
    if err != nil {
        b.Fatalf("Failed to create temp dir: %v", err)
    }
    defer os.RemoveAll(tmpDir)
    
    cfg := config.DefaultConfig()
    cfg.DataDir = tmpDir
    cfg.WAL.Dir = filepath.Join(tmpDir, "wal")
    
    log := logger.New(io.Discard, logger.LevelError, "")
    memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
    memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
    pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
    
    db := docdb.NewLogicalDB(1, "benchdb", cfg, memCaps, pool, log)
    if err := db.Open(tmpDir, cfg.WAL.Dir); err != nil {
        b.Fatalf("Failed to open database: %v", err)
    }
    defer db.Close()
    
    payload := []byte("benchmark payload")
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        docID := uint64(1)
        for pb.Next() {
            err := db.Create(docID, payload)
            if err != nil {
                b.Fatalf("Failed to create document: %v", err)
            }
            docID++
        }
    })
}
```

### Running Benchmarks

```bash
# Run all benchmarks
go test -bench=. ./tests/benchmarks

# Run specific benchmark
go test -bench=BenchmarkCreateDocument ./tests/benchmarks

# With CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./tests/benchmarks

# With memory profiling
go test -bench=. -memprofile=mem.prof ./tests/benchmarks
```

---

## Test Utilities

### Helper Functions

**Create Test Payload:**
```go
func createTestPayload(size int) []byte {
    payload := make([]byte, size)
    for i := range payload {
        payload[i] = byte(i % 256)
    }
    return payload
}
```

**Verify Document:**
```go
func verifyDocument(t *testing.T, db *docdb.LogicalDB, docID uint64, expected []byte) {
    t.Helper()
    
    actual, err := db.Read(docID)
    if err != nil {
        t.Fatalf("Failed to read document %d: %v", docID, err)
    }
    
    if !bytes.Equal(actual, expected) {
        t.Fatalf("Document %d mismatch: got %v, want %v", docID, actual, expected)
    }
}
```

**Wait for Condition:**
```go
func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration) {
    t.Helper()
    
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if condition() {
            return
        }
        time.Sleep(10 * time.Millisecond)
    }
    
    t.Fatal("Condition not met within timeout")
}
```

---

## Best Practices

### 1. Use Table-Driven Tests

```go
func TestUpdateDocument(t *testing.T) {
    testCases := []struct {
        name    string
        initial []byte
        update  []byte
        want    []byte
    }{
        {"small", []byte("a"), []byte("b"), []byte("b")},
        {"large", make([]byte, 1024), make([]byte, 2048), make([]byte, 2048)},
        {"empty", []byte(""), []byte("new"), []byte("new")},
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            db, _, cleanup := setupTestDB(t)
            defer cleanup()
            
            docID := uint64(1)
            db.Create(docID, tc.initial)
            db.Update(docID, tc.update)
            
            got, _ := db.Read(docID)
            if !bytes.Equal(got, tc.want) {
                t.Errorf("got %v, want %v", got, tc.want)
            }
        })
    }
}
```

### 2. Clean Up Resources

```go
func TestSomething(t *testing.T) {
    db, tmpDir, cleanup := setupTestDB(t)
    defer cleanup() // Always defer cleanup
    
    // Test code
}
```

### 3. Use Subtests

```go
func TestDatabaseOperations(t *testing.T) {
    p, _, cleanup := setupTestPool(t)
    defer cleanup()
    
    t.Run("Create", func(t *testing.T) {
        // Test create
    })
    
    t.Run("Read", func(t *testing.T) {
        // Test read
    })
    
    t.Run("Update", func(t *testing.T) {
        // Test update
    })
}
```

### 4. Test Error Cases

```go
func TestErrorCases(t *testing.T) {
    db, _, cleanup := setupTestDB(t)
    defer cleanup()
    
    // Test duplicate create
    db.Create(1, []byte("doc1"))
    err := db.Create(1, []byte("doc2"))
    if err == nil {
        t.Fatal("Expected error for duplicate document")
    }
    
    // Test read non-existent
    _, err = db.Read(999)
    if err != docdb.ErrDocNotFound {
        t.Fatalf("Expected ErrDocNotFound, got: %v", err)
    }
}
```

### 5. Use Parallel Tests (When Safe)

```go
func TestParallelReads(t *testing.T) {
    db, _, cleanup := setupTestDB(t)
    defer cleanup()
    
    // Create test data
    for i := 1; i <= 100; i++ {
        db.Create(uint64(i), []byte(fmt.Sprintf("doc%d", i)))
    }
    
    // Parallel reads are safe
    t.Run("Parallel", func(t *testing.T) {
        t.Parallel()
        
        for i := 1; i <= 100; i++ {
            _, err := db.Read(uint64(i))
            if err != nil {
                t.Errorf("Failed to read doc %d: %v", i, err)
            }
        }
    })
}
```

### 6. Test Edge Cases

```go
func TestEdgeCases(t *testing.T) {
    db, _, cleanup := setupTestDB(t)
    defer cleanup()
    
    // Empty payload
    db.Create(1, []byte{})
    
    // Very large payload
    large := make([]byte, 10*1024*1024)
    db.Create(2, large)
    
    // Maximum document ID
    db.Create(^uint64(0), []byte("max"))
    
    // Sequential operations
    for i := 1; i <= 1000; i++ {
        db.Create(uint64(i), []byte(fmt.Sprintf("doc%d", i)))
    }
}
```

---

## Running Tests

### Run All Tests

```bash
go test ./...
```

### Run Specific Test Package

```bash
go test ./tests/integration
go test ./tests/concurrency
go test ./tests/failure
```

### Run Specific Test

```bash
go test -run TestDatabaseOperations ./tests/integration
```

### Run with Verbose Output

```bash
go test -v ./tests/integration
```

### Run with Race Detector

```bash
go test -race ./tests/concurrency
```

### Run with Coverage

```bash
go test -cover ./tests/integration
go test -coverprofile=coverage.out ./tests/integration
go tool cover -html=coverage.out
```

---

## References

- [Go Testing Package](https://pkg.go.dev/testing)
- [Integration Tests](../tests/integration/integration_test.go)
- [Concurrency Tests](../tests/concurrency/concurrency_test.go)
- [Failure Tests](../tests/failure/failure_test.go)
- [Benchmarks](../tests/benchmarks/bench_test.go)

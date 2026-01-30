# Troubleshooting Guide

This document helps diagnose and resolve common issues with DocDB.

## Table of Contents

1. [Common Issues](#common-issues)
2. [Performance Problems](#performance-problems)
3. [Data Integrity Issues](#data-integrity-issues)
4. [Recovery Procedures](#recovery-procedures)
5. [Debugging Tips](#debugging-tips)
6. [Logging](#logging)
7. [Diagnostic Tools](#diagnostic-tools)

---

## Common Issues

### Issue: "Database not found" Error

**Symptoms:**
```
Error: database not found
```

**Causes:**
- Database was never created
- Database was deleted
- Database ID mismatch

**Solutions:**

1. **Verify Database Exists:**
```go
dbID, err := pool.CreateDB("mydb")
if err != nil {
    // Handle error
}
```

2. **Check Database ID:**
```go
// Store database ID after creation
dbID, err := pool.CreateDB("mydb")
// Use stored dbID, not a hardcoded value
```

3. **List All Databases:**
```go
// Check catalog for existing databases
// (Implementation depends on pool API)
```

---

### Issue: "Document not found" Error

**Symptoms:**
```
Error: document not found
```

**Causes:**
- Document was never created
- Document was deleted
- Document ID mismatch
- Reading from wrong database

**Solutions:**

1. **Verify Document Exists:**
```go
// Check if document exists before reading
data, err := db.Read(docID)
if err == docdb.ErrDocNotFound {
    // Document doesn't exist
}
```

2. **Check Document ID:**
```go
// Use correct document ID
// Document IDs are uint64, not strings
```

3. **Verify Database:**
```go
// Ensure you're reading from correct database
db, err := pool.OpenDB(dbID)
if err != nil {
    // Database not open
}
```

---

### Issue: "Memory limit exceeded" Error

**Symptoms:**
```
Error: memory limit exceeded
```

**Causes:**
- Document payload too large
- Too many documents
- Memory limit too low

**Solutions:**

1. **Increase Memory Limit:**
```go
cfg := config.DefaultConfig()
cfg.Memory.PerDBLimitMB = 1024  // Increase per-DB limit
cfg.Memory.GlobalCapacityMB = 4096  // Increase global limit
```

2. **Reduce Document Size:**
```go
// Split large documents into smaller chunks
// Or compress payload before storing
```

3. **Check Memory Usage:**
```go
stats := db.Stats()
fmt.Printf("Memory used: %d MB\n", stats.MemoryUsed / (1024 * 1024))
fmt.Printf("Memory limit: %d MB\n", stats.MemoryCapacity / (1024 * 1024))
```

4. **Delete Old Documents:**
```go
// Remove unused documents to free memory
db.Delete(docID)
```

---

### Issue: "Queue full" Error

**Symptoms:**
```
Error: queue full
```

**Causes:**
- Too many pending requests
- Workers overloaded
- Slow operations blocking queue

**Solutions:**

1. **Increase Queue Depth:**
```go
cfg := config.DefaultConfig()
cfg.Scheduler.QueueDepth = 1000  // Increase queue size
```

2. **Reduce Request Rate:**
```go
// Implement backoff or rate limiting
// Batch operations instead of many small requests
```

3. **Check Worker Status:**
```go
// Monitor worker pool health
// Ensure workers are processing requests
```

4. **Use Batch Operations:**
```go
// Instead of many individual operations:
for i := 0; i < 100; i++ {
    db.Create(uint64(i), data[i])
}

// Use batch:
tx := db.Begin()
for i := 0; i < 100; i++ {
    db.CreateInTx(tx, uint64(i), data[i])
}
db.Commit(tx)
```

---

### Issue: "WAL write failed" Error

**Symptoms:**
```
Error: failed to write WAL
```

**Causes:**
- Disk full
- Permission denied
- I/O error
- Disk failure

**Solutions:**

1. **Check Disk Space:**
```bash
df -h /path/to/data/dir
```

2. **Check Permissions:**
```bash
ls -la /path/to/data/dir
chmod 755 /path/to/data/dir
```

3. **Check Disk Health:**
```bash
# Check for disk errors
dmesg | grep -i error
```

4. **Handle Errors Gracefully:**
```go
err := db.Create(docID, payload)
if err != nil {
    if strings.Contains(err.Error(), "disk") {
        // Handle disk full
    }
    // Retry or fail gracefully
}
```

---

### Issue: "Database already exists" Error

**Symptoms:**
```
Error: database already exists
```

**Causes:**
- Attempting to create duplicate database
- Database name collision

**Solutions:**

1. **Check Before Creating:**
```go
// Check if database exists first
// (Implementation depends on pool API)
```

2. **Use Unique Names:**
```go
dbName := fmt.Sprintf("mydb_%d", time.Now().Unix())
dbID, err := pool.CreateDB(dbName)
```

3. **Delete and Recreate:**
```go
// Delete existing database first
pool.DeleteDB(existingDBID)
dbID, err := pool.CreateDB("mydb")
```

---

## Performance Problems

### Problem: Slow Writes

**Symptoms:**
- Write operations taking > 10ms
- High latency on commits

**Diagnosis:**

1. **Check Fsync Setting:**
```go
cfg := config.DefaultConfig()
if cfg.WAL.FsyncOnCommit {
    // Fsync adds latency (10-100ms per write)
    // Disable if durability not critical
    cfg.WAL.FsyncOnCommit = false
}
```

2. **Check Disk I/O:**
```bash
# Monitor disk I/O
iostat -x 1
```

3. **Check WAL Size:**
```go
stats := db.Stats()
fmt.Printf("WAL size: %d bytes\n", stats.WALSize)
// Large WAL may slow down writes
```

**Solutions:**

1. **Disable Fsync (if acceptable):**
```go
cfg.WAL.FsyncOnCommit = false
// Faster writes, but less durable
```

2. **Use Faster Storage:**
- Use SSD instead of HDD
- Use NVMe for best performance

3. **Batch Operations:**
```go
// Batch multiple operations
tx := db.Begin()
for i := 0; i < 100; i++ {
    db.CreateInTx(tx, uint64(i), data[i])
}
db.Commit(tx)  // Single WAL write
```

---

### Problem: Slow Reads

**Symptoms:**
- Read operations taking > 1ms
- High latency on lookups

**Diagnosis:**

1. **Check Shard Distribution:**
```go
// Document IDs should be distributed across shards
// Sequential IDs → all in same shard → contention
```

2. **Check Index Size:**
```go
size := db.IndexSize()
fmt.Printf("Index size: %d documents\n", size)
// Large index may slow lookups
```

3. **Check Memory Usage:**
```go
stats := db.Stats()
fmt.Printf("Memory usage: %d MB\n", stats.MemoryUsed / (1024 * 1024))
// High memory usage may cause swapping
```

**Solutions:**

1. **Distribute Document IDs:**
```go
// Use hash-based IDs instead of sequential
docID := hash(userID) % maxDocID
```

2. **Increase Memory:**
```go
cfg.Memory.PerDBLimitMB = 2048  // More memory for index
```

3. **Use Different Shards:**
```go
// Distribute reads across different shards
// Different shards = concurrent reads
```

---

### Problem: High Memory Usage

**Symptoms:**
- Memory usage approaching limit
- Frequent "memory limit exceeded" errors

**Diagnosis:**

1. **Check Memory Stats:**
```go
stats := db.Stats()
fmt.Printf("Memory used: %d MB\n", stats.MemoryUsed / (1024 * 1024))
fmt.Printf("Memory limit: %d MB\n", stats.MemoryCapacity / (1024 * 1024))
fmt.Printf("Usage: %.2f%%\n", float64(stats.MemoryUsed) / float64(stats.MemoryCapacity) * 100)
```

2. **Check Document Count:**
```go
size := db.IndexSize()
fmt.Printf("Document count: %d\n", size)
```

**Solutions:**

1. **Increase Memory Limit:**
```go
cfg.Memory.PerDBLimitMB = 4096  // Increase limit
cfg.Memory.GlobalCapacityMB = 16384  // Increase global limit
```

2. **Delete Unused Documents:**
```go
// Remove old or unused documents
db.Delete(docID)
```

3. **Compact Database:**
```go
// Run compaction to remove old versions
// (Implementation depends on compaction API)
```

4. **Reduce Document Size:**
```go
// Compress payloads before storing
compressed := compress(payload)
db.Create(docID, compressed)
```

---

## Data Integrity Issues

### Issue: Data Loss After Crash

**Symptoms:**
- Documents missing after restart
- Partial transactions lost

**Diagnosis:**

1. **Check WAL File:**
```bash
# Verify WAL file exists
ls -lh /path/to/wal/*.wal

# Check WAL size
stat /path/to/wal/mydb.wal
```

2. **Check Recovery Logs:**
```go
// Enable verbose logging
log := logger.New(os.Stdout, logger.LevelDebug, "")
```

**Solutions:**

1. **Enable Fsync:**
```go
cfg.WAL.FsyncOnCommit = true
// Ensures durability (slower but safer)
```

2. **Verify WAL Replay:**
```go
// Check that WAL replay completes successfully
// Look for errors in logs during startup
```

3. **Check Disk Health:**
```bash
# Verify disk is healthy
fsck /dev/sda1
```

---

### Issue: Corrupted WAL

**Symptoms:**
- Database fails to open
- "WAL corruption" errors
- Partial data recovery

**Diagnosis:**

1. **Check WAL File:**
```bash
# Check file size (should be > 0)
ls -lh /path/to/wal/mydb.wal

# Check for corruption
file /path/to/wal/mydb.wal
```

2. **Check Logs:**
```go
// Look for CRC32 errors in logs
// "WAL record CRC32 mismatch"
```

**Solutions:**

1. **Automatic Recovery:**
```go
// DocDB automatically truncates corrupted WAL
// Data up to last valid record is recovered
// Corrupted transaction is lost (by design)
```

2. **Manual Recovery:**
```bash
# Backup corrupted WAL
cp /path/to/wal/mydb.wal /path/to/wal/mydb.wal.backup

# Database will truncate on next open
# Lost data from corrupted transaction
```

3. **Prevention:**
```go
// Use reliable storage (SSD, RAID)
// Enable fsync for critical data
cfg.WAL.FsyncOnCommit = true
```

---

## Recovery Procedures

### Procedure: Recover from Crash

**Steps:**

1. **Check Database State:**
```bash
# Verify data directory exists
ls -la /path/to/data/

# Check WAL files
ls -la /path/to/wal/
```

2. **Restart Database:**
```go
// Database automatically replays WAL on startup
pool := pool.NewPool(cfg, log)
err := pool.Start()  // WAL replay happens here
```

3. **Verify Recovery:**
```go
// Check document count
size := db.IndexSize()
fmt.Printf("Recovered %d documents\n", size)

// Verify critical documents
data, err := db.Read(criticalDocID)
if err != nil {
    // Document not recovered
}
```

4. **Check Logs:**
```go
// Look for recovery messages
// "Replaying WAL: X records"
// "WAL replay complete"
```

---

### Procedure: Recover from Disk Full

**Steps:**

1. **Free Disk Space:**
```bash
# Remove unnecessary files
rm -rf /tmp/*

# Move data to larger disk
mv /path/to/data /new/path/to/data
```

2. **Update Configuration:**
```go
cfg.DataDir = "/new/path/to/data"
cfg.WAL.Dir = "/new/path/to/data/wal"
```

3. **Restart Database:**
```go
pool := pool.NewPool(cfg, log)
err := pool.Start()
```

4. **Verify Data:**
```go
// Check that all databases are accessible
// Verify document counts
```

---

## Debugging Tips

### Enable Verbose Logging

```go
log := logger.New(os.Stdout, logger.LevelDebug, "")
// Or
log := logger.New(os.Stdout, logger.LevelTrace, "")
```

### Check Internal State

```go
// Database stats
stats := db.Stats()
fmt.Printf("Stats: %+v\n", stats)

// Index size
size := db.IndexSize()
fmt.Printf("Index size: %d\n", size)

// Memory usage
memUsed := db.MemoryUsage()
fmt.Printf("Memory: %d bytes\n", memUsed)
```

### Use Race Detector

```bash
# Run tests with race detector
go test -race ./tests/concurrency

# Run server with race detector
go run -race ./cmd/docdb
```

### Profile Performance

```bash
# CPU profile
go test -bench=. -cpuprofile=cpu.prof ./tests/benchmarks
go tool pprof cpu.prof

# Memory profile
go test -bench=. -memprofile=mem.prof ./tests/benchmarks
go tool pprof mem.prof
```

### Profiling the server

To profile the **running** DocDB server (CPU, mutex, heap), start it with the pprof HTTP server enabled:

```bash
./docdb -data-dir ./data -socket /tmp/docdb.sock -debug-addr localhost:6060
```

Then use `go tool pprof` against the server:

```bash
# CPU profile (30-second sample)
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Heap snapshot
go tool pprof http://localhost:6060/debug/pprof/heap

# Mutex contention
go tool pprof http://localhost:6060/debug/pprof/mutex

# Goroutine block profile (requires GODEBUG=mutexprofile=1 or blockprofile=1 if needed)
go tool pprof http://localhost:6060/debug/pprof/block
```

Leave `-debug-addr` empty (default) to disable pprof. Bind to localhost only; do not expose the pprof port to the network.

---

## Logging

### Log Levels

- **Error**: Critical errors only
- **Warn**: Warnings and recoverable errors
- **Info**: General information
- **Debug**: Detailed debugging information
- **Trace**: Very detailed tracing

### Configure Logging

```go
log := logger.New(os.Stdout, logger.LevelInfo, "")
// Or
log := logger.New(logFile, logger.LevelDebug, "")
```

### Common Log Messages

**Startup:**
```
[INFO] Starting DocDB pool
[INFO] Loading catalog
[INFO] Replaying WAL for database: mydb
[INFO] WAL replay complete: 1000 records
```

**Operations:**
```
[DEBUG] Creating document: docID=1, size=1024
[DEBUG] Updating document: docID=1
[DEBUG] Deleting document: docID=1
```

**Errors:**
```
[ERROR] Failed to write WAL: disk full
[ERROR] Memory limit exceeded: dbID=1, limit=1024MB
[WARN] WAL record CRC32 mismatch, truncating
```

---

## Diagnostic Tools

### Check Database Files

```bash
# List all databases
ls -lh /path/to/data/*.data

# Check WAL files
ls -lh /path/to/wal/*.wal

# Check catalog
cat /path/to/data/.catalog
```

### Monitor Resource Usage

```bash
# CPU usage
top -p $(pgrep docdb)

# Memory usage
ps aux | grep docdb

# Disk I/O
iostat -x 1

# File descriptors
lsof -p $(pgrep docdb)
```

### Verify Data Integrity

```go
// Read all documents and verify
for i := 1; i <= maxDocID; i++ {
    data, err := db.Read(uint64(i))
    if err != nil && err != docdb.ErrDocNotFound {
        // Error reading document
    }
}
```

---

## Getting Help

### Collect Diagnostic Information

When reporting issues, include:

1. **Error Messages:**
   - Full error text
   - Stack traces (if available)

2. **Configuration:**
   - Memory limits
   - WAL settings
   - Data directory paths

3. **System Information:**
   - OS version
   - Go version
   - Disk space
   - Memory available

4. **Logs:**
   - Relevant log entries
   - Error logs
   - Debug logs (if enabled)

5. **Reproduction Steps:**
   - Steps to reproduce
   - Expected behavior
   - Actual behavior

---

## References

- [Failure Modes](failure_modes.md) - Failure handling details
- [Configuration Guide](configuration.md) - Configuration options
- [Usage Guide](usage.md) - Usage patterns
- [Architecture Guide](architecture.md) - System architecture

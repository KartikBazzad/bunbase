# Bundoc Configuration Guide

**Version:** 1.0  
**Last Updated:** February 1, 2026

---

## Configuration Options

### Options Struct

```go
type Options struct {
    Path           string
    BufferPoolSize int
    WALSegmentSize int64
    MetadataPath   string
}
```

---

## Path

**Type:** `string` (required)  
**Description:**: Directory where database files will be stored

**Example:**

```go
opts := bundoc.Options{
    Path: "./data/mydb",
}
```

**Notes:**

- Directory will be created if it doesn't exist
- Must have read/write permissions
- One bundoc instance per directory (no concurrent access)

---

## BufferPoolSize

**Type:** `int`  
**Default:** `256`  
**Unit:** Number of 8KB pages  
**Memory:** `BufferPoolSize × 8KB`

**Description:** Number of database pages to cache in memory

### Presets

| Preset      | Pages   | Memory   | Use Case                      |
| ----------- | ------- | -------- | ----------------------------- |
| Minimal     | 64      | 512 KB   | Embedded devices, low memory  |
| Small       | 128     | 1 MB     | Small databases (\<100MB)     |
| **Default** | **256** | **2 MB** | **General purpose**           |
| Medium      | 512     | 4 MB     | Medium databases (100MB-1GB)  |
| Large       | 1024    | 8 MB     | Large databases (1GB-10GB)    |
| XLarge      | 4096    | 32 MB    | Very large databases (\>10GB) |

### Tuning Formula

```
BufferPoolSize = (Available Memory × 0.25) / 8KB
```

**Example:**

- System with 1GB RAM
- Allocate 25% = 250MB to buffer pool
- 250MB / 8KB = ~32,000 pages

**Code:**

```go
opts.BufferPoolSize = 32000
```

### When to Increase

✅ **Increase when:**

- Read performance is slow
- Workload is read-heavy
- Database size \> current pool size
- Disk I/O is high

❌ **Don't increase when:**

- Memory is limited
- Workload is write-heavy
- OOM errors occur

### Performance Impact

| Pool Size           | Read Latency | Disk I/O  |
| ------------------- | ------------ | --------- |
| 64 pages            | High         | Very High |
| 256 pages (default) | Medium       | Medium    |
| 1024 pages          | Low          | Low       |
| 4096 pages          | Very Low     | Very Low  |

---

## WALSegmentSize

**Type:** `int64`  
**Default:** `67108864` (64MB)  
**Unit:** Bytes

**Description:** Size at which WAL segments rotate to new files

### Presets

| Preset      | Size      | Use Case                    |
| ----------- | --------- | --------------------------- |
| Small       | 16 MB     | Limited disk space          |
| Medium      | 32 MB     | Default for small databases |
| **Default** | **64 MB** | **Most workloads**          |
| Large       | 128 MB    | High-write workloads        |
| XLarge      | 256 MB    | Very high throughput        |

### Tuning Guidelines

**Smaller segments (16-32MB):**

- ✅ Faster recovery (less to replay)
- ✅ Less disk space usage
- ❌ More file handles
- ❌ More segment rotations

**Larger segments (128-256MB):**

- ✅ Fewer file handles
- ✅ Fewer segment rotations
- ❌ Slower recovery
- ❌ More disk space

### Code Examples

```go
// Small (16MB)
opts.WALSegmentSize = 16 * 1024 * 1024

// Default (64MB)
opts.WALSegmentSize = 64 * 1024 * 1024

// Large (128MB)
opts.WALSegmentSize = 128 * 1024 * 1024
```

---

---

## MetadataPath

**Type:** `string` (optional)  
**Default:** `<Path>/system_catalog.json`

**Description:** Location of the system catalog file which stores collection schemas and index locations.

**Example:**

```go
opts := bundoc.Options{
    Path:         "./data",
    MetadataPath: "./data/custom_catalog.json",
}
```

---

## Configuration Profiles

### Profile 1: Low Memory (Embedded)

**Scenario:** Embedded devices, IoT, mobile

```go
opts := bundoc.Options{
    Path:           "./data",
    BufferPoolSize: 64,   // 512KB
    WALSegmentSize: 16 * 1024 * 1024, // 16MB
}
```

**Characteristics:**

- Memory footprint: ~1MB
- Disk space: Minimal
- Performance: Basic

---

### Profile 2: Default (Balanced)

**Scenario:** General-purpose applications

```go
opts := bundoc.DefaultOptions("./data")
// or explicitly:
opts := bundoc.Options{
    Path:           "./data",
    BufferPoolSize: 256,  // 2MB
    WALSegmentSize: 64 * 1024 * 1024, // 64MB
}
```

**Characteristics:**

- Memory footprint: ~3MB
- Disk space: Moderate
- Performance: Good

---

### Profile 3: High Performance (Server)

**Scenario:** Server applications, high concurrency

```go
opts := bundoc.Options{
    Path:           "./data",
    BufferPoolSize: 4096, // 32MB
    WALSegmentSize: 128 * 1024 * 1024, // 128MB
}
```

**Characteristics:**

- Memory footprint: ~35MB
- Disk space: Higher
- Performance: Excellent

---

### Profile 4: Read-Heavy

**Scenario:** Analytics, reporting, mostly reads

```go
opts := bundoc.Options{
    Path:           "./data",
    BufferPoolSize: 8192, // 64MB - Large cache!
    WALSegmentSize: 32 * 1024 * 1024, // 32MB - Smaller WAL
}
```

**Why:**

- Large buffer pool → more hot data in memory
- Smaller WAL → faster recovery (reads don't generate many WAL entries)

---

### Profile 5: Write-Heavy

**Scenario:** Logging, event sourcing, high write throughput

```go
opts := bundoc.Options{
    Path:           "./data",
    BufferPoolSize: 512,  // 4MB - Moderate cache
    WALSegmentSize: 256 * 1024 * 1024, // 256MB - Large WAL
}
```

**Why:**

- Larger WAL → fewer segment rotations → less overhead
- Moderate cache → balance between reads and writes

---

## Environment-Specific Configurations

### Development

```go
opts := bundoc.DefaultOptions("./dev-data")
```

**Rationale:**

- Defaults are fine for local development
- Easy to delete and recreate

---

### Testing

```go
opts := bundoc.Options{
    Path:           t.TempDir(), // Unique per test
    BufferPoolSize: 128,  // Smaller for speed
    WALSegmentSize: 16 * 1024 * 1024,
}
```

**Rationale:**

- Isolated per test
- Smaller footprint → faster cleanup
- Auto-deleted after test

---

### Production

```go
opts := bundoc.Options{
    Path:           "/var/lib/myapp/bundoc",
    BufferPoolSize: 2048, // 16MB - Tuned for workload
    WALSegmentSize: 128 * 1024 * 1024,
}
```

**Rationale:**

- Larger buffer pool for performance
- Persistent storage location
- Tuned based on profiling

---

## Dynamic Tuning

### Monitoring

Track these metrics to inform tuning decisions:

```go
// (Future feature - not in v1.0)
stats := db.GetStats()
fmt.Printf("Buffer pool hit rate: %.2f%%\n", stats.HitRate*100)
fmt.Printf("WAL segments: %d\n", stats.WALSegments)
```

### Tuning Based on Metrics

| Metric          | Value | Action                      |
| --------------- | ----- | --------------------------- |
| Buffer hit rate | \<80% | Increase BufferPoolSize     |
| Buffer hit rate | \>95% | Can decrease BufferPoolSize |
| WAL segments    | \>100 | Increase WALSegmentSize     |
| Recovery time   | \>10s | Decrease WALSegmentSize     |

---

## Advanced Configuration

### Custom Paths

```go
opts := bundoc.Options{
    Path:           "/mnt/ssd/fast-data",  // Put on fast SSD
    BufferPoolSize: 1024,
    WALSegmentSize: 64 * 1024 * 1024,
}
```

**Tips:**

- Use SSD for better performance
- Separate data and WAL (future feature)
- Ensure sufficient disk space (DBsize + 2× WAL size)

---

### Multiple Databases

```go
// Database 1: Users
db1, _ := bundoc.Open(bundoc.Options{
    Path:           "./data/users",
    BufferPoolSize: 1024,
})

// Database 2: Logs (write-heavy)
db2, _ := bundoc.Open(bundoc.Options{
    Path:           "./data/logs",
    BufferPoolSize: 256,  // Smaller cache
    WALSegmentSize: 256 * 1024 * 1024, // Larger WAL
})
```

**Benefits:**

- Each database has independent configuration
- Shared global flusher still batches fsync calls
- Isolation between workloads

---

## Validation

### Validating Options

```go
func validateOptions(opts *Options) error {
    if opts.Path == "" {
        return errors.New("path is required")
    }

    if opts.BufferPoolSize < 1 {
        return errors.New("buffer pool must be >= 1")
    }

    if opts.WALSegmentSize < 1024*1024 {
        return errors.New("WAL segment must be >= 1MB")
    }

    return nil
}
```

---

## Migration

### Changing Configuration

**Current Limitation:** Options cannot be changed after database creation.

**Workaround:**

1. Export data from old database
2. Create new database with new options
3. Import data

**Example:**

```go
// Old database
oldDB, _ := bundoc.Open(bundoc.Options{Path: "./old", BufferPoolSize: 256})

// Export
oldColl, _ := oldDB.GetCollection("users")
// ... export logic ...

oldDB.Close()

// New database with different config
newDB, _ := bundoc.Open(bundoc.Options{Path: "./new", BufferPoolSize: 1024})

// Import
newColl, _ := newDB.CreateCollection("users")
// ... import logic ...
```

---

## Troubleshooting

### Issue: Out of Memory

**Symptom:** OOM errors, high memory usage

**Solution:**

```go
// Reduce buffer pool
opts.BufferPoolSize = 128  // Was 1024
```

---

### Issue: Slow Reads

**Symptom:** High read latency

**Solution:**

```go
// Increase buffer pool
opts.BufferPoolSize = 2048  // Was 256
```

---

### Issue: Slow Recovery

**Symptom:** Long startup time after crash

**Solution:**

```go
// Reduce WAL segment size
opts.WALSegmentSize = 32 * 1024 * 1024  // Was 128MB
```

---

### Issue: Too Many Open Files

**Symptom:** "too many open files" error

**Solution:**

```go
// Increase WAL segment size (fewer files)
opts.WALSegmentSize = 256 * 1024 * 1024  // Was 16MB

// Or increase system limit:
// ulimit -n 4096
```

---

## Checklist

### Before Production

- [ ] BufferPoolSize tuned for workload
- [ ] WALSegmentSize appropriate for write rate
- [ ] Sufficient disk space (3× database size)
- [ ] Path permissions verified
- [ ] Backup strategy in place
- [ ] Monitoring configured (future)

---

## Summary

| Parameter      | Default | Min | Max   | Impact                      |
| -------------- | ------- | --- | ----- | --------------------------- |
| BufferPoolSize | 256     | 1   | ~100k | Read performance            |
| WALSegmentSize | 64MB    | 1MB | ~1GB  | Write performance, recovery |

**Golden Rule:** Start with defaults, tune based on profiling!

---

**For API details**: See [API.md](./API.md)  
**For architecture**: See [ARCHITECTURE.md](./ARCHITECTURE.md)  
**For performance**: See [PERFORMANCE.md](./PERFORMANCE.md)

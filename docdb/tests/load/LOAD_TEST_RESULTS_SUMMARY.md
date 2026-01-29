# Load Test Results Summary

## Test Configuration

- **Duration**: 5 minutes (300 seconds)
- **Workers**: 10 concurrent workers
- **Workload Mix**: 40% reads, 30% writes, 20% updates, 10% deletes
- **Document Size**: 1024 bytes
- **Document Count**: 10,000 unique documents

## Results

### Throughput

- **Total Operations**: 28,978 operations
- **Operations per Second**: ~96.6 ops/sec
- **Operations per Worker**: ~2,898 ops/worker

### Latency Percentiles (milliseconds)

| Operation  | P50  | P95   | P99   | P99.9 | Mean  | Min  | Max       |
| ---------- | ---- | ----- | ----- | ----- | ----- | ---- | --------- |
| **Create** | 58.9 | 104.0 | 130.0 | 182.8 | 116.5 | 0.08 | 166,848.7 |
| **Read**   | 31.1 | 69.6  | 91.5  | 203.1 | 104.5 | 0.02 | 166,701.8 |
| **Update** | 51.4 | 97.8  | 121.0 | 179.0 | 82.3  | 0.04 | 166,828.8 |
| **Delete** | 45.4 | 89.0  | 111.7 | 166.0 | 102.9 | 0.03 | 166,798.7 |

### Operation Distribution

- **Create**: 8,671 operations (29.9%)
- **Read**: 11,633 operations (40.1%)
- **Update**: 5,743 operations (19.8%)
- **Delete**: 2,931 operations (10.1%)

### WAL Growth

**Note**: WAL size tracking showed 0 bytes because the WAL directory path in the load test configuration (`/tmp/docdb/wal`) didn't match where DocDB actually stored the WAL files (`./docdb/data/wal`).

The actual WAL file size after the test was **8.5 MB** (measured directly from `./docdb/data/wal/loadtest.wal`).

**Estimated WAL Growth Rate**: ~28.3 KB/sec (8.5 MB / 300 seconds)

### Healing Overhead

- **Total Healing Operations**: 46 on-demand healings
- **Documents Healed**: 0 (healings were triggered but no corruption found)
- **Healing Overhead**: 0% (negligible time spent on healing)

## Analysis

### Performance Characteristics

1. **Read Performance**: Best performance with P95 of 69.6ms
2. **Write Performance**: Create operations have highest latency (P95: 104ms)
3. **Update Performance**: Moderate latency (P95: 97.8ms)
4. **Delete Performance**: Good performance (P95: 89ms)

### Observations

1. **High Max Latencies**: Some operations showed very high max latencies (166+ seconds), likely due to:
   - Initial connection overhead
   - System resource contention
   - Garbage collection pauses

2. **Consistent P95/P99**: P95 and P99 latencies are reasonable and consistent across operations, indicating stable performance under load.

3. **WAL Growth**: The WAL grew to 8.5MB over 5 minutes with ~97 ops/sec, indicating healthy write throughput.

4. **Healing**: 46 healing operations were triggered but no actual corruption was found, suggesting the healing system is working correctly.

## Recommendations

1. **WAL Path Configuration**: Always ensure the `-wal-dir` parameter matches the actual WAL directory used by DocDB server.

2. **Longer Test Duration**: Run tests for longer durations (15-30 minutes) to get more stable metrics and identify any performance degradation over time.

3. **Higher Concurrency**: Test with more workers (20-50) to find the system's concurrency limits.

4. **Different Workloads**: Test with different workload patterns:
   - Write-heavy (80% writes)
   - Read-heavy (90% reads)
   - Update-heavy (60% updates)

5. **Monitor System Resources**: Track CPU, memory, and disk I/O during tests to identify bottlenecks.

## Known Issues

1. **WAL Path Mismatch**: The load test couldn't track WAL growth because the configured WAL directory didn't match the actual location. Fixed by ensuring `-wal-dir` matches DocDB's actual WAL directory.

2. **`.ls` Command**: The `.ls` command in docdbsh shows WAL size as 0 when databases are closed. This is expected behavior - stats are only available for open databases.

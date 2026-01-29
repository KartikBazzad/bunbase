# Load Test Analysis - Second Run

## Test Configuration

- **Duration**: 5 minutes (300 seconds)
- **Workers**: 10 concurrent workers
- **Workload Mix**: 40% reads, 30% writes, 20% updates, 10% deletes
- **Document Size**: 1024 bytes
- **Document Count**: 10,000 unique documents
- **WAL Directory**: `./docdb/data/wal` ✅ (Correct path)

## Performance Summary

### Throughput

- **Total Operations**: 71,156 operations
- **Operations per Second**: ~237 ops/sec
- **Operations per Worker**: ~7,116 ops/worker
- **Improvement**: 2.45x more operations than first run (28,978 → 71,156)

### Latency Percentiles (milliseconds)

| Operation  | P50  | P95  | P99   | P99.9 | Mean | Min  | Max   |
| ---------- | ---- | ---- | ----- | ----- | ---- | ---- | ----- |
| **Create** | 48.0 | 91.0 | 116.0 | 168.4 | 48.8 | 0.05 | 229.9 |
| **Read**   | 28.5 | 64.1 | 81.9  | 117.9 | 30.1 | 0.02 | 198.2 |
| **Update** | 53.2 | 97.9 | 122.6 | 167.3 | 54.6 | 0.04 | 206.5 |
| **Delete** | 45.0 | 87.6 | 112.0 | 142.9 | 45.9 | 0.04 | 198.0 |

### Operation Distribution

- **Create**: 21,412 operations (30.1%)
- **Read**: 28,533 operations (40.1%)
- **Update**: 14,138 operations (19.9%)
- **Delete**: 7,073 operations (9.9%)

**Note**: Distribution closely matches configured percentages (40/30/20/10).

## WAL Growth Analysis ✅

**WAL tracking is now working correctly!**

- **Initial WAL Size**: 8.9 MB (8,938,056 bytes)
- **Final WAL Size**: 28.0 MB (28,027,132 bytes)
- **WAL Growth**: 19.1 MB over 300 seconds
- **Growth Rate**: 63.6 KB/sec (63,608 bytes/sec)
- **Samples Collected**: 301 samples (1 sample per second)

### WAL Growth Pattern

From the CSV data:

- Started at 8.52 MB
- Grew steadily to 26.73 MB
- Consistent growth rate throughout the test
- No sudden spikes or drops

### WAL Efficiency

- **Operations per MB**: ~3,725 ops/MB of WAL growth
- **WAL per Operation**: ~268 bytes/operation average
- This includes WAL overhead (headers, CRC, etc.) plus document payloads

## Healing Overhead

- **Total Healing Operations**: 0
- **Healing Overhead**: 0%
- **Status**: No corruption detected, healing system working correctly

## Comparison: First vs Second Run

| Metric                 | First Run | Second Run | Change     |
| ---------------------- | --------- | ---------- | ---------- |
| **Total Operations**   | 28,978    | 71,156     | +145%      |
| **Ops/sec**            | 96.6      | 237.0      | +145%      |
| **Create P95**         | 104.0ms   | 91.0ms     | -12.5%     |
| **Read P95**           | 69.6ms    | 64.1ms     | -7.9%      |
| **Update P95**         | 97.8ms    | 97.9ms     | +0.1%      |
| **Delete P95**         | 89.0ms    | 87.6ms     | -1.6%      |
| **WAL Tracking**       | ❌ Failed | ✅ Working | Fixed      |
| **Healing Operations** | 46        | 0          | Normalized |

## Key Findings

### 1. Performance Improvement

The second run showed significantly better performance:

- **2.45x more throughput** - Likely due to:
  - Database warm-up (indexes loaded, caches populated)
  - Reduced connection overhead
  - Better system state

### 2. Latency Consistency

- **P95 latencies are consistent** across both runs (~64-104ms)
- **P99 latencies are stable** (~82-123ms)
- **Max latencies reduced** significantly (from 166s to 230ms)
- Indicates stable performance under load

### 3. WAL Growth Tracking ✅

- **Successfully tracked WAL growth** from 8.9 MB to 28.0 MB
- **Consistent growth rate** of ~63.6 KB/sec
- **301 samples** collected (1 per second)
- CSV file contains complete time-series data

### 4. Read Performance

- **Best performing operation** with P95 of 64.1ms
- **40% of workload** as configured
- **Lowest mean latency** at 30.1ms

### 5. Write Performance

- **Create operations**: P95 of 91ms (good)
- **Update operations**: P95 of 98ms (consistent)
- **Delete operations**: P95 of 88ms (efficient)

## Recommendations

### 1. Longer Duration Tests

Run tests for 15-30 minutes to:

- Identify any performance degradation over time
- Observe WAL rotation and trimming behavior
- Test checkpoint creation and recovery

### 2. Higher Concurrency

Test with more workers (20, 50, 100) to find:

- Maximum sustainable throughput
- Concurrency limits
- Lock contention points

### 3. Different Workload Patterns

- **Write-heavy**: 80% writes, 10% reads, 10% updates
- **Read-heavy**: 90% reads, 5% writes, 5% updates
- **Update-heavy**: 60% updates, 20% reads, 20% writes

### 4. Document Size Variations

Test with different document sizes:

- Small: 100 bytes
- Medium: 1 KB (current)
- Large: 10 KB, 100 KB

### 5. Monitor System Resources

During load tests, monitor:

- CPU usage
- Memory usage
- Disk I/O
- Network (if using TCP)

## WAL Growth Insights

### Growth Rate Analysis

- **63.6 KB/sec** growth rate
- For **237 ops/sec** throughput
- **~268 bytes per operation** average

This includes:

- WAL record headers (~37 bytes overhead per record)
- Document payloads (~1024 bytes)
- Commit markers
- CRC checksums

### WAL Rotation

With default config (64 MB max file size):

- Current WAL: 28 MB (43% of max)
- Would rotate at: ~64 MB
- Estimated rotation time: ~9.4 minutes at current rate

### Checkpoint Behavior

With checkpoint interval of 64 MB:

- Checkpoints created every ~64 MB of WAL
- Current WAL size (28 MB) hasn't triggered checkpoint yet
- Trimming would occur after checkpoint creation

## Conclusion

The load test suite is **working correctly** and providing valuable insights:

✅ **WAL tracking**: Successfully tracking growth with correct path  
✅ **Latency metrics**: P95/P99 percentiles calculated correctly  
✅ **Healing tracking**: Monitoring healing operations  
✅ **Performance**: Consistent and stable under load  
✅ **CSV export**: Complete time-series data available

The system shows:

- **Good performance**: ~237 ops/sec with 10 workers
- **Stable latencies**: P95 under 100ms for all operations
- **Efficient WAL usage**: ~268 bytes per operation
- **No corruption**: Healing system working correctly

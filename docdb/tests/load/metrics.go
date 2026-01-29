package load

import (
	"sort"
	"sync"
	"time"
)

// OperationType represents the type of database operation.
type OperationType string

const (
	OpCreate OperationType = "create"
	OpRead   OperationType = "read"
	OpUpdate OperationType = "update"
	OpDelete OperationType = "delete"
)

// LatencyMetrics tracks latency statistics for operations.
type LatencyMetrics struct {
	mu sync.RWMutex

	// Histogram buckets: [0-1ms), [1-2ms), [2-5ms), [5-10ms), [10-50ms), [50-100ms), [100ms+)
	buckets []int64
	counts  map[OperationType][]int64

	// Raw samples for exact percentile calculation
	samples map[OperationType][]time.Duration

	// Summary statistics
	totals map[OperationType]int64
	sums   map[OperationType]time.Duration
	mins   map[OperationType]time.Duration
	maxs   map[OperationType]time.Duration
}

// NewLatencyMetrics creates a new LatencyMetrics instance.
func NewLatencyMetrics() *LatencyMetrics {
	buckets := []int64{1, 2, 5, 10, 50, 100} // milliseconds
	ops := []OperationType{OpCreate, OpRead, OpUpdate, OpDelete}

	counts := make(map[OperationType][]int64)
	for _, op := range ops {
		counts[op] = make([]int64, len(buckets)+1) // +1 for overflow bucket
	}

	return &LatencyMetrics{
		buckets: buckets,
		counts:  counts,
		samples: make(map[OperationType][]time.Duration),
		totals:  make(map[OperationType]int64),
		sums:    make(map[OperationType]time.Duration),
		mins:    make(map[OperationType]time.Duration),
		maxs:    make(map[OperationType]time.Duration),
	}
}

// Record records a latency measurement for an operation.
func (lm *LatencyMetrics) Record(op OperationType, latency time.Duration) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Update histogram
	ms := latency.Milliseconds()
	bucketIdx := len(lm.buckets)
	for i, threshold := range lm.buckets {
		if ms < threshold {
			bucketIdx = i
			break
		}
	}
	lm.counts[op][bucketIdx]++

	// Store sample for exact percentile calculation
	lm.samples[op] = append(lm.samples[op], latency)

	// Update summary statistics
	lm.totals[op]++
	lm.sums[op] += latency

	if lm.mins[op] == 0 || latency < lm.mins[op] {
		lm.mins[op] = latency
	}
	if latency > lm.maxs[op] {
		lm.maxs[op] = latency
	}
}

// PercentileStats contains percentile statistics for an operation type.
type PercentileStats struct {
	P50   float64 // milliseconds
	P95   float64 // milliseconds
	P99   float64 // milliseconds
	P999  float64 // milliseconds
	Mean  float64 // milliseconds
	Min   float64 // milliseconds
	Max   float64 // milliseconds
	Count int64
}

// GetStats returns percentile statistics for all operation types.
func (lm *LatencyMetrics) GetStats() map[OperationType]PercentileStats {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	stats := make(map[OperationType]PercentileStats)
	ops := []OperationType{OpCreate, OpRead, OpUpdate, OpDelete}

	for _, op := range ops {
		stats[op] = lm.getStatsForOp(op)
	}

	return stats
}

// getStatsForOp calculates statistics for a single operation type.
func (lm *LatencyMetrics) getStatsForOp(op OperationType) PercentileStats {
	samples := lm.samples[op]
	if len(samples) == 0 {
		return PercentileStats{}
	}

	// Sort samples for percentile calculation
	sorted := make([]time.Duration, len(samples))
	copy(sorted, samples)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Calculate percentiles
	p50 := percentile(sorted, 0.50)
	p95 := percentile(sorted, 0.95)
	p99 := percentile(sorted, 0.99)
	p999 := percentile(sorted, 0.999)

	// Calculate mean
	mean := lm.sums[op] / time.Duration(lm.totals[op])

	return PercentileStats{
		P50:   float64(p50.Nanoseconds()) / 1e6, // convert to milliseconds
		P95:   float64(p95.Nanoseconds()) / 1e6,
		P99:   float64(p99.Nanoseconds()) / 1e6,
		P999:  float64(p999.Nanoseconds()) / 1e6,
		Mean:  float64(mean.Nanoseconds()) / 1e6,
		Min:   float64(lm.mins[op].Nanoseconds()) / 1e6,
		Max:   float64(lm.maxs[op].Nanoseconds()) / 1e6,
		Count: lm.totals[op],
	}
}

// percentile calculates the percentile value from sorted samples.
func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if p >= 1.0 {
		return sorted[len(sorted)-1]
	}
	if p <= 0.0 {
		return sorted[0]
	}

	idx := p * float64(len(sorted)-1)
	lower := int(idx)
	upper := lower + 1

	if upper >= len(sorted) {
		return sorted[lower]
	}

	// Linear interpolation
	weight := idx - float64(lower)
	lowerVal := float64(sorted[lower].Nanoseconds())
	upperVal := float64(sorted[upper].Nanoseconds())
	interpolated := lowerVal + weight*(upperVal-lowerVal)

	return time.Duration(interpolated)
}

// GetAllSamples returns all latency samples (for CSV export).
func (lm *LatencyMetrics) GetAllSamples() map[OperationType][]time.Duration {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	result := make(map[OperationType][]time.Duration)
	for op, samples := range lm.samples {
		result[op] = make([]time.Duration, len(samples))
		copy(result[op], samples)
	}
	return result
}

// Reset clears all metrics.
func (lm *LatencyMetrics) Reset() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	ops := []OperationType{OpCreate, OpRead, OpUpdate, OpDelete}
	for _, op := range ops {
		lm.counts[op] = make([]int64, len(lm.buckets)+1)
		lm.samples[op] = nil
		lm.totals[op] = 0
		lm.sums[op] = 0
		lm.mins[op] = 0
		lm.maxs[op] = 0
	}
}

package loadtest

import (
	"sort"
	"sync"
	"time"
)

// Stats collects latency samples and computes percentiles and throughput.
type Stats struct {
	mu        sync.Mutex
	latencies []time.Duration
	ops       int64
	errors    int64
}

// NewStats creates a new Stats collector.
func NewStats() *Stats {
	return &Stats{latencies: make([]time.Duration, 0, 65536)}
}

// Record records one operation with the given latency. If err is non-nil, it counts as an error.
func (s *Stats) Record(latency time.Duration, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ops++
	if err != nil {
		s.errors++
		return
	}
	s.latencies = append(s.latencies, latency)
}

// Report computes and returns a summary: total ops, errors, duration, ops/sec, P50/P95/P99 latency.
func (s *Stats) Report(duration time.Duration) Report {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := Report{
		TotalOps: s.ops,
		Errors:   s.errors,
		Duration: duration,
	}
	if duration > 0 {
		r.OpsPerSec = float64(s.ops) / duration.Seconds()
	}
	if len(s.latencies) == 0 {
		return r
	}
	lats := make([]time.Duration, len(s.latencies))
	copy(lats, s.latencies)
	s.mu.Unlock()
	sort.Slice(lats, func(i, j int) bool { return lats[i] < lats[j] })
	s.mu.Lock()
	n := len(lats)
	r.P50Latency = lats[(n*50)/100]
	r.P95Latency = lats[(n*95)/100]
	r.P99Latency = lats[(n*99)/100]
	return r
}

// Report is the result of a load test run.
type Report struct {
	TotalOps   int64
	Errors     int64
	Duration   time.Duration
	OpsPerSec  float64
	P50Latency time.Duration
	P95Latency time.Duration
	P99Latency time.Duration
}

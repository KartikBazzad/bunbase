// Bunder loadtest runs load tests against a Bunder server and prints ops/sec and latency percentiles.
//
// Usage:
//
//	go run ./cmd/loadtest -addr 127.0.0.1:6379 -duration 10s -clients 50
//	go run ./cmd/loadtest -addr 127.0.0.1:6379 -workload set -duration 5s
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/kartikbazzad/bunbase/bunder/internal/loadtest"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:6379", "Bunder server address")
	duration := flag.Duration("duration", 10*time.Second, "Load test duration")
	clients := flag.Int("clients", 50, "Number of concurrent clients")
	keyspace := flag.Int("keys", 10000, "Key space size (distinct keys)")
	valuesize := flag.Int("value-size", 64, "Value size in bytes")
	workload := flag.String("workload", "mixed", "Workload: set, get, or mixed")
	flag.Parse()

	var w loadtest.Workload
	switch *workload {
	case "set":
		w = loadtest.WorkloadSet
	case "get":
		w = loadtest.WorkloadGet
	case "mixed":
		w = loadtest.WorkloadMixed
	default:
		fmt.Fprintf(os.Stderr, "workload must be set, get, or mixed\n")
		os.Exit(1)
	}

	cfg := loadtest.DefaultConfig(*addr)
	cfg.Duration = *duration
	cfg.NumClients = *clients
	cfg.KeySpace = *keyspace
	cfg.ValueSize = *valuesize
	cfg.Workload = w

	ctx := context.Background()
	fmt.Printf("Bunder load test: addr=%s duration=%v clients=%d keys=%d value-size=%d workload=%s\n",
		*addr, *duration, *clients, *keyspace, *valuesize, *workload)

	report, err := loadtest.Run(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load test: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("--- results ---")
	fmt.Printf("total_ops=%d errors=%d duration=%v\n", report.TotalOps, report.Errors, report.Duration)
	fmt.Printf("ops_per_sec=%.2f\n", report.OpsPerSec)
	fmt.Printf("latency_p50=%v latency_p95=%v latency_p99=%v\n",
		report.P50Latency, report.P95Latency, report.P99Latency)
}

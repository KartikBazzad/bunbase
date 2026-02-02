// BunKMS loadtest runs load/stress tests against a BunKMS server.
//
// Usage:
//
//	# Start BunKMS first, then:
//	go run ./cmd/loadtest -url http://localhost:8080 -duration 10s -clients 20
//	go run ./cmd/loadtest -url http://localhost:8080 -workload encrypt -duration 5s
//	go run ./cmd/loadtest -url http://localhost:8080 -workload secrets -clients 10
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/kartikbazzad/bunbase/bun-kms/internal/loadtest"
)

func main() {
	url := flag.String("url", "http://localhost:8080", "BunKMS base URL")
	duration := flag.Duration("duration", 10*time.Second, "Load test duration")
	clients := flag.Int("clients", 20, "Number of concurrent clients")
	keyspace := flag.Int("keys", 1000, "Key/secret name space size")
	payloadSize := flag.Int("payload-size", 256, "Payload size in bytes (encrypt/secrets)")
	workload := flag.String("workload", "mixed", "Workload: encrypt, decrypt, mixed, secrets, keys")
	token := flag.String("token", os.Getenv("BUNKMS_TOKEN"), "Optional JWT Bearer token")
	flag.Parse()

	var w loadtest.Workload
	switch *workload {
	case "encrypt":
		w = loadtest.WorkloadEncrypt
	case "decrypt":
		w = loadtest.WorkloadDecrypt
	case "mixed":
		w = loadtest.WorkloadMixed
	case "secrets":
		w = loadtest.WorkloadSecrets
	case "keys":
		w = loadtest.WorkloadKeys
	default:
		fmt.Fprintf(os.Stderr, "workload must be encrypt, decrypt, mixed, secrets, or keys\n")
		os.Exit(1)
	}

	cfg := loadtest.DefaultConfig(*url)
	cfg.Duration = *duration
	cfg.NumClients = *clients
	cfg.KeySpace = *keyspace
	cfg.PayloadSize = *payloadSize
	cfg.Workload = w
	cfg.Token = *token

	fmt.Printf("BunKMS load test: url=%s duration=%v clients=%d keys=%d payload=%d workload=%s\n",
		*url, *duration, *clients, *keyspace, *payloadSize, *workload)

	ctx := context.Background()
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

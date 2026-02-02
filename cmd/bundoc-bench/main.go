package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc/client"
)

type Config struct {
	Addr        string
	Concurrency int
	TotalOps    int
	ReadRatio   float64 // 0.0 to 1.0 (e.g. 0.8 for 80% reads)
}

func main() {
	addr := flag.String("addr", "localhost:4321", "Server address")
	concurrency := flag.Int("c", 10, "Number of concurrent workers")
	ops := flag.Int("n", 10000, "Total number of operations")
	ratio := flag.Float64("ratio", 0.5, "Read ratio (0.0=Write Only, 1.0=Read Only)")

	flag.Parse()

	cfg := Config{
		Addr:        *addr,
		Concurrency: *concurrency,
		TotalOps:    *ops,
		ReadRatio:   *ratio,
	}

	fmt.Printf("🔥 Starting Bundoc Bench\n")
	fmt.Printf("   Server: %s\n   Workers: %d\n   Total Ops: %d\n   Read Ratio: %.2f\n",
		cfg.Addr, cfg.Concurrency, cfg.TotalOps, cfg.ReadRatio)

	runBenchmark(cfg)
}

func runBenchmark(cfg Config) {
	start := time.Now()

	var wg sync.WaitGroup
	opsPerWorker := cfg.TotalOps / cfg.Concurrency

	latencies := make(chan time.Duration, cfg.TotalOps)
	errors := make(chan error, cfg.TotalOps)

	// Single client setup per worker logic
	// Or shared client?
	// `client.Client` is thread-safe (uses mutex).
	// But network contention on single conn might be bottleneck.
	// Real world: multiple clients.
	// Let's create one client connection per worker to simulate real users.

	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Connect
			cli, err := client.Connect(cfg.Addr)
			if err != nil {
				log.Printf("Worker %d failed to connect: %v", id, err)
				return
			}
			defer cli.Close()

			col := cli.Database("bench_db").Collection("bench_col")

			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)))

			for j := 0; j < opsPerWorker; j++ {
				opStart := time.Now()

				isRead := r.Float64() < cfg.ReadRatio

				if isRead {
					// Read Operation
					// Find logic: maybe random ID if we had one?
					// For now, simple query
					_, err := col.FindQuery("bench_proj", map[string]interface{}{
						"key": "value", // Dummy query
					})
					if err != nil {
						errors <- err
					}
				} else {
					// Write Operation
					err := col.Insert("bench_proj", map[string]interface{}{
						"worker": id,
						"iter":   j,
						"data":   "some useful payload",
						"ts":     time.Now().UnixNano(),
					})
					if err != nil {
						errors <- err
					}
				}

				latencies <- time.Since(opStart)
			}
		}(i)
	}

	wg.Wait()
	close(latencies)
	close(errors)

	duration := time.Since(start)

	// Process Results
	var totalLatency time.Duration
	var latList []float64
	var errCount int

	for l := range latencies {
		totalLatency += l
		latList = append(latList, float64(l.Microseconds())/1000.0) // ms
	}

	for err := range errors {
		errCount++
		if errCount <= 5 {
			fmt.Printf("Error Sample: %v\n", err)
		}
	}

	opsCount := len(latList)
	throughput := float64(opsCount) / duration.Seconds()
	avgLatency := float64(totalLatency.Milliseconds()) / float64(opsCount)

	sort.Float64s(latList)
	p50 := 0.0
	p99 := 0.0
	if len(latList) > 0 {
		p50 = latList[int(float64(len(latList))*0.50)]
		p99 = latList[int(float64(len(latList))*0.99)]
	}

	fmt.Println("\n📊 Results:")
	fmt.Printf("   Duration:   %v\n", duration)
	fmt.Printf("   Throughput: %.2f ops/sec\n", throughput)
	fmt.Printf("   Avg Latency: %.2f ms\n", avgLatency)
	fmt.Printf("   P50 Latency: %.2f ms\n", p50)
	fmt.Printf("   P99 Latency: %.2f ms\n", p99)
	fmt.Printf("   Errors:     %d (%.2f%%)\n", errCount, float64(errCount)/float64(cfg.TotalOps)*100)
}

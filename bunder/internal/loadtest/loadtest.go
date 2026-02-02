package loadtest

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/kartikbazzad/bunbase/bunder/pkg/client"
)

// Workload is the type of operations to run: set-only, get-only, or mixed (50/50).
type Workload string

const (
	WorkloadSet   Workload = "set"
	WorkloadGet   Workload = "get"
	WorkloadMixed Workload = "mixed"
)

// Config configures a load test run.
type Config struct {
	Addr       string        // Bunder server address (e.g. "127.0.0.1:6379")
	Duration   time.Duration // How long to run (e.g. 10s)
	NumClients int           // Number of concurrent client connections
	KeySpace   int           // Number of distinct keys (0 = 10000)
	ValueSize  int           // Size of value in bytes (0 = 64)
	Workload   Workload      // set, get, or mixed
}

// DefaultConfig returns a default load test config.
func DefaultConfig(addr string) Config {
	return Config{
		Addr:       addr,
		Duration:   10 * time.Second,
		NumClients: 50,
		KeySpace:   10000,
		ValueSize:  64,
		Workload:   WorkloadMixed,
	}
}

// Run runs the load test: spawns NumClients workers, each running the chosen workload
// for Duration, then aggregates stats and returns a Report.
func Run(ctx context.Context, cfg Config) (*Report, error) {
	if cfg.KeySpace <= 0 {
		cfg.KeySpace = 10000
	}
	if cfg.ValueSize <= 0 {
		cfg.ValueSize = 64
	}
	if cfg.NumClients <= 0 {
		cfg.NumClients = 1
	}

	stats := NewStats()
	value := make([]byte, cfg.ValueSize)
	rand.Read(value)

	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < cfg.NumClients; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runWorker(ctx, cfg, workerID, value, stats, start)
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)
	report := stats.Report(elapsed)
	return &report, nil
}

func runWorker(ctx context.Context, cfg Config, workerID int, value []byte, stats *Stats, start time.Time) {
	opts := client.DefaultOptions(cfg.Addr)
	opts.Timeout = 5 * time.Second
	c, err := client.Connect(ctx, opts)
	if err != nil {
		stats.Record(0, err)
		return
	}
	defer c.Close()

	rng := rand.New(rand.NewSource(int64(workerID)))
	end := start.Add(cfg.Duration)

	for time.Now().Before(end) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		key := fmt.Sprintf("k%d", rng.Intn(cfg.KeySpace))
		opStart := time.Now()
		var err error
		switch cfg.Workload {
		case WorkloadSet:
			err = c.Set(ctx, key, value)
		case WorkloadGet:
			_, err = c.Get(ctx, key)
		case WorkloadMixed:
			if rng.Intn(2) == 0 {
				err = c.Set(ctx, key, value)
			} else {
				_, err = c.Get(ctx, key)
			}
		default:
			err = c.Set(ctx, key, value)
		}
		stats.Record(time.Since(opStart), err)
	}
}

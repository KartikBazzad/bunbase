package loadtest

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Workload is the type of operations: encrypt, decrypt, mixed (encrypt+decrypt), secrets, keys.
type Workload string

const (
	WorkloadEncrypt Workload = "encrypt"
	WorkloadDecrypt Workload = "decrypt"
	WorkloadMixed   Workload = "mixed"
	WorkloadSecrets Workload = "secrets"
	WorkloadKeys    Workload = "keys"
)

// Config configures a load test run.
type Config struct {
	BaseURL     string        // BunKMS base URL (e.g. "http://localhost:8080")
	Token       string        // Optional JWT Bearer token
	Duration    time.Duration // How long to run (e.g. 10s)
	NumClients  int           // Number of concurrent clients
	KeySpace    int           // Distinct key/secret names (0 = 1000)
	PayloadSize int           // Plaintext size in bytes (0 = 256)
	Workload    Workload      // encrypt, decrypt, mixed, secrets, keys
}

// DefaultConfig returns a default load test config.
func DefaultConfig(baseURL string) Config {
	return Config{
		BaseURL:     baseURL,
		Duration:    10 * time.Second,
		NumClients:  20,
		KeySpace:    1000,
		PayloadSize: 256,
		Workload:    WorkloadMixed,
	}
}

// Run runs the load test: ensures a key exists for encrypt/decrypt/mixed, spawns NumClients
// workers for Duration, then returns aggregated Report.
func Run(ctx context.Context, cfg Config) (*Report, error) {
	if cfg.NumClients <= 0 {
		cfg.NumClients = 1
	}
	if cfg.KeySpace <= 0 {
		cfg.KeySpace = 1000
	}
	if cfg.PayloadSize <= 0 {
		cfg.PayloadSize = 256
	}

	client := NewClient(cfg.BaseURL, cfg.Token)
	switch cfg.Workload {
	case WorkloadEncrypt, WorkloadDecrypt, WorkloadMixed:
		if err := client.CreateKey("loadtest-key"); err != nil {
			return nil, fmt.Errorf("create key: %w", err)
		}
	}

	stats := NewStats()
	payload := make([]byte, cfg.PayloadSize)
	rand.Read(payload)

	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < cfg.NumClients; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runWorker(ctx, cfg, workerID, payload, stats, start)
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)
	report := stats.Report(elapsed)
	return &report, nil
}

func runWorker(ctx context.Context, cfg Config, workerID int, payload []byte, stats *Stats, start time.Time) {
	client := NewClient(cfg.BaseURL, cfg.Token)
	rng := rand.New(rand.NewSource(int64(workerID)))
	end := start.Add(cfg.Duration)
	var lastCiphertext string

	for time.Now().Before(end) {
		select {
		case <-ctx.Done():
			return
		default:
		}

		opStart := time.Now()
		var err error

		switch cfg.Workload {
		case WorkloadEncrypt:
			_, err = client.Encrypt("loadtest-key", payload)
		case WorkloadDecrypt:
			if lastCiphertext == "" {
				lastCiphertext, err = client.Encrypt("loadtest-key", payload)
				if err == nil {
					_, err = client.Decrypt("loadtest-key", lastCiphertext)
				}
			} else {
				_, err = client.Decrypt("loadtest-key", lastCiphertext)
			}
		case WorkloadMixed:
			if rng.Intn(2) == 0 {
				lastCiphertext, err = client.Encrypt("loadtest-key", payload)
			} else {
				if lastCiphertext != "" {
					_, err = client.Decrypt("loadtest-key", lastCiphertext)
				} else {
					lastCiphertext, err = client.Encrypt("loadtest-key", payload)
				}
			}
		case WorkloadSecrets:
			key := fmt.Sprintf("loadtest-s-w%d-k%d", workerID, rng.Intn(cfg.KeySpace))
			if rng.Intn(2) == 0 {
				err = client.PutSecret(key, payload)
			} else {
				_, err = client.GetSecret(key)
			}
		case WorkloadKeys:
			key := fmt.Sprintf("loadtest-k-w%d-%d", workerID, rng.Intn(cfg.KeySpace))
			err = client.CreateKey(key)
			if err == nil {
				err = client.GetKey(key)
			}
			stats.Record(time.Since(opStart), err)
			continue
		default:
			lastCiphertext, err = client.Encrypt("loadtest-key", payload)
		}

		stats.Record(time.Since(opStart), err)
	}
}

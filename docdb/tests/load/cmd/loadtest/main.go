package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kartikbazzad/docdb/pkg/client"
	"github.com/kartikbazzad/docdb/tests/load"
)

func main() {
	cfg := load.DefaultConfig()
	cfg.ParseFlags()
	flag.Parse()

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize random number generator
	rng := rand.New(rand.NewSource(cfg.Seed))

	// Create client
	cli := client.New(cfg.SocketPath)
	if err := cli.Connect(); err != nil {
		log.Fatalf("Failed to connect to DocDB: %v", err)
	}
	defer cli.Close()

	// Open or create database
	dbID, err := cli.OpenDB(cfg.DBName)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer cli.CloseDB(dbID)

	log.Printf("Opened database '%s' with ID %d", cfg.DBName, dbID)

	// Initialize metrics collectors
	latencyMetrics := load.NewLatencyMetrics()
	walTracker := load.NewWALTracker(cfg.WALDir, cfg.DBName)
	healingClient := load.NewIPCHealingStatsClient(cfg.SocketPath)
	defer healingClient.Close()
	healingTracker := load.NewHealingTracker(healingClient, dbID)

	// Record initial state
	if err := healingTracker.Start(); err != nil {
		log.Printf("Warning: Failed to get initial healing stats: %v", err)
	}
	if err := walTracker.Sample(); err != nil {
		log.Printf("Warning: Failed to sample initial WAL size: %v", err)
	}

	// Generate document payloads
	payloads := generatePayloads(cfg.DocumentCount, cfg.DocumentSize, rng)

	// Start metrics collection goroutine
	stopMetrics := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(cfg.MetricsInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := walTracker.Sample(); err != nil {
					log.Printf("Warning: Failed to sample WAL size: %v", err)
				}
			case <-stopMetrics:
				return
			}
		}
	}()

	// Determine test duration
	testDuration := cfg.Duration
	if testDuration == 0 && cfg.Operations > 0 {
		// Estimate duration based on operations (will be stopped when operations complete)
		testDuration = 1 * time.Hour
	}

	// Start load test
	startTime := time.Now()
	var totalOps int64
	var stopFlag int32

	// Create worker pool
	var workerWg sync.WaitGroup

	// Start workers
	for i := 0; i < cfg.Workers; i++ {
		workerWg.Add(1)
		workerRNG := rand.New(rand.NewSource(cfg.Seed + int64(i)))
		go func(workerID int) {
			defer workerWg.Done()
			workerCli := client.New(cfg.SocketPath)
			if err := workerCli.Connect(); err != nil {
				log.Printf("Worker %d: Failed to connect: %v", workerID, err)
				return
			}
			defer workerCli.Close()

			for {
				// Check if we should stop
				if atomic.LoadInt32(&stopFlag) == 1 {
					return
				}

				// Check duration limit
				if cfg.Duration > 0 && time.Since(startTime) >= cfg.Duration {
					atomic.StoreInt32(&stopFlag, 1)
					return
				}

				// Check operations limit
				if cfg.Operations > 0 && atomic.LoadInt64(&totalOps) >= int64(cfg.Operations) {
					atomic.StoreInt32(&stopFlag, 1)
					return
				}

				// Generate and execute work item
				item := generateWorkItem(cfg, payloads, workerRNG)
				executeOperation(workerCli, dbID, item, latencyMetrics, workerRNG)
				atomic.AddInt64(&totalOps, 1)
			}
		}(i)
	}

	// Run test
	if cfg.Duration > 0 {
		log.Printf("Running load test for %v with %d workers...", cfg.Duration, cfg.Workers)
		time.Sleep(cfg.Duration)
		atomic.StoreInt32(&stopFlag, 1)
	} else {
		log.Printf("Running load test until %d operations complete with %d workers...", cfg.Operations, cfg.Workers)
		// Wait until operations complete
		for atomic.LoadInt64(&totalOps) < int64(cfg.Operations) {
			time.Sleep(100 * time.Millisecond)
		}
		atomic.StoreInt32(&stopFlag, 1)
	}

	// Stop metrics collection
	close(stopMetrics)
	wg.Wait()

	// Wait for workers to finish
	workerWg.Wait()

	endTime := time.Now()
	actualDuration := endTime.Sub(startTime)
	totalOpsFinal := atomic.LoadInt64(&totalOps)

	log.Printf("Load test completed: %d operations in %v", totalOpsFinal, actualDuration)

	// Record final state
	if err := healingTracker.Stop(); err != nil {
		log.Printf("Warning: Failed to get final healing stats: %v", err)
	}
	if err := walTracker.Sample(); err != nil {
		log.Printf("Warning: Failed to sample final WAL size: %v", err)
	}

	// Collect results
	results := collectResults(cfg, latencyMetrics, walTracker, healingTracker, actualDuration, totalOpsFinal)

	// Write results
	if err := writeResults(results, cfg); err != nil {
		log.Fatalf("Failed to write results: %v", err)
	}

	log.Printf("Results written to %s", cfg.OutputPath)

	// Write CSV if requested
	if cfg.CSVOutput {
		outputDir := filepath.Dir(cfg.OutputPath)
		if outputDir == "" {
			outputDir = "."
		}
		if err := load.WriteCSV(results, outputDir); err != nil {
			log.Printf("Warning: Failed to write CSV files: %v", err)
		} else {
			log.Printf("CSV files written to %s", outputDir)
		}
	}
}

type workItem struct {
	opType     load.OperationType
	docID      uint64
	payload    []byte
	collection string
}

func generateWorkItem(cfg *load.LoadTestConfig, payloads [][]byte, rng *rand.Rand) workItem {
	docID := uint64(rng.Intn(cfg.DocumentCount)) + 1
	payload := payloads[rng.Intn(len(payloads))]

	// Determine operation type based on percentages
	roll := rng.Intn(100)
	var opType load.OperationType
	if roll < cfg.ReadPercent {
		opType = load.OpRead
	} else if roll < cfg.ReadPercent+cfg.WritePercent {
		opType = load.OpCreate
	} else if roll < cfg.ReadPercent+cfg.WritePercent+cfg.UpdatePercent {
		opType = load.OpUpdate
	} else {
		opType = load.OpDelete
	}

	return workItem{
		opType:     opType,
		docID:      docID,
		payload:    payload,
		collection: "_default",
	}
}

func executeOperation(cli *client.Client, dbID uint64, item workItem, metrics *load.LatencyMetrics, rng *rand.Rand) {
	start := time.Now()
	var err error

	switch item.opType {
	case load.OpCreate:
		err = cli.Create(dbID, item.collection, item.docID, item.payload)
	case load.OpRead:
		_, err = cli.Read(dbID, item.collection, item.docID)
	case load.OpUpdate:
		err = cli.Update(dbID, item.collection, item.docID, item.payload)
	case load.OpDelete:
		err = cli.Delete(dbID, item.collection, item.docID)
	}

	latency := time.Since(start)

	// Record latency (even for errors, to measure error handling overhead)
	metrics.Record(item.opType, latency)

	if err != nil && item.opType != load.OpRead {
		// For reads, NotFound is expected sometimes
		log.Printf("Operation failed: %s docID=%d: %v", item.opType, item.docID, err)
	}
}

func generatePayloads(count, size int, rng *rand.Rand) [][]byte {
	payloads := make([][]byte, count)
	for i := 0; i < count; i++ {
		// Generate valid JSON payload
		// Format: {"id": <i>, "data": "<base64_encoded_random_data>"}
		// Reserve space for JSON structure: {"id":0,"data":""}
		baseStructureSize := 20                    // Approximate size of JSON structure
		dataSize := size - baseStructureSize - 100 // Reserve extra for base64 overhead and padding
		if dataSize < 10 {
			dataSize = 10
		}

		// Generate random binary data
		randomData := make([]byte, dataSize)
		rng.Read(randomData)

		// Encode as base64 to ensure valid JSON string
		encodedData := base64.StdEncoding.EncodeToString(randomData)

		// Create JSON payload
		payload := fmt.Sprintf(`{"id":%d,"data":"%s","timestamp":%d}`,
			i, encodedData, time.Now().UnixNano())

		// Adjust size to match exactly
		if len(payload) > size {
			// Truncate the base64 data to fit
			availableSize := size - baseStructureSize - len(fmt.Sprintf(`%d`, i)) - len(fmt.Sprintf(`%d`, time.Now().UnixNano()))
			if availableSize > 0 {
				truncatedData := encodedData
				if len(truncatedData) > availableSize {
					truncatedData = truncatedData[:availableSize]
				}
				payload = fmt.Sprintf(`{"id":%d,"data":"%s","timestamp":%d}`,
					i, truncatedData, time.Now().UnixNano())
				if len(payload) > size {
					payload = payload[:size-1] + "}"
				}
			} else {
				payload = fmt.Sprintf(`{"id":%d}`, i)
				if len(payload) > size {
					payload = payload[:size]
				}
			}
		} else if len(payload) < size {
			// Pad with additional fields to reach target size
			padding := size - len(payload) - 1
			if padding > 0 {
				// Add padding as base64-encoded random data
				paddingData := make([]byte, padding/2) // base64 expands ~4/3, so use padding/2
				rng.Read(paddingData)
				paddingEncoded := base64.StdEncoding.EncodeToString(paddingData)
				if len(paddingEncoded) > padding {
					paddingEncoded = paddingEncoded[:padding]
				}
				payload = fmt.Sprintf(`{"id":%d,"data":"%s","timestamp":%d,"padding":"%s"}`,
					i, encodedData, time.Now().UnixNano(), paddingEncoded)
				if len(payload) > size {
					payload = payload[:size-1] + "}"
				}
			}
		}

		// Validate JSON before adding to payloads
		var test interface{}
		if err := json.Unmarshal([]byte(payload), &test); err != nil {
			// Fallback to simple valid JSON if generation failed
			payload = fmt.Sprintf(`{"id":%d,"size":%d}`, i, size)
			if len(payload) > size {
				payload = payload[:size]
			}
		}

		payloads[i] = []byte(payload)
	}
	return payloads
}

func collectResults(cfg *load.LoadTestConfig, metrics *load.LatencyMetrics, walTracker *load.WALTracker, healingTracker *load.HealingTracker, duration time.Duration, totalOps int64) *load.TestResults {
	return &load.TestResults{
		TestConfig:      cfg,
		DurationSeconds: duration.Seconds(),
		TotalOperations: totalOps,
		Latency:         metrics.GetStats(),
		WALGrowth:       walTracker.GetSummary(),
		WALSamples:      walTracker.GetSamples(),
		Healing:         healingTracker.GetSummary(duration),
		HealingEvents:   healingTracker.GetEvents(),
		LatencySamples:  metrics.GetAllSamples(),
	}
}

func writeResults(results *load.TestResults, cfg *load.LoadTestConfig) error {
	// Write JSON
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(cfg.OutputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}

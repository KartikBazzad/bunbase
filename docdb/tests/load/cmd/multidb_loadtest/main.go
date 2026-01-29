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
	"strings"
	"sync"
	"time"

	"github.com/kartikbazzad/docdb/pkg/client"
	"github.com/kartikbazzad/docdb/tests/load"
)

func main() {
	configFile := flag.String("config", "", "Path to configuration file (JSON)")
	databasesFlag := flag.String("databases", "", "Comma-separated database names (alternative to config file)")
	workersPerDB := flag.Int("workers-per-db", 10, "Workers per database (when using -databases)")
	duration := flag.Duration("duration", 5*time.Minute, "Test duration")
	socketPath := flag.String("socket", "/tmp/docdb.sock", "IPC socket path")
	walDir := flag.String("wal-dir", "./data/wal", "WAL directory path")
	outputPath := flag.String("output", "multidb_results.json", "Output JSON file path")
	csvOutput := flag.Bool("csv", false, "Generate CSV output files")
	seed := flag.Int64("seed", time.Now().UnixNano(), "Random seed")
	flag.Parse()

	var cfg *load.MultiDBLoadTestConfig

	// Load from config file or create from flags
	if *configFile != "" {
		var err error
		cfg, err = load.LoadConfigFromFile(*configFile)
		if err != nil {
			log.Fatalf("Failed to load config file: %v", err)
		}
	} else if *databasesFlag != "" {
		// Create config from command-line flags
		cfg = createConfigFromFlags(*databasesFlag, *workersPerDB, *duration, *socketPath, *walDir, *outputPath, *csvOutput, *seed)
	} else {
		log.Fatalf("Must specify either -config or -databases")
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	log.Printf("Starting multi-database load test with %d databases", len(cfg.Databases))

	// Initialize base client
	baseClient := client.New(cfg.SocketPath)
	if err := baseClient.Connect(); err != nil {
		log.Fatalf("Failed to connect to DocDB: %v", err)
	}
	defer baseClient.Close()

	// Create database manager
	dbManager := load.NewDatabaseManager(baseClient, cfg.SocketPath)
	defer dbManager.CloseAll()

	// Create healing stats client
	healingClient := load.NewIPCHealingStatsClient(cfg.SocketPath)
	defer healingClient.Close()

	// Initialize databases
	rng := rand.New(rand.NewSource(cfg.Seed))
	for _, dbConfig := range cfg.Databases {
		ctx, err := dbManager.AddDatabase(dbConfig, healingClient, cfg.WALDir)
		if err != nil {
			log.Fatalf("Failed to add database %s: %v", dbConfig.Name, err)
		}

		// Generate payloads for this database
		ctx.Payloads = generatePayloads(dbConfig.DocumentCount, dbConfig.DocumentSize, rng)

		log.Printf("Initialized database '%s' with ID %d", dbConfig.Name, ctx.DBID)
	}

	// Create workload profile manager
	var profileMgr *load.WorkloadProfileManager
	if cfg.WorkloadProfile != nil {
		profileMgr = load.NewWorkloadProfileManager(cfg.WorkloadProfile)
	}

	// Create worker pool manager
	workerPoolMgr := load.NewWorkerPoolManager(cfg, dbManager, profileMgr)

	// Create metrics collector
	metrics := load.NewMultiDBMetrics()

	// Record initial state
	if err := dbManager.StartHealingTracking(); err != nil {
		log.Printf("Warning: Failed to start healing tracking: %v", err)
	}
	if err := dbManager.SampleWAL(); err != nil {
		log.Printf("Warning: Failed to sample initial WAL size: %v", err)
	}

	// Start metrics collection goroutine
	stopMetrics := make(chan struct{})
	var metricsWg sync.WaitGroup
	metricsWg.Add(1)
	go func() {
		defer metricsWg.Done()
		ticker := time.NewTicker(cfg.MetricsInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := dbManager.SampleWAL(); err != nil {
					log.Printf("Warning: Failed to sample WAL size: %v", err)
				}
				// Check for phase transitions and scale workers
				if profileMgr != nil && profileMgr.IsPhaseTransition() {
					if err := workerPoolMgr.ScaleWorkers(); err != nil {
						log.Printf("Warning: Failed to scale workers: %v", err)
					}
					phaseInfo := profileMgr.GetPhaseInfo()
					if phaseInfo != nil {
						log.Printf("Phase transition: %s (workers: %d)", phaseInfo.Name, phaseInfo.Workers)
					}
				}
			case <-stopMetrics:
				return
			}
		}
	}()

	// Determine test duration
	testDuration := cfg.Duration
	if testDuration == 0 {
		testDuration = 5 * time.Minute
	}

	// Start test
	startTime := time.Now()
	if profileMgr != nil {
		profileMgr.Start(startTime)
	}

	log.Printf("Running load test for %v...", testDuration)

	// Start workers
	if err := workerPoolMgr.Start(); err != nil {
		log.Fatalf("Failed to start workers: %v", err)
	}

	// Run test
	time.Sleep(testDuration)
	workerPoolMgr.Stop()

	// Stop metrics collection
	close(stopMetrics)
	metricsWg.Wait()

	endTime := time.Now()
	actualDuration := endTime.Sub(startTime)
	totalOps := workerPoolMgr.GetTotalOperations()

	log.Printf("Load test completed: %d operations in %v", totalOps, actualDuration)

	// Record final state
	if err := dbManager.StopHealingTracking(); err != nil {
		log.Printf("Warning: Failed to stop healing tracking: %v", err)
	}
	if err := dbManager.SampleWAL(); err != nil {
		log.Printf("Warning: Failed to sample final WAL size: %v", err)
	}

	// Collect metrics
	metrics.CollectMetrics(dbManager, profileMgr, actualDuration)

	// Collect results
	results := collectMultiDBResults(cfg, metrics, actualDuration, totalOps)

	// Write results
	if err := writeMultiDBResults(results, cfg); err != nil {
		log.Fatalf("Failed to write results: %v", err)
	}

	log.Printf("Results written to %s", cfg.OutputPath)

	// Write CSV if requested
	if cfg.CSVOutput {
		outputDir := filepath.Dir(cfg.OutputPath)
		if outputDir == "" {
			outputDir = "."
		}
		csvResults := &load.MultiDBTestResults{
			TestConfig:      results.TestConfig,
			DurationSeconds: results.DurationSeconds,
			TotalOperations: results.TotalOperations,
			Databases:       results.Databases,
			Global:          results.Global,
		}
		if err := load.WriteMultiDBCSV(csvResults, outputDir); err != nil {
			log.Printf("Warning: Failed to write CSV files: %v", err)
		} else {
			log.Printf("CSV files written to %s", outputDir)
		}
	}
}

func createConfigFromFlags(databasesStr string, workersPerDB int, duration time.Duration, socketPath, walDir, outputPath string, csvOutput bool, seed int64) *load.MultiDBLoadTestConfig {
	cfg := load.NewMultiDBConfig()
	cfg.Duration = duration
	cfg.SocketPath = socketPath
	cfg.WALDir = walDir
	cfg.OutputPath = outputPath
	cfg.CSVOutput = csvOutput
	cfg.Seed = seed
	cfg.MetricsInterval = 1 * time.Second

	// Parse database names
	dbNames := strings.Split(databasesStr, ",")
	for _, dbName := range dbNames {
		dbName = strings.TrimSpace(dbName)
		if dbName == "" {
			continue
		}
		cfg.AddDatabase(load.DatabaseConfig{
			Name:          dbName,
			Workers:       workersPerDB,
			DocumentSize:  1024,
			DocumentCount: 10000,
			CRUDPercent:   nil, // Use default or profile
			WALDir:        "",
		})
	}

	return cfg
}

func generatePayloads(count, size int, rng *rand.Rand) [][]byte {
	payloads := make([][]byte, count)
	for i := 0; i < count; i++ {
		baseStructureSize := 20
		dataSize := size - baseStructureSize - 100
		if dataSize < 10 {
			dataSize = 10
		}

		randomData := make([]byte, dataSize)
		rng.Read(randomData)

		encodedData := base64.StdEncoding.EncodeToString(randomData)

		payload := fmt.Sprintf(`{"id":%d,"data":"%s","timestamp":%d}`,
			i, encodedData, time.Now().UnixNano())

		if len(payload) > size {
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
			padding := size - len(payload) - 1
			if padding > 0 {
				paddingData := make([]byte, padding/2)
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

		var test interface{}
		if err := json.Unmarshal([]byte(payload), &test); err != nil {
			payload = fmt.Sprintf(`{"id":%d,"size":%d}`, i, size)
			if len(payload) > size {
				payload = payload[:size]
			}
		}

		payloads[i] = []byte(payload)
	}
	return payloads
}

func collectMultiDBResults(cfg *load.MultiDBLoadTestConfig, metrics *load.MultiDBMetrics, duration time.Duration, totalOps int64) *load.MultiDBTestResults {
	return &load.MultiDBTestResults{
		TestConfig:      cfg,
		DurationSeconds: duration.Seconds(),
		TotalOperations: totalOps,
		Databases:       metrics.Databases,
		Global:          metrics.Global,
	}
}

func writeMultiDBResults(results *load.MultiDBTestResults, cfg *load.MultiDBLoadTestConfig) error {
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(cfg.OutputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}

package load

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// MatrixRunner runs the full test matrix.
type MatrixRunner struct {
	config      *TestMatrixConfig
	testConfigs []TestConfiguration
	results     []TestResult
}

// TestResult holds the result of a single test configuration.
type TestResult struct {
	Config     TestConfiguration
	OutputPath string
	Success    bool
	Error      error
	Duration   time.Duration
	StartTime  time.Time
	EndTime    time.Time
}

// NewMatrixRunner creates a new matrix runner.
func NewMatrixRunner(config *TestMatrixConfig) *MatrixRunner {
	return &MatrixRunner{
		config:      config,
		testConfigs: config.GenerateTestConfigurations(),
		results:     make([]TestResult, 0),
	}
}

// Run executes all test configurations in the matrix.
func (mr *MatrixRunner) Run() error {
	// Create output directory layout
	if _, _, _, _, err := EnsureOutputDirs(mr.config.OutputDir); err != nil {
		return fmt.Errorf("failed to create output directories: %w", err)
	}

	// Open SQLite DB for this run and insert run row
	dbPath := MatrixDBPath(mr.config.OutputDir)
	db, err := OpenMatrixDB(dbPath)
	if err != nil {
		log.Printf("Warning: Failed to open matrix DB at %s: %v (results will not be stored in SQLite)", dbPath, err)
		db = nil
	}
	var runID int64
	if db != nil {
		defer db.Close()
		runID, err = InsertRun(db, mr.config.OutputDir)
		if err != nil {
			log.Printf("Warning: Failed to insert run: %v", err)
			runID = 0
		}
	}

	totalTests := len(mr.testConfigs)
	log.Printf("Starting test matrix: %d configurations", totalTests)

	matrixDBPath := ""
	if db != nil && runID != 0 {
		matrixDBPath = dbPath
	}
	for i, testConfig := range mr.testConfigs {
		log.Printf("[%d/%d] Running test: %s", i+1, totalTests, testConfig.Name)

		// Phase E.10: Verbose mode - log test configuration details
		if mr.config.Verbose {
			log.Printf("[VERBOSE] Test config: %d databases, %d connections/DB, %d workers/DB, duration=%v",
				testConfig.Databases, testConfig.ConnectionsPerDB, testConfig.WorkersPerDB, mr.config.Duration)
			log.Printf("[VERBOSE] Workload: Read=%d%% Write=%d%% Update=%d%% Delete=%d%%",
				mr.config.ReadPercent, mr.config.WritePercent, mr.config.UpdatePercent, mr.config.DeletePercent)
			log.Printf("[VERBOSE] Document: size=%d bytes, count=%d per DB",
				mr.config.DocumentSize, mr.config.DocumentCount)
		}

		result := mr.runTestConfiguration(testConfig, matrixDBPath, runID)
		mr.results = append(mr.results, result)

		if result.Success {
			log.Printf("[%d/%d] ✓ Completed: %s (duration: %v)", i+1, totalTests, testConfig.Name, result.Duration)
		} else {
			log.Printf("[%d/%d] ✗ Failed: %s - %v", i+1, totalTests, testConfig.Name, result.Error)
		}

		// Optional: Restart DocDB between tests if configured
		if mr.config.RestartDB && i < totalTests-1 {
			log.Printf("Restarting DocDB server...")
			// Note: This would require external script or process management
			// For now, we'll log a warning
			log.Printf("Warning: RestartDB=true but automatic restart not implemented. Please restart manually if needed.")
		}
	}

	// Update run row with final counts
	if db != nil && runID != 0 {
		successCount, failCount := 0, 0
		for _, r := range mr.results {
			if r.Success {
				successCount++
			} else {
				failCount++
			}
		}
		if err := UpdateRun(db, runID, len(mr.results), successCount, failCount); err != nil {
			log.Printf("Warning: Failed to update run in SQLite: %v", err)
		}
	}

	// Generate summary
	mr.generateSummary()

	return nil
}

// runTestConfiguration runs a single test configuration.
// When matrixDBPath != "" and matrixRunID != 0, the subprocess inserts into the matrix DB only (no JSON/CSV).
func (mr *MatrixRunner) runTestConfiguration(config TestConfiguration, matrixDBPath string, matrixRunID int64) TestResult {
	result := TestResult{
		Config:    config,
		StartTime: time.Now(),
	}

	// Generate database names
	dbNames := generateDatabaseNames(config.Databases)

	// Output: when using matrix DB, pass config name only (subprocess inserts, no file). Otherwise JSON filename.
	jsonDir := filepath.Join(mr.config.OutputDir, "json")
	if err := os.MkdirAll(jsonDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create json directory: %w", err)
		result.Success = false
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}
	useMatrixOnly := matrixDBPath != "" && matrixRunID != 0
	var outputArg string
	if useMatrixOnly {
		outputArg = config.Name
		result.OutputPath = ""
	} else {
		outputArg = mr.config.GetOutputPath(config)
		result.OutputPath = filepath.Join(jsonDir, outputArg)
	}

	// Build command - use relative path from project root
	cmd := exec.Command(
		"go", "run", "./docdb/tests/load/cmd/multidb_loadtest/main.go",
		"-databases", strings.Join(dbNames, ","),
		"-workers-per-db", fmt.Sprintf("%d", config.WorkersPerDB),
		"-connections-per-db", fmt.Sprintf("%d", config.ConnectionsPerDB),
		"-duration", mr.config.Duration.String(),
		"-socket", mr.config.SocketPath,
		"-wal-dir", mr.config.WALDir,
		"-output-dir", mr.config.OutputDir,
		"-output", outputArg,
		"-doc-size", fmt.Sprintf("%d", mr.config.DocumentSize),
		"-doc-count", fmt.Sprintf("%d", mr.config.DocumentCount),
		"-read-percent", fmt.Sprintf("%d", mr.config.ReadPercent),
		"-write-percent", fmt.Sprintf("%d", mr.config.WritePercent),
		"-update-percent", fmt.Sprintf("%d", mr.config.UpdatePercent),
		"-delete-percent", fmt.Sprintf("%d", mr.config.DeletePercent),
	)
	if useMatrixOnly {
		cmd.Args = append(cmd.Args, "-matrix-db", matrixDBPath, "-matrix-run-id", fmt.Sprintf("%d", matrixRunID))
	}
	if mr.config.CSVOutput && !useMatrixOnly {
		cmd.Args = append(cmd.Args, "-csv")
	}

	if mr.config.Seed != 0 {
		cmd.Args = append(cmd.Args, "-seed", fmt.Sprintf("%d", mr.config.Seed))
	}

	// Set working directory to project root
	cmd.Dir = findProjectRoot()

	// Run command
	output, err := cmd.CombinedOutput()
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("command failed: %w\nOutput: %s", err, string(output))
		return result
	}

	result.Success = true
	return result
}

// generateDatabaseNames creates database names for a test configuration.
func generateDatabaseNames(count int) []string {
	names := make([]string, count)
	for i := 0; i < count; i++ {
		names[i] = fmt.Sprintf("db%d", i+1)
	}
	return names
}

// generateSummary generates a summary report of all test results.
func (mr *MatrixRunner) generateSummary() {
	summaryPath := filepath.Join(mr.config.OutputDir, "summary.txt")
	file, err := os.Create(summaryPath)
	if err != nil {
		log.Printf("Warning: Failed to create summary file: %v", err)
		return
	}
	defer file.Close()

	fmt.Fprintf(file, "Test Matrix Summary\n")
	fmt.Fprintf(file, "==================\n\n")
	fmt.Fprintf(file, "Total Tests: %d\n", len(mr.results))

	successCount := 0
	failCount := 0
	for _, result := range mr.results {
		if result.Success {
			successCount++
		} else {
			failCount++
		}
	}

	fmt.Fprintf(file, "Successful: %d\n", successCount)
	fmt.Fprintf(file, "Failed: %d\n\n", failCount)

	fmt.Fprintf(file, "Test Results:\n")
	fmt.Fprintf(file, "-------------\n")
	for _, result := range mr.results {
		status := "✓"
		if !result.Success {
			status = "✗"
		}
		fmt.Fprintf(file, "%s %s (duration: %v)", status, result.Config.Name, result.Duration)
		if result.Error != nil {
			fmt.Fprintf(file, " - %v", result.Error)
		}
		fmt.Fprintf(file, "\n")
	}

	log.Printf("Summary written to %s", summaryPath)
}

// findProjectRoot finds the project root directory.
func findProjectRoot() string {
	// Start from current directory and walk up
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}

	for {
		// Check if this directory contains go.mod
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}

	return "."
}

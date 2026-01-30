package load

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// MatrixAnalysis analyzes results from a test matrix run.
type MatrixAnalysis struct {
	ResultsDir string
	Results    []TestResultData
}

// TestResultData holds parsed test result data.
type TestResultData struct {
	Config     TestConfiguration
	ResultFile string
	Throughput float64 // ops/sec
	P95Latency float64 // ms
	P99Latency float64 // ms
	TotalOps   int64
	Duration   float64
	Success    bool
}

// AnalyzeMatrix analyzes all test results in a directory.
// If matrix_results.db exists in resultsDir, it loads results from the latest run.
// Otherwise it scans resultsDir/json/*.json (backward compatibility).
func AnalyzeMatrix(resultsDir string) (*MatrixAnalysis, error) {
	analysis := &MatrixAnalysis{
		ResultsDir: resultsDir,
		Results:    make([]TestResultData, 0),
	}

	dbPath := MatrixDBPath(resultsDir)
	if _, err := os.Stat(dbPath); err == nil {
		// SQLite DB exists: load from latest run
		db, err := OpenMatrixDB(dbPath)
		if err != nil {
			return nil, fmt.Errorf("open matrix db: %w", err)
		}
		defer db.Close()
		runID, err := QueryLatestRunID(db)
		if err != nil {
			return nil, fmt.Errorf("query latest run: %w", err)
		}
		if runID == 0 {
			return analysis, nil
		}
		results, err := QueryResultsByRunID(db, runID)
		if err != nil {
			return nil, fmt.Errorf("query results for run %d: %w", runID, err)
		}
		analysis.Results = results
		return analysis, nil
	}

	// No DB: fall back to scanning json/
	jsonDir := filepath.Join(resultsDir, "json")
	files, err := os.ReadDir(jsonDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read results directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}
		if file.Name() == "summary.txt" {
			continue
		}
		resultPath := filepath.Join(jsonDir, file.Name())
		resultData, err := parseResultFile(resultPath)
		if err != nil {
			fmt.Printf("Warning: Failed to parse %s: %v\n", file.Name(), err)
			continue
		}
		analysis.Results = append(analysis.Results, *resultData)
	}

	sort.Slice(analysis.Results, func(i, j int) bool {
		return analysis.Results[i].Config.Name < analysis.Results[j].Config.Name
	})
	return analysis, nil
}

// parseResultFile parses a single result JSON file.
func parseResultFile(path string) (*TestResultData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var result MultiDBTestResults
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	// Extract test configuration from filename
	config := extractConfigFromFilename(filepath.Base(path))

	// Calculate metrics
	duration := result.DurationSeconds
	if duration == 0 {
		duration = 300 // Default 5 minutes
	}

	throughput := float64(result.TotalOperations) / duration
	p95Latency := 0.0
	p99Latency := 0.0

	if result.Global != nil && result.Global.Latency != nil {
		// Calculate average P95/P99 across all operation types
		count := 0
		if createLatency, ok := result.Global.Latency[OpCreate]; ok {
			p95Latency += createLatency.P95
			p99Latency += createLatency.P99
			count++
		}
		if readLatency, ok := result.Global.Latency[OpRead]; ok {
			p95Latency += readLatency.P95
			p99Latency += readLatency.P99
			count++
		}
		if updateLatency, ok := result.Global.Latency[OpUpdate]; ok {
			p95Latency += updateLatency.P95
			p99Latency += updateLatency.P99
			count++
		}
		if deleteLatency, ok := result.Global.Latency[OpDelete]; ok {
			p95Latency += deleteLatency.P95
			p99Latency += deleteLatency.P99
			count++
		}
		if count > 0 {
			p95Latency /= float64(count)
			p99Latency /= float64(count)
		}
	}

	return &TestResultData{
		Config:     config,
		ResultFile: path,
		Throughput: throughput,
		P95Latency: p95Latency,
		P99Latency: p99Latency,
		TotalOps:   result.TotalOperations,
		Duration:   duration,
		Success:    true,
	}, nil
}

// BuildTestResultDataFromMultiDB builds TestResultData from in-memory MultiDBTestResults.
// Used when inserting into the matrix SQLite DB without writing a JSON file.
// resultFile is stored in the DB (use "" when no file was written).
func BuildTestResultDataFromMultiDB(results *MultiDBTestResults, configName string, databases, connectionsPerDB, workersPerDB int, resultFile string) *TestResultData {
	duration := results.DurationSeconds
	if duration == 0 {
		duration = 300
	}
	throughput := float64(results.TotalOperations) / duration
	p95Latency := 0.0
	p99Latency := 0.0
	if results.Global != nil && results.Global.Latency != nil {
		count := 0
		if l, ok := results.Global.Latency[OpCreate]; ok {
			p95Latency += l.P95
			p99Latency += l.P99
			count++
		}
		if l, ok := results.Global.Latency[OpRead]; ok {
			p95Latency += l.P95
			p99Latency += l.P99
			count++
		}
		if l, ok := results.Global.Latency[OpUpdate]; ok {
			p95Latency += l.P95
			p99Latency += l.P99
			count++
		}
		if l, ok := results.Global.Latency[OpDelete]; ok {
			p95Latency += l.P95
			p99Latency += l.P99
			count++
		}
		if count > 0 {
			p95Latency /= float64(count)
			p99Latency /= float64(count)
		}
	}
	return &TestResultData{
		Config: TestConfiguration{
			Name:             configName,
			Databases:        databases,
			ConnectionsPerDB: connectionsPerDB,
			WorkersPerDB:     workersPerDB,
		},
		ResultFile: resultFile,
		Throughput: throughput,
		P95Latency: p95Latency,
		P99Latency: p99Latency,
		TotalOps:   results.TotalOperations,
		Duration:   duration,
		Success:    true,
	}
}

// extractConfigFromFilename extracts test configuration from filename.
func extractConfigFromFilename(filename string) TestConfiguration {
	// Filename format: "1db_1conn_1w.json"
	// Remove extension
	name := filename[:len(filename)-5] // Remove ".json"

	var dbs, conns, workers int
	fmt.Sscanf(name, "%ddb_%dconn_%dw", &dbs, &conns, &workers)

	return TestConfiguration{
		Databases:        dbs,
		ConnectionsPerDB: conns,
		WorkersPerDB:     workers,
		Name:             name,
	}
}

// GenerateReport generates an analysis report.
func (ma *MatrixAnalysis) GenerateReport(outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, "# DocDB Scaling Matrix Analysis\n\n")
	fmt.Fprintf(file, "## Summary\n\n")
	fmt.Fprintf(file, "Total Tests: %d\n\n", len(ma.Results))

	// Find baseline (1db_1conn_1w)
	var baseline *TestResultData
	for i := range ma.Results {
		if ma.Results[i].Config.Name == "1db_1conn_1w" {
			baseline = &ma.Results[i]
			break
		}
	}

	if baseline != nil {
		fmt.Fprintf(file, "Baseline (1db_1conn_1w):\n")
		fmt.Fprintf(file, "- Throughput: %.2f ops/sec\n", baseline.Throughput)
		fmt.Fprintf(file, "- P95 Latency: %.2f ms\n", baseline.P95Latency)
		fmt.Fprintf(file, "- P99 Latency: %.2f ms\n\n", baseline.P99Latency)
	}

	// Group by category
	fmt.Fprintf(file, "## Results by Category\n\n")

	// Connection scaling (1 DB, 1 worker, varying connections)
	fmt.Fprintf(file, "### Connection Scaling (1 DB, 1 Worker)\n\n")
	fmt.Fprintf(file, "| Connections | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |\n")
	fmt.Fprintf(file, "|-------------|----------------------|------------------|------------------|\n")
	for _, result := range ma.Results {
		if result.Config.Databases == 1 && result.Config.WorkersPerDB == 1 {
			fmt.Fprintf(file, "| %d | %.2f | %.2f | %.2f |\n",
				result.Config.ConnectionsPerDB, result.Throughput, result.P95Latency, result.P99Latency)
		}
	}
	fmt.Fprintf(file, "\n")

	// Worker scaling (1 DB, 1 connection, varying workers)
	fmt.Fprintf(file, "### Worker Scaling (1 DB, 1 Connection)\n\n")
	fmt.Fprintf(file, "| Workers | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |\n")
	fmt.Fprintf(file, "|---------|----------------------|------------------|------------------|\n")
	for _, result := range ma.Results {
		if result.Config.Databases == 1 && result.Config.ConnectionsPerDB == 1 {
			fmt.Fprintf(file, "| %d | %.2f | %.2f | %.2f |\n",
				result.Config.WorkersPerDB, result.Throughput, result.P95Latency, result.P99Latency)
		}
	}
	fmt.Fprintf(file, "\n")

	// Database scaling (1 connection, 1 worker, varying DBs)
	fmt.Fprintf(file, "### Database Scaling (1 Connection, 1 Worker)\n\n")
	fmt.Fprintf(file, "| Databases | Throughput (ops/sec) | P95 Latency (ms) | P99 Latency (ms) |\n")
	fmt.Fprintf(file, "|-----------|----------------------|------------------|------------------|\n")
	for _, result := range ma.Results {
		if result.Config.ConnectionsPerDB == 1 && result.Config.WorkersPerDB == 1 {
			fmt.Fprintf(file, "| %d | %.2f | %.2f | %.2f |\n",
				result.Config.Databases, result.Throughput, result.P95Latency, result.P99Latency)
		}
	}
	fmt.Fprintf(file, "\n")

	// All results table
	fmt.Fprintf(file, "## All Results\n\n")
	fmt.Fprintf(file, "| Test | DBs | Conn/DB | W/DB | Throughput | P95 | P99 |\n")
	fmt.Fprintf(file, "|------|-----|---------|------|------------|-----|-----|\n")
	for _, result := range ma.Results {
		fmt.Fprintf(file, "| %s | %d | %d | %d | %.2f | %.2f | %.2f |\n",
			result.Config.Name,
			result.Config.Databases,
			result.Config.ConnectionsPerDB,
			result.Config.WorkersPerDB,
			result.Throughput,
			result.P95Latency,
			result.P99Latency)
	}

	return nil
}

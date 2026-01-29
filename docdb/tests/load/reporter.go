package load

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// WriteCSV writes CSV files for the test results.
func WriteCSV(results *TestResults, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write latency samples CSV
	if err := writeLatencyCSV(results, outputDir); err != nil {
		return fmt.Errorf("failed to write latency CSV: %w", err)
	}

	// Write WAL growth CSV
	if err := writeWALGrowthCSV(results, outputDir); err != nil {
		return fmt.Errorf("failed to write WAL growth CSV: %w", err)
	}

	// Write healing events CSV
	if err := writeHealingEventsCSV(results, outputDir); err != nil {
		return fmt.Errorf("failed to write healing events CSV: %w", err)
	}

	return nil
}

// TestResults contains all test results data.
type TestResults struct {
	TestConfig      *LoadTestConfig
	DurationSeconds float64
	TotalOperations int64
	Latency         map[OperationType]PercentileStats
	WALGrowth       WALGrowthSummary
	WALSamples      []WALSample
	Healing         HealingSummary
	HealingEvents   []HealingEvent
	LatencySamples  map[OperationType][]time.Duration
}

func writeLatencyCSV(results *TestResults, outputDir string) error {
	file, err := os.Create(filepath.Join(outputDir, "latency_samples.csv"))
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"operation", "latency_ms", "timestamp"}); err != nil {
		return err
	}

	// Write samples
	for opType, samples := range results.LatencySamples {
		for _, sample := range samples {
			ms := float64(sample.Nanoseconds()) / 1e6
			record := []string{
				string(opType),
				strconv.FormatFloat(ms, 'f', 3, 64),
				time.Now().Format(time.RFC3339Nano),
			}
			if err := writer.Write(record); err != nil {
				return err
			}
		}
	}

	return nil
}

func writeWALGrowthCSV(results *TestResults, outputDir string) error {
	file, err := os.Create(filepath.Join(outputDir, "wal_growth.csv"))
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"timestamp", "size_bytes", "size_mb"}); err != nil {
		return err
	}

	// Write samples
	for _, sample := range results.WALSamples {
		sizeMB := float64(sample.SizeBytes) / (1024 * 1024)
		record := []string{
			sample.Timestamp.Format(time.RFC3339Nano),
			strconv.FormatUint(sample.SizeBytes, 10),
			strconv.FormatFloat(sizeMB, 'f', 2, 64),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

func writeHealingEventsCSV(results *TestResults, outputDir string) error {
	file, err := os.Create(filepath.Join(outputDir, "healing_events.csv"))
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"timestamp", "duration_ms", "documents_healed", "type"}); err != nil {
		return err
	}

	// Write events
	for _, event := range results.HealingEvents {
		durationMS := float64(event.Duration.Nanoseconds()) / 1e6
		record := []string{
			event.Timestamp.Format(time.RFC3339Nano),
			strconv.FormatFloat(durationMS, 'f', 3, 64),
			strconv.Itoa(event.DocumentsHealed),
			event.Type,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// MultiDBTestResults contains multi-database test results.
type MultiDBTestResults struct {
	TestConfig      interface{} // Can be MultiDBLoadTestConfig or LoadTestConfig
	DurationSeconds float64
	TotalOperations int64
	Databases       map[string]*DatabaseMetrics
	Global          *GlobalMetrics
}

// WriteMultiDBCSV writes CSV files for multi-database test results.
func WriteMultiDBCSV(results *MultiDBTestResults, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write per-database CSV files
	for dbName, dbMetrics := range results.Databases {
		dbDir := filepath.Join(outputDir, dbName)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return fmt.Errorf("failed to create database directory: %w", err)
		}

		// Write latency summary
		if err := writeDatabaseLatencyCSV(dbName, dbMetrics, dbDir); err != nil {
			return fmt.Errorf("failed to write latency CSV for %s: %w", dbName, err)
		}
	}

	// Write global WAL growth CSV (aggregated)
	if err := writeGlobalWALGrowthCSV(results, outputDir); err != nil {
		return fmt.Errorf("failed to write global WAL growth CSV: %w", err)
	}

	// Write phase statistics CSV
	if results.Global != nil && len(results.Global.PhaseStats) > 0 {
		if err := writePhaseStatsCSV(results.Global.PhaseStats, outputDir); err != nil {
			return fmt.Errorf("failed to write phase stats CSV: %w", err)
		}
	}

	return nil
}

func writeDatabaseLatencyCSV(dbName string, metrics *DatabaseMetrics, outputDir string) error {
	file, err := os.Create(filepath.Join(outputDir, "latency_summary.csv"))
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"operation", "p50_ms", "p95_ms", "p99_ms", "mean_ms", "min_ms", "max_ms", "count"}); err != nil {
		return err
	}

	// Write metrics for each operation type
	for opType, stats := range metrics.Latency {
		record := []string{
			string(opType),
			strconv.FormatFloat(stats.P50, 'f', 3, 64),
			strconv.FormatFloat(stats.P95, 'f', 3, 64),
			strconv.FormatFloat(stats.P99, 'f', 3, 64),
			strconv.FormatFloat(stats.Mean, 'f', 3, 64),
			strconv.FormatFloat(stats.Min, 'f', 3, 64),
			strconv.FormatFloat(stats.Max, 'f', 3, 64),
			strconv.FormatInt(stats.Count, 10),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

func writeGlobalWALGrowthCSV(results *MultiDBTestResults, outputDir string) error {
	file, err := os.Create(filepath.Join(outputDir, "global_wal_growth.csv"))
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"database", "initial_mb", "final_mb", "growth_mb", "growth_rate_kbps"}); err != nil {
		return err
	}

	// Write per-database WAL growth
	for dbName, dbMetrics := range results.Databases {
		initialMB := float64(dbMetrics.WALGrowth.InitialSizeBytes) / (1024 * 1024)
		finalMB := float64(dbMetrics.WALGrowth.FinalSizeBytes) / (1024 * 1024)
		growthMB := finalMB - initialMB
		growthRateKBps := dbMetrics.WALGrowth.GrowthRateBytesPerSec / 1024

		record := []string{
			dbName,
			strconv.FormatFloat(initialMB, 'f', 2, 64),
			strconv.FormatFloat(finalMB, 'f', 2, 64),
			strconv.FormatFloat(growthMB, 'f', 2, 64),
			strconv.FormatFloat(growthRateKBps, 'f', 2, 64),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

func writePhaseStatsCSV(phaseStats []PhaseStats, outputDir string) error {
	file, err := os.Create(filepath.Join(outputDir, "phase_stats.csv"))
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"phase", "start_time", "duration", "operations", "workers", "read_percent", "write_percent", "update_percent", "delete_percent"}); err != nil {
		return err
	}

	// Write phase statistics
	for _, phase := range phaseStats {
		record := []string{
			phase.Name,
			phase.StartTime.String(),
			phase.Duration.String(),
			strconv.FormatInt(phase.Operations, 10),
			strconv.Itoa(phase.Workers),
			strconv.Itoa(phase.CRUDPercent.ReadPercent),
			strconv.Itoa(phase.CRUDPercent.WritePercent),
			strconv.Itoa(phase.CRUDPercent.UpdatePercent),
			strconv.Itoa(phase.CRUDPercent.DeletePercent),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

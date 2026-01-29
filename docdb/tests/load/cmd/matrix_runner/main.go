package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/kartikbazzad/docdb/tests/load"
)

func main() {
	// Configuration flags
	databasesFlag := flag.String("databases", "1,3,6,12", "Comma-separated list of database counts to test")
	connectionsFlag := flag.String("connections", "1,5,10,20", "Comma-separated list of connections per DB to test")
	workersFlag := flag.String("workers", "1,2,5,10", "Comma-separated list of workers per DB to test")
	duration := flag.Duration("duration", 5*time.Minute, "Duration per test configuration")
	socketPath := flag.String("socket", "/tmp/docdb.sock", "IPC socket path")
	walDir := flag.String("wal-dir", "./docdb/data/wal", "WAL directory path")
	outputDir := flag.String("output-dir", "./matrix_results", "Output directory for results")
	docSize := flag.Int("doc-size", 1024, "Document size in bytes")
	docCount := flag.Int("doc-count", 10000, "Documents per database")
	readPercent := flag.Int("read-percent", 40, "Read operation percentage")
	writePercent := flag.Int("write-percent", 30, "Write operation percentage")
	updatePercent := flag.Int("update-percent", 20, "Update operation percentage")
	deletePercent := flag.Int("delete-percent", 10, "Delete operation percentage")
	csvOutput := flag.Bool("csv", true, "Generate CSV output files")
	seed := flag.Int64("seed", 0, "Random seed (0 = use timestamp)")
	restartDB := flag.Bool("restart-db", false, "Restart DocDB server between tests (not implemented)")
	flag.Parse()

	// Parse comma-separated lists
	databases := parseIntList(*databasesFlag)
	connections := parseIntList(*connectionsFlag)
	workers := parseIntList(*workersFlag)

	if len(databases) == 0 || len(connections) == 0 || len(workers) == 0 {
		log.Fatalf("Must specify at least one value for databases, connections, and workers")
	}

	// Create matrix configuration
	config := &load.TestMatrixConfig{
		Databases:        databases,
		ConnectionsPerDB: connections,
		WorkersPerDB:     workers,
		Duration:         *duration,
		DocumentSize:     *docSize,
		DocumentCount:    *docCount,
		ReadPercent:      *readPercent,
		WritePercent:     *writePercent,
		UpdatePercent:    *updatePercent,
		DeletePercent:    *deletePercent,
		SocketPath:       *socketPath,
		WALDir:           *walDir,
		OutputDir:        *outputDir,
		CSVOutput:        *csvOutput,
		Seed:             *seed,
		RestartDB:        *restartDB,
	}

	// Create and run matrix runner
	runner := load.NewMatrixRunner(config)
	if err := runner.Run(); err != nil {
		log.Fatalf("Matrix runner failed: %v", err)
	}

	log.Printf("Matrix test completed. Results in %s", *outputDir)
}

// parseIntList parses a comma-separated list of integers.
func parseIntList(s string) []int {
	parts := strings.Split(s, ",")
	result := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var val int
		if _, err := fmt.Sscanf(part, "%d", &val); err != nil {
			log.Fatalf("Invalid integer in list: %s", part)
		}
		result = append(result, val)
	}
	return result
}

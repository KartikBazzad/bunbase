package load

import (
	"fmt"
	"time"
)

// TestMatrixConfig defines the test matrix configuration.
type TestMatrixConfig struct {
	// Test variables
	Databases        []int // Number of databases to test [1, 3, 6, 12]
	ConnectionsPerDB []int // Connections per database [1, 5, 10, 20]
	WorkersPerDB     []int // Workers per database [1, 2, 5, 10]

	// Test parameters
	Duration      time.Duration // Duration per test configuration
	DocumentSize  int           // Document size in bytes
	DocumentCount int           // Documents per database
	ReadPercent   int           // Read operation percentage
	WritePercent  int           // Write operation percentage
	UpdatePercent int           // Update operation percentage
	DeletePercent int           // Delete operation percentage

	// Environment
	SocketPath string // IPC socket path
	WALDir     string // WAL directory path
	OutputDir  string // Directory for output files

	// Options
	CSVOutput bool  // Generate CSV files
	Seed      int64 // Random seed (0 = use timestamp)
	RestartDB bool  // Restart DocDB server between tests
}

// DefaultTestMatrixConfig returns a default test matrix configuration.
func DefaultTestMatrixConfig() *TestMatrixConfig {
	return &TestMatrixConfig{
		Databases:        []int{1, 3, 6, 12},
		ConnectionsPerDB: []int{1, 5, 10, 20},
		WorkersPerDB:     []int{1, 2, 5, 10},
		Duration:         5 * time.Minute,
		DocumentSize:     1024,
		DocumentCount:    10000,
		ReadPercent:      40,
		WritePercent:     30,
		UpdatePercent:    20,
		DeletePercent:    10,
		SocketPath:       "/tmp/docdb.sock",
		WALDir:           "./docdb/data/wal",
		OutputDir:        "./matrix_results",
		CSVOutput:        true,
		Seed:             0,
		RestartDB:        false,
	}
}

// TestConfiguration represents a single test configuration.
type TestConfiguration struct {
	Databases        int
	ConnectionsPerDB int
	WorkersPerDB     int
	Name             string // e.g., "1db_1conn_1w"
}

// GenerateTestConfigurations generates all test configurations from the matrix.
func (tmc *TestMatrixConfig) GenerateTestConfigurations() []TestConfiguration {
	var configs []TestConfiguration

	for _, dbCount := range tmc.Databases {
		for _, connCount := range tmc.ConnectionsPerDB {
			for _, workerCount := range tmc.WorkersPerDB {
				name := generateTestName(dbCount, connCount, workerCount)
				configs = append(configs, TestConfiguration{
					Databases:        dbCount,
					ConnectionsPerDB: connCount,
					WorkersPerDB:     workerCount,
					Name:             name,
				})
			}
		}
	}

	return configs
}

// generateTestName creates a test name from configuration.
func generateTestName(dbs, conns, workers int) string {
	return fmt.Sprintf("%ddb_%dconn_%dw", dbs, conns, workers)
}

// GetOutputPath returns the output file path (JSON) for a test configuration
// relative to the matrix OutputDir. The caller is responsible for joining it
// with OutputDir.
func (tmc *TestMatrixConfig) GetOutputPath(config TestConfiguration) string {
	return fmt.Sprintf("%s.json", config.Name)
}

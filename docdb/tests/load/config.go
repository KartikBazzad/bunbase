package load

import (
	"flag"
	"time"
)

// LoadTestConfig holds configuration for load tests.
type LoadTestConfig struct {
	// Test parameters
	Duration   time.Duration // Test duration (0 = run until Operations count)
	Workers    int           // Concurrent workers
	Operations int           // Total operations (if Duration=0)

	// Workload mix (percentages should sum to 100)
	ReadPercent   int // Percentage of read operations
	WritePercent  int // Percentage of write operations
	UpdatePercent int // Percentage of update operations
	DeletePercent int // Percentage of delete operations

	// Document parameters
	DocumentSize  int // Size of document payload (bytes)
	DocumentCount int // Number of unique documents

	// Metrics collection
	MetricsInterval time.Duration // How often to collect metrics

	// Database config
	SocketPath string // IPC socket path
	DBName     string // Database name
	WALDir     string // WAL directory for size tracking

	// Output
	OutputPath string // Path to output JSON file
	CSVOutput  bool   // Also generate CSV files

	// Random seed for reproducibility
	Seed int64
}

// DefaultConfig returns a default load test configuration.
func DefaultConfig() *LoadTestConfig {
	return &LoadTestConfig{
		Duration:        5 * time.Minute,
		Workers:         10,
		Operations:      0,
		ReadPercent:     40,
		WritePercent:    30,
		UpdatePercent:   20,
		DeletePercent:   10,
		DocumentSize:    1024, // 1KB
		DocumentCount:   10000,
		MetricsInterval: 1 * time.Second,
		SocketPath:      "/tmp/docdb.sock",
		DBName:          "loadtest",
		WALDir:          "/tmp/docdb/wal",
		OutputPath:      "loadtest_results.json",
		CSVOutput:       false,
		Seed:            time.Now().UnixNano(),
	}
}

// ParseFlags parses command-line flags and updates the config.
func (c *LoadTestConfig) ParseFlags() {
	flag.DurationVar(&c.Duration, "duration", c.Duration, "Test duration (e.g., 5m)")
	flag.IntVar(&c.Workers, "workers", c.Workers, "Number of concurrent workers")
	flag.IntVar(&c.Operations, "operations", c.Operations, "Total operations (0 = use duration)")
	flag.IntVar(&c.ReadPercent, "read-percent", c.ReadPercent, "Percentage of read operations")
	flag.IntVar(&c.WritePercent, "write-percent", c.WritePercent, "Percentage of write operations")
	flag.IntVar(&c.UpdatePercent, "update-percent", c.UpdatePercent, "Percentage of update operations")
	flag.IntVar(&c.DeletePercent, "delete-percent", c.DeletePercent, "Percentage of delete operations")
	flag.IntVar(&c.DocumentSize, "doc-size", c.DocumentSize, "Document payload size in bytes")
	flag.IntVar(&c.DocumentCount, "doc-count", c.DocumentCount, "Number of unique documents")
	flag.DurationVar(&c.MetricsInterval, "metrics-interval", c.MetricsInterval, "Metrics collection interval")
	flag.StringVar(&c.SocketPath, "socket", c.SocketPath, "IPC socket path")
	flag.StringVar(&c.DBName, "db-name", c.DBName, "Database name")
	flag.StringVar(&c.WALDir, "wal-dir", c.WALDir, "WAL directory path")
	flag.StringVar(&c.OutputPath, "output", c.OutputPath, "Output JSON file path")
	flag.BoolVar(&c.CSVOutput, "csv", c.CSVOutput, "Generate CSV output files")
	flag.Int64Var(&c.Seed, "seed", c.Seed, "Random seed for reproducibility")
}

// Validate checks if the configuration is valid.
func (c *LoadTestConfig) Validate() error {
	if c.Workers <= 0 {
		return &ConfigError{Field: "Workers", Message: "must be > 0"}
	}
	if c.Duration <= 0 && c.Operations <= 0 {
		return &ConfigError{Field: "Duration/Operations", Message: "must specify either duration or operations"}
	}
	if c.ReadPercent < 0 || c.WritePercent < 0 || c.UpdatePercent < 0 || c.DeletePercent < 0 {
		return &ConfigError{Field: "Percentages", Message: "cannot be negative"}
	}
	total := c.ReadPercent + c.WritePercent + c.UpdatePercent + c.DeletePercent
	if total != 100 {
		return &ConfigError{Field: "Percentages", Message: "must sum to 100"}
	}
	if c.DocumentSize <= 0 {
		return &ConfigError{Field: "DocumentSize", Message: "must be > 0"}
	}
	if c.DocumentCount <= 0 {
		return &ConfigError{Field: "DocumentCount", Message: "must be > 0"}
	}
	if c.SocketPath == "" {
		return &ConfigError{Field: "SocketPath", Message: "cannot be empty"}
	}
	if c.DBName == "" {
		return &ConfigError{Field: "DBName", Message: "cannot be empty"}
	}
	return nil
}

// ConfigError represents a configuration error.
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return "config error: " + e.Field + " - " + e.Message
}

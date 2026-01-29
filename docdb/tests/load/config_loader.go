package load

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ConfigFile represents the structure of a configuration file.
type ConfigFile struct {
	Databases       []DatabaseConfigFile `json:"databases" yaml:"databases"`
	WorkloadProfile *WorkloadProfileFile `json:"workload_profile" yaml:"workload_profile"`
	Test            TestConfigFile       `json:"test" yaml:"test"`
}

// DatabaseConfigFile represents database configuration in a file.
type DatabaseConfigFile struct {
	Name     string           `json:"name" yaml:"name"`
	Workers  int              `json:"workers" yaml:"workers"`
	DocSize  int              `json:"doc_size" yaml:"doc_size"`
	DocCount int              `json:"doc_count" yaml:"doc_count"`
	CRUD     *CRUDPercentages `json:"crud" yaml:"crud"`
	WALDir   string           `json:"wal_dir" yaml:"wal_dir"`
}

// WorkloadProfileFile represents workload profile in a file.
type WorkloadProfileFile struct {
	Phases []WorkloadPhaseFile `json:"phases" yaml:"phases"`
}

// WorkloadPhaseFile represents a workload phase in a file.
type WorkloadPhaseFile struct {
	Name           string              `json:"name" yaml:"name"`
	StartTime      string              `json:"start_time" yaml:"start_time"` // e.g., "0s", "1m"
	Duration       string              `json:"duration" yaml:"duration"`     // e.g., "1m", "30s"
	Workers        int                 `json:"workers" yaml:"workers"`
	CRUD           *CRUDPercentages    `json:"crud" yaml:"crud"`
	CRUDTransition *CRUDTransitionFile `json:"crud_transition" yaml:"crud_transition"`
	OperationRate  int                 `json:"operation_rate" yaml:"operation_rate"`
}

// CRUDTransitionFile represents a CRUD transition in a file.
type CRUDTransitionFile struct {
	Start CRUDPercentages `json:"start" yaml:"start"`
	End   CRUDPercentages `json:"end" yaml:"end"`
}

// TestConfigFile represents test configuration in a file.
type TestConfigFile struct {
	Duration        string `json:"duration" yaml:"duration"`
	Socket          string `json:"socket" yaml:"socket"`
	MetricsInterval string `json:"metrics_interval" yaml:"metrics_interval"`
	Output          string `json:"output" yaml:"output"`
	CSVOutput       bool   `json:"csv_output" yaml:"csv_output"`
	Seed            int64  `json:"seed" yaml:"seed"`
}

// LoadConfigFromFile loads configuration from a JSON or YAML file.
func LoadConfigFromFile(filePath string) (*MultiDBLoadTestConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var configFile ConfigFile
	if err := json.Unmarshal(data, &configFile); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return convertConfigFile(&configFile)
}

// convertConfigFile converts a ConfigFile to MultiDBLoadTestConfig.
func convertConfigFile(cf *ConfigFile) (*MultiDBLoadTestConfig, error) {
	cfg := NewMultiDBConfig()

	// Convert databases
	for _, dbFile := range cf.Databases {
		dbConfig := DatabaseConfig{
			Name:          dbFile.Name,
			Workers:       dbFile.Workers,
			DocumentSize:  dbFile.DocSize,
			DocumentCount: dbFile.DocCount,
			CRUDPercent:   dbFile.CRUD,
			WALDir:        dbFile.WALDir,
		}
		cfg.AddDatabase(dbConfig)
	}

	// Convert workload profile
	if cf.WorkloadProfile != nil {
		profile := &WorkloadProfile{
			Phases: make([]WorkloadPhase, 0, len(cf.WorkloadProfile.Phases)),
		}

		for _, phaseFile := range cf.WorkloadProfile.Phases {
			startTime, err := parseDuration(phaseFile.StartTime)
			if err != nil {
				return nil, fmt.Errorf("invalid start_time for phase %s: %w", phaseFile.Name, err)
			}

			duration, err := parseDuration(phaseFile.Duration)
			if err != nil {
				return nil, fmt.Errorf("invalid duration for phase %s: %w", phaseFile.Name, err)
			}

			phase := WorkloadPhase{
				Name:          phaseFile.Name,
				StartTime:     startTime,
				Duration:      duration,
				Workers:       phaseFile.Workers,
				OperationRate: phaseFile.OperationRate,
			}

			// Handle CRUD percentages
			if phaseFile.CRUDTransition != nil {
				// Gradual transition
				phase.CRUDTransition = NewCRUDTransition(
					phaseFile.CRUDTransition.Start,
					phaseFile.CRUDTransition.End,
					time.Now(), // Will be set to actual start time
					duration,
				)
				// Set initial CRUD to start
				phase.CRUDPercent = phaseFile.CRUDTransition.Start
			} else if phaseFile.CRUD != nil {
				// Fixed CRUD percentages
				phase.CRUDPercent = *phaseFile.CRUD
			} else {
				// Default CRUD
				phase.CRUDPercent = CRUDPercentages{
					ReadPercent:   40,
					WritePercent:  30,
					UpdatePercent: 20,
					DeletePercent: 10,
				}
			}

			profile.Phases = append(profile.Phases, phase)
		}

		cfg.SetWorkloadProfile(profile)
	}

	// Convert test configuration
	if cf.Test.Duration != "" {
		duration, err := parseDuration(cf.Test.Duration)
		if err != nil {
			return nil, fmt.Errorf("invalid test duration: %w", err)
		}
		cfg.Duration = duration
	}

	if cf.Test.Socket != "" {
		cfg.SocketPath = cf.Test.Socket
	}

	if cf.Test.MetricsInterval != "" {
		interval, err := parseDuration(cf.Test.MetricsInterval)
		if err != nil {
			return nil, fmt.Errorf("invalid metrics_interval: %w", err)
		}
		cfg.MetricsInterval = interval
	}

	if cf.Test.Output != "" {
		cfg.OutputPath = cf.Test.Output
	}

	cfg.CSVOutput = cf.Test.CSVOutput

	if cf.Test.Seed != 0 {
		cfg.Seed = cf.Test.Seed
	}

	// Set per-database CRUD if any database has CRUD config
	cfg.PerDatabaseCRUD = false
	for _, db := range cfg.Databases {
		if db.CRUDPercent != nil {
			cfg.PerDatabaseCRUD = true
			break
		}
	}

	return cfg, nil
}

// parseDuration parses a duration string (e.g., "5m", "30s", "1h").
func parseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

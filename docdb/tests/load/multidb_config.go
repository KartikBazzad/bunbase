package load

import (
	"time"
)

// CRUDPercentages holds operation mix percentages.
type CRUDPercentages struct {
	ReadPercent   int
	WritePercent  int
	UpdatePercent int
	DeletePercent int
}

// Validate checks if CRUD percentages sum to 100.
func (c *CRUDPercentages) Validate() error {
	total := c.ReadPercent + c.WritePercent + c.UpdatePercent + c.DeletePercent
	if total != 100 {
		return &ConfigError{
			Field:   "CRUDPercentages",
			Message: "must sum to 100",
		}
	}
	if c.ReadPercent < 0 || c.WritePercent < 0 || c.UpdatePercent < 0 || c.DeletePercent < 0 {
		return &ConfigError{
			Field:   "CRUDPercentages",
			Message: "cannot be negative",
		}
	}
	return nil
}

// DatabaseConfig defines a single database's test parameters.
type DatabaseConfig struct {
	Name          string
	Workers       int
	DocumentSize  int
	DocumentCount int
	CRUDPercent   *CRUDPercentages // Optional: per-database CRUD mix
	WALDir        string           // Optional: per-database WAL dir (defaults to base WALDir)
}

// WorkloadPhase defines a time period with specific characteristics.
type WorkloadPhase struct {
	Name           string
	StartTime      time.Duration // Relative to test start
	Duration       time.Duration
	Workers        int // Worker count for this phase
	CRUDPercent    CRUDPercentages
	CRUDTransition *CRUDTransition // Optional: gradual transition
	OperationRate  int             // Optional: target ops/sec (0 = unlimited)
}

// GetCRUDPercent returns CRUD percentages for the given time within this phase.
func (wp *WorkloadPhase) GetCRUDPercent(elapsed time.Duration) CRUDPercentages {
	if wp.CRUDTransition != nil {
		phaseStart := time.Now().Add(-elapsed)
		return wp.CRUDTransition.GetCurrentPercent(phaseStart)
	}
	return wp.CRUDPercent
}

// WorkloadProfile defines time-based workload changes.
type WorkloadProfile struct {
	Phases []WorkloadPhase
}

// GetCurrentPhase returns the active phase for the given elapsed time.
func (wp *WorkloadProfile) GetCurrentPhase(elapsed time.Duration) *WorkloadPhase {
	for i := range wp.Phases {
		phase := &wp.Phases[i]
		if elapsed >= phase.StartTime && elapsed < phase.StartTime+phase.Duration {
			return phase
		}
	}
	// Return last phase if past all phases
	if len(wp.Phases) > 0 {
		lastPhase := &wp.Phases[len(wp.Phases)-1]
		if elapsed >= lastPhase.StartTime+lastPhase.Duration {
			return lastPhase
		}
	}
	return nil
}

// Validate checks if the workload profile is valid.
func (wp *WorkloadProfile) Validate() error {
	if len(wp.Phases) == 0 {
		return &ConfigError{
			Field:   "WorkloadProfile",
			Message: "must have at least one phase",
		}
	}

	for i, phase := range wp.Phases {
		if phase.Duration <= 0 {
			return &ConfigError{
				Field:   "WorkloadProfile.Phases",
				Message: "phase duration must be > 0",
			}
		}
		if phase.Workers <= 0 {
			return &ConfigError{
				Field:   "WorkloadProfile.Phases",
				Message: "phase workers must be > 0",
			}
		}

		// Validate CRUD percentages (either fixed or transition)
		if phase.CRUDTransition == nil {
			if err := phase.CRUDPercent.Validate(); err != nil {
				return err
			}
		} else {
			if err := phase.CRUDTransition.StartPercent.Validate(); err != nil {
				return err
			}
			if err := phase.CRUDTransition.EndPercent.Validate(); err != nil {
				return err
			}
		}

		// Check phase ordering
		if i > 0 {
			prevEnd := wp.Phases[i-1].StartTime + wp.Phases[i-1].Duration
			if phase.StartTime < prevEnd {
				return &ConfigError{
					Field:   "WorkloadProfile.Phases",
					Message: "phases must be non-overlapping and in order",
				}
			}
		}
	}

	return nil
}

// MultiDBLoadTestConfig extends LoadTestConfig for multi-database testing.
type MultiDBLoadTestConfig struct {
	// Base configuration
	*LoadTestConfig

	// Multi-database support
	Databases []DatabaseConfig

	// Workload profile (phases/spikes)
	WorkloadProfile *WorkloadProfile

	// Global vs per-database settings
	PerDatabaseCRUD bool // If true, each DB has own CRUD percentages (overrides phase CRUD)
}

// NewMultiDBConfig creates a new multi-database configuration.
func NewMultiDBConfig() *MultiDBLoadTestConfig {
	return &MultiDBLoadTestConfig{
		LoadTestConfig:  DefaultConfig(),
		Databases:       make([]DatabaseConfig, 0),
		WorkloadProfile: nil,
		PerDatabaseCRUD: false,
	}
}

// AddDatabase adds a database configuration.
func (c *MultiDBLoadTestConfig) AddDatabase(dbConfig DatabaseConfig) {
	c.Databases = append(c.Databases, dbConfig)
}

// SetWorkloadProfile sets the workload profile.
func (c *MultiDBLoadTestConfig) SetWorkloadProfile(profile *WorkloadProfile) {
	c.WorkloadProfile = profile
}

// Validate checks if the multi-database configuration is valid.
func (c *MultiDBLoadTestConfig) Validate() error {
	// Validate base config
	if err := c.LoadTestConfig.Validate(); err != nil {
		return err
	}

	// Validate databases
	if len(c.Databases) == 0 {
		return &ConfigError{
			Field:   "Databases",
			Message: "must specify at least one database",
		}
	}

	dbNames := make(map[string]bool)
	for i, db := range c.Databases {
		if db.Name == "" {
			return &ConfigError{
				Field:   "Databases",
				Message: "database name cannot be empty",
			}
		}
		if dbNames[db.Name] {
			return &ConfigError{
				Field:   "Databases",
				Message: "duplicate database name: " + db.Name,
			}
		}
		dbNames[db.Name] = true

		if db.Workers <= 0 {
			return &ConfigError{
				Field:   "Databases",
				Message: "database workers must be > 0",
			}
		}
		if db.DocumentSize <= 0 {
			return &ConfigError{
				Field:   "Databases",
				Message: "database document size must be > 0",
			}
		}
		if db.DocumentCount <= 0 {
			return &ConfigError{
				Field:   "Databases",
				Message: "database document count must be > 0",
			}
		}

		// Validate per-database CRUD if specified
		if c.PerDatabaseCRUD && db.CRUDPercent != nil {
			if err := db.CRUDPercent.Validate(); err != nil {
				return err
			}
		}

		// Set default WAL dir if not specified
		if db.WALDir == "" {
			c.Databases[i].WALDir = c.WALDir
		}
	}

	// Validate workload profile if specified
	if c.WorkloadProfile != nil {
		if err := c.WorkloadProfile.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// GetTotalWorkers calculates total workers across all databases.
func (c *MultiDBLoadTestConfig) GetTotalWorkers() int {
	if c.WorkloadProfile != nil && len(c.WorkloadProfile.Phases) > 0 {
		// Use max workers from phases
		maxWorkers := 0
		for _, phase := range c.WorkloadProfile.Phases {
			if phase.Workers > maxWorkers {
				maxWorkers = phase.Workers
			}
		}
		return maxWorkers * len(c.Databases)
	}

	// Sum workers from databases
	total := 0
	for _, db := range c.Databases {
		total += db.Workers
	}
	return total
}

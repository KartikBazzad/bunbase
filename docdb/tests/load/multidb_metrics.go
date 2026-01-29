package load

import (
	"fmt"
	"sync"
	"time"
)

// DatabaseMetrics holds metrics for a single database.
type DatabaseMetrics struct {
	Name            string
	Latency         map[OperationType]PercentileStats
	WALGrowth       WALGrowthSummary
	Healing         HealingSummary
	OperationCounts map[OperationType]int64
	TotalOperations int64
}

// GlobalMetrics aggregates metrics across all databases.
type GlobalMetrics struct {
	TotalOperations int64
	Latency         map[OperationType]PercentileStats // Aggregated
	TotalWALGrowth  uint64
	TotalHealing    HealingSummary
	PhaseStats      []PhaseStats
	phaseOps        map[string]int64 // Internal: phase name -> operation count
}

// PhaseStats contains statistics for a workload phase.
type PhaseStats struct {
	Name        string
	StartTime   time.Duration
	Duration    time.Duration
	Operations  int64
	Workers     int
	CRUDPercent CRUDPercentages
}

// MultiDBMetrics aggregates metrics across databases.
type MultiDBMetrics struct {
	Databases map[string]*DatabaseMetrics
	Global    *GlobalMetrics
	mu        sync.RWMutex
	phaseOps  map[string]int64 // Phase name -> operation count
}

// NewMultiDBMetrics creates a new multi-database metrics collector.
func NewMultiDBMetrics() *MultiDBMetrics {
	return &MultiDBMetrics{
		Databases: make(map[string]*DatabaseMetrics),
		Global: &GlobalMetrics{
			Latency: make(map[OperationType]PercentileStats),
		},
		phaseOps: make(map[string]int64),
	}
}

// RecordPhaseOperation records an operation for a specific phase.
func (m *MultiDBMetrics) RecordPhaseOperation(phaseName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.phaseOps[phaseName]++
}

// CollectMetrics collects metrics from all databases.
func (m *MultiDBMetrics) CollectMetrics(dbManager *DatabaseManager, profileMgr *WorkloadProfileManager, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Collect per-database metrics
	for _, ctx := range dbManager.GetAllDatabases() {
		dbMetrics := &DatabaseMetrics{
			Name:            ctx.Name,
			Latency:         ctx.LatencyMetrics.GetStats(),
			WALGrowth:       ctx.WALTracker.GetSummary(),
			Healing:         ctx.HealingTracker.GetSummary(duration),
			OperationCounts: make(map[OperationType]int64),
		}

		// Count operations per type
		for opType, latencyStats := range dbMetrics.Latency {
			dbMetrics.OperationCounts[opType] = latencyStats.Count
			dbMetrics.TotalOperations += latencyStats.Count
		}

		m.Databases[ctx.Name] = dbMetrics
	}

	// Aggregate global metrics
	m.aggregateGlobalMetrics(duration, profileMgr)
}

// aggregateGlobalMetrics aggregates metrics across all databases.
func (m *MultiDBMetrics) aggregateGlobalMetrics(duration time.Duration, profileMgr *WorkloadProfileManager) {
	// Aggregate latency samples
	allSamples := make(map[OperationType][]time.Duration)
	for _, dbMetrics := range m.Databases {
		for opType := range dbMetrics.Latency {
			// For aggregation, we need to combine samples
			// Since we don't have raw samples here, we'll use weighted averages
			// In a real implementation, we'd combine raw samples
			if _, exists := allSamples[opType]; !exists {
				allSamples[opType] = make([]time.Duration, 0)
			}
		}
	}

	// Calculate aggregated percentiles
	// Note: This is simplified - real implementation would combine all samples
	aggregatedLatency := make(map[OperationType]PercentileStats)
	for opType := range allSamples {
		// Aggregate by taking weighted average of percentiles
		var totalCount int64
		var sumP50, sumP95, sumP99, sumP999, sumMean float64
		var minVal, maxVal float64
		minVal = 1e9 // Initialize to large value
		maxVal = 0
		firstStats := true

		for _, dbMetrics := range m.Databases {
			if stats, exists := dbMetrics.Latency[opType]; exists && stats.Count > 0 {
				totalCount += stats.Count
				sumP50 += stats.P50 * float64(stats.Count)
				sumP95 += stats.P95 * float64(stats.Count)
				sumP99 += stats.P99 * float64(stats.Count)
				sumP999 += stats.P999 * float64(stats.Count)
				sumMean += stats.Mean * float64(stats.Count)

				// Track min/max across databases
				if firstStats {
					minVal = stats.Min
					maxVal = stats.Max
					firstStats = false
				} else {
					if stats.Min < minVal {
						minVal = stats.Min
					}
					if stats.Max > maxVal {
						maxVal = stats.Max
					}
				}
			}
		}

		if totalCount > 0 {
			aggregatedLatency[opType] = PercentileStats{
				P50:   sumP50 / float64(totalCount),
				P95:   sumP95 / float64(totalCount),
				P99:   sumP99 / float64(totalCount),
				P999:  sumP999 / float64(totalCount),
				Mean:  sumMean / float64(totalCount),
				Min:   minVal,
				Max:   maxVal,
				Count: totalCount,
			}
		}
	}

	// Aggregate WAL growth
	var totalWALGrowth uint64
	for _, dbMetrics := range m.Databases {
		totalWALGrowth += dbMetrics.WALGrowth.FinalSizeBytes - dbMetrics.WALGrowth.InitialSizeBytes
	}

	// Aggregate healing
	totalHealing := HealingSummary{}
	for _, dbMetrics := range m.Databases {
		totalHealing.TotalHealings += dbMetrics.Healing.TotalHealings
		totalHealing.TotalDocumentsHealed += dbMetrics.Healing.TotalDocumentsHealed
		totalHealing.HealingTimeSeconds += dbMetrics.Healing.HealingTimeSeconds
	}
	totalHealing.OverheadPercent = (totalHealing.HealingTimeSeconds / duration.Seconds()) * 100.0

	// Calculate total operations
	var totalOps int64
	for _, dbMetrics := range m.Databases {
		totalOps += dbMetrics.TotalOperations
	}

	// Collect phase statistics
	phaseStats := make([]PhaseStats, 0)
	if profileMgr != nil && profileMgr.profile != nil {
		for _, phase := range profileMgr.profile.Phases {
			ops := m.phaseOps[phase.Name]
			phaseStats = append(phaseStats, PhaseStats{
				Name:        phase.Name,
				StartTime:   phase.StartTime,
				Duration:    phase.Duration,
				Operations:  ops,
				Workers:     phase.Workers,
				CRUDPercent: phase.CRUDPercent,
			})
		}
	}

	m.Global = &GlobalMetrics{
		TotalOperations: totalOps,
		Latency:         aggregatedLatency,
		TotalWALGrowth:  totalWALGrowth,
		TotalHealing:    totalHealing,
		PhaseStats:      phaseStats,
		phaseOps:        m.phaseOps,
	}
}

// GetDatabaseMetrics returns metrics for a specific database.
func (m *MultiDBMetrics) GetDatabaseMetrics(dbName string) (*DatabaseMetrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics, exists := m.Databases[dbName]
	if !exists {
		return nil, fmt.Errorf("database %s not found", dbName)
	}
	return metrics, nil
}

package metrics

import (
	"fmt"
	"sync"
	"time"

	"github.com/kartikbazzad/docdb/internal/errors"
	"github.com/kartikbazzad/docdb/internal/types"
)

// PrometheusExporter provides Prometheus/OpenMetrics format metrics.
type PrometheusExporter struct {
	mu sync.RWMutex

	// Operation counters
	operationsTotal map[string]map[string]uint64 // operation -> status -> count

	// Operation durations (histogram buckets in seconds)
	operationDurations map[string][]float64 // operation -> durations

	// System gauges
	documentsTotal uint64
	memoryBytes    uint64
	walSizeBytes   uint64

	// Error counters
	errorsTotal map[errors.ErrorCategory]uint64

	// Healing metrics
	healingOperationsTotal uint64
	documentsHealedTotal   uint64
}

// NewPrometheusExporter creates a new Prometheus metrics exporter.
func NewPrometheusExporter() *PrometheusExporter {
	return &PrometheusExporter{
		operationsTotal:    make(map[string]map[string]uint64),
		operationDurations: make(map[string][]float64),
		errorsTotal:        make(map[errors.ErrorCategory]uint64),
	}
}

// RecordOperation records an operation with its status and duration.
func (pe *PrometheusExporter) RecordOperation(operation string, status string, duration time.Duration) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if pe.operationsTotal[operation] == nil {
		pe.operationsTotal[operation] = make(map[string]uint64)
	}
	pe.operationsTotal[operation][status]++

	// Record duration in seconds
	if pe.operationDurations[operation] == nil {
		pe.operationDurations[operation] = make([]float64, 0, 100)
	}
	pe.operationDurations[operation] = append(pe.operationDurations[operation], duration.Seconds())

	// Keep only last 1000 durations per operation
	if len(pe.operationDurations[operation]) > 1000 {
		pe.operationDurations[operation] = pe.operationDurations[operation][len(pe.operationDurations[operation])-1000:]
	}
}

// SetDocumentsTotal sets the total number of documents.
func (pe *PrometheusExporter) SetDocumentsTotal(count uint64) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.documentsTotal = count
}

// SetMemoryBytes sets the memory usage in bytes.
func (pe *PrometheusExporter) SetMemoryBytes(bytes uint64) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.memoryBytes = bytes
}

// SetWALSizeBytes sets the WAL size in bytes.
func (pe *PrometheusExporter) SetWALSizeBytes(bytes uint64) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.walSizeBytes = bytes
}

// RecordError records an error occurrence.
func (pe *PrometheusExporter) RecordError(category errors.ErrorCategory) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.errorsTotal[category]++
}

// RecordHealingOperation records a healing operation.
func (pe *PrometheusExporter) RecordHealingOperation(documentsHealed uint64) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.healingOperationsTotal++
	pe.documentsHealedTotal += documentsHealed
}

// Export returns metrics in Prometheus/OpenMetrics format.
func (pe *PrometheusExporter) Export(stats *types.Stats) string {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	var output string

	// Operation counters
	output += "# HELP docdb_operations_total Total number of operations by type and status\n"
	output += "# TYPE docdb_operations_total counter\n"
	for operation, statuses := range pe.operationsTotal {
		for status, count := range statuses {
			output += fmt.Sprintf("docdb_operations_total{operation=\"%s\",status=\"%s\"} %d\n", operation, status, count)
		}
	}

	// Operation durations (histogram - simplified as summary)
	output += "# HELP docdb_operation_duration_seconds Operation duration in seconds\n"
	output += "# TYPE docdb_operation_duration_seconds summary\n"
	for operation, durations := range pe.operationDurations {
		if len(durations) == 0 {
			continue
		}

		// Calculate summary statistics
		var sum float64
		var min, max float64 = durations[0], durations[0]
		for _, d := range durations {
			sum += d
			if d < min {
				min = d
			}
			if d > max {
				max = d
			}
		}
		avg := sum / float64(len(durations))

		// Export as summary with count, sum, min, avg, max
		output += fmt.Sprintf("docdb_operation_duration_seconds{operation=\"%s\",quantile=\"0\"} %f\n", operation, min)
		output += fmt.Sprintf("docdb_operation_duration_seconds{operation=\"%s\",quantile=\"0.5\"} %f\n", operation, avg)
		output += fmt.Sprintf("docdb_operation_duration_seconds{operation=\"%s\",quantile=\"1\"} %f\n", operation, max)
		output += fmt.Sprintf("docdb_operation_duration_seconds_sum{operation=\"%s\"} %f\n", operation, sum)
		output += fmt.Sprintf("docdb_operation_duration_seconds_count{operation=\"%s\"} %d\n", operation, len(durations))
	}

	// System gauges
	output += "# HELP docdb_documents_total Total number of documents\n"
	output += "# TYPE docdb_documents_total gauge\n"
	output += fmt.Sprintf("docdb_documents_total %d\n", stats.DocsLive)

	output += "# HELP docdb_memory_bytes Memory usage in bytes\n"
	output += "# TYPE docdb_memory_bytes gauge\n"
	output += fmt.Sprintf("docdb_memory_bytes %d\n", stats.MemoryUsed)

	output += "# HELP docdb_wal_size_bytes WAL size in bytes\n"
	output += "# TYPE docdb_wal_size_bytes gauge\n"
	output += fmt.Sprintf("docdb_wal_size_bytes %d\n", stats.WALSize)

	// Error counters
	output += "# HELP docdb_errors_total Total number of errors by category\n"
	output += "# TYPE docdb_errors_total counter\n"
	for category, count := range pe.errorsTotal {
		categoryName := categoryString(category)
		output += fmt.Sprintf("docdb_errors_total{category=\"%s\"} %d\n", categoryName, count)
	}

	// Healing metrics
	output += "# HELP docdb_healing_operations_total Total number of healing operations\n"
	output += "# TYPE docdb_healing_operations_total counter\n"
	output += fmt.Sprintf("docdb_healing_operations_total %d\n", pe.healingOperationsTotal)

	output += "# HELP docdb_documents_healed_total Total number of documents healed\n"
	output += "# TYPE docdb_documents_healed_total counter\n"
	output += fmt.Sprintf("docdb_documents_healed_total %d\n", pe.documentsHealedTotal)

	return output
}

// categoryString converts ErrorCategory to string.
func categoryString(category errors.ErrorCategory) string {
	switch category {
	case errors.ErrorTransient:
		return "transient"
	case errors.ErrorPermanent:
		return "permanent"
	case errors.ErrorCritical:
		return "critical"
	case errors.ErrorValidation:
		return "validation"
	case errors.ErrorNetwork:
		return "network"
	default:
		return "unknown"
	}
}

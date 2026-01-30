package metrics

import (
	"fmt"
	"sync"
	"time"

	"github.com/kartikbazzad/docdb/internal/errors"
	"github.com/kartikbazzad/docdb/internal/types"
)

// globalExporter is set by the IPC handler so docdb can record partition metrics.
var globalExporter *PrometheusExporter
var globalExporterMu sync.RWMutex

// SetGlobalExporter sets the global exporter for partition metrics (called by IPC handler).
func SetGlobalExporter(pe *PrometheusExporter) {
	globalExporterMu.Lock()
	defer globalExporterMu.Unlock()
	globalExporter = pe
}

// RecordPartitionLockWait records lock wait time (called from docdb).
func RecordPartitionLockWait(db, partition string, duration time.Duration) {
	globalExporterMu.RLock()
	pe := globalExporter
	globalExporterMu.RUnlock()
	if pe != nil {
		pe.RecordPartitionLockWait(db, partition, duration)
	}
}

// RecordPartitionWALFsync records WAL fsync latency (called from docdb).
func RecordPartitionWALFsync(db, partition string, duration time.Duration) {
	globalExporterMu.RLock()
	pe := globalExporter
	globalExporterMu.RUnlock()
	if pe != nil {
		pe.RecordPartitionWALFsync(db, partition, duration)
	}
}

// RecordPartitionDatafileSync records datafile fsync latency (called from docdb).
func RecordPartitionDatafileSync(db, partition string, duration time.Duration) {
	globalExporterMu.RLock()
	pe := globalExporter
	globalExporterMu.RUnlock()
	if pe != nil {
		pe.RecordPartitionDatafileSync(db, partition, duration)
	}
}

// RecordPartitionWALRotation records WAL rotation duration (called from docdb).
func RecordPartitionWALRotation(db, partition string, duration time.Duration) {
	globalExporterMu.RLock()
	pe := globalExporter
	globalExporterMu.RUnlock()
	if pe != nil {
		pe.RecordPartitionWALRotation(db, partition, duration)
	}
}

// RecordCommitMuWait records time spent waiting to acquire commitMu (called from docdb).
func RecordCommitMuWait(db string, duration time.Duration) {
	globalExporterMu.RLock()
	pe := globalExporter
	globalExporterMu.RUnlock()
	if pe != nil {
		pe.RecordCommitMuWait(db, duration)
	}
}

// RecordCommitMuHold records time commitMu was held (called from docdb).
func RecordCommitMuHold(db string, duration time.Duration) {
	globalExporterMu.RLock()
	pe := globalExporter
	globalExporterMu.RUnlock()
	if pe != nil {
		pe.RecordCommitMuHold(db, duration)
	}
}

// RecordPartitionReplay records replay duration (called from docdb).
func RecordPartitionReplay(db, partition string, duration time.Duration) {
	globalExporterMu.RLock()
	pe := globalExporter
	globalExporterMu.RUnlock()
	if pe != nil {
		pe.RecordPartitionReplay(db, partition, duration)
	}
}

// RecordPartitionIndexScan records index scan duration (called from docdb).
func RecordPartitionIndexScan(db, partition string, duration time.Duration) {
	globalExporterMu.RLock()
	pe := globalExporter
	globalExporterMu.RUnlock()
	if pe != nil {
		pe.RecordPartitionIndexScan(db, partition, duration)
	}
}

// SetPartitionQueueDepth sets queue depth (called from docdb).
func SetPartitionQueueDepth(db, partition string, depth int) {
	globalExporterMu.RLock()
	pe := globalExporter
	globalExporterMu.RUnlock()
	if pe != nil {
		pe.SetPartitionQueueDepth(db, partition, depth)
	}
}

// RecordQueryMetrics records query execution metrics (Phase C.6).
func RecordQueryMetrics(db string, partitionsScanned, rowsScanned, rowsReturned uint64, executionTime time.Duration) {
	globalExporterMu.RLock()
	pe := globalExporter
	globalExporterMu.RUnlock()
	if pe != nil {
		pe.RecordQueryMetrics(db, partitionsScanned, rowsScanned, rowsReturned, executionTime)
	}
}

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

	// Partition metrics (Phase C.5)
	partitionLockWaitDurations    map[string]map[string][]float64 // db -> partition -> durations
	partitionWALFsyncDurations    map[string]map[string][]float64 // db -> partition -> durations
	partitionDatafileSyncDurations map[string]map[string][]float64 // db -> partition -> durations
	partitionWALRotationDurations  map[string]map[string][]float64 // db -> partition -> durations
	partitionReplayDurations       map[string]map[string][]float64 // db -> partition -> durations
	commitMuWaitDurations          map[string][]float64            // db -> wait durations (bottleneck profiling)
	commitMuHoldDurations          map[string][]float64             // db -> hold durations
	partitionIndexScanDurations map[string]map[string][]float64 // db -> partition -> durations
	partitionQueueDepth         map[string]map[string]int       // db -> partition -> depth

	// Query metrics (Phase C.6)
	queryPartitionsScanned map[string]uint64    // db -> count
	queryRowsScanned       map[string]uint64    // db -> count
	queryRowsReturned      map[string]uint64    // db -> count
	queryExecutionTimes    map[string][]float64 // db -> durations (seconds)
}

// NewPrometheusExporter creates a new Prometheus metrics exporter.
func NewPrometheusExporter() *PrometheusExporter {
	return &PrometheusExporter{
		operationsTotal:             make(map[string]map[string]uint64),
		operationDurations:          make(map[string][]float64),
		errorsTotal:                 make(map[errors.ErrorCategory]uint64),
		partitionLockWaitDurations:    make(map[string]map[string][]float64),
		partitionWALFsyncDurations:    make(map[string]map[string][]float64),
		partitionDatafileSyncDurations: make(map[string]map[string][]float64),
		partitionWALRotationDurations:  make(map[string]map[string][]float64),
		partitionReplayDurations:       make(map[string]map[string][]float64),
		commitMuWaitDurations:          make(map[string][]float64),
		commitMuHoldDurations:          make(map[string][]float64),
		partitionIndexScanDurations:    make(map[string]map[string][]float64),
		partitionQueueDepth:         make(map[string]map[string]int),
		queryPartitionsScanned:      make(map[string]uint64),
		queryRowsScanned:            make(map[string]uint64),
		queryRowsReturned:           make(map[string]uint64),
		queryExecutionTimes:         make(map[string][]float64),
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

	// Partition metrics (Phase C.5)
	output += exportPartitionMetrics(pe)

	// Query metrics (Phase C.6)
	output += exportQueryMetrics(pe)

	return output
}

// exportPartitionMetrics exports partition-level metrics.
func exportPartitionMetrics(pe *PrometheusExporter) string {
	var output string

	// Queue depth (gauge)
	output += "# HELP docdb_partition_queue_depth Current queue depth per partition\n"
	output += "# TYPE docdb_partition_queue_depth gauge\n"
	for db, partitions := range pe.partitionQueueDepth {
		for partition, depth := range partitions {
			output += fmt.Sprintf("docdb_partition_queue_depth{db=\"%s\",partition=\"%s\"} %d\n", db, partition, depth)
		}
	}

	// Lock wait (summary)
	output += exportPartitionDurations("docdb_partition_lock_wait_seconds", "Write lock wait time per partition", pe.partitionLockWaitDurations)

	// WAL fsync (summary)
	output += exportPartitionDurations("docdb_partition_wal_fsync_seconds", "WAL fsync latency per partition", pe.partitionWALFsyncDurations)

	// Datafile fsync (summary)
	output += exportPartitionDurations("docdb_partition_datafile_sync_seconds", "Datafile fsync latency per partition", pe.partitionDatafileSyncDurations)

	// WAL rotation (summary)
	output += exportPartitionDurations("docdb_partition_wal_rotation_seconds", "WAL rotation duration per partition", pe.partitionWALRotationDurations)

	// Commit mutex wait/hold (per DB, bottleneck profiling)
	output += exportDBDurations("docdb_commit_mu_wait_seconds", "Time waiting to acquire commit mutex per DB", pe.commitMuWaitDurations)
	output += exportDBDurations("docdb_commit_mu_hold_seconds", "Time commit mutex was held per DB", pe.commitMuHoldDurations)

	// Replay (summary)
	output += exportPartitionDurations("docdb_partition_replay_seconds", "Recovery replay duration per partition", pe.partitionReplayDurations)

	// Index scan (summary)
	output += exportPartitionDurations("docdb_partition_index_scan_seconds", "Index scan duration per partition", pe.partitionIndexScanDurations)

	return output
}

func exportPartitionDurations(metricName, help string, data map[string]map[string][]float64) string {
	var output string
	output += fmt.Sprintf("# HELP %s %s\n", metricName, help)
	output += fmt.Sprintf("# TYPE %s summary\n", metricName)
	for db, partitions := range data {
		for partition, durations := range partitions {
			if len(durations) == 0 {
				continue
			}
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
			labels := fmt.Sprintf("db=\"%s\",partition=\"%s\"", db, partition)
			output += fmt.Sprintf("%s{%s,quantile=\"0\"} %f\n", metricName, labels, min)
			output += fmt.Sprintf("%s{%s,quantile=\"0.5\"} %f\n", metricName, labels, avg)
			output += fmt.Sprintf("%s{%s,quantile=\"1\"} %f\n", metricName, labels, max)
			output += fmt.Sprintf("%s_sum{%s} %f\n", metricName, labels, sum)
			output += fmt.Sprintf("%s_count{%s} %d\n", metricName, labels, len(durations))
		}
	}
	return output
}

func exportDBDurations(metricName, help string, data map[string][]float64) string {
	var output string
	output += fmt.Sprintf("# HELP %s %s\n", metricName, help)
	output += fmt.Sprintf("# TYPE %s summary\n", metricName)
	for db, durations := range data {
		if len(durations) == 0 {
			continue
		}
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
		labels := fmt.Sprintf("db=\"%s\"", db)
		output += fmt.Sprintf("%s{%s,quantile=\"0\"} %f\n", metricName, labels, min)
		output += fmt.Sprintf("%s{%s,quantile=\"0.5\"} %f\n", metricName, labels, avg)
		output += fmt.Sprintf("%s{%s,quantile=\"1\"} %f\n", metricName, labels, max)
		output += fmt.Sprintf("%s_sum{%s} %f\n", metricName, labels, sum)
		output += fmt.Sprintf("%s_count{%s} %d\n", metricName, labels, len(durations))
	}
	return output
}

// RecordPartitionLockWait records lock wait time for a partition.
func (pe *PrometheusExporter) RecordPartitionLockWait(db, partition string, duration time.Duration) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	if pe.partitionLockWaitDurations[db] == nil {
		pe.partitionLockWaitDurations[db] = make(map[string][]float64)
	}
	durations := pe.partitionLockWaitDurations[db][partition]
	durations = append(durations, duration.Seconds())
	if len(durations) > 1000 {
		durations = durations[len(durations)-1000:]
	}
	pe.partitionLockWaitDurations[db][partition] = durations
}

// RecordPartitionWALFsync records WAL fsync latency for a partition.
func (pe *PrometheusExporter) RecordPartitionWALFsync(db, partition string, duration time.Duration) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	if pe.partitionWALFsyncDurations[db] == nil {
		pe.partitionWALFsyncDurations[db] = make(map[string][]float64)
	}
	durations := pe.partitionWALFsyncDurations[db][partition]
	durations = append(durations, duration.Seconds())
	if len(durations) > 1000 {
		durations = durations[len(durations)-1000:]
	}
	pe.partitionWALFsyncDurations[db][partition] = durations
}

// RecordPartitionDatafileSync records datafile fsync latency for a partition.
func (pe *PrometheusExporter) RecordPartitionDatafileSync(db, partition string, duration time.Duration) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	if pe.partitionDatafileSyncDurations[db] == nil {
		pe.partitionDatafileSyncDurations[db] = make(map[string][]float64)
	}
	durations := pe.partitionDatafileSyncDurations[db][partition]
	durations = append(durations, duration.Seconds())
	if len(durations) > 1000 {
		durations = durations[len(durations)-1000:]
	}
	pe.partitionDatafileSyncDurations[db][partition] = durations
}

// RecordPartitionWALRotation records WAL rotation duration for a partition.
func (pe *PrometheusExporter) RecordPartitionWALRotation(db, partition string, duration time.Duration) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	if pe.partitionWALRotationDurations[db] == nil {
		pe.partitionWALRotationDurations[db] = make(map[string][]float64)
	}
	durations := pe.partitionWALRotationDurations[db][partition]
	durations = append(durations, duration.Seconds())
	if len(durations) > 1000 {
		durations = durations[len(durations)-1000:]
	}
	pe.partitionWALRotationDurations[db][partition] = durations
}

// RecordCommitMuWait records time spent waiting to acquire commitMu per DB.
func (pe *PrometheusExporter) RecordCommitMuWait(db string, duration time.Duration) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	durations := pe.commitMuWaitDurations[db]
	durations = append(durations, duration.Seconds())
	if len(durations) > 1000 {
		durations = durations[len(durations)-1000:]
	}
	pe.commitMuWaitDurations[db] = durations
}

// RecordCommitMuHold records time commitMu was held per DB.
func (pe *PrometheusExporter) RecordCommitMuHold(db string, duration time.Duration) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	durations := pe.commitMuHoldDurations[db]
	durations = append(durations, duration.Seconds())
	if len(durations) > 1000 {
		durations = durations[len(durations)-1000:]
	}
	pe.commitMuHoldDurations[db] = durations
}

// RecordPartitionReplay records replay duration for a partition.
func (pe *PrometheusExporter) RecordPartitionReplay(db, partition string, duration time.Duration) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	if pe.partitionReplayDurations[db] == nil {
		pe.partitionReplayDurations[db] = make(map[string][]float64)
	}
	durations := pe.partitionReplayDurations[db][partition]
	durations = append(durations, duration.Seconds())
	if len(durations) > 1000 {
		durations = durations[len(durations)-1000:]
	}
	pe.partitionReplayDurations[db][partition] = durations
}

// RecordPartitionIndexScan records index scan duration for a partition.
func (pe *PrometheusExporter) RecordPartitionIndexScan(db, partition string, duration time.Duration) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	if pe.partitionIndexScanDurations[db] == nil {
		pe.partitionIndexScanDurations[db] = make(map[string][]float64)
	}
	durations := pe.partitionIndexScanDurations[db][partition]
	durations = append(durations, duration.Seconds())
	if len(durations) > 1000 {
		durations = durations[len(durations)-1000:]
	}
	pe.partitionIndexScanDurations[db][partition] = durations
}

// SetPartitionQueueDepth sets the queue depth for a partition.
func (pe *PrometheusExporter) SetPartitionQueueDepth(db, partition string, depth int) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	if pe.partitionQueueDepth[db] == nil {
		pe.partitionQueueDepth[db] = make(map[string]int)
	}
	pe.partitionQueueDepth[db][partition] = depth
}

// RecordQueryMetrics records query execution metrics (Phase C.6).
func (pe *PrometheusExporter) RecordQueryMetrics(db string, partitionsScanned, rowsScanned, rowsReturned uint64, executionTime time.Duration) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.queryPartitionsScanned[db] += partitionsScanned
	pe.queryRowsScanned[db] += rowsScanned
	pe.queryRowsReturned[db] += rowsReturned
	durations := pe.queryExecutionTimes[db]
	durations = append(durations, executionTime.Seconds())
	if len(durations) > 1000 {
		durations = durations[len(durations)-1000:]
	}
	pe.queryExecutionTimes[db] = durations
}

// exportQueryMetrics exports query metrics in Prometheus format.
func exportQueryMetrics(pe *PrometheusExporter) string {
	var output string

	// Query partitions scanned (counter)
	output += "# HELP docdb_query_partitions_scanned_total Total number of partitions scanned across all queries\n"
	output += "# TYPE docdb_query_partitions_scanned_total counter\n"
	for db, count := range pe.queryPartitionsScanned {
		output += fmt.Sprintf("docdb_query_partitions_scanned_total{db=\"%s\"} %d\n", db, count)
	}

	// Query rows scanned (counter)
	output += "# HELP docdb_query_rows_scanned_total Total number of rows scanned across all queries\n"
	output += "# TYPE docdb_query_rows_scanned_total counter\n"
	for db, count := range pe.queryRowsScanned {
		output += fmt.Sprintf("docdb_query_rows_scanned_total{db=\"%s\"} %d\n", db, count)
	}

	// Query rows returned (counter)
	output += "# HELP docdb_query_rows_returned_total Total number of rows returned across all queries\n"
	output += "# TYPE docdb_query_rows_returned_total counter\n"
	for db, count := range pe.queryRowsReturned {
		output += fmt.Sprintf("docdb_query_rows_returned_total{db=\"%s\"} %d\n", db, count)
	}

	// Query execution time (summary)
	output += "# HELP docdb_query_execution_seconds Query execution time\n"
	output += "# TYPE docdb_query_execution_seconds summary\n"
	for db, durations := range pe.queryExecutionTimes {
		if len(durations) == 0 {
			continue
		}
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
		labels := fmt.Sprintf("db=\"%s\"", db)
		output += fmt.Sprintf("docdb_query_execution_seconds{%s,quantile=\"0\"} %f\n", labels, min)
		output += fmt.Sprintf("docdb_query_execution_seconds{%s,quantile=\"0.5\"} %f\n", labels, avg)
		output += fmt.Sprintf("docdb_query_execution_seconds{%s,quantile=\"1\"} %f\n", labels, max)
		output += fmt.Sprintf("docdb_query_execution_seconds_sum{%s} %f\n", labels, sum)
		output += fmt.Sprintf("docdb_query_execution_seconds_count{%s} %d\n", labels, len(durations))
	}

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

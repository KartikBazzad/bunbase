// Package metrics provides counters and gauges for Bunder, exportable in Prometheus text format.
package metrics

import (
	"strconv"
	"sync"
	"sync/atomic"
)

// Metrics holds command counters (Get/Set/Del), connection gauge, and keys gauge.
// PrometheusFormat() returns the exposition format for /metrics.
type Metrics struct {
	CommandsTotal atomic.Uint64
	GetTotal      atomic.Uint64
	SetTotal      atomic.Uint64
	DelTotal      atomic.Uint64
	Connections   atomic.Int64
	KeysTotal     atomic.Uint64
	mu            sync.Mutex
}

var defaultMetrics Metrics

// Default returns the default metrics instance.
func Default() *Metrics {
	return &defaultMetrics
}

// IncCommands increments the total command counter.
func (m *Metrics) IncCommands() {
	m.CommandsTotal.Add(1)
}

// IncGet increments the GET counter.
func (m *Metrics) IncGet() {
	m.GetTotal.Add(1)
}

// IncSet increments the SET counter.
func (m *Metrics) IncSet() {
	m.SetTotal.Add(1)
}

// IncDel increments the DEL counter.
func (m *Metrics) IncDel() {
	m.DelTotal.Add(1)
}

// SetConnections sets the current connection count.
func (m *Metrics) SetConnections(n int64) {
	m.Connections.Store(n)
}

// SetKeys sets the total keys gauge (optional, updated on demand).
func (m *Metrics) SetKeys(n uint64) {
	m.KeysTotal.Store(n)
}

// PrometheusFormat returns the metrics in Prometheus text exposition format.
func (m *Metrics) PrometheusFormat() string {
	return "# HELP bunder_commands_total Total commands processed\n" +
		"# TYPE bunder_commands_total counter\n" +
		"bunder_commands_total " + formatUint64(m.CommandsTotal.Load()) + "\n" +
		"# HELP bunder_get_total GET commands\n" +
		"# TYPE bunder_get_total counter\n" +
		"bunder_get_total " + formatUint64(m.GetTotal.Load()) + "\n" +
		"# HELP bunder_set_total SET commands\n" +
		"# TYPE bunder_set_total counter\n" +
		"bunder_set_total " + formatUint64(m.SetTotal.Load()) + "\n" +
		"# HELP bunder_del_total DEL commands\n" +
		"# TYPE bunder_del_total counter\n" +
		"bunder_del_total " + formatUint64(m.DelTotal.Load()) + "\n" +
		"# HELP bunder_connections_current Current connections\n" +
		"# TYPE bunder_connections_current gauge\n" +
		"bunder_connections_current " + formatInt64(m.Connections.Load()) + "\n" +
		"# HELP bunder_keys_total Total keys (if set)\n" +
		"# TYPE bunder_keys_total gauge\n" +
		"bunder_keys_total " + formatUint64(m.KeysTotal.Load()) + "\n"
}

func formatUint64(n uint64) string {
	return strconv.FormatUint(n, 10)
}

func formatInt64(n int64) string {
	return strconv.FormatInt(n, 10)
}

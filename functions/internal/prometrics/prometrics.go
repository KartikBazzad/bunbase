package prometrics

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	once     sync.Once
	logLines *prometheus.CounterVec
)

func init() {
	logLines = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fn_log_lines_total",
			Help: "Total number of function log lines emitted",
		},
		[]string{"function_id", "level"},
	)
}

// IncLogLines increments the log line counter for the given function and level.
func IncLogLines(functionID, level string) {
	if functionID == "" {
		functionID = "unknown"
	}
	if level == "" {
		level = "info"
	}
	logLines.WithLabelValues(functionID, level).Inc()
}

// Handler returns the Prometheus HTTP handler for /metrics.
func Handler() http.Handler {
	return promhttp.Handler()
}

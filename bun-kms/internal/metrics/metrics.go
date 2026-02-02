package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestTotal counts HTTP requests by method and path prefix.
	RequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bunkms_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
	// RequestDuration is the latency of HTTP requests.
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "bunkms_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
	// OperationsTotal counts KMS operations (encrypt, decrypt, key create, etc.).
	OperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bunkms_operations_total",
			Help: "Total number of KMS operations",
		},
		[]string{"operation", "status"},
	)
)

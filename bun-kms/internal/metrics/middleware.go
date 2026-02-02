package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Middleware records request count and duration for the given handler.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := pathLabel(r.URL.Path)
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(rec.status)
		RequestTotal.WithLabelValues(r.Method, path, status).Inc()
		RequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

func pathLabel(p string) string {
	p = strings.Trim(p, "/")
	parts := strings.SplitN(p, "/", 3)
	if len(parts) >= 2 {
		return parts[0] + "_" + parts[1]
	}
	if len(parts) == 1 && parts[0] != "" {
		return parts[0]
	}
	return "root"
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

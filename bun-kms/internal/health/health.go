package health

import (
	"encoding/json"
	"net/http"
	"sync"
)

// Checker is a function that returns an error if the check fails.
type Checker func() error

// Handler returns an HTTP handler for health and readiness.
// masterKeyPresent is true when master key is loaded; storeCheck is optional (nil = skip).
func Handler(masterKeyPresent bool, storeCheck Checker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := http.StatusOK
		checks := map[string]string{
			"master_key": "ok",
		}
		if !masterKeyPresent {
			checks["master_key"] = "not_loaded"
			status = http.StatusServiceUnavailable
		}
		if storeCheck != nil {
			if err := storeCheck(); err != nil {
				checks["storage"] = err.Error()
				status = http.StatusServiceUnavailable
			} else {
				checks["storage"] = "ok"
			}
		}
		resp := map[string]any{
			"status": "ok",
			"checks": checks,
		}
		if status != http.StatusOK {
			resp["status"] = "degraded"
		}
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(resp)
	})
}

// PingHandler returns a simple 200 OK for liveness (no checks).
func PingHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
}

// Readiness wraps a checker and runs it on each request (with optional caching).
type Readiness struct {
	mu     sync.RWMutex
	check  Checker
	cached bool
	ok     bool
}

// NewReadiness returns a readiness checker.
func NewReadiness(check Checker) *Readiness {
	return &Readiness{check: check}
}

// ServeHTTP runs the check and returns 200 if nil, 503 otherwise.
func (r *Readiness) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.check == nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	err := r.check()
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "not_ready", "error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

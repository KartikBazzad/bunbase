package errors

import (
	"sync"
	"time"
)

// ErrorTracker tracks error metrics for observability.
type ErrorTracker struct {
	mu             sync.RWMutex
	errorCounts    map[ErrorCategory]uint64
	errorRates     map[ErrorCategory]float64 // Errors per second
	lastOccurrence map[ErrorCategory]time.Time
	criticalAlerts []CriticalAlert
}

// CriticalAlert represents a critical error that requires attention.
type CriticalAlert struct {
	Category    ErrorCategory
	Error       error
	OccurredAt  time.Time
	Description string
}

// NewErrorTracker creates a new error tracker.
func NewErrorTracker() *ErrorTracker {
	return &ErrorTracker{
		errorCounts:    make(map[ErrorCategory]uint64),
		errorRates:     make(map[ErrorCategory]float64),
		lastOccurrence: make(map[ErrorCategory]time.Time),
		criticalAlerts: make([]CriticalAlert, 0),
	}
}

// RecordError records an error occurrence.
func (et *ErrorTracker) RecordError(err error, category ErrorCategory) {
	et.mu.Lock()
	defer et.mu.Unlock()

	et.errorCounts[category]++
	et.lastOccurrence[category] = time.Now()

	// Calculate error rate (simplified: errors per second over last minute)
	// For v0.1, we'll use a simple counter-based approach
	// In production, this would use a sliding window

	// Alert on critical errors
	if category == ErrorCritical {
		alert := CriticalAlert{
			Category:    category,
			Error:       err,
			OccurredAt:  time.Now(),
			Description: err.Error(),
		}
		et.criticalAlerts = append(et.criticalAlerts, alert)

		// Keep only last 100 alerts
		if len(et.criticalAlerts) > 100 {
			et.criticalAlerts = et.criticalAlerts[len(et.criticalAlerts)-100:]
		}
	}
}

// GetErrorCount returns the count of errors for a category.
func (et *ErrorTracker) GetErrorCount(category ErrorCategory) uint64 {
	et.mu.RLock()
	defer et.mu.RUnlock()
	return et.errorCounts[category]
}

// GetErrorRate returns the error rate for a category.
func (et *ErrorTracker) GetErrorRate(category ErrorCategory) float64 {
	et.mu.RLock()
	defer et.mu.RUnlock()
	return et.errorRates[category]
}

// GetLastOccurrence returns the last occurrence time for a category.
func (et *ErrorTracker) GetLastOccurrence(category ErrorCategory) time.Time {
	et.mu.RLock()
	defer et.mu.RUnlock()
	return et.lastOccurrence[category]
}

// GetCriticalAlerts returns all critical alerts.
func (et *ErrorTracker) GetCriticalAlerts() []CriticalAlert {
	et.mu.RLock()
	defer et.mu.RUnlock()

	alerts := make([]CriticalAlert, len(et.criticalAlerts))
	copy(alerts, et.criticalAlerts)
	return alerts
}

// Reset clears all error tracking data.
func (et *ErrorTracker) Reset() {
	et.mu.Lock()
	defer et.mu.Unlock()

	et.errorCounts = make(map[ErrorCategory]uint64)
	et.errorRates = make(map[ErrorCategory]float64)
	et.lastOccurrence = make(map[ErrorCategory]time.Time)
	et.criticalAlerts = make([]CriticalAlert, 0)
}

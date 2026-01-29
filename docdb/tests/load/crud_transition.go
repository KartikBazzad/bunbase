package load

import (
	"math"
	"time"
)

// CRUDTransition handles gradual CRUD percentage changes.
type CRUDTransition struct {
	StartPercent CRUDPercentages
	EndPercent   CRUDPercentages
	StartTime    time.Time
	Duration     time.Duration
}

// NewCRUDTransition creates a new CRUD transition.
func NewCRUDTransition(start, end CRUDPercentages, startTime time.Time, duration time.Duration) *CRUDTransition {
	return &CRUDTransition{
		StartPercent: start,
		EndPercent:   end,
		StartTime:    startTime,
		Duration:     duration,
	}
}

// GetCurrentPercent returns CRUD percentages for current time.
func (ct *CRUDTransition) GetCurrentPercent(now time.Time) CRUDPercentages {
	elapsed := now.Sub(ct.StartTime)
	if elapsed <= 0 {
		return ct.StartPercent
	}
	if elapsed >= ct.Duration {
		return ct.EndPercent
	}

	progress := float64(elapsed) / float64(ct.Duration)

	// Linear interpolation
	return CRUDPercentages{
		ReadPercent:   interpolate(ct.StartPercent.ReadPercent, ct.EndPercent.ReadPercent, progress),
		WritePercent:  interpolate(ct.StartPercent.WritePercent, ct.EndPercent.WritePercent, progress),
		UpdatePercent: interpolate(ct.StartPercent.UpdatePercent, ct.EndPercent.UpdatePercent, progress),
		DeletePercent: interpolate(ct.StartPercent.DeletePercent, ct.EndPercent.DeletePercent, progress),
	}
}

// interpolate performs linear interpolation between start and end values.
func interpolate(start, end int, progress float64) int {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	value := float64(start) + (float64(end)-float64(start))*progress
	return int(math.Round(value))
}

// Validate checks if the transition is valid.
func (ct *CRUDTransition) Validate() error {
	if ct.Duration <= 0 {
		return &ConfigError{
			Field:   "CRUDTransition",
			Message: "duration must be > 0",
		}
	}
	if err := ct.StartPercent.Validate(); err != nil {
		return err
	}
	if err := ct.EndPercent.Validate(); err != nil {
		return err
	}
	return nil
}

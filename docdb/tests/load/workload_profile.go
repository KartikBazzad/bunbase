package load

import (
	"sync"
	"time"
)

// WorkloadProfileManager manages workload phases and transitions.
type WorkloadProfileManager struct {
	profile      *WorkloadProfile
	startTime    time.Time
	currentPhase *WorkloadPhase
	mu           sync.RWMutex
}

// NewWorkloadProfileManager creates a new workload profile manager.
func NewWorkloadProfileManager(profile *WorkloadProfile) *WorkloadProfileManager {
	return &WorkloadProfileManager{
		profile:   profile,
		startTime: time.Now(),
	}
}

// Start initializes the profile manager with test start time.
func (wpm *WorkloadProfileManager) Start(startTime time.Time) {
	wpm.mu.Lock()
	defer wpm.mu.Unlock()
	wpm.startTime = startTime
}

// GetCurrentPhase returns the active phase for the current time.
func (wpm *WorkloadProfileManager) GetCurrentPhase() *WorkloadPhase {
	wpm.mu.RLock()
	defer wpm.mu.RUnlock()

	if wpm.profile == nil {
		return nil
	}

	elapsed := time.Since(wpm.startTime)
	return wpm.profile.GetCurrentPhase(elapsed)
}

// GetCRUDPercent returns CRUD percentages for the current time.
func (wpm *WorkloadProfileManager) GetCRUDPercent() CRUDPercentages {
	wpm.mu.RLock()
	defer wpm.mu.RUnlock()

	if wpm.profile == nil {
		// Default CRUD mix
		return CRUDPercentages{
			ReadPercent:   40,
			WritePercent:  30,
			UpdatePercent: 20,
			DeletePercent: 10,
		}
	}

	elapsed := time.Since(wpm.startTime)
	phase := wpm.profile.GetCurrentPhase(elapsed)
	if phase == nil {
		// Return default if no phase active
		return CRUDPercentages{
			ReadPercent:   40,
			WritePercent:  30,
			UpdatePercent: 20,
			DeletePercent: 10,
		}
	}

	phaseElapsed := elapsed - phase.StartTime
	return phase.GetCRUDPercent(phaseElapsed)
}

// GetWorkerCount returns the worker count for the current phase.
func (wpm *WorkloadProfileManager) GetWorkerCount() int {
	wpm.mu.RLock()
	defer wpm.mu.RUnlock()

	if wpm.profile == nil {
		return 10 // Default
	}

	elapsed := time.Since(wpm.startTime)
	phase := wpm.profile.GetCurrentPhase(elapsed)
	if phase == nil {
		return 10 // Default
	}

	return phase.Workers
}

// GetOperationRate returns the target operation rate for the current phase.
func (wpm *WorkloadProfileManager) GetOperationRate() int {
	wpm.mu.RLock()
	defer wpm.mu.RUnlock()

	if wpm.profile == nil {
		return 0 // Unlimited
	}

	elapsed := time.Since(wpm.startTime)
	phase := wpm.profile.GetCurrentPhase(elapsed)
	if phase == nil {
		return 0 // Unlimited
	}

	return phase.OperationRate
}

// IsPhaseTransition checks if we've transitioned to a new phase.
func (wpm *WorkloadProfileManager) IsPhaseTransition() bool {
	wpm.mu.Lock()
	defer wpm.mu.Unlock()

	if wpm.profile == nil {
		return false
	}

	elapsed := time.Since(wpm.startTime)
	newPhase := wpm.profile.GetCurrentPhase(elapsed)

	if wpm.currentPhase == nil {
		wpm.currentPhase = newPhase
		return newPhase != nil
	}

	if newPhase == nil {
		return false
	}

	transitioned := wpm.currentPhase.Name != newPhase.Name
	if transitioned {
		wpm.currentPhase = newPhase
	}
	return transitioned
}

// GetPhaseInfo returns information about the current phase.
func (wpm *WorkloadProfileManager) GetPhaseInfo() *PhaseInfo {
	wpm.mu.RLock()
	defer wpm.mu.RUnlock()

	if wpm.profile == nil {
		return nil
	}

	elapsed := time.Since(wpm.startTime)
	phase := wpm.profile.GetCurrentPhase(elapsed)
	if phase == nil {
		return nil
	}

	phaseElapsed := elapsed - phase.StartTime
	remaining := phase.Duration - phaseElapsed
	if remaining < 0 {
		remaining = 0
	}

	return &PhaseInfo{
		Name:          phase.Name,
		Elapsed:       phaseElapsed,
		Remaining:     remaining,
		Workers:       phase.Workers,
		CRUDPercent:   phase.GetCRUDPercent(phaseElapsed),
		OperationRate: phase.OperationRate,
	}
}

// PhaseInfo contains information about the current phase.
type PhaseInfo struct {
	Name          string
	Elapsed       time.Duration
	Remaining     time.Duration
	Workers       int
	CRUDPercent   CRUDPercentages
	OperationRate int
}

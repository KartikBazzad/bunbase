package memory

import (
	"sync"
	"sync/atomic"
)

type Caps struct {
	mu             sync.RWMutex
	globalCapacity uint64
	perDBLimit     map[uint64]uint64
	perDBUsage     map[uint64]*uint64
	globalUsage    uint64
	replayBudget   map[uint64]uint64 // Per-DB replay budget (bytes)
	replayUsage    map[uint64]*uint64 // Per-DB replay usage (bytes)
}

func NewCaps(globalCapacityMB uint64, perDBLimitMB uint64) *Caps {
	return &Caps{
		globalCapacity: globalCapacityMB * 1024 * 1024,
		perDBLimit:     make(map[uint64]uint64),
		perDBUsage:     make(map[uint64]*uint64),
		replayBudget:   make(map[uint64]uint64),
		replayUsage:    make(map[uint64]*uint64),
	}
}

func (c *Caps) RegisterDB(dbID uint64, limitMB uint64) {
	if _, exists := c.perDBLimit[dbID]; exists {
		return
	}

	limit := limitMB * 1024 * 1024
	if limitMB == 0 {
		limit = c.globalCapacity / 10
	}

	c.perDBLimit[dbID] = limit
	usage := uint64(0)
	c.perDBUsage[dbID] = &usage
}

func (c *Caps) UnregisterDB(dbID uint64) {
	delete(c.perDBLimit, dbID)
	delete(c.perDBUsage, dbID)
}

func (c *Caps) TryAllocate(dbID uint64, size uint64) bool {
	currentUsage := atomic.LoadUint64(&c.globalUsage)
	if currentUsage+size > c.globalCapacity {
		return false
	}

	if dbUsagePtr, exists := c.perDBUsage[dbID]; exists {
		dbUsage := atomic.LoadUint64(dbUsagePtr)
		if dbUsage+size > c.perDBLimit[dbID] {
			return false
		}
		atomic.AddUint64(dbUsagePtr, size)
	}

	atomic.AddUint64(&c.globalUsage, size)
	return true
}

func (c *Caps) Free(dbID uint64, size uint64) {
	if size > atomic.LoadUint64(&c.globalUsage) {
		size = atomic.LoadUint64(&c.globalUsage)
	}
	atomic.AddUint64(&c.globalUsage, ^uint64(size-1))

	if dbUsagePtr, exists := c.perDBUsage[dbID]; exists {
		dbUsage := atomic.LoadUint64(dbUsagePtr)
		if size > dbUsage {
			size = dbUsage
		}
		atomic.AddUint64(dbUsagePtr, ^uint64(size-1))
	}
}

func (c *Caps) GlobalUsage() uint64 {
	return atomic.LoadUint64(&c.globalUsage)
}

func (c *Caps) GlobalCapacity() uint64 {
	return c.globalCapacity
}

func (c *Caps) DBUsage(dbID uint64) uint64 {
	if dbUsagePtr, exists := c.perDBUsage[dbID]; exists {
		return atomic.LoadUint64(dbUsagePtr)
	}
	return 0
}

func (c *Caps) DBLimit(dbID uint64) uint64 {
	return c.perDBLimit[dbID]
}

func (c *Caps) CanAllocate(dbID uint64, size uint64) bool {
	currentUsage := atomic.LoadUint64(&c.globalUsage)
	if currentUsage+size > c.globalCapacity {
		return false
	}

	if dbUsagePtr, exists := c.perDBUsage[dbID]; exists {
		dbUsage := atomic.LoadUint64(dbUsagePtr)
		if dbUsage+size > c.perDBLimit[dbID] {
			return false
		}
	}

	return true
}

// SetReplayBudget sets the replay budget for a database.
// If budgetMB is 0, uses the per-DB limit instead.
func (c *Caps) SetReplayBudget(dbID uint64, budgetMB uint64, perDBLimitMB uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var budget uint64
	if budgetMB > 0 {
		budget = budgetMB * 1024 * 1024
	} else {
		budget = perDBLimitMB * 1024 * 1024
		if budget == 0 {
			budget = c.globalCapacity / 10 // Fallback to 10% of global capacity
		}
	}

	c.replayBudget[dbID] = budget
	usage := uint64(0)
	c.replayUsage[dbID] = &usage
}

// TryAllocateReplay attempts to allocate memory from the replay budget.
// Returns true if allocation succeeds, false if replay budget is exceeded.
func (c *Caps) TryAllocateReplay(dbID uint64, size uint64) bool {
	c.mu.RLock()
	budget, hasBudget := c.replayBudget[dbID]
	usagePtr, hasUsage := c.replayUsage[dbID]
	c.mu.RUnlock()

	if !hasBudget || !hasUsage {
		// No replay budget set, fall back to normal allocation
		return c.TryAllocate(dbID, size)
	}

	currentReplayUsage := atomic.LoadUint64(usagePtr)
	if currentReplayUsage+size > budget {
		return false
	}

	// Check global capacity (replay still respects global limit)
	currentGlobalUsage := atomic.LoadUint64(&c.globalUsage)
	if currentGlobalUsage+size > c.globalCapacity {
		return false
	}

	// Allocate from replay budget
	atomic.AddUint64(usagePtr, size)
	atomic.AddUint64(&c.globalUsage, size)
	return true
}

// MergeReplayUsage merges replay usage into normal usage and clears replay tracking.
func (c *Caps) MergeReplayUsage(dbID uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	usagePtr, hasUsage := c.replayUsage[dbID]
	if !hasUsage {
		return
	}

	replayUsage := atomic.LoadUint64(usagePtr)

	// Merge into normal usage
	if dbUsagePtr, exists := c.perDBUsage[dbID]; exists {
		atomic.AddUint64(dbUsagePtr, replayUsage)
	}

	// Clear replay tracking
	delete(c.replayBudget, dbID)
	delete(c.replayUsage, dbID)
}

// GetReplayBudget returns the current replay budget for a database (in bytes).
func (c *Caps) GetReplayBudget(dbID uint64) uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.replayBudget[dbID]
}

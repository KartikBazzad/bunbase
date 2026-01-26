package memory

import (
	"sync/atomic"
)

type Caps struct {
	globalCapacity uint64
	perDBLimit     map[uint64]uint64
	perDBUsage     map[uint64]*uint64
	globalUsage    uint64
}

func NewCaps(globalCapacityMB uint64, perDBLimitMB uint64) *Caps {
	return &Caps{
		globalCapacity: globalCapacityMB * 1024 * 1024,
		perDBLimit:     make(map[uint64]uint64),
		perDBUsage:     make(map[uint64]*uint64),
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

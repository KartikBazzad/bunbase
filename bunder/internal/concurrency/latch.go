package concurrency

import (
	"sync"

	"github.com/kartikbazzad/bunbase/bunder/internal/storage"
)

// LatchPool provides per-page RWMutex latches for B+Tree concurrency.
// RLock/RUnlock for reads; Lock/Unlock for writes. Used for fine-grained locking (future use).
type LatchPool struct {
	mu      sync.Mutex
	latches map[storage.PageID]*sync.RWMutex
}

// NewLatchPool creates a latch pool.
func NewLatchPool() *LatchPool {
	return &LatchPool{latches: make(map[storage.PageID]*sync.RWMutex)}
}

func (p *LatchPool) get(pageID storage.PageID) *sync.RWMutex {
	p.mu.Lock()
	defer p.mu.Unlock()
	if l, ok := p.latches[pageID]; ok {
		return l
	}
	l := &sync.RWMutex{}
	p.latches[pageID] = l
	return l
}

// RLock acquires a read lock on the page.
func (p *LatchPool) RLock(pageID storage.PageID) {
	p.get(pageID).RLock()
}

// RUnlock releases a read lock.
func (p *LatchPool) RUnlock(pageID storage.PageID) {
	p.get(pageID).RUnlock()
}

// Lock acquires a write lock on the page.
func (p *LatchPool) Lock(pageID storage.PageID) {
	p.get(pageID).Lock()
}

// Unlock releases a write lock.
func (p *LatchPool) Unlock(pageID storage.PageID) {
	p.get(pageID).Unlock()
}

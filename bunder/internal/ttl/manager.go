// Package ttl provides TTL (time-to-live) management for keys using a timing wheel and sweeper.
package ttl

import (
	"sync"
	"time"
)

// Manager manages TTL for keys: Set/Get/Remove expiry, TTLSeconds, and a background sweeper
// that calls onExpire for each expired key at TTLCheckInterval.
type Manager struct {
	mu       sync.RWMutex
	ttls     map[string]time.Time
	wheel    *TimingWheel
	onExpire func(key string)
	stop     chan struct{}
	interval time.Duration
}

// NewManager creates a TTL manager. onExpire is called for each expired key.
func NewManager(onExpire func(key string), checkInterval time.Duration) *Manager {
	if checkInterval <= 0 {
		checkInterval = time.Second
	}
	m := &Manager{
		ttls:     make(map[string]time.Time),
		wheel:    NewTimingWheel(time.Second),
		onExpire: onExpire,
		stop:     make(chan struct{}),
		interval: checkInterval,
	}
	go m.run()
	return m
}

// Set sets the TTL for key to expire at expires.
func (m *Manager) Set(key string, expires time.Time) {
	m.mu.Lock()
	m.ttls[key] = expires
	m.mu.Unlock()
	m.wheel.Add(key, expires)
}

// Get returns the expiry time for key, and true if set.
func (m *Manager) Get(key string) (time.Time, bool) {
	m.mu.RLock()
	t, ok := m.ttls[key]
	m.mu.RUnlock()
	return t, ok
}

// Remove removes TTL for key.
func (m *Manager) Remove(key string) {
	m.mu.Lock()
	delete(m.ttls, key)
	m.mu.Unlock()
	m.wheel.Remove(key)
}

// TTLSeconds returns remaining seconds until key expires; -1 if no TTL, -2 if key not found.
func (m *Manager) TTLSeconds(key string, now time.Time) int {
	m.mu.RLock()
	exp, ok := m.ttls[key]
	m.mu.RUnlock()
	if !ok {
		return -2
	}
	if exp.Before(now) || exp.Equal(now) {
		return -1
	}
	sec := int(exp.Sub(now).Seconds())
	if sec < 0 {
		return -1
	}
	return sec
}

func (m *Manager) run() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	for {
		select {
		case <-m.stop:
			return
		case now := <-ticker.C:
			expired := m.wheel.Expired(now)
			for _, key := range expired {
				m.mu.Lock()
				delete(m.ttls, key)
				m.mu.Unlock()
				if m.onExpire != nil {
					m.onExpire(key)
				}
			}
		}
	}
}

// Stop stops the TTL sweeper.
func (m *Manager) Stop() {
	close(m.stop)
}

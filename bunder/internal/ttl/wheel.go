package ttl

import (
	"container/list"
	"sync"
	"time"
)

// Slot is one bucket in the timing wheel holding keys that expire in that time slot.
type Slot struct {
	mu    sync.Mutex
	keys  map[string]*list.Element
	order *list.List
}

func newSlot() *Slot {
	return &Slot{keys: make(map[string]*list.Element), order: list.New()}
}

// entry is a key with its expiry time.
type entry struct {
	key     string
	expires time.Time
}

// TimingWheel is a simple timing wheel for TTL: slots by second (or minute) and sweep expired keys.
type TimingWheel struct {
	mu          sync.RWMutex
	slots       map[int64]*Slot
	granularity time.Duration
}

// NewTimingWheel creates a timing wheel with the given granularity (e.g. time.Second).
func NewTimingWheel(granularity time.Duration) *TimingWheel {
	if granularity <= 0 {
		granularity = time.Second
	}
	return &TimingWheel{slots: make(map[int64]*Slot), granularity: granularity}
}

// Add adds a key that expires at expires.
func (tw *TimingWheel) Add(key string, expires time.Time) {
	slotID := expires.Unix()
	if tw.granularity > time.Second {
		slotID = expires.Unix() / int64(tw.granularity/time.Second)
	}
	tw.mu.Lock()
	s, ok := tw.slots[slotID]
	if !ok {
		s = newSlot()
		tw.slots[slotID] = s
	}
	s.mu.Lock()
	tw.mu.Unlock()
	if e, ok := s.keys[key]; ok {
		s.order.Remove(e)
	}
	e := s.order.PushBack(&entry{key: key, expires: expires})
	s.keys[key] = e
	s.mu.Unlock()
}

// Remove removes a key from the wheel.
func (tw *TimingWheel) Remove(key string) {
	tw.mu.RLock()
	slots := make([]*Slot, 0, len(tw.slots))
	for _, s := range tw.slots {
		slots = append(slots, s)
	}
	tw.mu.RUnlock()
	for _, s := range slots {
		s.mu.Lock()
		if e, ok := s.keys[key]; ok {
			s.order.Remove(e)
			delete(s.keys, key)
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()
	}
}

// Expired returns keys that have expired (before now) and removes them from the wheel.
func (tw *TimingWheel) Expired(now time.Time) []string {
	var expired []string
	slotID := now.Unix()
	if tw.granularity > time.Second {
		slotID = now.Unix() / int64(tw.granularity/time.Second)
	}
	tw.mu.Lock()
	toRemove := make([]int64, 0)
	for id, s := range tw.slots {
		if id > slotID {
			continue
		}
		s.mu.Lock()
		for e := s.order.Front(); e != nil; {
			ent := e.Value.(*entry)
			if ent.expires.Before(now) || ent.expires.Equal(now) {
				expired = append(expired, ent.key)
				next := e.Next()
				s.order.Remove(e)
				delete(s.keys, ent.key)
				e = next
			} else {
				e = e.Next()
			}
		}
		if s.order.Len() == 0 {
			toRemove = append(toRemove, id)
		}
		s.mu.Unlock()
	}
	for _, id := range toRemove {
		delete(tw.slots, id)
	}
	tw.mu.Unlock()
	return expired
}

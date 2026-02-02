package data_structures

import (
	"encoding/binary"
	"sync"
)

// Set implements a Redis-like set: SADD, SREM, SMEMBERS, SISMEMBER, SCARD, SINTER, SUNION, SDIFF.
// Serialization: count(4) + sorted members as len(4)+data for deterministic encoding.
type Set struct {
	mu   sync.RWMutex
	data map[string]struct{}
}

// NewSet creates an empty set.
func NewSet() *Set {
	return &Set{data: make(map[string]struct{})}
}

// setEncode serializes set to bytes (sorted keys for determinism).
func setEncode(m map[string]struct{}) []byte {
	if len(m) == 0 {
		return binary.LittleEndian.AppendUint32(nil, 0)
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sortStrings(keys)
	buf := make([]byte, 0, 4+len(keys)*8)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(keys)))
	for _, k := range keys {
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(k)))
		buf = append(buf, k...)
	}
	return buf
}

func setDecode(b []byte) (map[string]struct{}, error) {
	if len(b) < 4 {
		return make(map[string]struct{}), nil
	}
	n := binary.LittleEndian.Uint32(b[0:4])
	out := make(map[string]struct{}, n)
	off := 4
	for i := uint32(0); i < n && off+4 <= len(b); i++ {
		el := binary.LittleEndian.Uint32(b[off : off+4])
		off += 4
		if off+int(el) > len(b) {
			break
		}
		out[string(b[off:off+int(el)])] = struct{}{}
		off += int(el)
	}
	return out, nil
}

func sortStrings(s []string) {
	// simple bubble sort for small sets
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j] < s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

// SAdd adds members; returns number of new members added.
func (s *Set) SAdd(members ...[]byte) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	added := 0
	for _, m := range members {
		k := string(m)
		if _, ok := s.data[k]; !ok {
			s.data[k] = struct{}{}
			added++
		}
	}
	return added
}

// SRem removes members; returns number removed.
func (s *Set) SRem(members ...[]byte) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	removed := 0
	for _, m := range members {
		k := string(m)
		if _, ok := s.data[k]; ok {
			delete(s.data, k)
			removed++
		}
	}
	return removed
}

// SIsMember returns true if member is in the set.
func (s *Set) SIsMember(member []byte) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.data[string(member)]
	return ok
}

// SCard returns the set size.
func (s *Set) SCard() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

// SMembers returns all members (copy).
func (s *Set) SMembers() [][]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([][]byte, 0, len(s.data))
	for k := range s.data {
		out = append(out, []byte(k))
	}
	return out
}

// SInter returns intersection of this set with others.
func (s *Set) SInter(others ...*Set) [][]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]struct{})
	for k := range s.data {
		all := true
		for _, o := range others {
			o.mu.RLock()
			_, ok := o.data[k]
			o.mu.RUnlock()
			if !ok {
				all = false
				break
			}
		}
		if all {
			result[k] = struct{}{}
		}
	}
	out := make([][]byte, 0, len(result))
	for k := range result {
		out = append(out, []byte(k))
	}
	return out
}

// SUnion returns union of this set with others.
func (s *Set) SUnion(others ...*Set) [][]byte {
	s.mu.RLock()
	result := make(map[string]struct{})
	for k := range s.data {
		result[k] = struct{}{}
	}
	s.mu.RUnlock()
	for _, o := range others {
		o.mu.RLock()
		for k := range o.data {
			result[k] = struct{}{}
		}
		o.mu.RUnlock()
	}
	out := make([][]byte, 0, len(result))
	for k := range result {
		out = append(out, []byte(k))
	}
	return out
}

// SDiff returns members in this set but not in others.
func (s *Set) SDiff(others ...*Set) [][]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]struct{})
	for k := range s.data {
		result[k] = struct{}{}
	}
	for _, o := range others {
		o.mu.RLock()
		for k := range o.data {
			delete(result, k)
		}
		o.mu.RUnlock()
	}
	out := make([][]byte, 0, len(result))
	for k := range result {
		out = append(out, []byte(k))
	}
	return out
}

// LoadFromBytes deserializes set from bytes.
func (s *Set) LoadFromBytes(b []byte) error {
	m, err := setDecode(b)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = m
	return nil
}

// Bytes serializes the set for storage.
func (s *Set) Bytes() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return setEncode(s.data)
}

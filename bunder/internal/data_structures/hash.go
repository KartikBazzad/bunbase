package data_structures

import (
	"encoding/binary"
	"sync"
)

// Hash implements a Redis-like hash: HSET, HGET, HGETALL, HDEL, HEXISTS, HKEYS, HVALS, HLEN.
// Serialization: count(4) + sorted fields as keyLen(4)+key+valLen(4)+val.
type Hash struct {
	mu   sync.RWMutex
	data map[string][]byte
}

// NewHash creates an empty hash.
func NewHash() *Hash {
	return &Hash{data: make(map[string][]byte)}
}

func hashEncode(m map[string][]byte) []byte {
	if len(m) == 0 {
		return binary.LittleEndian.AppendUint32(nil, 0)
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	hashSortStrings(keys)
	buf := make([]byte, 0, 4)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(keys)))
	for _, k := range keys {
		v := m[k]
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(k)))
		buf = append(buf, k...)
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(v)))
		buf = append(buf, v...)
	}
	return buf
}

func hashSortStrings(s []string) {
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j] < s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

func hashDecode(b []byte) (map[string][]byte, error) {
	if len(b) < 4 {
		return make(map[string][]byte), nil
	}
	n := binary.LittleEndian.Uint32(b[0:4])
	out := make(map[string][]byte, n)
	off := 4
	for i := uint32(0); i < n && off+8 <= len(b); i++ {
		kl := binary.LittleEndian.Uint32(b[off : off+4])
		off += 4
		if off+int(kl)+4 > len(b) {
			break
		}
		key := string(b[off : off+int(kl)])
		off += int(kl)
		vl := binary.LittleEndian.Uint32(b[off : off+4])
		off += 4
		if off+int(vl) > len(b) {
			break
		}
		out[key] = append([]byte(nil), b[off:off+int(vl)]...)
		off += int(vl)
	}
	return out, nil
}

// HSet sets field to value; returns 1 if new field, 0 if updated.
func (h *Hash) HSet(field, value []byte) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	k := string(field)
	if _, ok := h.data[k]; ok {
		h.data[k] = append([]byte(nil), value...)
		return 0
	}
	h.data[k] = append([]byte(nil), value...)
	return 1
}

// HGet returns the value for field, or nil.
func (h *Hash) HGet(field []byte) []byte {
	h.mu.RLock()
	defer h.mu.RUnlock()
	v, ok := h.data[string(field)]
	if !ok {
		return nil
	}
	return append([]byte(nil), v...)
}

// HDel removes fields; returns number removed.
func (h *Hash) HDel(fields ...[]byte) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	removed := 0
	for _, f := range fields {
		k := string(f)
		if _, ok := h.data[k]; ok {
			delete(h.data, k)
			removed++
		}
	}
	return removed
}

// HExists returns true if field exists.
func (h *Hash) HExists(field []byte) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.data[string(field)]
	return ok
}

// HLen returns the number of fields.
func (h *Hash) HLen() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.data)
}

// HKeys returns all field names.
func (h *Hash) HKeys() [][]byte {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([][]byte, 0, len(h.data))
	for k := range h.data {
		out = append(out, []byte(k))
	}
	return out
}

// HVals returns all values.
func (h *Hash) HVals() [][]byte {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([][]byte, 0, len(h.data))
	for _, v := range h.data {
		out = append(out, append([]byte(nil), v...))
	}
	return out
}

// HGetAll returns all field-value pairs (field, value, field, value, ...).
func (h *Hash) HGetAll() [][]byte {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([][]byte, 0, len(h.data)*2)
	for k, v := range h.data {
		out = append(out, []byte(k), append([]byte(nil), v...))
	}
	return out
}

// LoadFromBytes deserializes hash from bytes.
func (h *Hash) LoadFromBytes(b []byte) error {
	m, err := hashDecode(b)
	if err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.data = m
	return nil
}

// Bytes serializes the hash for storage.
func (h *Hash) Bytes() []byte {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return hashEncode(h.data)
}

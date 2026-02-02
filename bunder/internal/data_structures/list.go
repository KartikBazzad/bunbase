package data_structures

import (
	"encoding/binary"
	"sync"
)

// List implements a Redis-like list: LPUSH, RPUSH, LPOP, RPOP, LRANGE, LLEN, LINDEX, LSET, LTRIM.
// Serialization: count(4) + for each element len(4)+data. Bytes/LoadFromBytes for persistence.
type List struct {
	mu   sync.RWMutex
	data [][]byte
}

// NewList creates an empty list.
func NewList() *List {
	return &List{data: nil}
}

// listCodec serializes/deserializes list to/from bytes.
func listEncode(elems [][]byte) []byte {
	if len(elems) == 0 {
		return binary.LittleEndian.AppendUint32(nil, 0)
	}
	buf := make([]byte, 0, 4+len(elems)*8)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(elems)))
	for _, e := range elems {
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(e)))
		buf = append(buf, e...)
	}
	return buf
}

func listDecode(b []byte) ([][]byte, error) {
	if len(b) < 4 {
		return nil, nil
	}
	n := binary.LittleEndian.Uint32(b[0:4])
	if n == 0 {
		return [][]byte{}, nil
	}
	elems := make([][]byte, 0, n)
	off := 4
	for i := uint32(0); i < n && off+4 <= len(b); i++ {
		el := binary.LittleEndian.Uint32(b[off : off+4])
		off += 4
		if off+int(el) > len(b) {
			break
		}
		elems = append(elems, append([]byte(nil), b[off:off+int(el)]...))
		off += int(el)
	}
	return elems, nil
}

// LPush prepends values (last argument ends up at head, Redis semantics); returns new length.
func (l *List) LPush(values ...[]byte) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, v := range values {
		l.data = append([][]byte{v}, l.data...)
	}
	return len(l.data)
}

// RPush appends values; returns new length.
func (l *List) RPush(values ...[]byte) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.data = append(l.data, values...)
	return len(l.data)
}

// LPop removes and returns the first element, or nil if empty.
func (l *List) LPop() []byte {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.data) == 0 {
		return nil
	}
	v := l.data[0]
	l.data = l.data[1:]
	return v
}

// RPop removes and returns the last element, or nil if empty.
func (l *List) RPop() []byte {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.data) == 0 {
		return nil
	}
	v := l.data[len(l.data)-1]
	l.data = l.data[:len(l.data)-1]
	return v
}

// LLen returns the number of elements.
func (l *List) LLen() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.data)
}

// LIndex returns the element at index (0-based); negative index from end. Nil if out of range.
func (l *List) LIndex(index int) []byte {
	l.mu.RLock()
	defer l.mu.RUnlock()
	n := len(l.data)
	if n == 0 {
		return nil
	}
	if index < 0 {
		index = n + index
	}
	if index < 0 || index >= n {
		return nil
	}
	return append([]byte(nil), l.data[index]...)
}

// LSet sets the element at index; returns false if out of range.
func (l *List) LSet(index int, value []byte) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := len(l.data)
	if index < 0 {
		index = n + index
	}
	if index < 0 || index >= n {
		return false
	}
	l.data[index] = append([]byte(nil), value...)
	return true
}

// LRange returns elements from start to stop (inclusive). Negative indices from end.
func (l *List) LRange(start, stop int) [][]byte {
	l.mu.RLock()
	defer l.mu.RUnlock()
	n := len(l.data)
	if n == 0 {
		return nil
	}
	if start < 0 {
		start = n + start
	}
	if stop < 0 {
		stop = n + stop
	}
	if start > stop || start >= n {
		return [][]byte{}
	}
	if start < 0 {
		start = 0
	}
	if stop >= n {
		stop = n - 1
	}
	out := make([][]byte, 0, stop-start+1)
	for i := start; i <= stop; i++ {
		out = append(out, append([]byte(nil), l.data[i]...))
	}
	return out
}

// LTrim keeps only elements in [start, stop]; discards the rest.
func (l *List) LTrim(start, stop int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := len(l.data)
	if n == 0 {
		return
	}
	if start < 0 {
		start = n + start
	}
	if stop < 0 {
		stop = n + stop
	}
	if start > stop || start >= n {
		l.data = nil
		return
	}
	if start < 0 {
		start = 0
	}
	if stop >= n {
		stop = n - 1
	}
	l.data = append([][]byte(nil), l.data[start:stop+1]...)
}

// LoadFromBytes deserializes a list from bytes (e.g. from KV store).
func (l *List) LoadFromBytes(b []byte) error {
	elems, err := listDecode(b)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.data = elems
	return nil
}

// Bytes serializes the list to bytes for storage.
func (l *List) Bytes() []byte {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return listEncode(l.data)
}

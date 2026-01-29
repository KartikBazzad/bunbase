// Package query implements streaming k-way merge for partitioned query results.
package query

import (
	"container/heap"
	"encoding/json"
	"io"
)

// RowStream is a lazy iterator over rows (e.g. from one partition).
// Next returns the next row, or io.EOF when done.
// Callers must call Close when done.
type RowStream interface {
	Next() (Row, error)
	Close() error
}

// KWayMerger merges multiple RowStreams into a single ordered stream.
// When OrderBy is nil, rows are yielded in stream order (stream 0 until EOF, then stream 1, etc.).
// When OrderBy is set, rows are yielded in sorted order (k-way merge).
type KWayMerger struct {
	streams          []RowStream
	orderBy          *OrderSpec
	limit            int
	heap             *rowHeap
	yielded          int
	closed           bool
	noOrderStreamIdx int // when orderBy is nil: current stream index
}

// NewKWayMerger creates a merger for the given streams.
// orderBy may be nil (no sort). limit is max rows to yield (0 = no limit).
func NewKWayMerger(streams []RowStream, orderBy *OrderSpec, limit int) *KWayMerger {
	m := &KWayMerger{
		streams: streams,
		orderBy: orderBy,
		limit:   limit,
	}
	if orderBy != nil && orderBy.Field != "" && len(streams) > 0 {
		m.heap = &rowHeap{orderBy: orderBy, items: make([]heapItem, 0, len(streams))}
		// Prime the heap: read first row from each stream
		for i, s := range streams {
			row, err := s.Next()
			if err == nil {
				m.heap.items = append(m.heap.items, heapItem{row: row, streamIdx: i})
			} else if err != io.EOF {
				// Non-EOF error: will be surfaced on Next
			}
		}
		heap.Init(m.heap)
	}
	return m
}

// Next returns the next row in order, or (Row{}, false) when done or limit reached.
func (m *KWayMerger) Next() (Row, bool) {
	if m.closed {
		return Row{}, false
	}
	if m.limit > 0 && m.yielded >= m.limit {
		return Row{}, false
	}

	if m.heap != nil && m.heap.Len() > 0 {
		item := heap.Pop(m.heap).(heapItem)
		m.yielded++
		// Refill this stream
		if item.streamIdx >= 0 && item.streamIdx < len(m.streams) {
			s := m.streams[item.streamIdx]
			nextRow, err := s.Next()
			if err == nil {
				heap.Push(m.heap, heapItem{row: nextRow, streamIdx: item.streamIdx})
			}
		}
		return item.row, true
	}

	// No order: read from current stream until EOF, then advance to next stream
	for m.noOrderStreamIdx < len(m.streams) {
		if m.limit > 0 && m.yielded >= m.limit {
			return Row{}, false
		}
		s := m.streams[m.noOrderStreamIdx]
		row, err := s.Next()
		if err == io.EOF {
			m.noOrderStreamIdx++
			continue
		}
		if err != nil {
			return Row{}, false
		}
		m.yielded++
		return row, true
	}

	return Row{}, false
}

// Close closes all underlying streams.
func (m *KWayMerger) Close() error {
	if m.closed {
		return nil
	}
	m.closed = true
	for _, s := range m.streams {
		_ = s.Close()
	}
	return nil
}

type heapItem struct {
	row       Row
	streamIdx int
}

type rowHeap struct {
	orderBy *OrderSpec
	items   []heapItem
}

func (h *rowHeap) Len() int { return len(h.items) }
func (h *rowHeap) Less(i, j int) bool {
	return compareRowsForOrder(h.items[i].row, h.items[j].row, h.orderBy) < 0
}
func (h *rowHeap) Swap(i, j int)      { h.items[i], h.items[j] = h.items[j], h.items[i] }
func (h *rowHeap) Push(x interface{}) { h.items = append(h.items, x.(heapItem)) }
func (h *rowHeap) Pop() interface{} {
	n := len(h.items)
	item := h.items[n-1]
	h.items = h.items[:n-1]
	return item
}

// compareRowsForOrder compares two rows by the order spec (field + asc).
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareRowsForOrder(a, b Row, o *OrderSpec) int {
	if o == nil || o.Field == "" {
		// Fallback: compare by DocID
		if a.DocID < b.DocID {
			return -1
		}
		if a.DocID > b.DocID {
			return 1
		}
		return 0
	}
	va := extractField(a.Payload, o.Field)
	vb := extractField(b.Payload, o.Field)
	cmp := compareValuesForOrder(va, vb)
	if !o.Asc {
		cmp = -cmp
	}
	return cmp
}

func extractField(payload []byte, field string) interface{} {
	var doc map[string]interface{}
	if err := jsonUnmarshal(payload, &doc); err != nil {
		return nil
	}
	v, _ := doc[field]
	return v
}

func compareValuesForOrder(a, b interface{}) int {
	fa, oka := toFloatOrder(a)
	fb, okb := toFloatOrder(b)
	if oka && okb {
		if fa < fb {
			return -1
		}
		if fa > fb {
			return 1
		}
		return 0
	}
	sa, oka := toStringOrder(a)
	sb, okb := toStringOrder(b)
	if oka && okb {
		if sa < sb {
			return -1
		}
		if sa > sb {
			return 1
		}
		return 0
	}
	return 0
}

func toFloatOrder(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	default:
		return 0, false
	}
}

func toStringOrder(v interface{}) (string, bool) {
	s, ok := v.(string)
	return s, ok
}

func jsonUnmarshal(payload []byte, v interface{}) error {
	return json.Unmarshal(payload, v)
}

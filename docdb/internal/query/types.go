// Package query implements server-side query engine for v0.4.
package query

// Query represents a server-side query (v0.4).
type Query struct {
	Filter     Expression // Optional predicate
	Projection []Field    // Fields to return (empty = all)
	Limit      int        // Max rows (0 = no limit)
	OrderBy    *OrderSpec // Optional sort
}

// Expression is a simple predicate (field op value).
type Expression struct {
	Field string
	Op    string // "eq", "neq", "gt", "gte", "lt", "lte"
	Value interface{}
}

// Field names a projected field.
type Field struct {
	Name string
}

// OrderSpec specifies sort order.
type OrderSpec struct {
	Field string
	Asc   bool
}

// Row is a single result row (docID + payload).
type Row struct {
	DocID   uint64
	Payload []byte
}

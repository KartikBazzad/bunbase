// Package query implements server-side query engine for v0.4.
package query

import "context"

// Engine executes queries with partition fan-out and merge.
// The actual execution is invoked from docdb.LogicalDB which holds the engine.
type Engine struct{}

// NewEngine creates a new query engine.
func NewEngine() *Engine {
	return &Engine{}
}

// ExecuteContext is the signature for execution; implementation lives in docdb to avoid import cycles.
// Snapshot → partition fan-out → parallel execution → stream merge.
// v0.4: No ORDER → concat; ORDER BY → k-way merge; LIMIT → early termination; FILTER → apply per row.
type ExecuteContext func(ctx context.Context, q Query) ([]Row, error)

// Package docdb implements task-based execution for v0.4.
//
// Tasks are submitted to a LogicalDB's worker pool and routed to partitions.
package docdb

import (
	"github.com/kartikbazzad/docdb/internal/types"
)

// Task represents a database operation to be executed on a partition.
//
// Tasks are bound to a partition by PartitionID. Workers pull tasks,
// lock the partition, execute, unlock, and send results.
type Task struct {
	PartitionID int                    // Partition to execute on
	Op          types.OperationType    // Operation type (Create, Read, Update, Delete, Patch)
	Collection  string                 // Collection name
	DocID       uint64                 // Document ID
	Payload     []byte                 // Payload (for Create, Update)
	PatchOps    []types.PatchOperation // Patch operations (for Patch)
	ResultCh    chan *Result           // Channel for result
}

// Result represents the result of a task execution.
type Result struct {
	Status types.Status // Operation status
	Data   []byte       // Response data (for Read)
	Error  error        // Error if operation failed
}

// NewTask creates a new task.
func NewTask(partitionID int, op types.OperationType, collection string, docID uint64) *Task {
	return &Task{
		PartitionID: partitionID,
		Op:          op,
		Collection:  collection,
		DocID:       docID,
		ResultCh:    make(chan *Result, 1),
	}
}

// NewTaskWithPayload creates a new task with payload.
func NewTaskWithPayload(partitionID int, op types.OperationType, collection string, docID uint64, payload []byte) *Task {
	t := NewTask(partitionID, op, collection, docID)
	t.Payload = payload
	return t
}

// NewTaskWithPatch creates a new task with patch operations.
func NewTaskWithPatch(partitionID int, collection string, docID uint64, patchOps []types.PatchOperation) *Task {
	t := NewTask(partitionID, types.OpPatch, collection, docID)
	t.PatchOps = patchOps
	return t
}

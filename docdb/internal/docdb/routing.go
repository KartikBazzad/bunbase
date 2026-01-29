// Package docdb implements routing logic for v0.4 partitions.
package docdb

// RouteToPartition computes which partition a document belongs to.
//
// Routing formula: partitionID = Hash(docID) % partitionCount
//
// Properties:
//   - Deterministic: same docID always routes to same partition
//   - Stable: partition assignment doesn't change unless partitionCount changes
//   - Versioned: hash function change = breaking change
func RouteToPartition(docID uint64, partitionCount int) int {
	if partitionCount <= 0 {
		return 0
	}
	return int(docID % uint64(partitionCount))
}

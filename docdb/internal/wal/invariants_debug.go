//go:build debug

package wal

import "fmt"

// checkLSNMonotonic verifies LSN strictly increases per partition.
// Panics if newLSN != prevLSN+1.
func checkLSNMonotonic(prevLSN, newLSN uint64) {
	if newLSN != prevLSN+1 {
		panic(fmt.Sprintf("wal invariant: LSN not monotonic prev=%d new=%d", prevLSN, newLSN))
	}
}

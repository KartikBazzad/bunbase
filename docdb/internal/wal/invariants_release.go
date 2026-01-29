//go:build !debug

package wal

func checkLSNMonotonic(prevLSN, newLSN uint64) {
	_ = prevLSN
	_ = newLSN
}

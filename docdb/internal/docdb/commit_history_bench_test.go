package docdb

import (
	"testing"
)

// BenchmarkCommitsAfter measures performance of the binary search optimization
func BenchmarkCommitsAfter(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}

	for _, size := range sizes {
		h := NewCommitHistory(size + 1000)

		// Populate commit history with ordered records
		for i := 0; i < size; i++ {
			readSet := make(map[string]struct{})
			writeSet := make(map[string]struct{})
			readSet[docKey("test", uint64(i))] = struct{}{}
			writeSet[docKey("test", uint64(i+1))] = struct{}{}
			h.Append(uint64(i+1), readSet, writeSet)
		}

		b.Run("size_"+string(rune(size)), func(b *testing.B) {
			// Test worst case: looking for very early txID (returns most records)
			snapshotTxID := uint64(10)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = h.CommitsAfter(snapshotTxID)
			}
		})

		b.Run("size_"+string(rune(size))+"_recent", func(b *testing.B) {
			// Test best case: looking for recent txID (returns few records)
			snapshotTxID := uint64(size - 10)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = h.CommitsAfter(snapshotTxID)
			}
		})

		b.Run("size_"+string(rune(size))+"_mid", func(b *testing.B) {
			// Test middle case
			snapshotTxID := uint64(size / 2)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = h.CommitsAfter(snapshotTxID)
			}
		})
	}
}

// BenchmarkCommitsAfterContention measures performance under lock contention
func BenchmarkCommitsAfterContention(b *testing.B) {
	h := NewCommitHistory(100000)

	// Populate with 50k commits
	for i := 0; i < 50000; i++ {
		readSet := make(map[string]struct{})
		writeSet := make(map[string]struct{})
		readSet[docKey("test", uint64(i))] = struct{}{}
		writeSet[docKey("test", uint64(i+1))] = struct{}{}
		h.Append(uint64(i+1), readSet, writeSet)
	}

	b.RunParallel(func(pb *testing.PB) {
		snapshotTxID := uint64(1000)
		for pb.Next() {
			_ = h.CommitsAfter(snapshotTxID)
		}
	})
}

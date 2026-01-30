package docdb

import (
	"strconv"
	"sync"
)

const defaultCommitHistoryMaxSize = 100_000

// maxConflictWindow limits how far back we check for conflicts (SSI-lite).
// Only the last N commits are checked for read-write conflicts.
// This bounds the critical section time under commitMu.
const maxConflictWindow = 1000

// docKey returns a stable key for (collection, docID) for read/write set storage.
func docKey(collection string, docID uint64) string {
	if collection == "" {
		collection = DefaultCollection
	}
	return collection + ":" + strconv.FormatUint(docID, 10)
}

// commitRecord holds read and write sets for a committed transaction (for SSI conflict detection).
type commitRecord struct {
	txID     uint64
	readSet  map[string]struct{}
	writeSet map[string]struct{}
}

// CommitHistory stores recent commit records for SSI-lite conflict detection.
// Bounded size; oldest records are dropped when over capacity.
type CommitHistory struct {
	mu      sync.Mutex
	records []commitRecord
	maxSize int
}

// NewCommitHistory creates a commit history with the given max size (number of commit records).
// If maxSize <= 0, defaultCommitHistoryMaxSize is used.
func NewCommitHistory(maxSize int) *CommitHistory {
	if maxSize <= 0 {
		maxSize = defaultCommitHistoryMaxSize
	}
	return &CommitHistory{
		records: make([]commitRecord, 0, maxSize+1),
		maxSize: maxSize,
	}
}

// CommitsAfter returns copies of commit records for transactions that committed after snapshotTxID
// (i.e. txID > snapshotTxID). Caller must not modify the returned maps.
// Only checks the last maxConflictWindow commits to bound critical section time.
func (h *CommitHistory) CommitsAfter(snapshotTxID uint64) []commitRecord {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Start from max(0, len-maxConflictWindow) to cap the scan window
	startIdx := 0
	if len(h.records) > maxConflictWindow {
		startIdx = len(h.records) - maxConflictWindow
	}

	var out []commitRecord
	for i := startIdx; i < len(h.records); i++ {
		if h.records[i].txID > snapshotTxID {
			out = append(out, h.records[i])
		}
	}
	return out
}

// Append records a committed transaction's read and write sets. readSet and writeSet are copied.
func (h *CommitHistory) Append(txID uint64, readSet, writeSet map[string]struct{}) {
	r := commitRecord{
		txID:     txID,
		readSet:  make(map[string]struct{}),
		writeSet: make(map[string]struct{}),
	}
	for k := range readSet {
		r.readSet[k] = struct{}{}
	}
	for k := range writeSet {
		r.writeSet[k] = struct{}{}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r)
	for len(h.records) > h.maxSize {
		h.records = h.records[1:]
	}
}

// hasConflict returns true if (ourRead ∩ theirWrite) ∪ (ourWrite ∩ theirRead) is non-empty.
func hasConflict(ourRead, ourWrite, theirRead, theirWrite map[string]struct{}) bool {
	for k := range ourRead {
		if _, ok := theirWrite[k]; ok {
			return true
		}
	}
	for k := range ourWrite {
		if _, ok := theirRead[k]; ok {
			return true
		}
	}
	return false
}

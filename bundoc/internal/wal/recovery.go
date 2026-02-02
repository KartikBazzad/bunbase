package wal

import (
	"fmt"

	"github.com/kartikbazzad/bunbase/bundoc/internal/util"
)

// Recovery handles WAL recovery after a crash
type Recovery struct {
	wal *WAL
}

// NewRecovery creates a new recovery instance
func NewRecovery(wal *WAL) *Recovery {
	return &Recovery{wal: wal}
}

// Recover reads all WAL records and returns them for replay
func (r *Recovery) Recover() ([]*Record, error) {
	// Read all records from WAL
	records, err := r.wal.ReadAllRecords()
	if err != nil {
		return nil, fmt.Errorf("recovery failed: %w", err)
	}

	// Filter and validate records
	validRecords := r.filterValidRecords(records)

	return validRecords, nil
}

// filterValidRecords filters out invalid or incomplete transactions
func (r *Recovery) filterValidRecords(records []*Record) []*Record {
	// Build transaction map to track committed transactions
	committedTxns := make(map[uint64]bool)

	// First pass: identify committed transactions
	for _, record := range records {
		if record.Type == RecordTypeCommit {
			committedTxns[record.TxnID] = true
		} else if record.Type == RecordTypeAbort {
			committedTxns[record.TxnID] = false
		}
	}

	// Second pass: collect records from committed transactions
	var validRecords []*Record
	for _, record := range records {
		// Skip commit/abort markers (not data records)
		if record.Type == RecordTypeCommit || record.Type == RecordTypeAbort {
			continue
		}

		// Only include records from committed transactions
		if committed, exists := committedTxns[record.TxnID]; exists && committed {
			validRecords = append(validRecords, record)
		}
	}

	return validRecords
}

// RecoverToLSN recovers up to a specific LSN
func (r *Recovery) RecoverToLSN(targetLSN LSN) ([]*Record, error) {
	allRecords, err := r.Recover()
	if err != nil {
		return nil, err
	}

	// Filter records up to target LSN
	var records []*Record
	for _, record := range allRecords {
		if record.LSN <= targetLSN {
			records = append(records, record)
		}
	}

	return records, nil
}

// VerifyIntegrity checks WAL integrity
func (r *Recovery) VerifyIntegrity() error {
	records, err := r.wal.ReadAllRecords()
	if err != nil {
		return fmt.Errorf("%w: %v", util.ErrWALCorrupt, err)
	}

	// Check LSN monotonicity
	var prevLSN LSN
	for i, record := range records {
		if record.LSN <= prevLSN {
			return fmt.Errorf("%w: LSN not monotonic at record %d (prev=%d, current=%d)",
				util.ErrWALCorrupt, i, prevLSN, record.LSN)
		}
		prevLSN = record.LSN
	}

	return nil
}

// GetLastCommittedLSN returns the LSN of the last committed transaction
func (r *Recovery) GetLastCommittedLSN() (LSN, error) {
	records, err := r.wal.ReadAllRecords()
	if err != nil {
		return 0, err
	}

	var lastLSN LSN
	for _, record := range records {
		if record.Type == RecordTypeCommit && record.LSN > lastLSN {
			lastLSN = record.LSN
		}
	}

	return lastLSN, nil
}

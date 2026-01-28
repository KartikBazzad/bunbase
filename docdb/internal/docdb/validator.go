package docdb

import (
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/types"
)

// DocumentHealth represents the health status of a document.
type DocumentHealth int

const (
	HealthUnknown DocumentHealth = iota
	HealthValid
	HealthCorrupt
	HealthMissing
)

// Validator validates document integrity and detects corruption.
type Validator struct {
	db     *LogicalDB
	logger *logger.Logger
}

// NewValidator creates a new document validator.
func NewValidator(db *LogicalDB, log *logger.Logger) *Validator {
	return &Validator{
		db:     db,
		logger: log,
	}
}

// ValidateDocument checks the health of a document by reading it
// and verifying CRC32 checksum.
func (v *Validator) ValidateDocument(docID uint64) (DocumentHealth, error) {
	v.db.mu.RLock()
	defer v.db.mu.RUnlock()

	if v.db.closed {
		return HealthUnknown, ErrDBNotOpen
	}

	version, exists := v.db.index.Get(docID, v.db.mvcc.CurrentSnapshot())
	if !exists {
		return HealthMissing, nil
	}

	if version.DeletedTxID != nil {
		return HealthMissing, nil
	}

	// Try to read the document - this will verify CRC32
	_, err := v.db.dataFile.Read(version.Offset, version.Length)
	if err != nil {
		v.logger.Warn("Document %d validation failed: %v", docID, err)
		return HealthCorrupt, err
	}

	return HealthValid, nil
}

// ValidateAllDocuments validates all documents in the database.
func (v *Validator) ValidateAllDocuments() (map[uint64]DocumentHealth, error) {
	v.db.mu.RLock()
	defer v.db.mu.RUnlock()

	if v.db.closed {
		return nil, ErrDBNotOpen
	}

	results := make(map[uint64]DocumentHealth)

	v.db.index.ForEach(func(docID uint64, version *types.DocumentVersion) {
		if version.DeletedTxID != nil {
			return // Skip deleted documents
		}

		_, err := v.db.dataFile.Read(version.Offset, version.Length)
		if err != nil {
			results[docID] = HealthCorrupt
		} else {
			results[docID] = HealthValid
		}
	})

	return results, nil
}

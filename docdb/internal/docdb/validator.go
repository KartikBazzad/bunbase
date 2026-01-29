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
// and verifying CRC32 checksum. Uses the partition that owns the document.
func (v *Validator) ValidateDocument(collection string, docID uint64) (DocumentHealth, error) {
	v.db.mu.RLock()
	defer v.db.mu.RUnlock()

	if v.db.closed {
		return HealthUnknown, ErrDBNotOpen
	}
	if v.db.partitions == nil {
		return HealthUnknown, ErrDBNotOpen
	}

	if collection == "" {
		collection = DefaultCollection
	}

	partitionID := RouteToPartition(docID, v.db.PartitionCount())
	if partitionID >= len(v.db.partitions) {
		return HealthUnknown, nil
	}
	partition := v.db.partitions[partitionID]
	index := partition.GetIndex()
	dataFile := partition.GetDataFile()
	if dataFile == nil {
		return HealthUnknown, nil
	}

	version, exists := index.Get(collection, docID, v.db.mvcc.CurrentSnapshot())
	if !exists {
		return HealthMissing, nil
	}
	if version.DeletedTxID != nil {
		return HealthMissing, nil
	}

	_, err := dataFile.Read(version.Offset, version.Length)
	if err != nil {
		v.logger.Warn("Document %d in collection %s validation failed: %v", docID, collection, err)
		return HealthCorrupt, err
	}
	return HealthValid, nil
}

// ValidateAllDocuments validates all documents in the database across all partitions.
func (v *Validator) ValidateAllDocuments() (map[string]map[uint64]DocumentHealth, error) {
	v.db.mu.RLock()
	defer v.db.mu.RUnlock()

	if v.db.closed {
		return nil, ErrDBNotOpen
	}
	if v.db.partitions == nil {
		return nil, ErrDBNotOpen
	}

	results := make(map[string]map[uint64]DocumentHealth)

	for _, partition := range v.db.partitions {
		dataFile := partition.GetDataFile()
		if dataFile == nil {
			continue
		}
		index := partition.GetIndex()
		index.ForEachCollection(func(collection string, ci *CollectionIndex) {
			if results[collection] == nil {
				results[collection] = make(map[uint64]DocumentHealth)
			}
			ci.ForEach(func(docID uint64, version *types.DocumentVersion) {
				if version.DeletedTxID != nil {
					return
				}
				_, err := dataFile.Read(version.Offset, version.Length)
				if err != nil {
					results[collection][docID] = HealthCorrupt
				} else {
					results[collection][docID] = HealthValid
				}
			})
		})
	}

	return results, nil
}

package docdb

import (
	"fmt"
	"path/filepath"

	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/types"
	"github.com/kartikbazzad/docdb/internal/wal"
)

// Healer repairs corrupted documents by finding the latest valid version from WAL.
type Healer struct {
	db     *LogicalDB
	logger *logger.Logger
}

// NewHealer creates a new document healer.
func NewHealer(db *LogicalDB, log *logger.Logger) *Healer {
	return &Healer{
		db:     db,
		logger: log,
	}
}

// HealDocument attempts to heal a corrupted document by finding the latest
// valid version from the partition's WAL records.
func (h *Healer) HealDocument(collection string, docID uint64) error {
	h.db.mu.Lock()
	defer h.db.mu.Unlock()

	if h.db.closed {
		return ErrDBNotOpen
	}
	if h.db.partitions == nil {
		return ErrDBNotOpen
	}

	if collection == "" {
		collection = DefaultCollection
	}

	partitionID := RouteToPartition(docID, h.db.PartitionCount())
	if partitionID >= len(h.db.partitions) {
		return fmt.Errorf("invalid partition %d for doc %d", partitionID, docID)
	}
	partition := h.db.partitions[partitionID]
	walPath := filepath.Join(h.db.walDir, h.db.dbName, fmt.Sprintf("p%d.wal", partitionID))

	var latestValidVersion *types.WALRecord
	maxTxID := uint64(0)

	err := wal.ReplayPartitionWAL(walPath, h.db.logger, func(rec *types.WALRecord) error {
		if rec == nil || rec.DocID != docID {
			return nil
		}
		recCollection := rec.Collection
		if recCollection == "" {
			recCollection = DefaultCollection
		}
		if recCollection != collection {
			return nil
		}
		if rec.OpType == types.OpCreate || rec.OpType == types.OpUpdate || rec.OpType == types.OpPatch {
			if rec.TxID > maxTxID && len(rec.Payload) > 0 {
				maxTxID = rec.TxID
				latestValidVersion = rec
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan WAL for document %d: %w", docID, err)
	}
	if latestValidVersion == nil {
		return fmt.Errorf("no valid version found for document %d in collection %s", docID, collection)
	}

	dataFile := partition.GetDataFile()
	if dataFile == nil {
		return fmt.Errorf("partition %d data file not available", partitionID)
	}
	offset, err := dataFile.Write(latestValidVersion.Payload)
	if err != nil {
		return fmt.Errorf("failed to write healed payload: %w", err)
	}

	version := h.db.mvcc.CreateVersion(docID, latestValidVersion.TxID, offset, latestValidVersion.PayloadLen)
	partition.GetIndex().Set(collection, version)

	h.logger.Info("Healed document %d in collection %s using version from tx_id=%d", docID, collection, latestValidVersion.TxID)
	return nil
}

// HealAllCorruptedDocuments finds and heals all corrupted documents.
func (h *Healer) HealAllCorruptedDocuments() ([]uint64, error) {
	validator := NewValidator(h.db, h.db.logger)
	healthMap, err := validator.ValidateAllDocuments()
	if err != nil {
		return nil, err
	}

	healed := make([]uint64, 0)
	for collection, docs := range healthMap {
		for docID, health := range docs {
			if health == HealthCorrupt {
				if err := h.HealDocument(collection, docID); err != nil {
					h.logger.Warn("Failed to heal document %d in collection %s: %v", docID, collection, err)
					continue
				}
				healed = append(healed, docID)
			}
		}
	}

	return healed, nil
}

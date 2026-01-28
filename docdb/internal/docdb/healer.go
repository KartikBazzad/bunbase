package docdb

import (
	"fmt"

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
// valid version from WAL records.
func (h *Healer) HealDocument(docID uint64) error {
	h.db.mu.Lock()
	defer h.db.mu.Unlock()

	if h.db.closed {
		return ErrDBNotOpen
	}

	// Get all WAL records for this document
	walBasePath := fmt.Sprintf("%s/%s.wal", h.db.walDir, h.db.dbName)
	recovery := wal.NewRecovery(walBasePath, h.db.logger)

	var latestValidVersion *types.WALRecord
	maxTxID := uint64(0)

	// Scan WAL for all versions of this document
	err := recovery.Replay(func(rec *types.WALRecord) error {
		if rec == nil || rec.DocID != docID {
			return nil
		}

		// Only consider committed transactions
		// (We'd need to track commit markers, but for simplicity,
		// we'll check if we can read the payload)
		if rec.OpType == types.OpCreate || rec.OpType == types.OpUpdate {
			// Try to validate the payload by attempting to read it
			// For now, we'll use the latest record with valid payload
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
		return fmt.Errorf("no valid version found for document %d", docID)
	}

	// Write the valid payload to data file
	offset, err := h.db.dataFile.Write(latestValidVersion.Payload)
	if err != nil {
		return fmt.Errorf("failed to write healed payload: %w", err)
	}

	// Update index with healed version
	version := h.db.mvcc.CreateVersion(docID, latestValidVersion.TxID, offset, latestValidVersion.PayloadLen)
	h.db.index.Set(version)

	h.logger.Info("Healed document %d using version from tx_id=%d", docID, latestValidVersion.TxID)
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
	for docID, health := range healthMap {
		if health == HealthCorrupt {
			if err := h.HealDocument(docID); err != nil {
				h.logger.Warn("Failed to heal document %d: %v", docID, err)
				continue
			}
			healed = append(healed, docID)
		}
	}

	return healed, nil
}

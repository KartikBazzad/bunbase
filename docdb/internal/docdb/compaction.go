package docdb

import (
	"os"
	"time"

	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/types"
)

type Compactor struct {
	db     *LogicalDB
	logger *logger.Logger
}

func NewCompactor(db *LogicalDB, log *logger.Logger) *Compactor {
	return &Compactor{
		db:     db,
		logger: log,
	}
}

func (c *Compactor) ShouldCompact() bool {
	if c.db.dataFile == nil {
		return false
	}

	fileSize := c.db.dataFile.Size()
	sizeThreshold := uint64(c.db.cfg.DB.CompactionSizeThresholdMB * 1024 * 1024)

	if fileSize >= sizeThreshold {
		return true
	}

	tombstoneCount := 0
	totalCount := 0

	c.db.index.ForEach(func(docID uint64, version *types.DocumentVersion) {
		totalCount++
		if version.DeletedTxID != nil {
			tombstoneCount++
		}
	})

	if totalCount > 0 && float64(tombstoneCount)/float64(totalCount) > c.db.cfg.DB.CompactionTombstoneRatio {
		return true
	}

	return false
}

func (c *Compactor) Compact() error {
	c.logger.Info("Starting compaction for db: %s", c.db.Name())

	c.db.mu.Lock()
	defer c.db.mu.Unlock()

	if c.db.closed {
		return ErrDBNotOpen
	}

	compactPath := c.db.dataFile.path + ".compact"
	oldPath := c.db.dataFile.path

	compactFile := NewDataFile(compactPath, c.logger)
	if err := compactFile.Open(); err != nil {
		return err
	}
	defer compactFile.Close()

	newOffsets := make(map[uint64]uint64)

	c.db.index.ForEach(func(docID uint64, version *types.DocumentVersion) {
		if version.DeletedTxID == nil {
			payload, err := c.db.dataFile.Read(version.Offset, version.Length)
			if err != nil {
				c.logger.Error("Failed to read doc %d during compaction: %v", docID, err)
				return
			}

			newOffset, err := compactFile.Write(payload)
			if err != nil {
				c.logger.Error("Failed to write doc %d during compaction: %v", docID, err)
				return
			}

			newOffsets[docID] = newOffset
		}
	})

	if err := compactFile.Close(); err != nil {
		return err
	}

	for _, shard := range c.db.index.shards {
		shard.mu.Lock()
		for docID, version := range shard.data {
			if version.DeletedTxID == nil {
				if newOffset, exists := newOffsets[docID]; exists {
					version.Offset = newOffset
				}
			} else {
				delete(shard.data, docID)
			}
		}
		shard.mu.Unlock()
	}

	if err := c.db.dataFile.Close(); err != nil {
		c.logger.Error("Failed to close old data file: %v", err)
	}

	if err := os.Rename(compactPath, oldPath); err != nil {
		c.logger.Error("Failed to rename compacted file: %v", err)
		if err := os.Rename(oldPath, oldPath+".old"); err != nil {
			c.logger.Error("Failed to rename old file: %v", err)
		}
	}

	c.db.dataFile = NewDataFile(oldPath, c.logger)
	if err := c.db.dataFile.Open(); err != nil {
		return err
	}

	c.logger.Info("Compaction complete for db: %s", c.db.Name())

	return nil
}

func (c *Compactor) RunPeriodically(interval int) {
	if interval <= 0 {
		return
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if c.db.closed {
			return
		}

		if c.ShouldCompact() {
			if err := c.Compact(); err != nil {
				c.logger.Error("Compaction failed: %v", err)
			}
		}
	}
}

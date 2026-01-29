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
	c.db.mu.RLock()
	defer c.db.mu.RUnlock()

	if c.db.partitions == nil || c.db.cfg == nil {
		return false
	}

	sizeThreshold := uint64(c.db.cfg.DB.CompactionSizeThresholdMB * 1024 * 1024)
	tombstoneRatio := c.db.cfg.DB.CompactionTombstoneRatio

	for _, partition := range c.db.partitions {
		dataFile := partition.GetDataFile()
		if dataFile == nil {
			continue
		}
		fileSize := dataFile.Size()
		if sizeThreshold > 0 && fileSize >= sizeThreshold {
			return true
		}

		tombstoneCount := 0
		totalCount := 0
		partition.GetIndex().ForEachCollection(func(collection string, ci *CollectionIndex) {
			ci.ForEach(func(docID uint64, version *types.DocumentVersion) {
				totalCount++
				if version.DeletedTxID != nil {
					tombstoneCount++
				}
			})
		})
		if totalCount > 0 && tombstoneRatio > 0 && float64(tombstoneCount)/float64(totalCount) > tombstoneRatio {
			return true
		}
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
	if c.db.partitions == nil {
		return ErrDBNotOpen
	}

	for i, partition := range c.db.partitions {
		if err := c.compactPartition(partition, i); err != nil {
			c.logger.Error("Compaction failed for partition %d: %v", i, err)
			return err
		}
	}

	// Aggregate doc counts per collection across all partitions
	collectionCounts := make(map[string]uint64)
	for _, partition := range c.db.partitions {
		partition.GetIndex().ForEachCollection(func(collection string, ci *CollectionIndex) {
			collectionCounts[collection] += uint64(ci.LiveCount())
		})
	}
	for name, count := range collectionCounts {
		c.db.collections.SetDocCount(name, count)
	}

	c.db.lastCompaction = time.Now()
	c.logger.Info("Compaction complete for db: %s", c.db.Name())
	return nil
}

func (c *Compactor) compactPartition(partition *Partition, partitionID int) error {
	dataFile := partition.GetDataFile()
	if dataFile == nil {
		return nil
	}
	index := partition.GetIndex()

	compactPath := dataFile.path + ".compact"
	oldPath := dataFile.path

	compactFile := NewDataFile(compactPath, c.logger)
	if err := compactFile.Open(); err != nil {
		return err
	}
	defer compactFile.Close()

	newOffsets := make(map[string]map[uint64]uint64)

	index.ForEachCollection(func(collection string, ci *CollectionIndex) {
		newOffsets[collection] = make(map[uint64]uint64)
		ci.ForEach(func(docID uint64, version *types.DocumentVersion) {
			if version.DeletedTxID == nil {
				payload, err := dataFile.Read(version.Offset, version.Length)
				if err != nil {
					c.logger.Error("Failed to read doc %d in collection %s during compaction: %v", docID, collection, err)
					return
				}
				newOffset, err := compactFile.Write(payload)
				if err != nil {
					c.logger.Error("Failed to write doc %d in collection %s during compaction: %v", docID, collection, err)
					return
				}
				newOffsets[collection][docID] = newOffset
			}
		})
	})

	if err := compactFile.Close(); err != nil {
		return err
	}

	index.ForEachCollection(func(collection string, ci *CollectionIndex) {
		collectionOffsets := newOffsets[collection]
		if collectionOffsets == nil {
			return
		}
		ci.ForEach(func(docID uint64, version *types.DocumentVersion) {
			if version.DeletedTxID == nil {
				if newOffset, exists := collectionOffsets[docID]; exists {
					updatedVersion := *version
					updatedVersion.Offset = newOffset
					ci.Set(&updatedVersion)
				}
			}
		})
	})

	if err := dataFile.Close(); err != nil {
		c.logger.Error("Failed to close old data file: %v", err)
	}

	if err := os.Rename(compactPath, oldPath); err != nil {
		c.logger.Error("Failed to rename compacted file: %v", err)
		if err := os.Rename(oldPath, oldPath+".old"); err != nil {
			c.logger.Error("Failed to rename old file: %v", err)
		}
		return err
	}

	newDataFile := NewDataFile(oldPath, c.logger)
	if err := newDataFile.Open(); err != nil {
		return err
	}
	partition.SetDataFile(newDataFile)

	return nil
}

func (c *Compactor) RunPeriodically(interval int) {
	if interval <= 0 {
		return
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.db.mu.RLock()
		closed := c.db.closed
		c.db.mu.RUnlock()
		if closed {
			return
		}
		if c.ShouldCompact() {
			if err := c.Compact(); err != nil {
				c.logger.Error("Compaction failed: %v", err)
			}
		}
	}
}

package wal

import (
	"fmt"
	"io"
	"os"

	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/types"
)

type Recovery struct {
	basePath string
	logger   *logger.Logger
}

func NewRecovery(basePath string, log *logger.Logger) *Recovery {
	return &Recovery{
		basePath: basePath,
		logger:   log,
	}
}

type RecoveryHandler func(record *types.WALRecord) error

func (r *Recovery) Replay(handler RecoveryHandler) error {
	rotator := NewRotator(r.basePath, 0, false, r.logger)

	walPaths, err := rotator.GetAllWALPaths()
	if err != nil {
		return fmt.Errorf("failed to list WAL segments: %w", err)
	}

	if len(walPaths) == 0 {
		r.logger.Info("No WAL segments found")
		return nil
	}

	r.logger.Info("Found %d WAL segment(s) to replay", len(walPaths))

	totalRecords := 0
	var lastError error

	for i, walPath := range walPaths {
		r.logger.Info("Replaying WAL segment %d/%d: %s", i+1, len(walPaths), walPath)

		segRecords, err := r.replaySegment(walPath, handler, i == len(walPaths)-1)
		if err != nil {
			lastError = err
			r.logger.Error("Failed to replay segment %s: %v", walPath, err)
			if !r.isActiveWAL(walPath) {
				continue
			}
			break
		}

		totalRecords += segRecords
		r.logger.Info("Replayed %d records from segment %s", segRecords, walPath)
	}

	r.logger.Info("WAL replay complete: %d total records across %d segments", totalRecords, len(walPaths))
	return lastError
}

func (r *Recovery) replaySegment(walPath string, handler RecoveryHandler, truncateOnError bool) (int, error) {
	reader := NewReader(walPath, r.logger)

	if err := reader.Open(); err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	defer reader.Close()

	count := 0
	var lastError error

	for {
		record, err := reader.Next()
		if err != nil {
			lastError = err
			r.logger.Error("WAL record error at offset %d: %v", count, err)

			if truncateOnError {
				if err := r.truncateSegment(walPath, reader); err != nil {
					r.logger.Error("Failed to truncate WAL segment %s: %v", walPath, err)
				}
			}

			break
		}

		if record == nil {
			break
		}

		if handler != nil {
			if err := handler(record); err != nil {
				r.logger.Error("Handler error for record %d: %v", count, err)
				lastError = err
			}
		}

		count++
	}

	return count, lastError
}

func (r *Recovery) truncateSegment(walPath string, reader *Reader) error {
	if reader == nil || reader.file == nil {
		return nil
	}

	offset, err := reader.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}

	if err := reader.Close(); err != nil {
		return err
	}

	if err := os.Truncate(walPath, offset); err != nil {
		return err
	}

	r.logger.Info("WAL segment truncated to offset: %d", offset)
	return nil
}

func (r *Recovery) isActiveWAL(walPath string) bool {
	return walPath == r.basePath
}

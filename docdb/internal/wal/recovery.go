package wal

import (
	"io"
	"os"

	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/types"
)

type Recovery struct {
	path   string
	reader *Reader
	logger *logger.Logger
}

func NewRecovery(path string, log *logger.Logger) *Recovery {
	return &Recovery{
		path:   path,
		logger: log,
	}
}

type RecoveryHandler func(record *types.WALRecord) error

func (r *Recovery) Replay(handler RecoveryHandler) error {
	r.reader = NewReader(r.path, r.logger)

	if err := r.reader.Open(); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer r.reader.Close()

	count := 0
	var lastError error

	for {
		record, err := r.reader.Next()
		if err != nil {
			lastError = err
			r.logger.Error("WAL record error at offset %d: %v", count, err)
			if err := r.Truncate(); err != nil {
				r.logger.Error("Failed to truncate WAL: %v", err)
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

	r.logger.Info("WAL replay complete: %d records", count)
	return lastError
}

func (r *Recovery) Truncate() error {
	if r.reader == nil || r.reader.file == nil {
		return nil
	}

	offset, err := r.reader.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}

	if err := r.reader.Close(); err != nil {
		return err
	}

	if err := os.Truncate(r.path, offset); err != nil {
		return err
	}

	r.logger.Info("WAL truncated to offset: %d", offset)
	return nil
}

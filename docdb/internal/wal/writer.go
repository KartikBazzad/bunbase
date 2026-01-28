package wal

import (
	"os"
	"sync"

	"github.com/kartikbazzad/docdb/internal/errors"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/types"
)

// Writer appends WAL records for a single logical database.
//
// It is responsible for:
//   - Encoding records using the canonical on-disk format (see format.go)
//   - Optional fsync-after-write semantics
//   - Tracking current WAL size
//   - Cooperating with Rotator for multi-segment WALs
type Writer struct {
	mu            sync.Mutex
	file          *os.File
	path          string
	dbID          uint64
	size          uint64
	maxSize       uint64
	fsync         bool
	logger        *logger.Logger
	rotator       *Rotator
	isClosed      bool
	checkpointMgr *CheckpointManager
	retryCtrl     *errors.RetryController
	classifier    *errors.Classifier
	errorTracker  *errors.ErrorTracker
}

// NewWriter creates a new WAL writer for a specific database.
//
// maxSize controls automatic rotation:
//   - 0 disables rotation
//   - >0 triggers rotation once the WAL reaches or exceeds this size (bytes)
func NewWriter(path string, dbID uint64, maxSize uint64, fsync bool, log *logger.Logger) *Writer {
	return &Writer{
		path:         path,
		dbID:         dbID,
		maxSize:      maxSize,
		fsync:        fsync,
		logger:       log,
		rotator:      NewRotator(path, maxSize, fsync, log),
		retryCtrl:    errors.NewRetryController(),
		classifier:   errors.NewClassifier(),
		errorTracker: errors.NewErrorTracker(),
	}
}

// SetCheckpointManager sets the checkpoint manager for this writer.
func (w *Writer) SetCheckpointManager(mgr *CheckpointManager) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.checkpointMgr = mgr
}

func (w *Writer) Open() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	file, err := os.OpenFile(w.path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return errors.ErrFileOpen
	}

	w.file = file
	w.size = getWalSize(file)
	w.isClosed = false

	return nil
}

func getWalSize(file *os.File) uint64 {
	info, _ := file.Stat()
	return uint64(info.Size())
}

// Write encodes and appends a single WAL record.
//
// The on-disk format is defined in format.go and ondisk_format.md.
func (w *Writer) Write(txID, dbID, docID uint64, opType types.OperationType, payload []byte) error {
	return w.retryCtrl.Retry(func() error {
		w.mu.Lock()
		defer w.mu.Unlock()

		if w.file == nil {
			err := errors.ErrFileWrite
			category := w.classifier.Classify(err)
			w.errorTracker.RecordError(err, category)
			return err
		}

		record, err := EncodeRecord(txID, dbID, docID, opType, payload)
		if err != nil {
			category := w.classifier.Classify(err)
			w.errorTracker.RecordError(err, category)
			return err
		}

		if _, err := w.file.Write(record); err != nil {
			err = errors.ErrFileWrite
			category := w.classifier.Classify(err)
			w.errorTracker.RecordError(err, category)
			return err
		}

		w.size += uint64(len(record))

		if w.fsync {
			if err := w.file.Sync(); err != nil {
				err = errors.ErrFileSync
				category := w.classifier.Classify(err)
				w.errorTracker.RecordError(err, category)
				return err
			}
		}

		// Perform rotation if enabled and threshold reached.
		if w.rotator != nil && w.rotator.ShouldRotate(w.size) {
			if err := w.rotateLocked(); err != nil {
				category := w.classifier.Classify(err)
				w.errorTracker.RecordError(err, category)
				return err
			}
		}

		// Check if checkpoint should be created (after rotation check)
		if w.checkpointMgr != nil && w.checkpointMgr.ShouldCreateCheckpoint(w.size) {
			// Note: Checkpoint creation is handled by the caller (core.go)
			// after commit markers are written, to ensure checkpoint includes
			// only committed transactions.
		}

		return nil
	}, w.classifier)
}

// WriteCommitMarker writes a transaction commit marker to WAL.
//
// This uses the same record format as normal operations, with:
//   - OpType = OpCommit
//   - DocID  = 0
//   - PayloadLen = 0
func (w *Writer) WriteCommitMarker(txID uint64) error {
	return w.retryCtrl.Retry(func() error {
		w.mu.Lock()
		defer w.mu.Unlock()

		if w.file == nil {
			err := errors.ErrFileWrite
			category := w.classifier.Classify(err)
			w.errorTracker.RecordError(err, category)
			return err
		}

		record, err := EncodeRecord(txID, w.dbID, 0, types.OpCommit, nil)
		if err != nil {
			category := w.classifier.Classify(err)
			w.errorTracker.RecordError(err, category)
			return err
		}

		offset := w.size

		if _, err := w.file.Write(record); err != nil {
			err = errors.ErrFileWrite
			category := w.classifier.Classify(err)
			w.errorTracker.RecordError(err, category)
			return err
		}

		w.size += uint64(len(record))

		if w.fsync {
			if err := w.file.Sync(); err != nil {
				err = errors.ErrFileSync
				category := w.classifier.Classify(err)
				w.errorTracker.RecordError(err, category)
				return err
			}
		}

		w.logger.Debug("WAL commit marker written: tx_id=%d, offset=%d", txID, offset)

		if w.rotator != nil && w.rotator.ShouldRotate(w.size) {
			if err := w.rotateLocked(); err != nil {
				category := w.classifier.Classify(err)
				w.errorTracker.RecordError(err, category)
				return err
			}
		}

		return nil
	}, w.classifier)
}

// WriteCheckpoint writes a checkpoint record to WAL.
//
// Checkpoints mark a consistent point in the WAL where recovery can start
// from, bounding recovery time. The checkpoint record contains:
//   - OpType = OpCheckpoint
//   - TxID = highest committed transaction ID at checkpoint time
//   - DocID = 0
//   - PayloadLen = 0
func (w *Writer) WriteCheckpoint(txID uint64) error {
	return w.retryCtrl.Retry(func() error {
		w.mu.Lock()
		defer w.mu.Unlock()

		if w.file == nil {
			err := errors.ErrFileWrite
			category := w.classifier.Classify(err)
			w.errorTracker.RecordError(err, category)
			return err
		}

		record, err := EncodeRecord(txID, w.dbID, 0, types.OpCheckpoint, nil)
		if err != nil {
			category := w.classifier.Classify(err)
			w.errorTracker.RecordError(err, category)
			return err
		}

		offset := w.size

		if _, err := w.file.Write(record); err != nil {
			err = errors.ErrFileWrite
			category := w.classifier.Classify(err)
			w.errorTracker.RecordError(err, category)
			return err
		}

		w.size += uint64(len(record))

		if w.fsync {
			if err := w.file.Sync(); err != nil {
				err = errors.ErrFileSync
				category := w.classifier.Classify(err)
				w.errorTracker.RecordError(err, category)
				return err
			}
		}

		w.logger.Info("WAL checkpoint written: tx_id=%d, offset=%d, size=%d", txID, offset, w.size)

		// Record checkpoint in manager if available
		if w.checkpointMgr != nil {
			w.checkpointMgr.RecordCheckpoint(txID, w.size)
		}

		return nil
	}, w.classifier)
}

func (w *Writer) rotateLocked() error {
	// Assumes w.mu is already held.
	if w.file == nil || w.maxSize == 0 {
		return nil
	}

	// Close the current WAL file before rotating.
	if err := w.file.Sync(); err != nil {
		return errors.ErrFileSync
	}
	if err := w.file.Close(); err != nil {
		return errors.ErrFileWrite
	}

	rotatedPath, err := w.rotator.Rotate()
	if err != nil {
		// Best-effort: try to reopen original path even on rotation error.
		w.logger.Error("WAL rotation failed for %s: %v", w.path, err)
		file, openErr := os.OpenFile(w.path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if openErr != nil {
			return errors.ErrFileOpen
		}
		w.file = file
		w.size = getWalSize(file)
		return err
	}

	w.logger.Info("Rotated WAL segment: %s", rotatedPath)

	// Open a fresh active WAL at the base path and reset size counter.
	file, err := os.OpenFile(w.path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return errors.ErrFileOpen
	}

	w.file = file
	w.size = getWalSize(file)

	return nil
}

func (w *Writer) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}

	return w.file.Sync()
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil || w.isClosed {
		return nil
	}

	if err := w.file.Sync(); err != nil {
		return err
	}

	if err := w.file.Close(); err != nil {
		return err
	}

	w.file = nil
	w.isClosed = true
	return nil
}

func (w *Writer) Size() uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.size
}

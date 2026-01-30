package docdb

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sync"
	"time"

	docdberrors "github.com/kartikbazzad/docdb/internal/errors"
	"github.com/kartikbazzad/docdb/internal/logger"
)

const (
	PayloadLenSize    = 4
	CRCLenSize        = 4
	VerificationSize  = 1
	MaxPayloadSize    = 16 * 1024 * 1024
	VerificationValue = byte(1) // Verified records have this value
)

type DataFile struct {
	mu           sync.Mutex
	path         string
	file         *os.File
	offset       uint64
	logger       *logger.Logger
	retryCtrl    *docdberrors.RetryController
	classifier   *docdberrors.Classifier
	errorTracker *docdberrors.ErrorTracker
	onSync       func(d time.Duration) // optional callback after Sync (for metrics)
}

// SetSyncCallback sets a callback invoked after each Sync() with the fsync duration.
func (df *DataFile) SetSyncCallback(cb func(d time.Duration)) {
	df.mu.Lock()
	defer df.mu.Unlock()
	df.onSync = cb
}

func NewDataFile(path string, log *logger.Logger) *DataFile {
	return &DataFile{
		path:         path,
		logger:       log,
		retryCtrl:    docdberrors.NewRetryController(),
		classifier:   docdberrors.NewClassifier(),
		errorTracker: docdberrors.NewErrorTracker(),
	}
}

func (df *DataFile) Open() error {
	df.mu.Lock()
	defer df.mu.Unlock()

	file, err := os.OpenFile(df.path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return docdberrors.ErrFileOpen
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return docdberrors.ErrFileOpen
	}

	df.file = file
	df.offset = uint64(info.Size())

	return nil
}

func (df *DataFile) Write(payload []byte) (uint64, error) {
	if uint32(len(payload)) > MaxPayloadSize {
		err := docdberrors.ErrPayloadTooLarge
		category := df.classifier.Classify(err)
		df.errorTracker.RecordError(err, category)
		return 0, err
	}

	var resultOffset uint64

	err := df.retryCtrl.Retry(func() error {
		df.mu.Lock()
		defer df.mu.Unlock()

		// Sync in-memory offset with on-disk size so that after external truncation
		// (e.g. in healing tests) we write at the correct position and return the right offset.
		if df.file != nil {
			if info, err := df.file.Stat(); err == nil {
				df.offset = uint64(info.Size())
			}
		}

		payloadLen := uint32(len(payload))
		crc32Value := crc32.ChecksumIEEE(payload)

		header := make([]byte, PayloadLenSize+CRCLenSize)
		binary.LittleEndian.PutUint32(header[0:], payloadLen)
		binary.LittleEndian.PutUint32(header[4:], crc32Value)

		offset := df.offset

		// Write header (len + crc32)
		if _, err := df.file.Write(header); err != nil {
			return docdberrors.ErrFileWrite
		}

		// Write payload
		if _, err := df.file.Write(payload); err != nil {
			return docdberrors.ErrFileWrite
		}

		// Write verification flag LAST - this is a critical part
		// for partial write protection. If crash occurs before this, record is unverified.
		verificationFlag := []byte{VerificationValue}
		if _, err := df.file.Write(verificationFlag); err != nil {
			return docdberrors.ErrFileWrite
		}

	// Single sync at the end to ensure all data is durable.
	// This reduces fsync overhead by ~50% while maintaining durability.
	// Trade-off: Slightly larger window (0.1-1ms) where crash could lose data,
	// but verification flag prevents partial records from being read.
		fsyncStart := time.Now()
		if err := df.file.Sync(); err != nil {
			return docdberrors.ErrFileSync
		}
		if df.onSync != nil {
			df.onSync(time.Since(fsyncStart))
		}

		df.offset += uint64(PayloadLenSize + CRCLenSize + len(payload) + VerificationSize)
		resultOffset = offset
		return nil
	}, df.classifier)

	if err != nil {
		return 0, err
	}

	return resultOffset, nil
}

// WriteNoSync appends a record to the data file without calling fsync.
// Caller must call Sync() after a batch of writes (e.g. at end of WAL replay).
// Used during recovery to avoid ~N fsyncs; one Sync at end is sufficient.
func (df *DataFile) WriteNoSync(payload []byte) (uint64, error) {
	if uint32(len(payload)) > MaxPayloadSize {
		err := docdberrors.ErrPayloadTooLarge
		category := df.classifier.Classify(err)
		df.errorTracker.RecordError(err, category)
		return 0, err
	}

	df.mu.Lock()
	defer df.mu.Unlock()

	payloadLen := uint32(len(payload))
	crc32Value := crc32.ChecksumIEEE(payload)

	header := make([]byte, PayloadLenSize+CRCLenSize)
	binary.LittleEndian.PutUint32(header[0:], payloadLen)
	binary.LittleEndian.PutUint32(header[4:], crc32Value)

	offset := df.offset

	if _, err := df.file.Write(header); err != nil {
		return 0, docdberrors.ErrFileWrite
	}
	if _, err := df.file.Write(payload); err != nil {
		return 0, docdberrors.ErrFileWrite
	}
	verificationFlag := []byte{VerificationValue}
	if _, err := df.file.Write(verificationFlag); err != nil {
		return 0, docdberrors.ErrFileWrite
	}

	df.offset += uint64(PayloadLenSize + CRCLenSize + len(payload) + VerificationSize)
	return offset, nil
}

func (df *DataFile) Read(offset uint64, length uint32) ([]byte, error) {
	df.mu.Lock()
	defer df.mu.Unlock()

	if _, err := df.file.Seek(int64(offset), io.SeekStart); err != nil {
		return nil, docdberrors.ErrFileRead
	}

	header := make([]byte, PayloadLenSize+CRCLenSize)
	if _, err := io.ReadFull(df.file, header); err != nil {
		return nil, df.readErrorToCategory(err)
	}

	storedLen := binary.LittleEndian.Uint32(header[0:])
	storedCRC := binary.LittleEndian.Uint32(header[4:])

	if storedLen != length {
		return nil, fmt.Errorf("payload length mismatch: stored=%d, expected=%d", storedLen, length)
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(df.file, payload); err != nil {
		return nil, df.readErrorToCategory(err)
	}

	// Read verification flag
	verificationFlag := make([]byte, VerificationSize)
	if _, err := io.ReadFull(df.file, verificationFlag); err != nil {
		// If we can't read verification flag, record is incomplete/unverified
		df.logger.Warn("Failed to read verification flag at offset %d: %v", offset, err)
		return nil, docdberrors.ErrCorruptRecord
	}

	// Check verification flag - only verified records should be read
	if verificationFlag[0] != VerificationValue {
		df.logger.Warn("Unverified record at offset %d (verification flag=%d)", offset, verificationFlag[0])
		return nil, docdberrors.ErrCorruptRecord
	}

	// Verify CRC32 checksum
	computedCRC := crc32.ChecksumIEEE(payload)
	if storedCRC != computedCRC {
		df.logger.Error("CRC mismatch at offset %d: stored=%x, computed=%x", offset, storedCRC, computedCRC)
		err := docdberrors.ErrCRCMismatch
		category := df.classifier.Classify(err)
		df.errorTracker.RecordError(err, category)
		return nil, err
	}

	return payload, nil
}

// readErrorToCategory maps read failures to the right error for classification.
// EOF/short read indicates truncated/corrupt data (ErrorValidation); other read errors stay ErrFileRead (transient).
func (df *DataFile) readErrorToCategory(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return docdberrors.ErrCorruptRecord
	}
	return docdberrors.ErrFileRead
}

func (df *DataFile) Sync() error {
	df.mu.Lock()
	defer df.mu.Unlock()

	if df.file == nil {
		return nil
	}

	return df.file.Sync()
}

func (df *DataFile) Close() error {
	df.mu.Lock()
	defer df.mu.Unlock()

	if df.file == nil {
		return nil
	}

	if err := df.file.Sync(); err != nil {
		return err
	}

	if err := df.file.Close(); err != nil {
		return err
	}

	df.file = nil
	return nil
}

func (df *DataFile) Size() uint64 {
	df.mu.Lock()
	defer df.mu.Unlock()
	return df.offset
}

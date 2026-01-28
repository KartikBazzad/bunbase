package docdb

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sync"

	"github.com/kartikbazzad/docdb/internal/errors"
	"github.com/kartikbazzad/docdb/internal/logger"
)

const (
	PayloadLenSize = 4
	CRCLenSize     = 4
	MaxPayloadSize = 16 * 1024 * 1024
)

type DataFile struct {
	mu     sync.Mutex
	path   string
	file   *os.File
	offset uint64
	logger *logger.Logger
}

func NewDataFile(path string, log *logger.Logger) *DataFile {
	return &DataFile{
		path:   path,
		logger: log,
	}
}

func (df *DataFile) Open() error {
	df.mu.Lock()
	defer df.mu.Unlock()

	file, err := os.OpenFile(df.path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return errors.ErrFileOpen
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return errors.ErrFileOpen
	}

	df.file = file
	df.offset = uint64(info.Size())

	return nil
}

func (df *DataFile) Write(payload []byte) (uint64, error) {
	if uint32(len(payload)) > MaxPayloadSize {
		return 0, errors.ErrPayloadTooLarge
	}

	df.mu.Lock()
	defer df.mu.Unlock()

	payloadLen := uint32(len(payload))
	crc32 := crc32.ChecksumIEEE(payload)

	header := make([]byte, PayloadLenSize+CRCLenSize)
	binary.LittleEndian.PutUint32(header[0:], payloadLen)
	binary.LittleEndian.PutUint32(header[4:], crc32)

	offset := df.offset

	if _, err := df.file.Write(header); err != nil {
		return 0, errors.ErrFileWrite
	}

	if _, err := df.file.Write(payload); err != nil {
		return 0, errors.ErrFileWrite
	}

	df.offset += uint64(PayloadLenSize + CRCLenSize + len(payload))

	return offset, nil
}

func (df *DataFile) Read(offset uint64, length uint32) ([]byte, error) {
	df.mu.Lock()
	defer df.mu.Unlock()

	if _, err := df.file.Seek(int64(offset), io.SeekStart); err != nil {
		return nil, errors.ErrFileRead
	}

	header := make([]byte, PayloadLenSize+CRCLenSize)
	if _, err := io.ReadFull(df.file, header); err != nil {
		return nil, errors.ErrFileRead
	}

	storedLen := binary.LittleEndian.Uint32(header[0:])
	storedCRC := binary.LittleEndian.Uint32(header[4:])

	if storedLen != length {
		return nil, fmt.Errorf("payload length mismatch: stored=%d, expected=%d", storedLen, length)
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(df.file, payload); err != nil {
		return nil, errors.ErrFileRead
	}

	computedCRC := crc32.ChecksumIEEE(payload)
	if storedCRC != computedCRC {
		df.logger.Error("CRC mismatch at offset %d: stored=%x, computed=%x", offset, storedCRC, computedCRC)
		return nil, errors.ErrCorruptRecord
	}

	return payload, nil
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

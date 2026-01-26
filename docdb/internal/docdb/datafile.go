package docdb

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"sync"

	"github.com/kartikbazzad/docdb/internal/logger"
)

var (
	ErrDataFileOpen    = errors.New("failed to open data file")
	ErrDataFileWrite   = errors.New("failed to write data file")
	ErrDataFileRead    = errors.New("failed to read data file")
	ErrPayloadTooLarge = errors.New("payload exceeds maximum size")
)

const (
	PayloadLenSize = 4
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
		return ErrDataFileOpen
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return ErrDataFileOpen
	}

	df.file = file
	df.offset = uint64(info.Size())

	return nil
}

func (df *DataFile) Write(payload []byte) (uint64, error) {
	if uint32(len(payload)) > MaxPayloadSize {
		return 0, ErrPayloadTooLarge
	}

	df.mu.Lock()
	defer df.mu.Unlock()

	payloadLen := uint32(len(payload))
	header := make([]byte, PayloadLenSize)
	binary.LittleEndian.PutUint32(header, payloadLen)

	offset := df.offset

	if _, err := df.file.Write(header); err != nil {
		return 0, ErrDataFileWrite
	}

	if _, err := df.file.Write(payload); err != nil {
		return 0, ErrDataFileWrite
	}

	df.offset += uint64(PayloadLenSize + len(payload))

	return offset, nil
}

func (df *DataFile) Read(offset uint64, length uint32) ([]byte, error) {
	df.mu.Lock()
	defer df.mu.Unlock()

	if _, err := df.file.Seek(int64(offset), io.SeekStart); err != nil {
		return nil, ErrDataFileRead
	}

	header := make([]byte, PayloadLenSize)
	if _, err := io.ReadFull(df.file, header); err != nil {
		return nil, ErrDataFileRead
	}

	storedLen := binary.LittleEndian.Uint32(header)
	if storedLen != length {
		return nil, ErrDataFileRead
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(df.file, payload); err != nil {
		return nil, ErrDataFileRead
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

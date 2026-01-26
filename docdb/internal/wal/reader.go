package wal

import (
	"io"
	"os"

	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/types"
)

type Reader struct {
	file   *os.File
	path   string
	logger *logger.Logger
}

func NewReader(path string, log *logger.Logger) *Reader {
	return &Reader{
		path:   path,
		logger: log,
	}
}

func (r *Reader) Open() error {
	file, err := os.Open(r.path)
	if err != nil {
		return ErrFileOpen
	}

	r.file = file
	return nil
}

func (r *Reader) Next() (*types.WALRecord, error) {
	if r.file == nil {
		return nil, ErrFileRead
	}

	lenBuf := make([]byte, RecordLenSize)
	_, err := io.ReadFull(r.file, lenBuf)
	if err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, ErrCorruptRecord
	}

	recordLen := byteOrder.Uint64(lenBuf)

	if recordLen < RecordLenSize || recordLen > MaxPayloadSize+RecordOverhead {
		return nil, ErrCorruptRecord
	}

	remaining := recordLen - RecordLenSize
	buf := make([]byte, remaining)

	_, err = io.ReadFull(r.file, buf)
	if err != nil {
		return nil, ErrCorruptRecord
	}

	fullRecord := make([]byte, recordLen)
	copy(fullRecord[:RecordLenSize], lenBuf)
	copy(fullRecord[RecordLenSize:], buf)

	record, err := DecodeRecord(fullRecord)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (r *Reader) Close() error {
	if r.file == nil {
		return nil
	}

	err := r.file.Close()
	r.file = nil
	return err
}

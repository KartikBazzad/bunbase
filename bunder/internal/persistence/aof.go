package persistence

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// AOF is an append-only file for durability (Redis-like). Each record is length(4)+cmd+args.
// Append(cmd, args) encodes and appends; ReadAOF(path) decodes for replay.
type AOF struct {
	path   string
	file   *os.File
	writer *bufio.Writer
	mu     sync.Mutex
}

// NewAOF creates or opens an AOF file.
func NewAOF(path string) (*AOF, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &AOF{path: path, file: f, writer: bufio.NewWriterSize(f, 256*1024)}, nil
}

// Append encodes and appends a single command (e.g. SET key value, DEL key).
func (a *AOF) Append(cmd string, args ...[]byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	// Format: cmdLen(4) + cmd + argc(4) + for each arg: len(4)+data
	buf := make([]byte, 0, 8+len(cmd)+4+32)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(cmd)))
	buf = append(buf, cmd...)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(args)))
	for _, arg := range args {
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(arg)))
		buf = append(buf, arg...)
	}
	// Record: totalLen(4) + buf
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, uint32(len(buf)))
	if _, err := a.writer.Write(lenBuf); err != nil {
		return err
	}
	if _, err := a.writer.Write(buf); err != nil {
		return err
	}
	return nil
}

// Sync flushes the buffer to disk.
func (a *AOF) Sync() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.writer.Flush(); err != nil {
		return err
	}
	return a.file.Sync()
}

// Close closes the AOF file.
func (a *AOF) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.file == nil {
		return nil
	}
	_ = a.writer.Flush()
	_ = a.file.Sync()
	err := a.file.Close()
	a.file = nil
	return err
}

// AOFRecord is one decoded AOF record for replay.
type AOFRecord struct {
	Cmd  string
	Args [][]byte
}

// ReadAOF reads all records from an AOF file (for replay).
func ReadAOF(path string) ([]AOFRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []AOFRecord
	var lenBuf [4]byte
	for {
		if _, err := f.Read(lenBuf[:]); err != nil {
			break
		}
		recLen := binary.LittleEndian.Uint32(lenBuf[:])
		if recLen == 0 || recLen > 64*1024*1024 {
			return nil, fmt.Errorf("invalid aof record length %d", recLen)
		}
		buf := make([]byte, recLen)
		if _, err := f.Read(buf); err != nil {
			return nil, err
		}
		off := 0
		if off+4 > len(buf) {
			break
		}
		cmdLen := binary.LittleEndian.Uint32(buf[off : off+4])
		off += 4
		if off+int(cmdLen)+4 > len(buf) {
			break
		}
		cmd := string(buf[off : off+int(cmdLen)])
		off += int(cmdLen)
		argc := binary.LittleEndian.Uint32(buf[off : off+4])
		off += 4
		var args [][]byte
		for i := uint32(0); i < argc && off+4 <= len(buf); i++ {
			al := binary.LittleEndian.Uint32(buf[off : off+4])
			off += 4
			if off+int(al) > len(buf) {
				break
			}
			args = append(args, append([]byte(nil), buf[off:off+int(al)]...))
			off += int(al)
		}
		out = append(out, AOFRecord{Cmd: cmd, Args: args})
	}
	return out, nil
}

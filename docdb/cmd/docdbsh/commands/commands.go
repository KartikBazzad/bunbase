package commands

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/kartikbazzad/docdb/cmd/docdbsh/parser"
	"github.com/kartikbazzad/docdb/internal/ipc"
	"github.com/kartikbazzad/docdb/internal/types"
)

type Result interface {
	Print(w io.Writer)
	IsExit() bool
}

type ErrorResult struct {
	Err string
}

func (e ErrorResult) Print(w io.Writer) {
	fmt.Fprintln(w, "ERROR")
	fmt.Fprintln(w, e.Err)
}

func (e ErrorResult) IsExit() bool {
	return false
}

type ExitResult struct{}

func (e ExitResult) Print(w io.Writer) {}

func (e ExitResult) IsExit() bool {
	return true
}

type HelpResult struct{}

func (h HelpResult) Print(w io.Writer) {
	fmt.Fprintln(w, "DocDB Shell Commands:")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Meta Commands:")
	fmt.Fprintln(w, "  .help     Show this help message")
	fmt.Fprintln(w, "  .exit     Exit the shell")
	fmt.Fprintln(w, "  .clear    Clear shell state (db + tx)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Database Lifecycle:")
	fmt.Fprintln(w, "  .open <db_name>    Open or create database")
	fmt.Fprintln(w, "  .close             Close current database")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "CRUD Operations:")
	fmt.Fprintln(w, "  .create <doc_id> <payload>      Create document")
	fmt.Fprintln(w, "  .read <doc_id>                   Read document")
	fmt.Fprintln(w, "  .update <doc_id> <payload>      Update document")
	fmt.Fprintln(w, "  .delete <doc_id>                 Delete document")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Payload Formats:")
	fmt.Fprintln(w, "  raw:\"Hello world\"   Raw string")
	fmt.Fprintln(w, "  hex:48656c6c6f      Hex bytes")
	fmt.Fprintln(w, "  json:{\"key\":\"val\"}  JSON object")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Introspection:")
	fmt.Fprintln(w, "  .stats    Print pool statistics")
	fmt.Fprintln(w, "  .mem      Print memory usage")
	fmt.Fprintln(w, "  .wal      Print WAL info")
}

func (h HelpResult) IsExit() bool {
	return false
}

type ClearResult struct{}

func (c ClearResult) Print(w io.Writer) {
	fmt.Fprintln(w, "OK")
}

func (c ClearResult) IsExit() bool {
	return false
}

type OpenResult struct {
	DBID uint64
}

func (o OpenResult) Print(w io.Writer) {
	fmt.Fprintln(w, "OK")
	fmt.Fprintf(w, "db_id=%d\n", o.DBID)
}

func (o OpenResult) IsExit() bool {
	return false
}

type CloseResult struct{}

func (c CloseResult) Print(w io.Writer) {
	fmt.Fprintln(w, "OK")
}

func (c CloseResult) IsExit() bool {
	return false
}

type ReadResult struct {
	Data   []byte
	IsJSON bool
}

func (r ReadResult) Print(w io.Writer) {
	fmt.Fprintln(w, "OK")
	fmt.Fprintf(w, "len=%d\n", len(r.Data))
	fmt.Fprintf(w, "hex=%s\n", hex.EncodeToString(r.Data))

	if r.IsJSON {
		var v interface{}
		if err := json.Unmarshal(r.Data, &v); err == nil {
			pretty, _ := json.Marshal(v)
			fmt.Fprintf(w, "json=%s\n", string(pretty))
		}
	}
}

func (r ReadResult) IsExit() bool {
	return false
}

type OKResult struct{}

func (o OKResult) Print(w io.Writer) {
	fmt.Fprintln(w, "OK")
}

func (o OKResult) IsExit() bool {
	return false
}

type StatsResult struct {
	Stats *types.Stats
}

func (s StatsResult) Print(w io.Writer) {
	fmt.Fprintln(w, "OK")
	fmt.Fprintf(w, "total_dbs=%d\n", s.Stats.TotalDBs)
	fmt.Fprintf(w, "active_dbs=%d\n", s.Stats.ActiveDBs)
	fmt.Fprintf(w, "total_txns=%d\n", s.Stats.TotalTxns)
	fmt.Fprintf(w, "wal_size=%d\n", s.Stats.WALSize)
	fmt.Fprintf(w, "memory_used=%d\n", s.Stats.MemoryUsed)
}

func (s StatsResult) IsExit() bool {
	return false
}

type MemResult struct {
	Stats *types.Stats
}

func (m MemResult) Print(w io.Writer) {
	fmt.Fprintln(w, "OK")
	fmt.Fprintf(w, "memory_used=%d\n", m.Stats.MemoryUsed)
	fmt.Fprintf(w, "memory_capacity=%d\n", m.Stats.MemoryCapacity)
	if m.Stats.MemoryCapacity > 0 {
		fmt.Fprintf(w, "usage_percent=%.2f\n", float64(m.Stats.MemoryUsed)/float64(m.Stats.MemoryCapacity)*100)
	}
}

func (m MemResult) IsExit() bool {
	return false
}

type WALResult struct {
	Stats *types.Stats
}

func (w WALResult) Print(out io.Writer) {
	fmt.Fprintln(out, "OK")
	fmt.Fprintf(out, "wal_size=%d\n", w.Stats.WALSize)
}

func (w WALResult) IsExit() bool {
	return false
}

func Help() Result {
	return HelpResult{}
}

func Exit() Result {
	return ExitResult{}
}

func Clear(s Shell) Result {
	s.ClearDB()
	return ClearResult{}
}

func Open(s Shell, cmd *parser.Command) Result {
	if err := parser.ValidateArgs(cmd, 1); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	name := cmd.Args[0]

	dbID, err := s.GetClient().OpenDB(name)
	if err != nil {
		return ErrorResult{Err: err.Error()}
	}

	s.SetDB(dbID)
	return OpenResult{DBID: dbID}
}

func Close(s Shell) Result {
	dbID := s.GetDB()

	if dbID == 0 {
		return ErrorResult{Err: "no database open"}
	}

	if err := s.GetClient().CloseDB(dbID); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	s.ClearDB()
	return CloseResult{}
}

func Create(s Shell, cmd *parser.Command) Result {
	if err := parser.ValidateDB(s.GetDB()); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	if err := parser.ValidateArgs(cmd, 2); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	docID, err := parser.ParseUint64(cmd.Args[0])
	if err != nil {
		return ErrorResult{Err: fmt.Sprintf("invalid doc_id: %s", err)}
	}

	payload, err := parser.DecodePayload(strings.Join(cmd.Args[1:], " "))
	if err != nil {
		return ErrorResult{Err: err.Error()}
	}

	ops := []ipc.Operation{
		{
			OpType:  types.OpCreate,
			DocID:   docID,
			Payload: payload,
		},
	}

	if _, err := s.GetClient().Execute(s.GetDB(), ops); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	return OKResult{}
}

func Read(s Shell, cmd *parser.Command) Result {
	if err := parser.ValidateDB(s.GetDB()); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	if err := parser.ValidateArgs(cmd, 1); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	docID, err := parser.ParseUint64(cmd.Args[0])
	if err != nil {
		return ErrorResult{Err: fmt.Sprintf("invalid doc_id: %s", err)}
	}

	ops := []ipc.Operation{
		{
			OpType:  types.OpRead,
			DocID:   docID,
			Payload: nil,
		},
	}

	data, err := s.GetClient().Execute(s.GetDB(), ops)
	if err != nil {
		return ErrorResult{Err: err.Error()}
	}

	resultData, err := parseReadResponse(data)
	if err != nil {
		return ErrorResult{Err: err.Error()}
	}

	isJSON := false
	if len(resultData) > 0 && json.Valid(resultData) {
		isJSON = true
	}

	return ReadResult{Data: resultData, IsJSON: isJSON}
}

func Update(s Shell, cmd *parser.Command) Result {
	if err := parser.ValidateDB(s.GetDB()); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	if err := parser.ValidateArgs(cmd, 2); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	docID, err := parser.ParseUint64(cmd.Args[0])
	if err != nil {
		return ErrorResult{Err: fmt.Sprintf("invalid doc_id: %s", err)}
	}

	payload, err := parser.DecodePayload(strings.Join(cmd.Args[1:], " "))
	if err != nil {
		return ErrorResult{Err: err.Error()}
	}

	ops := []ipc.Operation{
		{
			OpType:  types.OpUpdate,
			DocID:   docID,
			Payload: payload,
		},
	}

	if _, err := s.GetClient().Execute(s.GetDB(), ops); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	return OKResult{}
}

func Delete(s Shell, cmd *parser.Command) Result {
	if err := parser.ValidateDB(s.GetDB()); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	if err := parser.ValidateArgs(cmd, 1); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	docID, err := parser.ParseUint64(cmd.Args[0])
	if err != nil {
		return ErrorResult{Err: fmt.Sprintf("invalid doc_id: %s", err)}
	}

	ops := []ipc.Operation{
		{
			OpType:  types.OpDelete,
			DocID:   docID,
			Payload: nil,
		},
	}

	if _, err := s.GetClient().Execute(s.GetDB(), ops); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	return OKResult{}
}

func Stats(s Shell) Result {
	stats, err := s.GetClient().Stats()
	if err != nil {
		return ErrorResult{Err: err.Error()}
	}

	return StatsResult{Stats: stats}
}

func Mem(s Shell) Result {
	stats, err := s.GetClient().Stats()
	if err != nil {
		return ErrorResult{Err: err.Error()}
	}

	return MemResult{Stats: stats}
}

func WAL(s Shell) Result {
	stats, err := s.GetClient().Stats()
	if err != nil {
		return ErrorResult{Err: err.Error()}
	}

	return WALResult{Stats: stats}
}

func parseReadResponse(data []byte) ([]byte, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("invalid response: too short")
	}

	count := binary.LittleEndian.Uint32(data[0:])
	if count != 1 {
		return nil, fmt.Errorf("invalid response: expected 1 result, got %d", count)
	}

	offset := 4
	payloadLen := binary.LittleEndian.Uint32(data[offset:])
	offset += 4

	if offset+int(payloadLen) > len(data) {
		return nil, fmt.Errorf("invalid response: payload length mismatch")
	}

	return data[offset : offset+int(payloadLen)], nil
}

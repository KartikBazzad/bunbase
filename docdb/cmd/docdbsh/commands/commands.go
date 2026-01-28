package commands

import (
	"encoding/binary"
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
	fmt.Fprintln(w, "  .use <db_name>      Alias for .open")
	fmt.Fprintln(w, "  .close               Close current database")
	fmt.Fprintln(w, "  .ls                  List databases")
	fmt.Fprintln(w, "  .pwd                 Show current database")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Display Options:")
	fmt.Fprintln(w, "  .pretty on|off       Toggle JSON formatting")
	fmt.Fprintln(w, "  .history             Show command history")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Collections:")
	fmt.Fprintln(w, "  .use <collection>                Set current collection")
	fmt.Fprintln(w, "  .collections                     List all collections")
	fmt.Fprintln(w, "  .create-collection <name>        Create collection")
	fmt.Fprintln(w, "  .drop-collection <name>          Delete collection")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "CRUD Operations:")
	fmt.Fprintln(w, "  .create <doc_id> <payload>      Create document")
	fmt.Fprintln(w, "  .read <doc_id>                   Read document")
	fmt.Fprintln(w, "  .update <doc_id> <payload>      Update document")
	fmt.Fprintln(w, "  .delete <doc_id>                 Delete document")
	fmt.Fprintln(w, "  .patch <doc_id> <patch-ops>      Patch document (path-based updates)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Payload Format:")
	fmt.Fprintln(w, "  Documents must be valid JSON:")
	fmt.Fprintln(w, "    .create 1 {\"name\":\"Alice\",\"age\":30}")
	fmt.Fprintln(w, "    .create 2 [1,2,3]")
	fmt.Fprintln(w, "    .create 3 \"hello world\"")
	fmt.Fprintln(w, "    .create 4 42")
	fmt.Fprintln(w, "    .create 5 true")
	fmt.Fprintln(w, "    .create 6 null")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Binary data (use base64 encoding):")
	fmt.Fprintln(w, "    .create 7 {\"_type\":\"bytes\",\"encoding\":\"base64\",\"data\":\"SGVsbG8=\"}")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Introspection:")
	fmt.Fprintln(w, "  .stats    Print pool statistics")
	fmt.Fprintln(w, "  .mem      Print memory usage")
	fmt.Fprintln(w, "  .wal      Print WAL info")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Healing:")
	fmt.Fprintln(w, "  .heal <doc_id>      Heal a specific document")
	fmt.Fprintln(w, "  .heal-all           Trigger full database healing scan")
	fmt.Fprintln(w, "  .heal-stats         Show healing statistics")
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
	Pretty bool
}

func (r ReadResult) Print(w io.Writer) {
	fmt.Fprintln(w, "OK")
	fmt.Fprintf(w, "len=%d\n", len(r.Data))

	var v interface{}
	if err := json.Unmarshal(r.Data, &v); err == nil {
		var jsonOutput []byte
		if r.Pretty {
			jsonOutput, _ = json.MarshalIndent(v, "", "  ")
		} else {
			jsonOutput, _ = json.Marshal(v)
		}
		fmt.Fprintf(w, "json=%s\n", string(jsonOutput))
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
	s.SetDBName(name)
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
			OpType:     types.OpCreate,
			Collection: s.GetCollection(),
			DocID:      docID,
			Payload:    payload,
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
			OpType:     types.OpRead,
			Collection: s.GetCollection(),
			DocID:      docID,
			Payload:    nil,
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

	return ReadResult{Data: resultData, Pretty: s.GetPretty()}
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
			OpType:     types.OpUpdate,
			Collection: s.GetCollection(),
			DocID:      docID,
			Payload:    payload,
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
			OpType:     types.OpDelete,
			Collection: s.GetCollection(),
			DocID:      docID,
			Payload:    nil,
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

type ListDBsResult struct {
	DBInfos []*types.DBInfo
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (l ListDBsResult) Print(w io.Writer) {
	fmt.Fprintln(w, "OK")
	if len(l.DBInfos) == 0 {
		fmt.Fprintln(w, "No databases found")
		fmt.Fprintln(w, "Use '.open <dbname>' or '.use <dbname>' to create a database")
	} else {
		fmt.Fprintf(w, "databases=%d\n", len(l.DBInfos))
		fmt.Fprintln(w)
		// Print header
		fmt.Fprintf(w, "%-20s %8s %19s %12s %12s %10s\n", "NAME", "ID", "CREATED", "WAL_SIZE", "MEMORY", "DOCS")
		fmt.Fprintln(w, strings.Repeat("-", 90))
		// Print each database
		for _, info := range l.DBInfos {
			createdStr := info.CreatedAt.Format("2006-01-02 15:04:05")
			fmt.Fprintf(w, "%-20s %8d %19s %12s %12s %10d\n",
				info.Name,
				info.ID,
				createdStr,
				formatBytes(info.WALSize),
				formatBytes(info.MemoryUsed),
				info.DocsLive)
		}
	}
}

func (l ListDBsResult) IsExit() bool {
	return false
}

func ListDBs(s Shell) Result {
	dbInfos, err := s.GetClient().ListDBs()
	if err != nil {
		return ErrorResult{Err: err.Error()}
	}

	return ListDBsResult{DBInfos: dbInfos}
}

type PWDResult struct {
	DBName string
	DBID   uint64
}

func (p PWDResult) Print(w io.Writer) {
	fmt.Fprintln(w, "OK")
	if p.DBID == 0 {
		fmt.Fprintln(w, "No database open")
	} else {
		fmt.Fprintf(w, "database=%s\n", p.DBName)
		fmt.Fprintf(w, "db_id=%d\n", p.DBID)
	}
}

func (p PWDResult) IsExit() bool {
	return false
}

func PWD(s Shell) Result {
	dbID := s.GetDB()
	dbName := s.GetDBName()

	return PWDResult{
		DBName: dbName,
		DBID:   dbID,
	}
}

type PrettyResult struct {
	Enabled bool
}

func (p PrettyResult) Print(w io.Writer) {
	fmt.Fprintln(w, "OK")
	fmt.Fprintf(w, "pretty=%t\n", p.Enabled)
}

func (p PrettyResult) IsExit() bool {
	return false
}

func Pretty(s Shell, cmd *parser.Command) Result {
	if len(cmd.Args) == 0 {
		return PrettyResult{Enabled: s.GetPretty()}
	}

	arg := strings.ToLower(cmd.Args[0])
	switch arg {
	case "on":
		s.SetPretty(true)
		return PrettyResult{Enabled: true}
	case "off":
		s.SetPretty(false)
		return PrettyResult{Enabled: false}
	default:
		return ErrorResult{Err: "usage: .pretty on|off"}
	}
}

type HistoryResult struct {
	Commands []string
}

func (h HistoryResult) Print(w io.Writer) {
	fmt.Fprintln(w, "OK")
	fmt.Fprintf(w, "total_commands=%d\n", len(h.Commands))
	for i, cmd := range h.Commands {
		fmt.Fprintf(w, "%3d: %s\n", i+1, cmd)
	}
}

func (h HistoryResult) IsExit() bool {
	return false
}

func History(s Shell) Result {
	hist := s.GetHistory()
	return HistoryResult{Commands: hist}
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

type HealResult struct {
	DocID   uint64
	Success bool
	Error   string
}

func (h HealResult) Print(w io.Writer) {
	if h.Success {
		fmt.Fprintln(w, "OK")
		fmt.Fprintf(w, "healed_doc_id=%d\n", h.DocID)
	} else {
		fmt.Fprintln(w, "ERROR")
		fmt.Fprintln(w, h.Error)
	}
}

func (h HealResult) IsExit() bool {
	return false
}

func Heal(s Shell, cmd *parser.Command) Result {
	if err := parser.ValidateDB(s.GetDB()); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	if len(cmd.Args) == 0 {
		return ErrorResult{Err: "usage: .heal <doc_id>"}
	}

	// Parse doc_id
	var docID uint64
	if _, err := fmt.Sscanf(cmd.Args[0], "%d", &docID); err != nil {
		return ErrorResult{Err: fmt.Sprintf("invalid doc_id: %s", cmd.Args[0])}
	}

	collection := s.GetCollection()
	if collection == "" {
		collection = "default"
	}

	err := s.GetClient().HealDocument(s.GetDB(), collection, docID)
	if err != nil {
		return ErrorResult{Err: err.Error()}
	}

	return HealResult{
		DocID:   docID,
		Success: true,
	}
}

type HealAllResult struct {
	HealedCount int
	Error       string
}

func (h HealAllResult) Print(w io.Writer) {
	if h.Error != "" {
		fmt.Fprintln(w, "ERROR")
		fmt.Fprintln(w, h.Error)
	} else {
		fmt.Fprintln(w, "OK")
		fmt.Fprintf(w, "healed_count=%d\n", h.HealedCount)
	}
}

func (h HealAllResult) IsExit() bool {
	return false
}

func HealAll(s Shell) Result {
	if err := parser.ValidateDB(s.GetDB()); err != nil {
		return HealAllResult{Error: err.Error()}
	}

	healed, err := s.GetClient().HealAll(s.GetDB())
	if err != nil {
		return HealAllResult{Error: err.Error()}
	}

	return HealAllResult{
		HealedCount: len(healed),
		Error:       "",
	}
}

type HealStatsResult struct {
	TotalScans         uint64
	DocumentsHealed    uint64
	DocumentsCorrupted uint64
	LastScanTime       string
	LastHealingTime    string
	Error              string
}

func (h HealStatsResult) Print(w io.Writer) {
	if h.Error != "" {
		fmt.Fprintln(w, "ERROR")
		fmt.Fprintln(w, h.Error)
	} else {
		fmt.Fprintln(w, "OK")
		fmt.Fprintf(w, "total_scans=%d\n", h.TotalScans)
		fmt.Fprintf(w, "documents_healed=%d\n", h.DocumentsHealed)
		fmt.Fprintf(w, "documents_corrupted=%d\n", h.DocumentsCorrupted)
		fmt.Fprintf(w, "last_scan_time=%s\n", h.LastScanTime)
		fmt.Fprintf(w, "last_healing_time=%s\n", h.LastHealingTime)
	}
}

func (h HealStatsResult) IsExit() bool {
	return false
}

func HealStats(s Shell) Result {
	if err := parser.ValidateDB(s.GetDB()); err != nil {
		return HealStatsResult{Error: err.Error()}
	}

	stats, err := s.GetClient().HealStats(s.GetDB())
	if err != nil {
		return HealStatsResult{Error: err.Error()}
	}

	result := HealStatsResult{
		Error: "",
	}

	// Extract stats from map
	if val, ok := stats["TotalScans"].(float64); ok {
		result.TotalScans = uint64(val)
	}
	if val, ok := stats["DocumentsHealed"].(float64); ok {
		result.DocumentsHealed = uint64(val)
	}
	if val, ok := stats["DocumentsCorrupted"].(float64); ok {
		result.DocumentsCorrupted = uint64(val)
	}
	if val, ok := stats["LastScanTime"].(string); ok {
		result.LastScanTime = val
	}
	if val, ok := stats["LastHealingTime"].(string); ok {
		result.LastHealingTime = val
	}

	return result
}

// Collection management commands

func UseCollection(s Shell, cmd *parser.Command) Result {
	if err := parser.ValidateDB(s.GetDB()); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	if err := parser.ValidateArgs(cmd, 1); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	collection := cmd.Args[0]
	s.SetCollection(collection)
	return OKResult{}
}

func ListCollections(s Shell) Result {
	if err := parser.ValidateDB(s.GetDB()); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	collections, err := s.GetClient().ListCollections(s.GetDB())
	if err != nil {
		return ErrorResult{Err: err.Error()}
	}

	return CollectionsResult{Collections: collections}
}

type CollectionsResult struct {
	Collections []string
}

func (c CollectionsResult) Print(w io.Writer) {
	fmt.Fprintln(w, "OK")
	for _, coll := range c.Collections {
		fmt.Fprintln(w, coll)
	}
}

func (c CollectionsResult) IsExit() bool {
	return false
}

func CreateCollection(s Shell, cmd *parser.Command) Result {
	if err := parser.ValidateDB(s.GetDB()); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	if err := parser.ValidateArgs(cmd, 1); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	name := cmd.Args[0]
	if err := s.GetClient().CreateCollection(s.GetDB(), name); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	return OKResult{}
}

func DropCollection(s Shell, cmd *parser.Command) Result {
	if err := parser.ValidateDB(s.GetDB()); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	if err := parser.ValidateArgs(cmd, 1); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	name := cmd.Args[0]
	if err := s.GetClient().DeleteCollection(s.GetDB(), name); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	return OKResult{}
}

func Patch(s Shell, cmd *parser.Command) Result {
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

	// Parse patch operations JSON array
	patchOpsJSON := strings.Join(cmd.Args[1:], " ")
	var patchOps []types.PatchOperation
	if err := json.Unmarshal([]byte(patchOpsJSON), &patchOps); err != nil {
		return ErrorResult{Err: fmt.Sprintf("invalid patch operations: %s", err)}
	}

	ops := []ipc.Operation{
		{
			OpType:     types.OpPatch,
			Collection: s.GetCollection(),
			DocID:      docID,
			PatchOps:   patchOps,
			Payload:    nil,
		},
	}

	if _, err := s.GetClient().Execute(s.GetDB(), ops); err != nil {
		return ErrorResult{Err: err.Error()}
	}

	return OKResult{}
}

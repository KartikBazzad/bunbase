package integration

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/docdb"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
	"github.com/kartikbazzad/docdb/internal/query"
	"github.com/kartikbazzad/docdb/internal/types"
	"github.com/kartikbazzad/docdb/tests/failure"
)

const defaultCollection = "_default"

// ReplayOp represents a single committed operation for building expected state.
type ReplayOp struct {
	Op      string `json:"op"` // "create", "update", "delete"
	DocID   uint64 `json:"doc_id"`
	Payload []byte `json:"payload,omitempty"`
}

// buildReferenceModel builds expected docID -> payload from a list of operations.
// Deleted docs are not present in the map.
func buildReferenceModel(ops []ReplayOp) map[uint64][]byte {
	expected := make(map[uint64][]byte)
	for _, op := range ops {
		switch op.Op {
		case "create", "update":
			expected[op.DocID] = op.Payload
		case "delete":
			delete(expected, op.DocID)
		}
	}
	return expected
}

// buildReferenceModelFromLog parses newline-delimited JSON lines (commit log)
// and returns expected state. Incomplete last line is skipped.
func buildReferenceModelFromLog(logPath string) (map[uint64][]byte, error) {
	f, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var ops []ReplayOp
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var op ReplayOp
		if err := json.Unmarshal([]byte(line), &op); err != nil {
			// Incomplete or corrupt line (e.g. last line when process was killed)
			continue
		}
		ops = append(ops, op)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return buildReferenceModel(ops), nil
}

// runWorkload executes ops on db via worker pool (partitioned). Returns committed ops for reference.
func runWorkload(t *testing.T, db *docdb.LogicalDB, ops []ReplayOp) []ReplayOp {
	t.Helper()
	partitionCount := db.PartitionCount()
	if partitionCount <= 1 {
		t.Fatal("runWorkload requires partitioned DB (PartitionCount > 1)")
	}

	var committed []ReplayOp
	for _, op := range ops {
		partitionID := docdb.RouteToPartition(op.DocID, partitionCount)
		var task *docdb.Task
		switch op.Op {
		case "create":
			task = docdb.NewTaskWithPayload(partitionID, types.OpCreate, defaultCollection, op.DocID, op.Payload)
		case "update":
			task = docdb.NewTaskWithPayload(partitionID, types.OpUpdate, defaultCollection, op.DocID, op.Payload)
		case "delete":
			task = docdb.NewTask(partitionID, types.OpDelete, defaultCollection, op.DocID)
		default:
			t.Fatalf("unknown op: %s", op.Op)
		}
		result := db.SubmitTaskAndWait(task)
		if result.Error != nil {
			// Document not found on update/delete or already exists on create - skip for deterministic test
			continue
		}
		committed = append(committed, op)
	}
	return committed
}

// verifyPartitionState runs a full-scan query and compares result to expected state.
func verifyPartitionState(t *testing.T, db *docdb.LogicalDB, expected map[uint64][]byte) {
	t.Helper()

	q := query.Query{Limit: 0}
	rows, err := db.ExecuteQuery(context.Background(), defaultCollection, q)
	if err != nil {
		t.Fatalf("ExecuteQuery: %v", err)
	}

	actual := make(map[uint64][]byte)
	for _, r := range rows {
		actual[r.DocID] = r.Payload
	}

	if len(actual) != len(expected) {
		t.Errorf("row count: got %d, want %d", len(actual), len(expected))
	}
	for docID, wantPayload := range expected {
		got, ok := actual[docID]
		if !ok {
			t.Errorf("doc %d: missing", docID)
			continue
		}
		if string(got) != string(wantPayload) {
			t.Errorf("doc %d: payload mismatch: got %q, want %q", docID, got, wantPayload)
		}
	}
	for docID := range actual {
		if _, ok := expected[docID]; !ok {
			t.Errorf("doc %d: unexpected (not in expected)", docID)
		}
	}
}

// deterministicOps returns a fixed sequence of create/update/delete for testing.
func deterministicOps() []ReplayOp {
	return []ReplayOp{
		{Op: "create", DocID: 1, Payload: []byte(`{"x":1}`)},
		{Op: "create", DocID: 2, Payload: []byte(`{"x":2}`)},
		{Op: "create", DocID: 3, Payload: []byte(`{"x":3}`)},
		{Op: "create", DocID: 4, Payload: []byte(`{"x":4}`)},
		{Op: "create", DocID: 5, Payload: []byte(`{"x":5}`)},
		{Op: "update", DocID: 2, Payload: []byte(`{"x":22}`)},
		{Op: "update", DocID: 4, Payload: []byte(`{"x":44}`)},
		{Op: "delete", DocID: 3},
		{Op: "create", DocID: 10, Payload: []byte(`{"x":10}`)},
		{Op: "create", DocID: 11, Payload: []byte(`{"x":11}`)},
	}
}

func TestPartitionReplay_GracefulClose(t *testing.T) {
	helper := failure.NewCrashTestHelper(t)
	defer helper.Cleanup()

	partitionCounts := []int{2, 4, 8}
	for _, n := range partitionCounts {
		t.Run("partitions_"+strconv.Itoa(n), func(t *testing.T) {
			dbName := "replay_grace_" + strconv.Itoa(n)
			db := helper.CreateDBWithPartitions(dbName, n)
			ops := deterministicOps()
			committed := runWorkload(t, db, ops)
			expected := buildReferenceModel(committed)

			if err := db.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}

			db2 := helper.ReopenDBWithPartitions(dbName, n)
			defer db2.Close()

			verifyPartitionState(t, db2, expected)
		})
	}
}

// TestPartitionReplayChild is run as a subprocess when DOCDB_REPLAY_CHILD=1.
// It opens the DB at env dir, runs a workload, appends each committed op to a log file, then blocks.
func TestPartitionReplayChild(t *testing.T) {
	if os.Getenv("DOCDB_REPLAY_CHILD") != "1" {
		t.Skip("not replay child process")
		return
	}

	dataDir := os.Getenv("DOCDB_REPLAY_DATADIR")
	walDir := os.Getenv("DOCDB_REPLAY_WALDIR")
	dbName := os.Getenv("DOCDB_REPLAY_DB")
	commitLogPath := os.Getenv("DOCDB_REPLAY_COMMITLOG")
	numOpsStr := os.Getenv("DOCDB_REPLAY_OPS")
	partitionsStr := os.Getenv("DOCDB_REPLAY_PARTITIONS")

	if dataDir == "" || walDir == "" || dbName == "" || commitLogPath == "" || numOpsStr == "" || partitionsStr == "" {
		t.Fatal("missing env: DOCDB_REPLAY_DATADIR, WALDIR, DB, COMMITLOG, OPS, PARTITIONS")
	}

	numOps, _ := strconv.Atoi(numOpsStr)
	if numOps <= 0 {
		numOps = 30
	}
	partitionCount, _ := strconv.Atoi(partitionsStr)
	if partitionCount <= 0 {
		partitionCount = 4
	}

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	dbCfg := config.DefaultLogicalDBConfig()
	dbCfg.PartitionCount = partitionCount

	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)

	db := docdb.NewLogicalDBWithConfig(1, dbName, cfg, dbCfg, memCaps, pool, log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("child open DB: %v", err)
	}

	commitLog, err := os.OpenFile(commitLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("child open commit log: %v", err)
	}

	// Deterministic ops: cycle through create/update/delete so we have a mix.
	ops := deterministicOps()
	for i := 0; i < numOps; i++ {
		tmpl := ops[i%len(ops)]
		op := ReplayOp{Op: tmpl.Op, DocID: uint64(i%20) + 1}
		if op.Op != "delete" {
			op.Payload = []byte(`{"i":` + strconv.Itoa(i) + `}`)
		}

		partitionID := docdb.RouteToPartition(op.DocID, partitionCount)
		var task *docdb.Task
		switch op.Op {
		case "create":
			task = docdb.NewTaskWithPayload(partitionID, types.OpCreate, defaultCollection, op.DocID, op.Payload)
		case "update":
			task = docdb.NewTaskWithPayload(partitionID, types.OpUpdate, defaultCollection, op.DocID, op.Payload)
		case "delete":
			task = docdb.NewTask(partitionID, types.OpDelete, defaultCollection, op.DocID)
		default:
			continue
		}
		result := db.SubmitTaskAndWait(task)
		if result.Error != nil {
			continue
		}
		line, _ := json.Marshal(op)
		if _, err := commitLog.Write(append(line, '\n')); err != nil {
			t.Fatalf("child write commit log: %v", err)
		}
		if err := commitLog.Sync(); err != nil {
			t.Fatalf("child sync commit log: %v", err)
		}
	}

	commitLog.Close()
	// Block until killed
	select {}
}

func TestPartitionReplay_KillAndRecover(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGKILL subprocess test skipped on Windows")
		return
	}

	tmpDir, err := os.MkdirTemp("", "docdb-replay-kill-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	walDir := filepath.Join(tmpDir, "wal")
	if err := os.MkdirAll(walDir, 0755); err != nil {
		t.Fatalf("MkdirAll wal: %v", err)
	}

	commitLogPath := filepath.Join(tmpDir, "commit.log")

	env := append(os.Environ(),
		"DOCDB_REPLAY_CHILD=1",
		"DOCDB_REPLAY_DATADIR="+tmpDir,
		"DOCDB_REPLAY_WALDIR="+walDir,
		"DOCDB_REPLAY_DB=replaykill",
		"DOCDB_REPLAY_COMMITLOG="+commitLogPath,
		"DOCDB_REPLAY_OPS=50",
		"DOCDB_REPLAY_PARTITIONS=4",
	)

	pm, err := failure.StartProcess(t, os.Args[0], []string{"-test.run=^TestPartitionReplayChild$", "-test.v"}, env)
	if err != nil {
		t.Fatalf("start child: %v", err)
	}
	// Child runs in tmpDir via env; StartProcess does not set cmd.Dir
	// DOCDB_REPLAY_DATADIR/WALDIR point to tmpDir so child uses correct paths

	// Let child run some ops then kill
	time.Sleep(400 * time.Millisecond)
	if err := pm.Kill(); err != nil {
		t.Fatalf("Kill: %v", err)
	}
	_ = pm.Wait(2 * time.Second)

	// Build expected from commit log (complete lines only)
	expected, err := buildReferenceModelFromLog(commitLogPath)
	if err != nil {
		t.Fatalf("buildReferenceModelFromLog: %v", err)
	}

	// Reopen and verify (reopen inline since we own tmpDir, not CrashTestHelper)
	cfg := config.DefaultConfig()
	cfg.DataDir = tmpDir
	cfg.WAL.Dir = walDir
	dbCfg := config.DefaultLogicalDBConfig()
	dbCfg.PartitionCount = 4

	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)

	db2 := docdb.NewLogicalDBWithConfig(1, "replaykill", cfg, dbCfg, memCaps, pool, log)
	if err := db2.Open(tmpDir, walDir); err != nil {
		t.Fatalf("reopen after kill: %v", err)
	}
	defer db2.Close()

	verifyPartitionState(t, db2, expected)
}

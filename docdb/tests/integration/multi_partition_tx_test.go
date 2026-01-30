package integration

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/docdb"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
	"github.com/kartikbazzad/docdb/internal/types"
)

// setupMultiPartitionDB creates a LogicalDB with partitionCount >= 2 for multi-partition tx tests.
func setupMultiPartitionDB(t *testing.T, name string, partitionCount int) (db *docdb.LogicalDB, dataDir, walDir string, cleanup func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "docdb-multipartition-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	dataDir = filepath.Join(tmpDir, "data")
	walDir = filepath.Join(tmpDir, "wal")
	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	dbCfg := config.DefaultLogicalDBConfig()
	dbCfg.PartitionCount = partitionCount
	db = docdb.NewLogicalDBWithConfig(1, name, cfg, dbCfg, memCaps, pool, log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	cleanup = func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}
	return db, dataDir, walDir, cleanup
}

func reopenMultiPartitionDB(t *testing.T, name, dataDir, walDir string, partitionCount int) *docdb.LogicalDB {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	dbCfg := config.DefaultLogicalDBConfig()
	dbCfg.PartitionCount = partitionCount
	db := docdb.NewLogicalDBWithConfig(1, name, cfg, dbCfg, memCaps, pool, log)
	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	return db
}

// TestMultiPartitionCommit_Success commits a transaction with ops on two partitions and verifies visibility.
func TestMultiPartitionCommit_Success(t *testing.T) {
	db, dataDir, walDir, cleanup := setupMultiPartitionDB(t, "mp-commit", 2)
	defer cleanup()
	// With 2 partitions: docID 1 -> partition 1, docID 2 -> partition 0
	tx := db.Begin()
	if err := db.AddOpToTx(tx, "_default", types.OpCreate, 1, []byte(`{"a":1}`)); err != nil {
		t.Fatalf("AddOpToTx: %v", err)
	}
	if err := db.AddOpToTx(tx, "_default", types.OpCreate, 2, []byte(`{"b":2}`)); err != nil {
		t.Fatalf("AddOpToTx: %v", err)
	}
	if err := db.Commit(tx); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	got1, err := db.Read("_default", 1)
	if err != nil {
		t.Fatalf("Read doc 1: %v", err)
	}
	if string(got1) != `{"a":1}` {
		t.Fatalf("doc 1: got %s", got1)
	}
	got2, err := db.Read("_default", 2)
	if err != nil {
		t.Fatalf("Read doc 2: %v", err)
	}
	if string(got2) != `{"b":2}` {
		t.Fatalf("doc 2: got %s", got2)
	}
	// Restart and verify both visible
	db.Close()
	db2 := reopenMultiPartitionDB(t, "mp-commit", dataDir, walDir, 2)
	defer db2.Close()
	got1, _ = db2.Read("_default", 1)
	got2, _ = db2.Read("_default", 2)
	if string(got1) != `{"a":1}` || string(got2) != `{"b":2}` {
		t.Fatalf("after restart: doc1=%s doc2=%s", got1, got2)
	}
}

// TestSinglePartitionCommit_Success uses single-partition fast path (no coordinator).
func TestSinglePartitionCommit_Success(t *testing.T) {
	db, _, _, cleanup := setupMultiPartitionDB(t, "sp-commit", 2)
	defer cleanup()
	// Both docs on same partition: docID 0 and 2 -> partition 0
	tx := db.Begin()
	if err := db.AddOpToTx(tx, "_default", types.OpCreate, 0, []byte(`{"x":0}`)); err != nil {
		t.Fatalf("AddOpToTx: %v", err)
	}
	if err := db.AddOpToTx(tx, "_default", types.OpCreate, 2, []byte(`{"x":2}`)); err != nil {
		t.Fatalf("AddOpToTx: %v", err)
	}
	if err := db.Commit(tx); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	got0, _ := db.Read("_default", 0)
	got2, _ := db.Read("_default", 2)
	if string(got0) != `{"x":0}` || string(got2) != `{"x":2}` {
		t.Fatalf("single-partition commit: got %s, %s", got0, got2)
	}
}

// TestMultiPartition_EmptyTx commits an empty transaction (no-op).
func TestMultiPartition_EmptyTx(t *testing.T) {
	db, _, _, cleanup := setupMultiPartitionDB(t, "mp-empty", 2)
	defer cleanup()
	tx := db.Begin()
	if err := db.Commit(tx); err != nil {
		t.Fatalf("Commit empty tx: %v", err)
	}
}

// TestMultiPartition_ConcurrentNoDeadlock runs two concurrent multi-partition transactions
// touching overlapping partition sets (different doc order). Both must commit successfully;
// deterministic lock order by partition ID prevents deadlock.
func TestMultiPartition_ConcurrentNoDeadlock(t *testing.T) {
	db, _, _, cleanup := setupMultiPartitionDB(t, "mp-concurrent", 2)
	defer cleanup()
	// With 2 partitions: doc 1,3 -> p1; doc 2,4 -> p0. Both tx touch p0 and p1; use non-overlapping docs so both commit.
	var wg sync.WaitGroup
	wg.Add(2)
	// Tx1: doc 1 then doc 2 (partition order sorted [0,1])
	go func() {
		defer wg.Done()
		tx := db.Begin()
		_ = db.AddOpToTx(tx, "_default", types.OpCreate, 1, []byte(`{"tx":1}`))
		_ = db.AddOpToTx(tx, "_default", types.OpCreate, 2, []byte(`{"tx":1}`))
		if err := db.Commit(tx); err != nil {
			t.Errorf("tx1 commit: %v", err)
		}
	}()
	// Tx2: doc 2 then doc 1 (different op order; partition order still [0,1])
	go func() {
		defer wg.Done()
		tx := db.Begin()
		_ = db.AddOpToTx(tx, "_default", types.OpCreate, 4, []byte(`{"tx":2}`))
		_ = db.AddOpToTx(tx, "_default", types.OpCreate, 3, []byte(`{"tx":2}`))
		if err := db.Commit(tx); err != nil {
			t.Errorf("tx2 commit: %v", err)
		}
	}()
	wg.Wait()
	// Verify both committed: docs 1,2 from tx1 and 3,4 from tx2
	for docID, want := range map[uint64]string{1: `{"tx":1}`, 2: `{"tx":1}`, 3: `{"tx":2}`, 4: `{"tx":2}`} {
		got, err := db.Read("_default", docID)
		if err != nil {
			t.Errorf("read doc %d: %v", docID, err)
			continue
		}
		if string(got) != want {
			t.Errorf("doc %d: got %s want %s", docID, got, want)
		}
	}
}

// TestReadInTx_SeesPendingWrites verifies ReadInTx sees the transaction's own pending writes.
func TestReadInTx_SeesPendingWrites(t *testing.T) {
	db, _, _, cleanup := setupMultiPartitionDB(t, "readintx", 2)
	defer cleanup()
	tx := db.Begin()
	if err := db.AddOpToTx(tx, "_default", types.OpCreate, 10, []byte(`{"v":1}`)); err != nil {
		t.Fatalf("AddOpToTx: %v", err)
	}
	// ReadInTx before commit must see pending create
	got, err := db.ReadInTx(tx, "_default", 10)
	if err != nil {
		t.Fatalf("ReadInTx: %v", err)
	}
	if string(got) != `{"v":1}` {
		t.Fatalf("ReadInTx: got %s", got)
	}
	// Normal Read must not see uncommitted doc
	_, err = db.Read("_default", 10)
	if err == nil {
		t.Fatal("Read should not see uncommitted doc")
	}
	if err := db.Commit(tx); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	got, _ = db.Read("_default", 10)
	if string(got) != `{"v":1}` {
		t.Fatalf("after commit: got %s", got)
	}
}

// TestReadInTx_AfterRollback verifies ReadInTx and Read do not see rolled-back writes.
func TestReadInTx_AfterRollback(t *testing.T) {
	db, _, _, cleanup := setupMultiPartitionDB(t, "readintx-rollback", 2)
	defer cleanup()
	tx := db.Begin()
	_ = db.AddOpToTx(tx, "_default", types.OpCreate, 20, []byte(`{"x":1}`))
	got, _ := db.ReadInTx(tx, "_default", 20)
	if string(got) != `{"x":1}` {
		t.Fatalf("ReadInTx before rollback: got %s", got)
	}
	if err := db.Rollback(tx); err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	// ReadInTx on rolled-back tx should error (tx not open)
	_, err := db.ReadInTx(tx, "_default", 20)
	if err == nil {
		t.Fatal("ReadInTx after rollback should error")
	}
	// Normal Read must not see the doc (never committed)
	_, err = db.Read("_default", 20)
	if err == nil {
		t.Fatal("Read should not see rolled-back doc")
	}
}

// TestReadInTx_MultiPartition pending write on one partition, ReadInTx for that doc sees it.
func TestReadInTx_MultiPartition(t *testing.T) {
	db, _, _, cleanup := setupMultiPartitionDB(t, "readintx-mp", 2)
	defer cleanup()
	// doc 1 -> p1, doc 2 -> p0; create both in one tx
	tx := db.Begin()
	_ = db.AddOpToTx(tx, "_default", types.OpCreate, 1, []byte(`{"p":1}`))
	_ = db.AddOpToTx(tx, "_default", types.OpCreate, 2, []byte(`{"p":2}`))
	// ReadInTx for doc 1 and 2 must see pending
	got1, err := db.ReadInTx(tx, "_default", 1)
	if err != nil {
		t.Fatalf("ReadInTx doc 1: %v", err)
	}
	if string(got1) != `{"p":1}` {
		t.Fatalf("doc 1: got %s", got1)
	}
	got2, err := db.ReadInTx(tx, "_default", 2)
	if err != nil {
		t.Fatalf("ReadInTx doc 2: %v", err)
	}
	if string(got2) != `{"p":2}` {
		t.Fatalf("doc 2: got %s", got2)
	}
	if err := db.Commit(tx); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	got1, _ = db.Read("_default", 1)
	got2, _ = db.Read("_default", 2)
	if string(got1) != `{"p":1}` || string(got2) != `{"p":2}` {
		t.Fatalf("after commit: %s %s", got1, got2)
	}
}

// TestSSI_ConflictTwoReadSameDocThenWrite: Tx2 begins first (snapshot 0), Tx1 begins and commits (txID 1), Tx2 commits.
// Tx2 read doc1 and Tx1 wrote doc1 after Tx2's snapshot, so Tx2 must get ErrSerializationFailure.
func TestSSI_ConflictTwoReadSameDocThenWrite(t *testing.T) {
	db, _, _, cleanup := setupMultiPartitionDB(t, "ssi-conflict", 2)
	defer cleanup()
	// Create doc 1 so both can read it
	tx0 := db.Begin()
	_ = db.AddOpToTx(tx0, "_default", types.OpCreate, 1, []byte(`{"v":0}`))
	if err := db.Commit(tx0); err != nil {
		t.Fatalf("setup commit: %v", err)
	}
	// Tx2 begins first (gets snapshot 0 / txID 2)
	tx2 := db.Begin()
	_, _ = db.ReadInTx(tx2, "_default", 1)
	_ = db.AddOpToTx(tx2, "_default", types.OpCreate, 2, []byte(`{"v":2}`))
	// Tx1 begins second (snapshot 1 / txID 1), commits
	tx1 := db.Begin()
	_, _ = db.ReadInTx(tx1, "_default", 1)
	_ = db.AddOpToTx(tx1, "_default", types.OpUpdate, 1, []byte(`{"v":1}`))
	if err := db.Commit(tx1); err != nil {
		t.Fatalf("tx1 commit: %v", err)
	}
	// Tx2 commits: CommitsAfter(Tx2.SnapshotTxID) includes Tx1; Tx2 read doc1, Tx1 wrote doc1 -> conflict
	err2 := db.Commit(tx2)
	if err2 == nil {
		t.Fatal("tx2 commit should fail with ErrSerializationFailure")
	}
	if err2 != docdb.ErrSerializationFailure {
		t.Errorf("tx2 commit: want ErrSerializationFailure, got %v", err2)
	}
}

// TestSSI_NoFalsePositive: two tx with non-overlapping read/write sets both commit.
func TestSSI_NoFalsePositive(t *testing.T) {
	db, _, _, cleanup := setupMultiPartitionDB(t, "ssi-noconflict", 2)
	defer cleanup()
	// Create doc 1 and doc 3 so each tx can read a different doc
	tx0 := db.Begin()
	_ = db.AddOpToTx(tx0, "_default", types.OpCreate, 1, []byte(`{"v":0}`))
	_ = db.AddOpToTx(tx0, "_default", types.OpCreate, 3, []byte(`{"v":0}`))
	if err := db.Commit(tx0); err != nil {
		t.Fatalf("setup commit: %v", err)
	}
	tx1 := db.Begin()
	_, _ = db.ReadInTx(tx1, "_default", 1)
	_ = db.AddOpToTx(tx1, "_default", types.OpUpdate, 1, []byte(`{"v":1}`))
	if err := db.Commit(tx1); err != nil {
		t.Fatalf("tx1 commit: %v", err)
	}
	tx2 := db.Begin()
	_, _ = db.ReadInTx(tx2, "_default", 3)
	_ = db.AddOpToTx(tx2, "_default", types.OpUpdate, 3, []byte(`{"v":3}`))
	if err := db.Commit(tx2); err != nil {
		t.Fatalf("tx2 commit: %v", err)
	}
	// Both should have committed; no conflict
	got1, _ := db.Read("_default", 1)
	got3, _ := db.Read("_default", 3)
	if string(got1) != `{"v":1}` || string(got3) != `{"v":3}` {
		t.Fatalf("after commits: doc1=%s doc3=%s", got1, got3)
	}
}

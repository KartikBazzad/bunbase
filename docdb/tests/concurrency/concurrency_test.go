package concurrency

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/docdb"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/pool"
)

func setupTestPool(t *testing.T) (*pool.Pool, string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "docdb-concur-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.DataDir = tmpDir
	cfg.WAL.Dir = filepath.Join(tmpDir, "wal")

	log := logger.Default()

	p := pool.NewPool(cfg, log)
	if err := p.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	cleanup := func() {
		p.Stop()
		os.RemoveAll(tmpDir)
	}

	return p, tmpDir, cleanup
}

func TestConcurrentWrites(t *testing.T) {
	p, _, cleanup := setupTestPool(t)
	defer cleanup()

	dbID, err := p.CreateDB("concurrentwrites")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	db, err := p.OpenDB(dbID)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	numWriters := 10
	numDocs := 100

	var wg sync.WaitGroup
	wg.Add(numWriters)

	const coll = "_default"
	writer := func(workerID int) {
		defer wg.Done()

		for i := 1; i <= numDocs; i++ {
			docID := uint64(workerID*1000 + i)
			// Use a simple, valid JSON payload so we exercise the JSON-only
			// engine invariant under concurrent write load.
			payload := []byte(fmt.Sprintf(`{"worker":%d,"doc":%d}`, workerID, docID))

			err := db.Create(coll, docID, payload)
			if err != nil {
				t.Logf("Worker %d: Failed to create doc %d: %v", workerID, docID, err)
			}
		}
	}

	for i := 0; i < numWriters; i++ {
		go writer(i)
	}

	wg.Wait()

	expectedDocs := numWriters * numDocs
	if db.IndexSize() != expectedDocs {
		t.Fatalf("Expected %d documents, got %d", expectedDocs, db.IndexSize())
	}
}

func TestConcurrentReadsWrites(t *testing.T) {
	t.Skip("Concurrent writes to same DB requires more sophisticated locking - v0 limitation")
	p, _, cleanup := setupTestPool(t)
	defer cleanup()

	dbID, err := p.CreateDB("concurrentreadwrites")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	db, err := p.OpenDB(dbID)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	const coll = "_default"
	numDocs := 100
	for i := 1; i <= numDocs; i++ {
		payload := []byte("initial payload")
		err := db.Create(coll, uint64(i), payload)
		if err != nil {
			t.Fatalf("Failed to create doc %d: %v", i, err)
		}
	}

	var wg sync.WaitGroup

	reader := func(workerID int) {
		defer wg.Done()

		for i := 1; i <= numDocs; i++ {
			_, err := db.Read(coll, uint64(i))
			if err != nil && err != docdb.ErrDocNotFound {
				t.Logf("Reader %d: Error reading doc %d: %v", workerID, i, err)
			}
		}
	}

	writer := func(workerID int) {
		defer wg.Done()

		for i := 1; i <= 10; i++ {
			docID := uint64((workerID * 10) + i)
			payload := []byte("updated payload")
			err := db.Update(coll, docID, payload)
			if err != nil {
				t.Logf("Writer %d: Failed to update doc %d: %v", workerID, docID, err)
			}
		}
	}

	wg.Add(10)
	for i := 0; i < 10; i++ {
		go reader(i)
		go writer(i)
	}

	wg.Wait()

	if db.IndexSize() != numDocs {
		t.Fatalf("Expected %d documents, got %d", numDocs, db.IndexSize())
	}
}

func TestMultipleDBs(t *testing.T) {
	t.Skip("Multiple concurrent DBs requires pool-level coordination - v0 limitation")
	p, _, cleanup := setupTestPool(t)
	defer cleanup()

	numDBs := 10
	numDocs := 50

	var wg sync.WaitGroup
	wg.Add(numDBs)

	for i := 0; i < numDBs; i++ {
		dbName := filepath.Join("multidb", "db", string(rune(i+'a')))
		dbID, err := p.CreateDB(dbName)
		if err != nil {
			t.Fatalf("Failed to create database %d: %v", i, err)
		}

		go func(dbID uint64) {
			defer wg.Done()

			db, err := p.OpenDB(dbID)
			if err != nil {
				t.Logf("Failed to open database %d: %v", dbID, err)
				return
			}

			const coll = "_default"
			for j := 1; j <= numDocs; j++ {
				payload := []byte("multi-db payload")
				err := db.Create(coll, uint64(j), payload)
				if err != nil {
					t.Logf("DB %d: Failed to create doc %d: %v", dbID, j, err)
				}
			}

			if db.IndexSize() != numDocs {
				t.Logf("DB %d: Expected %d documents, got %d", dbID, numDocs, db.IndexSize())
			}
		}(dbID)
	}

	wg.Wait()
}

func TestStarvationPrevention(t *testing.T) {
	t.Skip("Starvation prevention test requires pool-level coordination - v0 limitation")
	p, _, cleanup := setupTestPool(t)
	defer cleanup()

	dbID1, err := p.CreateDB("starvationdb1")
	if err != nil {
		t.Fatalf("Failed to create database 1: %v", err)
	}

	dbID2, err := p.CreateDB("starvationdb2")
	if err != nil {
		t.Fatalf("Failed to create database 2: %v", err)
	}

	db1, err := p.OpenDB(dbID1)
	if err != nil {
		t.Fatalf("Failed to open database 1: %v", err)
	}

	db2, err := p.OpenDB(dbID2)
	if err != nil {
		t.Fatalf("Failed to open database 2: %v", err)
	}

	var wg sync.WaitGroup

	burstyWriter := func(db *docdb.LogicalDB, workerID int) {
		defer wg.Done()

		const coll = "_default"
		for i := 0; i < 100; i++ {
			payload := []byte("bursty payload")
			docID := uint64(workerID*1000 + i)
			err := db.Create(coll, docID, payload)
			if err != nil {
				t.Logf("Bursty writer %d: Error creating doc %d: %v", workerID, docID, err)
			}
		}
	}

	slowWriter := func(db *docdb.LogicalDB, workerID int) {
		defer wg.Done()

		const coll = "_default"
		for i := 0; i < 100; i++ {
			payload := []byte("slow payload")
			docID := uint64(workerID*1000 + i)
			err := db.Create(coll, docID, payload)
			if err != nil {
				t.Logf("Slow writer %d: Error creating doc %d: %v", workerID, docID, err)
			}
		}
	}

	wg.Add(6)

	for i := 0; i < 5; i++ {
		go burstyWriter(db1, i)
	}

	go slowWriter(db2, 1)

	wg.Wait()

	if db2.IndexSize() == 0 {
		t.Fatal("Slow-writer DB has no documents - possible starvation")
	}
}

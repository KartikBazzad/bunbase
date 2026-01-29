package integration

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/pool"
	"github.com/kartikbazzad/docdb/internal/wal"
)

func createTestLogger(t *testing.T) *logger.Logger {
	return logger.Default()
}

// TestWALRotation tests WAL rotation at size threshold
func TestWALRotation(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "docdb-rotation-test-*")
	defer os.RemoveAll(tmpDir)

	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir

	cfg.WAL.MaxFileSizeMB = 1 // Rotate at 1MB
	log := createTestLogger(t)

	p := pool.NewPool(cfg, log)
	p.Start()
	defer p.Stop()

	dbID, err := p.CreateDB("testdb")
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	db, err := p.OpenDB(dbID)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Write enough data to trigger rotation.
	// Use a large, but valid, JSON string payload.
	const coll = "_default"
	payload := []byte(`{"data":"` + strings.Repeat("x", 100*1024-30) + `"}`) // ~100KB JSON

	for i := 0; i < 15; i++ {
		docID := uint64(i + 1)
		err = db.Create(coll, docID, payload)
		if err != nil {
			t.Fatalf("failed to create document %d: %v", docID, err)
		}
	}

	// Check for rotated segments
	rotator := wal.NewRotator(filepath.Join(walDir, "testdb.wal"), cfg.WAL.MaxFileSizeMB*1024*1024, false, log)
	segments, err := rotator.ListSegments()
	if err != nil {
		t.Fatalf("failed to list segments: %v", err)
	}

	if len(segments) == 0 {
		t.Errorf("expected WAL segments, got none")
	}

	t.Logf("Found %d WAL segments", len(segments))
}

// TestMultiSegmentRecovery tests recovery from multiple WAL segments
func TestMultiSegmentRecovery(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "docdb-multisegment-test-*")
	defer os.RemoveAll(tmpDir)

	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir

	cfg.WAL.MaxFileSizeMB = 1 // Rotate at 1MB
	log := createTestLogger(t)

	// Phase 1: Create database and write data
	p1 := pool.NewPool(cfg, log)
	p1.Start()

	dbID1, err := p1.CreateDB("testdb")
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	db1, err := p1.OpenDB(dbID1)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Large, valid JSON payload for multi-segment recovery.
	const coll = "_default"
	payload := []byte(`{"data":"` + strings.Repeat("x", 100*1024-30) + `"}`)

	for i := 0; i < 15; i++ {
		docID := uint64(i + 1)
		err = db1.Create(coll, docID, payload)
		if err != nil {
			t.Fatalf("failed to create document %d: %v", docID, err)
		}
	}

	p1.Stop()

	// Phase 2: Simulate crash and restart
	t.Log("Simulating crash and restart...")

	p2 := pool.NewPool(cfg, log)
	p2.Start()
	defer p2.Stop()

	db2, err := p2.OpenDB(dbID1)
	if err != nil {
		t.Fatalf("failed to open database after restart: %v", err)
	}

	// Verify all documents are recoverable
	for i := 0; i < 15; i++ {
		docID := uint64(i + 1)
		data, err := db2.Read(coll, docID)
		if err != nil {
			t.Errorf("failed to read document %d after recovery: %v", docID, err)
		}

		if len(data) != len(payload) {
			t.Errorf("document %d has incorrect size after recovery: got %d, want %d", docID, len(data), len(payload))
		}
	}

	t.Log("All documents successfully recovered from multi-segment WAL")
}

// TestRotationDuringCrash tests rotation safety during crash
func TestRotationDuringCrash(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "docdb-crash-rotation-test-*")
	defer os.RemoveAll(tmpDir)

	dataDir := filepath.Join(tmpDir, "data")
	walDir := filepath.Join(tmpDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir

	cfg.WAL.MaxFileSizeMB = 1
	log := createTestLogger(t)

	p := pool.NewPool(cfg, log)
	p.Start()

	dbID, err := p.CreateDB("testdb")
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	db, err := p.OpenDB(dbID)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Write data approaching rotation threshold using valid JSON payloads.
	const coll = "_default"
	payload := []byte(`{"data":"` + strings.Repeat("x", 800*1024-30) + `"}`) // ~800KB

	for i := 0; i < 10; i++ {
		docID := uint64(i + 1)
		err = db.Create(coll, docID, payload)
		if err != nil {
			t.Fatalf("failed to create document %d: %v", docID, err)
		}
	}

	// Get WAL size before potential rotation (v0.4: walDir/dbName/p0.wal)
	walPath := filepath.Join(walDir, "testdb", "p0.wal")
	walInfo, err := os.Stat(walPath)
	if err != nil {
		t.Fatalf("failed to stat WAL: %v", err)
	}
	sizeBefore := walInfo.Size()

	t.Logf("WAL size before crash: %d bytes", sizeBefore)

	// Simulate crash
	p.Stop()

	// Verify recovery works
	p2 := pool.NewPool(cfg, log)
	p2.Start()
	defer p2.Stop()

	db2, err := p2.OpenDB(dbID)
	if err != nil {
		t.Fatalf("failed to open database after crash: %v", err)
	}

	// Verify documents are recoverable
	for i := 0; i < 10; i++ {
		docID := uint64(i + 1)
		data, err := db2.Read(coll, docID)
		if err != nil {
			t.Errorf("failed to read document %d after crash recovery: %v", docID, err)
		}

		if len(data) != len(payload) {
			t.Errorf("document %d has incorrect size: got %d, want %d", docID, len(data), len(payload))
		}
	}

	t.Log("Recovery successful after rotation-crash scenario")
}

// TestSegmentNaming tests that segments are named correctly
func TestSegmentNaming(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "docdb-segment-naming-test-*")
	defer os.RemoveAll(tmpDir)

	walDir := filepath.Join(tmpDir, "wal")
	if err := os.MkdirAll(walDir, 0755); err != nil {
		t.Fatalf("failed to create WAL dir: %v", err)
	}

	log := createTestLogger(t)
	walPath := filepath.Join(walDir, "testdb.wal")
	rotator := wal.NewRotator(walPath, 64*1024*1024, false, log)

	// Create initial WAL
	file, err := os.Create(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	file.Close()

	// Perform multiple rotations
	for i := 1; i <= 5; i++ {
		_, err := rotator.Rotate()
		if err != nil {
			t.Fatalf("rotation %d failed: %v", i, err)
		}

		// Create new file for next rotation
		file, err := os.Create(walPath)
		if err != nil {
			t.Fatalf("failed to create WAL after rotation %d: %v", i, err)
		}
		file.Close()
	}

	// Verify segments exist
	segments, err := rotator.ListSegments()
	if err != nil {
		t.Fatalf("failed to list segments: %v", err)
	}

	if len(segments) != 5 {
		t.Errorf("expected 5 segments, got %d", len(segments))
	}

	// Verify segment names
	expectedSuffixes := []string{".wal.1", ".wal.2", ".wal.3", ".wal.4", ".wal.5"}
	for i, seg := range segments {
		baseName := filepath.Base(seg)
		expectedName := "testdb" + expectedSuffixes[i]
		if baseName != expectedName {
			t.Errorf("segment %d: expected name %s, got %s", i, expectedName, baseName)
		}
	}

	t.Logf("Segment naming test passed with %d segments", len(segments))
}

// TestSegmentOrdering tests that segments are returned in correct order
func TestSegmentOrdering(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "docdb-segment-ordering-test-*")
	defer os.RemoveAll(tmpDir)

	walDir := filepath.Join(tmpDir, "wal")
	if err := os.MkdirAll(walDir, 0755); err != nil {
		t.Fatalf("failed to create WAL dir: %v", err)
	}

	log := createTestLogger(t)
	walPath := filepath.Join(walDir, "testdb.wal")
	rotator := wal.NewRotator(walPath, 64*1024*1024, false, log)

	// Create segments in reverse order
	for i := 5; i >= 1; i-- {
		segPath := filepath.Join(walDir, "testdb.wal.1."+strconv.Itoa(i))
		file, err := os.Create(segPath)
		if err != nil {
			t.Fatalf("failed to create segment %s: %v", segPath, err)
		}
		file.Close()
	}

	// Verify segments are returned in ascending order
	segments, err := rotator.ListSegments()
	if err != nil {
		t.Fatalf("failed to list segments: %v", err)
	}

	t.Logf("Rotator returned %d segments: %v", len(segments), segments)

	if len(segments) != 5 {
		t.Errorf("expected 5 segments, got %d", len(segments))
	}

	// Verify ordering by checking segment numbers
	for i, seg := range segments {
		baseName := filepath.Base(seg)
		t.Logf("Segment %d: %s", i, baseName)

		expectedSuffix := "." + strconv.Itoa(i+1)
		if filepath.Ext(baseName) != expectedSuffix {
			t.Errorf("segment %d: expected suffix %s, got %s", i, expectedSuffix, filepath.Ext(baseName))
		}
	}

	t.Log("Segment ordering test passed")
}

// TestActiveWALInclusion tests that active WAL is included in GetAllWALPaths
func TestActiveWALInclusion(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "docdb-active-wal-test-*")
	defer os.RemoveAll(tmpDir)

	walDir := filepath.Join(tmpDir, "wal")
	if err := os.MkdirAll(walDir, 0755); err != nil {
		t.Fatalf("failed to create WAL dir: %v", err)
	}

	log := createTestLogger(t)
	walPath := filepath.Join(walDir, "testdb.wal")
	rotator := wal.NewRotator(walPath, 64*1024*1024, false, log)

	// Create active WAL
	activeFile, err := os.Create(walPath)
	if err != nil {
		t.Fatalf("failed to create active WAL: %v", err)
	}
	activeFile.Close()

	// Create a rotated segment
	segFile, err := os.Create(walPath + ".wal.1")
	if err != nil {
		t.Fatalf("failed to create rotated segment: %v", err)
	}
	segFile.Close()

	// Get all WAL paths
	allPaths, err := rotator.GetAllWALPaths()
	if err != nil {
		t.Fatalf("failed to get all WAL paths: %v", err)
	}

	if len(allPaths) != 2 {
		t.Errorf("expected 2 WAL paths, got %d", len(allPaths))
	}

	// Verify active WAL is last
	if allPaths[1] != walPath {
		t.Errorf("expected active WAL at index 1, got %s", allPaths[1])
	}

	t.Log("Active WAL inclusion test passed")
}

// TestInvalidSegmentNames tests that non-segment files are ignored
func TestInvalidSegmentNames(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "docdb-invalid-names-test-*")
	defer os.RemoveAll(tmpDir)

	walDir := filepath.Join(tmpDir, "wal")
	if err := os.MkdirAll(walDir, 0755); err != nil {
		t.Fatalf("failed to create WAL dir: %v", err)
	}

	log := createTestLogger(t)
	walPath := filepath.Join(walDir, "testdb.wal")
	rotator := wal.NewRotator(walPath, 64*1024*1024, false, log)

	// Create various files that should be ignored
	ignoredFiles := []string{
		"testdb.wal.bak",
		"testdb.wal.old",
		"testdb.wal.tmp",
		"testdb.wal.1a",
		"testdb.wal.1.txt",
		"otherdb.wal",
		"testdb.data",
	}

	for _, name := range ignoredFiles {
		filePath := filepath.Join(walDir, name)
		file, err := os.Create(filePath)
		if err != nil {
			t.Fatalf("failed to create file %s: %v", name, err)
		}
		file.Close()
	}

	// Create valid segments
	for i := 1; i <= 3; i++ {
		segPath := walPath + ".wal." + string('0'+byte(i))
		file, err := os.Create(segPath)
		if err != nil {
			t.Fatalf("failed to create segment %s: %v", segPath, err)
		}
		file.Close()
	}

	// Verify only valid segments are returned
	segments, err := rotator.ListSegments()
	if err != nil {
		t.Fatalf("failed to list segments: %v", err)
	}

	if len(segments) != 3 {
		t.Errorf("expected 3 valid segments, got %d", len(segments))
	}

	t.Log("Invalid segment names test passed")
}

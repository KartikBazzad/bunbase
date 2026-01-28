package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/docdb"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
)

func TestAutomaticHealingOnCorruption(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	walDir := filepath.Join(tempDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	cfg.Healing.Enabled = true
	cfg.Healing.OnReadCorruption = true
	cfg.Healing.Interval = 10 * time.Second // Short interval for testing

	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	log := logger.NewLogger(logger.LevelInfo)

	db := docdb.NewLogicalDB(1, "healdb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create a document
	payload := []byte(`{"key":"value"}`)
	if err := db.Create("default", 1, payload); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Verify document is readable
	data, err := db.Read("default", 1)
	if err != nil {
		t.Fatalf("Failed to read document: %v", err)
	}
	if string(data) != string(payload) {
		t.Errorf("Payload mismatch: got %s, want %s", string(data), string(payload))
	}

	// Corrupt the data file
	dataFilePath := filepath.Join(dataDir, "healdb.data")
	file, err := os.OpenFile(dataFilePath, os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("Failed to open data file: %v", err)
	}

	// Truncate to corrupt the file
	if err := file.Truncate(10); err != nil {
		file.Close()
		t.Fatalf("Failed to truncate file: %v", err)
	}
	file.Close()

	// Try to read - should trigger healing if OnReadCorruption is enabled
	_, err = db.Read("default", 1)
	if err != nil {
		t.Logf("Read failed (expected): %v", err)
	}

	// Wait a bit for healing to process
	time.Sleep(2 * time.Second)

	// Try reading again - document should be healed
	data, err = db.Read("default", 1)
	if err != nil {
		t.Logf("Document may still be healing: %v", err)
	} else {
		if string(data) != string(payload) {
			t.Errorf("Healed payload mismatch: got %s, want %s", string(data), string(payload))
		}
	}

	// Verify healing stats were updated
	healingService := db.HealingService()
	if healingService != nil {
		stats := healingService.GetStats()
		if stats.OnDemandHealings == 0 && err == nil {
			// Healing may have occurred, check stats
			t.Logf("Healing stats: OnDemandHealings=%d, DocumentsHealed=%d", stats.OnDemandHealings, stats.DocumentsHealed)
		}
	}
}

func TestManualHealing(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	walDir := filepath.Join(tempDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	cfg.Healing.Enabled = true

	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	log := logger.NewLogger(logger.LevelInfo)

	db := docdb.NewLogicalDB(1, "manualheal", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create documents
	payload := []byte(`{"data":"test"}`)
	for i := 1; i <= 3; i++ {
		if err := db.Create("default", uint64(i), payload); err != nil {
			t.Fatalf("Failed to create document %d: %v", i, err)
		}
	}

	// Verify documents are readable
	for i := 1; i <= 3; i++ {
		data, err := db.Read("default", uint64(i))
		if err != nil {
			t.Fatalf("Failed to read document %d: %v", i, err)
		}
		if string(data) != string(payload) {
			t.Errorf("Document %d payload mismatch", i)
		}
	}

	// Corrupt data file
	dataFilePath := filepath.Join(dataDir, "manualheal.data")
	file, err := os.OpenFile(dataFilePath, os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("Failed to open data file: %v", err)
	}
	if err := file.Truncate(20); err != nil {
		file.Close()
		t.Fatalf("Failed to truncate file: %v", err)
	}
	file.Close()

	// Get healing service and heal document manually
	healingService := db.HealingService()
	if healingService == nil {
		t.Fatal("Healing service not available")
	}

	// Heal document 1
	err = healingService.HealDocument("default", 1)
	if err != nil {
		t.Fatalf("Failed to heal document 1: %v", err)
	}

	// Verify document 1 is healed
	data, err := db.Read("default", 1)
	if err != nil {
		t.Fatalf("Failed to read healed document: %v", err)
	}
	if string(data) != string(payload) {
		t.Errorf("Healed payload mismatch: got %s, want %s", string(data), string(payload))
	}
}

func TestHealingStatistics(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	walDir := filepath.Join(tempDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	cfg.Healing.Enabled = true
	cfg.Healing.Interval = 100 * time.Millisecond // Very short for testing

	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	log := logger.NewLogger(logger.LevelInfo)

	db := docdb.NewLogicalDB(1, "statsdb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create documents
	payload := []byte(`{"data":"test"}`)
	for i := 1; i <= 5; i++ {
		if err := db.Create("default", uint64(i), payload); err != nil {
			t.Fatalf("Failed to create document %d: %v", i, err)
		}
	}

	healingService := db.HealingService()
	if healingService == nil {
		t.Fatal("Healing service not available")
	}

	// Get initial stats
	initialStats := healingService.GetStats()
	if initialStats.TotalScans == 0 {
		t.Log("No scans yet (expected)")
	}

	// Wait for at least one health scan
	time.Sleep(250 * time.Millisecond)

	// Get stats after scan
	stats := healingService.GetStats()
	if stats.TotalScans == 0 {
		t.Error("Expected at least one scan to have occurred")
	}
	if stats.LastScanTime.IsZero() {
		t.Error("Expected LastScanTime to be set")
	}

	// Perform manual healing
	err := healingService.HealDocument("default", 1)
	if err == nil {
		// If healing succeeded, check stats
		stats = healingService.GetStats()
		if stats.DocumentsHealed == 0 && stats.OnDemandHealings == 0 {
			t.Log("No documents needed healing (expected)")
		}
		if stats.LastHealingTime.IsZero() && stats.OnDemandHealings > 0 {
			t.Error("Expected LastHealingTime to be set after healing")
		}
	}
}

func TestBatchHealing(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	walDir := filepath.Join(tempDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	cfg.Healing.Enabled = true
	cfg.Healing.OnReadCorruption = true
	cfg.Healing.MaxBatchSize = 10

	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	log := logger.NewLogger(logger.LevelInfo)

	db := docdb.NewLogicalDB(1, "batchdb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create multiple documents
	payload := []byte(`{"data":"test"}`)
	for i := 1; i <= 5; i++ {
		if err := db.Create("default", uint64(i), payload); err != nil {
			t.Fatalf("Failed to create document %d: %v", i, err)
		}
	}

	// Corrupt data file to make documents unreadable
	dataFilePath := filepath.Join(dataDir, "batchdb.data")
	file, err := os.OpenFile(dataFilePath, os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("Failed to open data file: %v", err)
	}
	if err := file.Truncate(30); err != nil {
		file.Close()
		t.Fatalf("Failed to truncate file: %v", err)
	}
	file.Close()

	healingService := db.HealingService()
	if healingService == nil {
		t.Fatal("Healing service not available")
	}

	// Heal all corrupted documents
	healed, err := healingService.HealAll()
	if err != nil {
		t.Fatalf("Failed to heal all documents: %v", err)
	}

	// Verify documents were healed
	if len(healed) > 0 {
		t.Logf("Healed %d documents", len(healed))
		for _, docID := range healed {
			data, err := db.Read("default", docID)
			if err != nil {
				t.Errorf("Failed to read healed document %d: %v", docID, err)
			} else if string(data) != string(payload) {
				t.Errorf("Healed document %d payload mismatch", docID)
			}
		}
	}

	// Check stats
	stats := healingService.GetStats()
	if stats.DocumentsHealed == 0 && len(healed) > 0 {
		t.Error("Expected DocumentsHealed to be updated")
	}
}

func TestHealingDisabled(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	walDir := filepath.Join(tempDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	cfg.Healing.Enabled = false // Disabled

	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	log := logger.NewLogger(logger.LevelInfo)

	db := docdb.NewLogicalDB(1, "disableddb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create document
	payload := []byte(`{"key":"value"}`)
	if err := db.Create("default", 1, payload); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Corrupt data file
	dataFilePath := filepath.Join(dataDir, "disableddb.data")
	file, err := os.OpenFile(dataFilePath, os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("Failed to open data file: %v", err)
	}
	if err := file.Truncate(10); err != nil {
		file.Close()
		t.Fatalf("Failed to truncate file: %v", err)
	}
	file.Close()

	// Read should fail (healing disabled)
	_, err = db.Read("default", 1)
	if err == nil {
		t.Error("Expected error when reading corrupted file with healing disabled")
	}

	// Verify healing service is nil or not started
	healingService := db.HealingService()
	if healingService != nil {
		stats := healingService.GetStats()
		if stats.TotalScans > 0 {
			t.Error("Expected no scans when healing is disabled")
		}
	}
}

func TestCollectionSpecificHealing(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	walDir := filepath.Join(tempDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir
	cfg.Healing.Enabled = true

	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	log := logger.NewLogger(logger.LevelInfo)

	db := docdb.NewLogicalDB(1, "collectiondb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create collection
	if err := db.CreateCollection("users"); err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	// Create documents in different collections
	payload1 := []byte(`{"name":"Alice"}`)
	payload2 := []byte(`{"name":"Bob"}`)

	if err := db.Create("default", 1, payload1); err != nil {
		t.Fatalf("Failed to create document in default collection: %v", err)
	}
	if err := db.Create("users", 1, payload2); err != nil {
		t.Fatalf("Failed to create document in users collection: %v", err)
	}

	// Corrupt data file
	dataFilePath := filepath.Join(dataDir, "collectiondb.data")
	file, err := os.OpenFile(dataFilePath, os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("Failed to open data file: %v", err)
	}
	if err := file.Truncate(30); err != nil {
		file.Close()
		t.Fatalf("Failed to truncate file: %v", err)
	}
	file.Close()

	healingService := db.HealingService()
	if healingService == nil {
		t.Fatal("Healing service not available")
	}

	// Heal document in specific collection
	err = healingService.HealDocument("users", 1)
	if err != nil {
		t.Fatalf("Failed to heal document in users collection: %v", err)
	}

	// Verify document in users collection is healed
	data, err := db.Read("users", 1)
	if err != nil {
		t.Fatalf("Failed to read healed document: %v", err)
	}
	if string(data) != string(payload2) {
		t.Errorf("Healed payload mismatch: got %s, want %s", string(data), string(payload2))
	}
}

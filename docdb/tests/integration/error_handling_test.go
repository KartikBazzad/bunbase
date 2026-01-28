package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/docdb"
	"github.com/kartikbazzad/docdb/internal/errors"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
)

func TestErrorClassification(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	walDir := filepath.Join(tempDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir

	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	log := logger.NewLogger(logger.LevelInfo)

	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	classifier := errors.NewClassifier()

	// Test classification of various error types
	testCases := []struct {
		name     string
		err      error
		expected errors.ErrorCategory
	}{
		{"InvalidJSON", errors.ErrInvalidJSON, errors.ErrorValidation},
		{"CorruptRecord", errors.ErrCorruptRecord, errors.ErrorValidation},
		{"CRCMismatch", errors.ErrCRCMismatch, errors.ErrorValidation},
		{"DocNotFound", errors.ErrDocNotFound, errors.ErrorPermanent},
		{"DocExists", errors.ErrDocExists, errors.ErrorPermanent},
		{"MemoryLimit", errors.ErrMemoryLimit, errors.ErrorPermanent},
		{"PayloadTooLarge", errors.ErrPayloadTooLarge, errors.ErrorPermanent},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			category := classifier.Classify(tc.err)
			if category != tc.expected {
				t.Errorf("Expected category %v, got %v", tc.expected, category)
			}
		})
	}
}

func TestRetryLogic(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	walDir := filepath.Join(tempDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir

	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	log := logger.NewLogger(logger.LevelInfo)

	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test that permanent errors are not retried
	classifier := errors.NewClassifier()
	retryCtrl := errors.NewRetryController()

	attempts := 0
	err := retryCtrl.Retry(func() error {
		attempts++
		return errors.ErrDocNotFound // Permanent error
	}, classifier)

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt for permanent error, got %d", attempts)
	}

	// Test that validation errors are not retried
	attempts = 0
	err = retryCtrl.Retry(func() error {
		attempts++
		return errors.ErrInvalidJSON // Validation error
	}, classifier)

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt for validation error, got %d", attempts)
	}
}

func TestErrorTracking(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	walDir := filepath.Join(tempDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir

	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	log := logger.NewLogger(logger.LevelInfo)

	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create a document
	payload := []byte(`{"key":"value"}`)
	if err := db.Create(1, payload); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Try to create duplicate (should error)
	err := db.Create(1, payload)
	if err == nil {
		t.Error("Expected error for duplicate document")
	}

	// Error should be tracked (we can't directly access the tracker,
	// but the operation should complete without panic)
	if err != errors.ErrDocExists {
		t.Errorf("Expected ErrDocExists, got %v", err)
	}
}

func TestFileOperationRetry(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	walDir := filepath.Join(tempDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir

	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	log := logger.NewLogger(logger.LevelInfo)

	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create a document - this exercises retry logic in datafile write
	payload := []byte(`{"test":"data"}`)
	if err := db.Create(1, payload); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Read it back - this exercises retry logic in datafile read
	data, err := db.Read(1)
	if err != nil {
		t.Fatalf("Failed to read document: %v", err)
	}

	if string(data) != string(payload) {
		t.Errorf("Expected %s, got %s", string(payload), string(data))
	}
}

func TestCorruptionErrorTracking(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	walDir := filepath.Join(tempDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir

	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	log := logger.NewLogger(logger.LevelInfo)

	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create a document
	payload := []byte(`{"key":"value"}`)
	if err := db.Create(1, payload); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Corrupt the data file by truncating it
	dataFilePath := filepath.Join(dataDir, "testdb.data")
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

	// Try to read - should detect corruption
	_, err = db.Read(1)
	if err == nil {
		t.Error("Expected error when reading corrupted file")
	}

	// Error should be classified as validation/corruption
	classifier := errors.NewClassifier()
	category := classifier.Classify(err)
	if category != errors.ErrorValidation {
		t.Errorf("Expected ErrorValidation category, got %v", category)
	}
}

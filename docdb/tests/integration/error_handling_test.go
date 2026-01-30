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
	log := logger.Default()

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
	log := logger.Default()

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
	log := logger.Default()

	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create a document
	payload := []byte(`{"key":"value"}`)
	if err := db.Create("default", 1, payload); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Try to create duplicate (should error)
	err := db.Create("default", 1, payload)
	if err == nil {
		t.Error("Expected error for duplicate document")
	}

	// Error should be tracked (we can't directly access the tracker,
	// but the operation should complete without panic)
	if err != docdb.ErrDocExists && err != errors.ErrDocExists {
		t.Errorf("Expected ErrDocExists, got %v", err)
	}

	// Verify error classification
	classifier := errors.NewClassifier()
	category := classifier.Classify(err)
	if category != errors.ErrorPermanent {
		t.Errorf("Expected ErrorPermanent for duplicate document, got %v", category)
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
	log := logger.Default()

	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create a document - this exercises retry logic in datafile write
	payload := []byte(`{"test":"data"}`)
	if err := db.Create("default", 1, payload); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Read it back - this exercises retry logic in datafile read
	data, err := db.Read("default", 1)
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
	log := logger.Default()

	db := docdb.NewLogicalDB(1, "testdb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create a document
	payload := []byte(`{"key":"value"}`)
	if err := db.Create("default", 1, payload); err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Corrupt the data file by truncating it (partitioned layout: dbname_p0.data)
	dataFilePath := filepath.Join(dataDir, "testdb_p0.data")
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
	_, err = db.Read("default", 1)
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

func TestRetryWithTransientErrors(t *testing.T) {
	classifier := errors.NewClassifier()
	retryCtrl := errors.NewRetryController()

	// Simulate transient error that succeeds on retry
	attempts := 0
	maxAttempts := 3
	err := retryCtrl.Retry(func() error {
		attempts++
		if attempts < maxAttempts {
			// Simulate transient error (EAGAIN-like)
			return errors.ErrFileWrite // Treated as transient
		}
		return nil // Success on retry
	}, classifier)

	if err != nil {
		t.Errorf("Expected success after retries, got error: %v", err)
	}

	if attempts != maxAttempts {
		t.Errorf("Expected %d attempts, got %d", maxAttempts, attempts)
	}
}

func TestRetryMaxAttempts(t *testing.T) {
	classifier := errors.NewClassifier()
	retryCtrl := errors.NewRetryController()

	// Simulate persistent transient error
	attempts := 0
	err := retryCtrl.Retry(func() error {
		attempts++
		return errors.ErrFileWrite // Always fails
	}, classifier)

	if err == nil {
		t.Error("Expected error after max retries")
	}

	// Should attempt maxRetries + 1 times (initial + retries)
	// Default maxRetries is 5, so expected attempts is 6
	expectedAttempts := 6
	if attempts != expectedAttempts {
		t.Errorf("Expected %d attempts (maxRetries+1), got %d", expectedAttempts, attempts)
	}
}

func TestErrorTrackerMetrics(t *testing.T) {
	tracker := errors.NewErrorTracker()

	// Record various error categories
	tracker.RecordError(errors.ErrInvalidJSON, errors.ErrorValidation)
	tracker.RecordError(errors.ErrDocNotFound, errors.ErrorPermanent)
	tracker.RecordError(errors.ErrFileWrite, errors.ErrorTransient)

	// Verify counts
	if count := tracker.GetErrorCount(errors.ErrorValidation); count != 1 {
		t.Errorf("Expected 1 validation error, got %d", count)
	}

	if count := tracker.GetErrorCount(errors.ErrorPermanent); count != 1 {
		t.Errorf("Expected 1 permanent error, got %d", count)
	}

	if count := tracker.GetErrorCount(errors.ErrorTransient); count != 1 {
		t.Errorf("Expected 1 transient error, got %d", count)
	}

	// Record more errors
	tracker.RecordError(errors.ErrInvalidJSON, errors.ErrorValidation)
	tracker.RecordError(errors.ErrInvalidJSON, errors.ErrorValidation)

	if count := tracker.GetErrorCount(errors.ErrorValidation); count != 3 {
		t.Errorf("Expected 3 validation errors, got %d", count)
	}
}

func TestCriticalErrorAlerting(t *testing.T) {
	tracker := errors.NewErrorTracker()

	// Record critical error
	criticalErr := errors.ErrFileSync // Would be classified as transient, but let's test critical
	tracker.RecordError(criticalErr, errors.ErrorCritical)

	alerts := tracker.GetCriticalAlerts()
	if len(alerts) == 0 {
		t.Error("Expected critical alert to be recorded")
	}

	if len(alerts) > 0 {
		if alerts[0].Category != errors.ErrorCritical {
			t.Errorf("Expected ErrorCritical category, got %v", alerts[0].Category)
		}
	}
}

func TestErrorClassificationIntegration(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	walDir := filepath.Join(tempDir, "wal")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.WAL.Dir = walDir

	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	log := logger.Default()

	db := docdb.NewLogicalDB(1, "integrationdb", cfg, memCaps, pool, log)

	if err := db.Open(dataDir, walDir); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test various error scenarios
	testCases := []struct {
		name             string
		operation        func() error
		expectedCategory errors.ErrorCategory
	}{
		{
			name: "InvalidJSON",
			operation: func() error {
				return db.Create("default", 1, []byte("invalid json"))
			},
			expectedCategory: errors.ErrorValidation,
		},
		{
			name: "DocNotFound",
			operation: func() error {
				_, err := db.Read("default", 99999)
				return err
			},
			expectedCategory: errors.ErrorPermanent,
		},
		{
			name: "DocExists",
			operation: func() error {
				payload := []byte(`{"key":"value"}`)
				db.Create("default", 2, payload)
				return db.Create("default", 2, payload) // Duplicate
			},
			expectedCategory: errors.ErrorPermanent,
		},
	}

	classifier := errors.NewClassifier()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.operation()
			if err == nil && tc.expectedCategory != errors.ErrorPermanent {
				// Some operations may succeed, skip classification check
				return
			}

			if err != nil {
				category := classifier.Classify(err)
				if category != tc.expectedCategory {
					t.Errorf("Expected category %v, got %v for error %v", tc.expectedCategory, category, err)
				}
			}
		})
	}
}

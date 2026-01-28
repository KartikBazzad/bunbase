package failure

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/docdb"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
)

// CrashTestHelper provides utilities for crash simulation and recovery testing.
type CrashTestHelper struct {
	tempDir string
	t       *testing.T
}

// NewCrashTestHelper creates a new crash test helper.
func NewCrashTestHelper(t *testing.T) *CrashTestHelper {
	tempDir, err := os.MkdirTemp("", "docdb-crash-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	return &CrashTestHelper{
		tempDir: tempDir,
		t:       t,
	}
}

// Cleanup removes the temporary directory.
func (h *CrashTestHelper) Cleanup() {
	os.RemoveAll(h.tempDir)
}

// TempDir returns the temporary directory path.
func (h *CrashTestHelper) TempDir() string {
	return h.tempDir
}

// DataDir returns the data directory path.
func (h *CrashTestHelper) DataDir() string {
	return h.tempDir
}

// WALDir returns the WAL directory path.
func (h *CrashTestHelper) WALDir() string {
	return filepath.Join(h.tempDir, "wal")
}

// CreateDB creates a new database instance.
func (h *CrashTestHelper) CreateDB(dbName string) *docdb.LogicalDB {
	cfg := config.DefaultConfig()
	cfg.DataDir = h.tempDir
	cfg.WAL.Dir = h.WALDir()

	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)

	db := docdb.NewLogicalDB(1, dbName, cfg, memCaps, pool, log)
	if err := db.Open(h.tempDir, h.WALDir()); err != nil {
		h.t.Fatalf("Failed to open database: %v", err)
	}

	return db
}

// ReopenDB reopens a database after a crash.
func (h *CrashTestHelper) ReopenDB(dbName string) *docdb.LogicalDB {
	cfg := config.DefaultConfig()
	cfg.DataDir = h.tempDir
	cfg.WAL.Dir = h.WALDir()

	log := logger.Default()
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB*1024*1024, cfg.Memory.PerDBLimitMB*1024*1024)
	memCaps.RegisterDB(1, cfg.Memory.PerDBLimitMB*1024*1024)
	pool := memory.NewBufferPool(cfg.Memory.BufferSizes)

	db := docdb.NewLogicalDB(1, dbName, cfg, memCaps, pool, log)
	if err := db.Open(h.tempDir, h.WALDir()); err != nil {
		h.t.Fatalf("Failed to reopen database: %v", err)
	}

	return db
}

// WALPath returns the path to the WAL file for a database.
func (h *CrashTestHelper) WALPath(dbName string) string {
	return filepath.Join(h.WALDir(), fmt.Sprintf("%s.wal", dbName))
}

// DataFilePath returns the path to the data file for a database.
func (h *CrashTestHelper) DataFilePath(dbName string) string {
	return filepath.Join(h.tempDir, fmt.Sprintf("%s.data", dbName))
}

// TruncateWAL truncates the WAL file at the specified offset.
func (h *CrashTestHelper) TruncateWAL(dbName string, offset int64) error {
	walPath := h.WALPath(dbName)
	file, err := os.OpenFile(walPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open WAL: %w", err)
	}
	defer file.Close()

	if err := file.Truncate(offset); err != nil {
		return fmt.Errorf("failed to truncate WAL: %w", err)
	}

	return nil
}

// CorruptWAL corrupts the WAL file by writing random bytes at the specified offset.
func (h *CrashTestHelper) CorruptWAL(dbName string, offset int64, length int) error {
	walPath := h.WALPath(dbName)
	file, err := os.OpenFile(walPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open WAL: %w", err)
	}
	defer file.Close()

	if _, err := file.Seek(offset, 0); err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	corruptData := make([]byte, length)
	for i := range corruptData {
		corruptData[i] = 0xFF
	}

	if _, err := file.Write(corruptData); err != nil {
		return fmt.Errorf("failed to write corrupt data: %w", err)
	}

	return nil
}

// TruncateDataFile truncates the data file at the specified offset.
func (h *CrashTestHelper) TruncateDataFile(dbName string, offset int64) error {
	dataPath := h.DataFilePath(dbName)
	file, err := os.OpenFile(dataPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open data file: %w", err)
	}
	defer file.Close()

	if err := file.Truncate(offset); err != nil {
		return fmt.Errorf("failed to truncate data file: %w", err)
	}

	return nil
}

// CorruptDataFile corrupts the data file by writing random bytes at the specified offset.
func (h *CrashTestHelper) CorruptDataFile(dbName string, offset int64, length int) error {
	dataPath := h.DataFilePath(dbName)
	file, err := os.OpenFile(dataPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open data file: %w", err)
	}
	defer file.Close()

	if _, err := file.Seek(offset, 0); err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	corruptData := make([]byte, length)
	for i := range corruptData {
		corruptData[i] = 0xFF
	}

	if _, err := file.Write(corruptData); err != nil {
		return fmt.Errorf("failed to write corrupt data: %w", err)
	}

	return nil
}

// VerifyDocument verifies that a document exists and has the expected content.
func (h *CrashTestHelper) VerifyDocument(db *docdb.LogicalDB, docID uint64, expectedPayload []byte) error {
	data, err := db.Read(docID)
	if err != nil {
		return fmt.Errorf("failed to read document %d: %w", docID, err)
	}

	if !bytes.Equal(data, expectedPayload) {
		return fmt.Errorf("payload mismatch for document %d: got %s, want %s", docID, string(data), string(expectedPayload))
	}

	return nil
}

// VerifyDocumentMissing verifies that a document does not exist.
func (h *CrashTestHelper) VerifyDocumentMissing(db *docdb.LogicalDB, docID uint64) error {
	_, err := db.Read(docID)
	if err == nil {
		return fmt.Errorf("document %d should not exist", docID)
	}
	return nil
}

// CreateTestDocuments creates multiple test documents.
func (h *CrashTestHelper) CreateTestDocuments(db *docdb.LogicalDB, count int) error {
	payload := []byte(`{"test":"data"}`)
	for i := 1; i <= count; i++ {
		if err := db.Create(uint64(i), payload); err != nil {
			return fmt.Errorf("failed to create document %d: %w", i, err)
		}
	}
	return nil
}

// WALSize returns the size of the WAL file.
func (h *CrashTestHelper) WALSize(dbName string) (int64, error) {
	walPath := h.WALPath(dbName)
	info, err := os.Stat(walPath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// DataFileSize returns the size of the data file.
func (h *CrashTestHelper) DataFileSize(dbName string) (int64, error) {
	dataPath := h.DataFilePath(dbName)
	info, err := os.Stat(dataPath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// ProcessManager manages a subprocess for crash testing.
type ProcessManager struct {
	cmd    *exec.Cmd
	t      *testing.T
	doneCh chan error
}

// StartProcess starts a subprocess that will be killed during testing.
func StartProcess(t *testing.T, name string, args []string, env []string) (*ProcessManager, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	pm := &ProcessManager{
		cmd:    cmd,
		t:      t,
		doneCh: make(chan error, 1),
	}

	go func() {
		pm.doneCh <- cmd.Wait()
	}()

	return pm, nil
}

// Kill sends SIGKILL to the process.
func (pm *ProcessManager) Kill() error {
	if pm.cmd.Process == nil {
		return fmt.Errorf("process not started")
	}

	if runtime.GOOS == "windows" {
		// Windows doesn't support SIGKILL, use Kill() instead
		return pm.cmd.Process.Kill()
	}

	return pm.cmd.Process.Signal(syscall.SIGKILL)
}

// Wait waits for the process to finish (or timeout).
func (pm *ProcessManager) Wait(timeout time.Duration) error {
	select {
	case err := <-pm.doneCh:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("process did not finish within %v", timeout)
	}
}

// PID returns the process ID.
func (pm *ProcessManager) PID() int {
	if pm.cmd.Process == nil {
		return 0
	}
	return pm.cmd.Process.Pid
}

// SimulateCrashDuringWrite simulates a crash during a write operation by
// killing the process mid-operation. This is a helper for testing scenarios
// where we can't easily inject failures.
func SimulateCrashDuringWrite(t *testing.T, db *docdb.LogicalDB, docID uint64, payload []byte) error {
	// For testing purposes, we'll close the database abruptly
	// In a real crash scenario, the process would be killed
	if err := db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	// Simulate crash by truncating WAL (representing incomplete write)
	// This is a simplified simulation - real crash tests would use process management
	return nil
}

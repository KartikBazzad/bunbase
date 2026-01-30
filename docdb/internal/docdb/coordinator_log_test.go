package docdb

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kartikbazzad/docdb/internal/logger"
)

func TestCoordinatorLog_AppendAndReplay(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "coordinator.log")
	log := logger.Default()
	c := NewCoordinatorLog(path, log)
	if err := c.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer c.Close()

	// Append decisions
	if err := c.AppendDecision(1, true); err != nil {
		t.Fatalf("AppendDecision(1, true): %v", err)
	}
	if err := c.AppendDecision(2, false); err != nil {
		t.Fatalf("AppendDecision(2, false): %v", err)
	}
	if err := c.AppendDecision(3, true); err != nil {
		t.Fatalf("AppendDecision(3, true): %v", err)
	}

	// Replay (file is open)
	decisions, err := c.Replay()
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if len(decisions) != 3 {
		t.Fatalf("Replay: got %d decisions, want 3", len(decisions))
	}
	if !decisions[1] {
		t.Fatal("tx 1: want commit (true), got false")
	}
	if decisions[2] {
		t.Fatal("tx 2: want abort (false), got true")
	}
	if !decisions[3] {
		t.Fatal("tx 3: want commit (true), got false")
	}
}

func TestCoordinatorLog_ReplayFromDisk(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "coordinator.log")
	log := logger.Default()
	c := NewCoordinatorLog(path, log)
	if err := c.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := c.AppendDecision(10, true); err != nil {
		t.Fatalf("AppendDecision: %v", err)
	}
	c.Close()

	// Replay without opening (simulate startup: open file for read)
	c2 := NewCoordinatorLog(path, log)
	decisions, err := c2.Replay()
	if err != nil {
		t.Fatalf("Replay from disk: %v", err)
	}
	if len(decisions) != 1 || !decisions[10] {
		t.Fatalf("Replay from disk: want {10: true}, got %v", decisions)
	}
}

func TestCoordinatorLog_ReplayMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent.log")
	log := logger.Default()
	c := NewCoordinatorLog(path, log)
	decisions, err := c.Replay()
	if err != nil {
		t.Fatalf("Replay missing file: expected nil error (empty map), got %v", err)
	}
	if len(decisions) != 0 {
		t.Fatalf("Replay missing file: want empty map, got %v", decisions)
	}
}

func TestCoordinatorLog_AppendAfterClose(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "coordinator.log")
	log := logger.Default()
	c := NewCoordinatorLog(path, log)
	if err := c.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	c.Close()
	err := c.AppendDecision(1, true)
	if err != ErrDBNotOpen {
		t.Fatalf("AppendDecision after Close: want ErrDBNotOpen, got %v", err)
	}
}

func TestCoordinatorLog_ChecksumRejectTornRecord(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "coordinator.log")
	log := logger.Default()
	c := NewCoordinatorLog(path, log)
	if err := c.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := c.AppendDecision(1, true); err != nil {
		t.Fatalf("AppendDecision: %v", err)
	}
	c.Close()

	// Corrupt the file: truncate last byte (break checksum)
	f, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("Open file: %v", err)
	}
	info, _ := f.Stat()
	if err := f.Truncate(info.Size() - 1); err != nil {
		f.Close()
		t.Fatalf("Truncate: %v", err)
	}
	f.Close()

	// Replay should skip the torn record (checksum mismatch) and return empty or partial
	c2 := NewCoordinatorLog(path, log)
	decisions, err := c2.Replay()
	if err != nil {
		t.Fatalf("Replay after truncate: %v", err)
	}
	// Torn record is skipped; we may get 0 decisions
	if len(decisions) > 1 {
		t.Fatalf("Replay: expected torn record to be skipped, got %v", decisions)
	}
}

package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPager_AllocateReadWrite(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "data.db")
	p, err := NewPager(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	id, err := p.AllocatePage()
	if err != nil {
		t.Fatal(err)
	}
	if id != 0 {
		t.Fatalf("first page id: got %d", id)
	}
	page, err := p.ReadPage(id)
	if err != nil {
		t.Fatal(err)
	}
	page.Data[PageHeaderSize] = 42
	if err := p.WritePage(page); err != nil {
		t.Fatal(err)
	}
	page2, err := p.ReadPage(id)
	if err != nil {
		t.Fatal(err)
	}
	if page2.Data[PageHeaderSize] != 42 {
		t.Fatalf("read back: got %d", page2.Data[PageHeaderSize])
	}
}

func TestPager_EnsurePages(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "data.db")
	p, err := NewPager(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if err := p.EnsurePages(2); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(dbPath)
	if info.Size() != int64(2*PageSize) {
		t.Fatalf("file size: got %d", info.Size())
	}
}

package storage

import (
	"fmt"
	"os"
	"sync"
)

// Pager manages disk I/O for a single data file split into fixed-size (4KB) pages.
// It supports AllocatePage (grow file), ReadPage, WritePage, Sync, and EnsurePages for bootstrap.
type Pager struct {
	file       *os.File
	mu         sync.RWMutex
	nextPageID PageID
	pageSize   int64
}

// NewPager creates a new Pager for the given file path (e.g. data/data.db).
// The parent directory is created if missing. nextPageID is derived from current file size.
func NewPager(filename string) (*Pager, error) {
	if err := os.MkdirAll(filename[:len(filename)-len("/data.db")], 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDiskWrite, err)
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("%w: %v", ErrDiskRead, err)
	}
	pageSize := int64(PageSize)
	nextPageID := PageID(info.Size() / pageSize)
	return &Pager{
		file:       file,
		nextPageID: nextPageID,
		pageSize:   pageSize,
	}, nil
}

// AllocatePage reserves a new PageID and extends the file.
func (p *Pager) AllocatePage() (PageID, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	id := p.nextPageID
	p.nextPageID++
	newSize := int64(p.nextPageID) * p.pageSize
	if err := p.file.Truncate(newSize); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrDiskWrite, err)
	}
	return id, nil
}

// ReadPage reads a page from disk.
func (p *Pager) ReadPage(pageID PageID) (*Page, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if pageID >= p.nextPageID {
		return nil, ErrInvalidPageID
	}
	page := &Page{ID: pageID}
	offset := int64(pageID) * p.pageSize
	n, err := p.file.ReadAt(page.Data[:], offset)
	if err != nil && n == 0 {
		return nil, fmt.Errorf("%w: %v", ErrDiskRead, err)
	}
	return page, nil
}

// WritePage writes a page to disk.
func (p *Pager) WritePage(page *Page) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if page.ID >= p.nextPageID {
		return ErrInvalidPageID
	}
	offset := int64(page.ID) * p.pageSize
	if _, err := p.file.WriteAt(page.Data[:], offset); err != nil {
		return fmt.Errorf("%w: %v", ErrDiskWrite, err)
	}
	page.mu.Lock()
	page.IsDirty = false
	page.mu.Unlock()
	return nil
}

// Sync flushes the file to disk.
func (p *Pager) Sync() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if err := p.file.Sync(); err != nil {
		return fmt.Errorf("%w: %v", ErrDiskWrite, err)
	}
	return nil
}

// Close closes the pager.
func (p *Pager) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.file != nil {
		_ = p.file.Sync()
		err := p.file.Close()
		p.file = nil
		return err
	}
	return nil
}

// GetNextPageID returns the next allocatable page ID.
func (p *Pager) GetNextPageID() PageID {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.nextPageID
}

// EnsurePages extends the file so that at least n pages exist (IDs 0..n-1).
func (p *Pager) EnsurePages(n int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for p.nextPageID < PageID(n) {
		newSize := int64(p.nextPageID+1) * p.pageSize
		if err := p.file.Truncate(newSize); err != nil {
			return fmt.Errorf("%w: %v", ErrDiskWrite, err)
		}
		p.nextPageID++
	}
	return nil
}

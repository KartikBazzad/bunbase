// Package storage implements the low-level data storage layer of Bundoc.
//
// It is responsibly for:
// 1. Pager: Direct Disk I/O, managing a single data file split into 8KB pages.
// 2. BufferPool: In-memory LRU cache to minimize disk access.
// 3. BPlusTree: The core indexing data structure for fast data retrieval.
// 4. Page: The fundamental unit of storage, containing headers and raw data.
package storage

import (
	"fmt"
	"os"
	"sync"

	"github.com/kartikbazzad/bunbase/bundoc/internal/util"
)

// Pager manages disk I/O for fixed-size pages.
// It handles opening the database file, reading/writing pages at specific offsets,
// and extending the file when new pages are allocated.
type Pager struct {
	file       *os.File
	mu         sync.RWMutex
	nextPageID PageID
}

// NewPager creates a new Pager instance backed by the specified file.
// It creates the file and parent directories if they don't exist.
// Ideally, this should open the file with O_DIRECT for database usage, but for now
// standard buffered I/O is used.
func NewPager(filename string) (*Pager, error) {
	// Create parent directories if they don't exist
	dir := filename[:len(filename)-len("/data.db")]
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", util.ErrDiskWriteFailed, err)
	}

	// Get file size to determine next page ID
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("%w: %v", util.ErrDiskReadFailed, err)
	}

	nextPageID := PageID(info.Size() / PageSize)

	return &Pager{
		file:       file,
		nextPageID: nextPageID,
	}, nil
}

// AllocatePage reserves a new PageID and extends the file size.
// It returns the ID of the newly allocated page.
func (p *Pager) AllocatePage() (PageID, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	pageID := p.nextPageID
	p.nextPageID++

	// Extend the file to accommodate the new page
	newSize := int64(p.nextPageID) * PageSize
	if err := p.file.Truncate(newSize); err != nil {
		return 0, fmt.Errorf("%w: %v", util.ErrDiskWriteFailed, err)
	}

	return pageID, nil
}

// ReadPage reads the page data from disk into memory.
func (p *Pager) ReadPage(pageID PageID) (*Page, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if pageID >= p.nextPageID {
		return nil, util.ErrInvalidPageID
	}

	page := &Page{ID: pageID}
	offset := int64(pageID) * PageSize

	n, err := p.file.ReadAt(page.Data[:], offset)
	if err != nil && n == 0 {
		return nil, fmt.Errorf("%w: %v", util.ErrDiskReadFailed, err)
	}

	return page, nil
}

// WritePage writes a page to disk
func (p *Pager) WritePage(page *Page) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if page.ID >= p.nextPageID {
		return util.ErrInvalidPageID
	}

	offset := int64(page.ID) * PageSize
	_, err := p.file.WriteAt(page.Data[:], offset)
	if err != nil {
		return fmt.Errorf("%w: %v", util.ErrDiskWriteFailed, err)
	}

	// Mark as clean after writing
	page.mu.Lock()
	page.IsDirty = false
	page.mu.Unlock()

	return nil
}

// Sync flushes all pending writes to disk
func (p *Pager) Sync() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if err := p.file.Sync(); err != nil {
		return fmt.Errorf("%w: %v", util.ErrDiskWriteFailed, err)
	}
	return nil
}

// Close closes the pager
func (p *Pager) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.file != nil {
		if err := p.file.Sync(); err != nil {
			return fmt.Errorf("%w: %v", util.ErrDiskWriteFailed, err)
		}
		return p.file.Close()
	}
	return nil
}

// GetNextPageID returns the next available page ID
func (p *Pager) GetNextPageID() PageID {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.nextPageID
}

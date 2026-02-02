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
	"github.com/kartikbazzad/bunbase/bundoc/security"
)

// Pager manages disk I/O for fixed-size pages.
type Pager struct {
	file         *os.File
	mu           sync.RWMutex
	nextPageID   PageID
	encryptor    *security.Encryptor
	diskPageSize int64 // PageSize (+ Overhead if encrypted)
}

// NewPager creates a new Pager. If key is provided, enables encryption.
func NewPager(filename string, key []byte) (*Pager, error) {
	// Create parent directories
	dir := filename[:len(filename)-len("/data.db")]
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", util.ErrDiskWriteFailed, err)
	}

	var encryptor *security.Encryptor
	diskPageSize := int64(PageSize)

	if len(key) > 0 {
		encryptor, err = security.NewEncryptor(key)
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to init encryptor: %w", err)
		}
		diskPageSize += int64(security.Overhead)
	}

	// Get file size to determine next page ID
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("%w: %v", util.ErrDiskReadFailed, err)
	}

	nextPageID := PageID(info.Size() / diskPageSize)

	return &Pager{
		file:         file,
		nextPageID:   nextPageID,
		encryptor:    encryptor,
		diskPageSize: diskPageSize,
	}, nil
}

// AllocatePage reserves a new PageID and extends the file size.
func (p *Pager) AllocatePage() (PageID, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	pageID := p.nextPageID
	p.nextPageID++

	// Extend the file
	newSize := int64(p.nextPageID) * p.diskPageSize
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

	page := &Page{ID: pageID} // Data is zeroed [PageSize]
	offset := int64(pageID) * p.diskPageSize

	// Read Disk Data
	diskData := make([]byte, p.diskPageSize)
	n, err := p.file.ReadAt(diskData, offset)
	if err != nil && n == 0 {
		return nil, fmt.Errorf("%w: %v", util.ErrDiskReadFailed, err)
	}

	// Decrypt if needed
	if p.encryptor != nil {
		plaintext, err := p.encryptor.DecryptBlock(diskData)
		if err != nil {
			return nil, fmt.Errorf("decryption failed for page %d: %w", pageID, err)
		}
		// Copy plaintext to page.Data
		// Note: plaintext MUST be PageSize (8192)
		if len(plaintext) != PageSize {
			return nil, fmt.Errorf("corrupt page size after decrypt: %d", len(plaintext))
		}
		copy(page.Data[:], plaintext)
	} else {
		copy(page.Data[:], diskData)
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

	var dataToWrite []byte

	// Encrypt if needed
	if p.encryptor != nil {
		var err error
		dataToWrite, err = p.encryptor.EncryptBlock(page.Data[:])
		if err != nil {
			return fmt.Errorf("encryption failed: %w", err)
		}
	} else {
		dataToWrite = page.Data[:]
	}

	offset := int64(page.ID) * p.diskPageSize
	_, err := p.file.WriteAt(dataToWrite, offset)
	if err != nil {
		return fmt.Errorf("%w: %v", util.ErrDiskWriteFailed, err)
	}

	// Mark as clean
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

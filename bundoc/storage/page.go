package storage

import (
	"encoding/binary"
	"sync"
)

// PageID uniquely identifies a page in the database
type PageID uint64

// PageSize is the size of each page in bytes (8KB default)
const PageSize = 8192

// Page types
const (
	PageTypeInvalid = iota
	PageTypeMeta    // Database metadata
	PageTypeFree    // Free page list
	PageTypeIndex   // B+ tree index page
	PageTypeLeaf    // B+ tree leaf page (stores documents)
)

// Page header layout:
// - PageType (1 byte)
// - Flags (1 byte)
// - KeyCount (2 bytes) - number of keys in this page
// - FreeSpace (2 bytes) - offset to free space
// - LSN (8 bytes) - Log Sequence Number for WAL
// - NextPage (8 bytes) - for linked pages (leaf pages)
// - PrevPage (8 bytes) - for linked pages (leaf pages)
// Total: 30 bytes
const PageHeaderSize = 30

// Page represents a single page in the database
type Page struct {
	ID       PageID
	Data     [PageSize]byte
	IsDirty  bool
	PinCount int32
	mu       sync.RWMutex
}

// NewPage creates a new page with the given ID and type
func NewPage(id PageID, pageType byte) *Page {
	p := &Page{
		ID: id,
	}
	p.SetPageType(pageType)
	p.SetKeyCount(0)
	p.SetFreeSpace(PageHeaderSize)
	return p
}

// Pin increments the pin count (page is in use)
func (p *Page) Pin() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.PinCount++
}

// Unpin decrements the pin count (page is no longer in use)
func (p *Page) Unpin() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.PinCount > 0 {
		p.PinCount--
	}
}

// IsPinned returns true if the page is currently pinned
func (p *Page) IsPinned() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.PinCount > 0
}

// MarkDirty marks the page as modified
func (p *Page) MarkDirty() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.IsDirty = true
}

// GetPageType returns the page type
func (p *Page) GetPageType() byte {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Data[0]
}

// SetPageType sets the page type
func (p *Page) SetPageType(pageType byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Data[0] = pageType
	p.IsDirty = true
}

// GetKeyCount returns the number of keys in the page
func (p *Page) GetKeyCount() uint16 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return binary.LittleEndian.Uint16(p.Data[2:4])
}

// SetKeyCount sets the number of keys in the page
func (p *Page) SetKeyCount(count uint16) {
	p.mu.Lock()
	defer p.mu.Unlock()
	binary.LittleEndian.PutUint16(p.Data[2:4], count)
	p.IsDirty = true
}

// GetFreeSpace returns the offset to free space in the page
func (p *Page) GetFreeSpace() uint16 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return binary.LittleEndian.Uint16(p.Data[4:6])
}

// SetFreeSpace sets the offset to free space in the page
func (p *Page) SetFreeSpace(offset uint16) {
	p.mu.Lock()
	defer p.mu.Unlock()
	binary.LittleEndian.PutUint16(p.Data[4:6], offset)
	p.IsDirty = true
}

// GetLSN returns the Log Sequence Number
func (p *Page) GetLSN() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return binary.LittleEndian.Uint64(p.Data[6:14])
}

// SetLSN sets the Log Sequence Number
func (p *Page) SetLSN(lsn uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	binary.LittleEndian.PutUint64(p.Data[6:14], lsn)
	p.IsDirty = true
}

// GetNextPage returns the next page ID (for linked pages)
func (p *Page) GetNextPage() PageID {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return PageID(binary.LittleEndian.Uint64(p.Data[14:22]))
}

// SetNextPage sets the next page ID
func (p *Page) SetNextPage(pageID PageID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	binary.LittleEndian.PutUint64(p.Data[14:22], uint64(pageID))
	p.IsDirty = true
}

// GetPrevPage returns the previous page ID (for linked pages)
func (p *Page) GetPrevPage() PageID {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return PageID(binary.LittleEndian.Uint64(p.Data[22:30]))
}

// SetPrevPage sets the previous page ID
func (p *Page) SetPrevPage(pageID PageID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	binary.LittleEndian.PutUint64(p.Data[22:30], uint64(pageID))
	p.IsDirty = true
}

// RemainingSpace returns the available space in the page
func (p *Page) RemainingSpace() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	freeSpace := int(binary.LittleEndian.Uint16(p.Data[4:6]))
	return PageSize - freeSpace
}

// Copy creates a deep copy of the page data (useful for MVCC)
func (p *Page) Copy() *Page {
	p.mu.RLock()
	defer p.mu.RUnlock()

	newPage := &Page{
		ID:       p.ID,
		IsDirty:  p.IsDirty,
		PinCount: p.PinCount,
	}
	copy(newPage.Data[:], p.Data[:])
	return newPage
}

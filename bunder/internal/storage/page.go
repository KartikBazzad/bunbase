// Package storage implements the low-level storage layer for Bunder:
// fixed-size pages (4KB), pager for disk I/O, buffer pool (SLRU), freelist, and B+Tree.
package storage

import (
	"encoding/binary"
	"sync"
)

// PageID uniquely identifies a page in the data file.
type PageID uint64

// PageSize is 4KB (Bunder default, BoltDB-style).
const PageSize = 4096

// Page types
const (
	PageTypeInvalid = iota
	PageTypeMeta
	PageTypeFree
	PageTypeLeaf
	PageTypeBranch
)

// Page header: Type(1) + Flags(1) + KeyCount(2) + FreeSpace(2) + LSN(8) + Next(8) + Prev(8) = 30
const PageHeaderSize = 30

// Page is a single 4KB page.
type Page struct {
	ID       PageID
	Data     [PageSize]byte
	IsDirty  bool
	PinCount int32
	mu       sync.RWMutex
}

// NewPage creates a new page with the given ID and type.
func NewPage(id PageID, pageType byte) *Page {
	p := &Page{ID: id}
	p.SetPageType(pageType)
	p.SetKeyCount(0)
	p.SetFreeSpace(PageHeaderSize)
	return p
}

// Pin increments the pin count.
func (p *Page) Pin() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.PinCount++
}

// Unpin decrements the pin count.
func (p *Page) Unpin() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.PinCount > 0 {
		p.PinCount--
	}
}

// IsPinned returns true if the page is pinned.
func (p *Page) IsPinned() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.PinCount > 0
}

// MarkDirty marks the page as dirty.
func (p *Page) MarkDirty() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.IsDirty = true
}

// GetPageType returns the page type.
func (p *Page) GetPageType() byte {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Data[0]
}

// SetPageType sets the page type.
func (p *Page) SetPageType(t byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Data[0] = t
	p.IsDirty = true
}

// GetKeyCount returns the number of keys in the page.
func (p *Page) GetKeyCount() uint16 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return binary.LittleEndian.Uint16(p.Data[2:4])
}

// SetKeyCount sets the key count.
func (p *Page) SetKeyCount(count uint16) {
	p.mu.Lock()
	defer p.mu.Unlock()
	binary.LittleEndian.PutUint16(p.Data[2:4], count)
	p.IsDirty = true
}

// GetFreeSpace returns the offset to free space.
func (p *Page) GetFreeSpace() uint16 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return binary.LittleEndian.Uint16(p.Data[4:6])
}

// SetFreeSpace sets the free space offset.
func (p *Page) SetFreeSpace(offset uint16) {
	p.mu.Lock()
	defer p.mu.Unlock()
	binary.LittleEndian.PutUint16(p.Data[4:6], offset)
	p.IsDirty = true
}

// GetLSN returns the LSN.
func (p *Page) GetLSN() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return binary.LittleEndian.Uint64(p.Data[6:14])
}

// SetLSN sets the LSN.
func (p *Page) SetLSN(lsn uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	binary.LittleEndian.PutUint64(p.Data[6:14], lsn)
	p.IsDirty = true
}

// GetNextPage returns the next page ID (for leaf linking).
func (p *Page) GetNextPage() PageID {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return PageID(binary.LittleEndian.Uint64(p.Data[14:22]))
}

// SetNextPage sets the next page ID.
func (p *Page) SetNextPage(id PageID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	binary.LittleEndian.PutUint64(p.Data[14:22], uint64(id))
	p.IsDirty = true
}

// GetPrevPage returns the previous page ID.
func (p *Page) GetPrevPage() PageID {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return PageID(binary.LittleEndian.Uint64(p.Data[22:30]))
}

// SetPrevPage sets the previous page ID.
func (p *Page) SetPrevPage(id PageID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	binary.LittleEndian.PutUint64(p.Data[22:30], uint64(id))
	p.IsDirty = true
}

// RemainingSpace returns bytes available in the page.
func (p *Page) RemainingSpace() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return PageSize - int(binary.LittleEndian.Uint16(p.Data[4:6]))
}

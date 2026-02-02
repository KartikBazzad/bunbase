package storage

import (
	"encoding/binary"
	"sync"
)

// FreeList manages free page IDs (BoltDB-style) so pages can be reused instead of always growing the file.
// Page 0 is reserved for meta; page 1 holds the free list (count + up to MaxFreeIDs page IDs).
// Call EnsurePages(2) on the pager before using the freelist so page 1 exists.
const freeListPageID PageID = 1

// MaxFreeIDs is the number of free IDs we can store in one free-list page ( (PageSize - header - 8) / 8 ).
const MaxFreeIDs = (PageSize - PageHeaderSize - 8) / 8

// FreeList holds free page IDs.
type FreeList struct {
	mu     sync.Mutex
	ids    []PageID
	pager  *Pager
	pool   *BufferPool
	nextID PageID // next ID to use when ids is empty (grow file)
}

// NewFreeList creates a FreeList. Call EnsurePages(2) on the pager first so page 0 (meta) and page 1 (freelist) exist.
func NewFreeList(pager *Pager, pool *BufferPool) *FreeList {
	fl := &FreeList{pager: pager, pool: pool}
	fl.nextID = pager.GetNextPageID()
	if fl.nextID <= freeListPageID {
		fl.nextID = freeListPageID + 1
	}
	return fl
}

// Load reads the free list from disk (from the free list page).
func (fl *FreeList) Load() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	page, err := fl.pool.FetchPage(freeListPageID)
	if err != nil {
		// First run: no free list page yet
		fl.ids = nil
		return nil
	}
	defer fl.pool.UnpinPage(page.ID, false)
	n := binary.LittleEndian.Uint64(page.Data[PageHeaderSize : PageHeaderSize+8])
	if n > MaxFreeIDs {
		n = MaxFreeIDs
	}
	fl.ids = make([]PageID, 0, n)
	off := PageHeaderSize + 8
	for i := uint64(0); i < n; i++ {
		id := PageID(binary.LittleEndian.Uint64(page.Data[off : off+8]))
		fl.ids = append(fl.ids, id)
		off += 8
	}
	return nil
}

// Allocate returns a free page ID (either from the list or allocates a new one).
func (fl *FreeList) Allocate() (PageID, error) {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	if len(fl.ids) > 0 {
		id := fl.ids[len(fl.ids)-1]
		fl.ids = fl.ids[:len(fl.ids)-1]
		return id, nil
	}
	// Allocate new page from pager
	id, err := fl.pager.AllocatePage()
	if err != nil {
		return 0, err
	}
	fl.nextID = fl.pager.GetNextPageID()
	return id, nil
}

// Free adds a page ID to the free list (caller must have already truncated/cleared the page from use).
func (fl *FreeList) Free(pageID PageID) {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	if pageID == 0 || pageID == freeListPageID {
		return
	}
	fl.ids = append(fl.ids, pageID)
}

// Count returns the number of free pages.
func (fl *FreeList) Count() int {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	return len(fl.ids)
}

// Persist writes the free list to the free list page (for checkpoint/recovery).
// The pager must have been initialized with EnsurePages(2) so page 1 exists.
func (fl *FreeList) Persist(pool *BufferPool) error {
	fl.mu.Lock()
	ids := make([]PageID, len(fl.ids))
	copy(ids, fl.ids)
	fl.mu.Unlock()

	page, err := pool.FetchPage(freeListPageID)
	if err != nil {
		return err
	}
	defer pool.UnpinPage(page.ID, true)
	page.SetPageType(PageTypeFree)
	n := len(ids)
	if n > MaxFreeIDs {
		n = MaxFreeIDs
	}
	binary.LittleEndian.PutUint64(page.Data[PageHeaderSize:PageHeaderSize+8], uint64(n))
	off := PageHeaderSize + 8
	for i := 0; i < n; i++ {
		binary.LittleEndian.PutUint64(page.Data[off:off+8], uint64(ids[i]))
		off += 8
	}
	return nil
}

package storage

import (
	"container/list"
	"sync"

	"github.com/kartikbazzad/bunbase/bundoc/internal/util"
)

// BufferPool manages in-memory pages using Segmented LRU (SLRU) eviction policy.
// It caches pages to reduce expensive disk I/O.
//
// SLRU Mechanism:
// - **Probation Segment**: New pages start here. If accessed again, they move to Protected.
// - **Protected Segment**: Hot pages reside here. If full, pages are demoted back to Probation.
// - **Eviction**: Always happens from the tail of the Probation segment.
type BufferPool struct {
	capacity     int
	protectedCap int // Capacity of protected segment (e.g., 70-80%)
	pages        map[PageID]*bufferEntry
	protected    *list.List // Protected segment (hot pages)
	probation    *list.List // Probation segment (new/cold pages)
	pager        *Pager
	mu           sync.RWMutex
}

// bufferEntry represents an entry in the buffer pool
type bufferEntry struct {
	page        *Page
	element     *list.Element
	isProtected bool // Tracks which list the element is in
}

// NewBufferPool creates a new buffer pool with the given capacity
func NewBufferPool(capacity int, pager *Pager) *BufferPool {
	// 80% protected, 20% probation is a common split
	protectedCap := int(float64(capacity) * 0.8)
	if protectedCap < 1 {
		protectedCap = 1
	}

	return &BufferPool{
		capacity:     capacity,
		protectedCap: protectedCap,
		pages:        make(map[PageID]*bufferEntry),
		protected:    list.New(),
		probation:    list.New(),
		pager:        pager,
	}
}

// FetchPage retrieves a page. If it's in the cache, it's pinned and promoted (SLRU logic).
// If not, it's loaded from disk via the Pager.
//
// SLRU Logic:
// - If in Protected: Move to front (MRU).
// - If in Probation: Promote to Protected segment.
// - If loading from disk: Add to Probation segment.
func (bp *BufferPool) FetchPage(pageID PageID) (*Page, error) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	// Check if page is in buffer pool
	if entry, exists := bp.pages[pageID]; exists {
		entry.page.Pin()

		if entry.isProtected {
			// Already protected: Just MRU update
			bp.protected.MoveToFront(entry.element)
		} else {
			// In probation: Upgrade to protected (Second Chance)
			bp.probation.Remove(entry.element)
			entry.element = bp.protected.PushFront(pageID)
			entry.isProtected = true

			// Enforce protected capacity: Demote LRU of protected to probation
			if bp.protected.Len() > bp.protectedCap {
				demoteElem := bp.protected.Back()
				if demoteElem != nil {
					demoteID := demoteElem.Value.(PageID)
					demoteEntry := bp.pages[demoteID]

					bp.protected.Remove(demoteElem)
					demoteEntry.element = bp.probation.PushFront(demoteID)
					demoteEntry.isProtected = false
				}
			}
		}

		return entry.page, nil
	}

	// Page not in buffer pool, load from disk
	page, err := bp.pager.ReadPage(pageID)
	if err != nil {
		return nil, err
	}

	// Evict if necessary (Total capacity check)
	if len(bp.pages) >= bp.capacity {
		if err := bp.evictPage(); err != nil {
			return nil, err
		}
	}

	// Add to buffer pool (starts in Probation)
	element := bp.probation.PushFront(pageID)
	bp.pages[pageID] = &bufferEntry{
		page:        page,
		element:     element,
		isProtected: false,
	}

	page.Pin()
	return page, nil
}

// NewPage allocates a new page on disk and adds it to the buffer pool (pinned).
func (bp *BufferPool) NewPage(pageType byte) (*Page, error) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	// Allocate new page
	pageID, err := bp.pager.AllocatePage()
	if err != nil {
		return nil, err
	}

	// Create page
	page := NewPage(pageID, pageType)

	// Evict if necessary
	if len(bp.pages) >= bp.capacity {
		if err := bp.evictPage(); err != nil {
			return nil, err
		}
	}

	// Add to buffer pool (starts in Probation)
	element := bp.probation.PushFront(pageID)
	bp.pages[pageID] = &bufferEntry{
		page:        page,
		element:     element,
		isProtected: false,
	}

	page.Pin()
	page.MarkDirty()
	return page, nil
}

// UnpinPage unpins a page, making it eligible for eviction
func (bp *BufferPool) UnpinPage(pageID PageID, isDirty bool) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	entry, exists := bp.pages[pageID]
	if !exists {
		return util.ErrPageNotFound
	}

	if isDirty {
		entry.page.MarkDirty()
	}

	entry.page.Unpin()
	return nil
}

// FlushPage writes a page to disk if it's dirty
func (bp *BufferPool) FlushPage(pageID PageID) error {
	bp.mu.RLock()
	entry, exists := bp.pages[pageID]
	bp.mu.RUnlock()

	if !exists {
		return util.ErrPageNotFound
	}

	entry.page.mu.RLock()
	isDirty := entry.page.IsDirty
	entry.page.mu.RUnlock()

	if isDirty {
		return bp.pager.WritePage(entry.page)
	}

	return nil
}

// FlushAllPages writes all dirty pages to disk
func (bp *BufferPool) FlushAllPages() error {
	bp.mu.RLock()
	pageIDs := make([]PageID, 0, len(bp.pages))
	for pageID := range bp.pages {
		pageIDs = append(pageIDs, pageID)
	}
	bp.mu.RUnlock()

	for _, pageID := range pageIDs {
		if err := bp.FlushPage(pageID); err != nil {
			return err
		}
	}

	return bp.pager.Sync()
}

// evictPage evicts the least recently used unpinned page
// Caller must hold bp.mu
func (bp *BufferPool) evictPage() error {
	// Helper to try evicting from a list (LRU order = Back)
	evictFromList := func(l *list.List) (bool, error) {
		for element := l.Back(); element != nil; element = element.Prev() {
			pageID := element.Value.(PageID)
			entry := bp.pages[pageID]

			// Skip pinned pages
			if entry.page.IsPinned() {
				continue
			}

			// Flush if dirty
			entry.page.mu.RLock()
			isDirty := entry.page.IsDirty
			entry.page.mu.RUnlock()

			if isDirty {
				if err := bp.pager.WritePage(entry.page); err != nil {
					return false, err
				}
			}

			// Remove from buffer pool
			l.Remove(element)
			delete(bp.pages, pageID)

			return true, nil
		}
		return false, nil // No unpinned pages found in this list
	}

	// 1. Try evicting from Probation (Scan/New pages)
	evicted, err := evictFromList(bp.probation)
	if err != nil {
		return err
	}
	if evicted {
		return nil
	}

	// 2. Try evicting from Protected (Hot pages that became cold)
	evicted, err = evictFromList(bp.protected)
	if err != nil {
		return err
	}
	if evicted {
		return nil
	}

	// All pages are pinned - cannot evict
	return util.ErrPageFull
}

// Size returns the current number of pages in the buffer pool
func (bp *BufferPool) Size() int {
	bp.mu.RLock()
	defer bp.mu.RUnlock()
	return len(bp.pages)
}

// Close flushes all pages and closes the buffer pool
func (bp *BufferPool) Close() error {
	if err := bp.FlushAllPages(); err != nil {
		return err
	}
	return bp.pager.Close()
}

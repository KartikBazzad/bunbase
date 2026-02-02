package storage

import (
	"container/list"
	"sync"
)

// BufferPool caches pages in memory using SLRU (Segmented LRU) eviction:
// new pages enter the probation segment; on second access they move to protected.
// Eviction always happens from the probation tail. 80% of capacity is protected by default.
type BufferPool struct {
	capacity     int
	protectedCap int
	pages        map[PageID]*bufferEntry
	protected    *list.List
	probation    *list.List
	pager        *Pager
	mu           sync.RWMutex
}

// bufferEntry holds a page and its position in either the protected or probation list.
type bufferEntry struct {
	page        *Page
	element     *list.Element
	isProtected bool
}

// NewBufferPool creates a buffer pool with the given capacity (number of 4KB pages).
// protectedCap is set to 80% of capacity for the SLRU protected segment.
func NewBufferPool(capacity int, pager *Pager) *BufferPool {
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

// FetchPage returns a page from cache or loads from disk; pins the page.
func (bp *BufferPool) FetchPage(pageID PageID) (*Page, error) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	if entry, ok := bp.pages[pageID]; ok {
		entry.page.Pin()
		if entry.isProtected {
			bp.protected.MoveToFront(entry.element)
		} else {
			bp.probation.Remove(entry.element)
			entry.element = bp.protected.PushFront(pageID)
			entry.isProtected = true
			if bp.protected.Len() > bp.protectedCap {
				demote := bp.protected.Back()
				if demote != nil {
					did := demote.Value.(PageID)
					de := bp.pages[did]
					bp.protected.Remove(demote)
					de.element = bp.probation.PushFront(did)
					de.isProtected = false
				}
			}
		}
		return entry.page, nil
	}
	page, err := bp.pager.ReadPage(pageID)
	if err != nil {
		return nil, err
	}
	if len(bp.pages) >= bp.capacity {
		if err := bp.evictPage(); err != nil {
			return nil, err
		}
	}
	el := bp.probation.PushFront(pageID)
	bp.pages[pageID] = &bufferEntry{page: page, element: el, isProtected: false}
	page.Pin()
	return page, nil
}

// NewPage allocates a new page and adds it to the pool (pinned, dirty).
func (bp *BufferPool) NewPage(pageType byte) (*Page, error) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	pageID, err := bp.pager.AllocatePage()
	if err != nil {
		return nil, err
	}
	page := NewPage(pageID, pageType)
	if len(bp.pages) >= bp.capacity {
		if err := bp.evictPage(); err != nil {
			return nil, err
		}
	}
	el := bp.probation.PushFront(pageID)
	bp.pages[pageID] = &bufferEntry{page: page, element: el, isProtected: false}
	page.Pin()
	page.MarkDirty()
	return page, nil
}

// UnpinPage unpins a page; if isDirty, marks it dirty.
func (bp *BufferPool) UnpinPage(pageID PageID, isDirty bool) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	entry, ok := bp.pages[pageID]
	if !ok {
		return ErrPageNotFound
	}
	if isDirty {
		entry.page.MarkDirty()
	}
	entry.page.Unpin()
	return nil
}

// FlushPage writes a dirty page to disk.
func (bp *BufferPool) FlushPage(pageID PageID) error {
	bp.mu.RLock()
	entry, ok := bp.pages[pageID]
	bp.mu.RUnlock()
	if !ok {
		return ErrPageNotFound
	}
	entry.page.mu.RLock()
	dirty := entry.page.IsDirty
	entry.page.mu.RUnlock()
	if dirty {
		return bp.pager.WritePage(entry.page)
	}
	return nil
}

// FlushAllPages flushes all dirty pages and syncs.
func (bp *BufferPool) FlushAllPages() error {
	bp.mu.RLock()
	ids := make([]PageID, 0, len(bp.pages))
	for id := range bp.pages {
		ids = append(ids, id)
	}
	bp.mu.RUnlock()
	for _, id := range ids {
		if err := bp.FlushPage(id); err != nil {
			return err
		}
	}
	return bp.pager.Sync()
}

func (bp *BufferPool) evictPage() error {
	evict := func(l *list.List) (bool, error) {
		for e := l.Back(); e != nil; e = e.Prev() {
			pageID := e.Value.(PageID)
			entry := bp.pages[pageID]
			if entry.page.IsPinned() {
				continue
			}
			entry.page.mu.RLock()
			dirty := entry.page.IsDirty
			entry.page.mu.RUnlock()
			if dirty {
				if err := bp.pager.WritePage(entry.page); err != nil {
					return false, err
				}
			}
			l.Remove(e)
			delete(bp.pages, pageID)
			return true, nil
		}
		return false, nil
	}
	if ok, err := evict(bp.probation); err != nil || ok {
		return err
	}
	if ok, err := evict(bp.protected); err != nil || ok {
		return err
	}
	return ErrPageFull
}

// Size returns the number of pages in the pool.
func (bp *BufferPool) Size() int {
	bp.mu.RLock()
	defer bp.mu.RUnlock()
	return len(bp.pages)
}

// Close flushes all pages and closes the pager.
func (bp *BufferPool) Close() error {
	if err := bp.FlushAllPages(); err != nil {
		return err
	}
	return bp.pager.Close()
}

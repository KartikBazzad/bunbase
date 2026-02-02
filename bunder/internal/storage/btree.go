package storage

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"
)

// BTreeEntry is a key-value pair stored in B+Tree leaves.
type BTreeEntry struct {
	Key   []byte
	Value []byte
}

// branchEntry is an internal (branch) node entry: separator key and child PageID.
type branchEntry struct {
	key []byte
	pid PageID
}

// internalHeaderSize is the branch page header: LeftPtr(8) after PageHeaderSize.
const internalHeaderSize = PageHeaderSize + 8

// BTree is a simplified B+Tree for persistent KV storage using 4KB pages.
// Order is 32 (max keys per node). Leaves are linked for range scans.
type BTree struct {
	pool   *BufferPool
	rootID PageID
	order  int // max keys per node
	mu     sync.RWMutex
	onRoot func(PageID)
}

// NewBTree creates a new B+Tree with a single leaf root.
func NewBTree(pool *BufferPool) (*BTree, error) {
	root, err := pool.NewPage(PageTypeLeaf)
	if err != nil {
		return nil, err
	}
	t := &BTree{pool: pool, rootID: root.ID, order: 32}
	pool.UnpinPage(root.ID, true)
	return t, nil
}

// LoadBTree loads a B+Tree from a known root page ID.
func LoadBTree(pool *BufferPool, rootID PageID) (*BTree, error) {
	page, err := pool.FetchPage(rootID)
	if err != nil {
		return nil, err
	}
	defer pool.UnpinPage(rootID, false)
	t := page.GetPageType()
	if t != PageTypeLeaf && t != PageTypeBranch {
		return nil, fmt.Errorf("invalid root page type: %d", t)
	}
	return &BTree{pool: pool, rootID: rootID, order: 32}, nil
}

// SetOnRootChange sets callback when root changes.
func (bt *BTree) SetOnRootChange(fn func(PageID)) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	bt.onRoot = fn
}

// GetRootID returns the root page ID.
func (bt *BTree) GetRootID() PageID {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.rootID
}

// Get returns the value for key, or nil if not found.
func (bt *BTree) Get(key []byte) ([]byte, error) {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	root, err := bt.pool.FetchPage(bt.rootID)
	if err != nil {
		return nil, err
	}
	defer bt.pool.UnpinPage(root.ID, false)
	leaf, err := bt.findLeaf(root, key)
	if err != nil {
		return nil, err
	}
	if leaf.ID != root.ID {
		defer bt.pool.UnpinPage(leaf.ID, false)
	}
	return bt.searchLeaf(leaf, key)
}

func (bt *BTree) findLeaf(page *Page, key []byte) (*Page, error) {
	for page.GetPageType() == PageTypeBranch {
		childID, err := bt.searchBranch(page, key)
		if err != nil {
			return nil, err
		}
		next, err := bt.pool.FetchPage(childID)
		if err != nil {
			return nil, err
		}
		if page.ID != bt.rootID {
			bt.pool.UnpinPage(page.ID, false)
		}
		page = next
	}
	return page, nil
}

func (bt *BTree) searchBranch(page *Page, key []byte) (PageID, error) {
	leftPtr, entries := bt.getBranchEntries(page)
	if len(entries) == 0 {
		return leftPtr, nil
	}
	for i, e := range entries {
		if bytes.Compare(key, e.key) < 0 {
			if i == 0 {
				return leftPtr, nil
			}
			return entries[i-1].pid, nil
		}
	}
	return entries[len(entries)-1].pid, nil
}

func (bt *BTree) getBranchEntries(page *Page) (PageID, []branchEntry) {
	leftPtr := PageID(binary.LittleEndian.Uint64(page.Data[PageHeaderSize : PageHeaderSize+8]))
	keyCount := int(page.GetKeyCount())
	var entries []branchEntry
	off := internalHeaderSize
	for i := 0; i < keyCount && off+12 <= PageSize; i++ {
		kl := binary.LittleEndian.Uint16(page.Data[off : off+2])
		off += 2
		if off+int(kl)+10 > PageSize {
			break
		}
		k := make([]byte, kl)
		copy(k, page.Data[off:off+int(kl)])
		off += int(kl) + 2 // skip valLen
		pid := PageID(binary.LittleEndian.Uint64(page.Data[off : off+8]))
		off += 8
		entries = append(entries, branchEntry{key: k, pid: pid})
	}
	return leftPtr, entries
}

func (bt *BTree) searchLeaf(page *Page, key []byte) ([]byte, error) {
	entries := bt.getLeafEntries(page)
	for _, e := range entries {
		if bytes.Equal(e.Key, key) {
			return e.Value, nil
		}
		if bytes.Compare(e.Key, key) > 0 {
			break
		}
	}
	return nil, nil // not found
}

func (bt *BTree) getLeafEntries(page *Page) []BTreeEntry {
	var out []BTreeEntry
	keyCount := int(page.GetKeyCount())
	off := PageHeaderSize
	for i := 0; i < keyCount && off+4 <= PageSize; i++ {
		kl := binary.LittleEndian.Uint16(page.Data[off : off+2])
		off += 2
		if off+int(kl)+2 > PageSize {
			break
		}
		key := make([]byte, kl)
		copy(key, page.Data[off:off+int(kl)])
		off += int(kl)
		vl := binary.LittleEndian.Uint16(page.Data[off : off+2])
		off += 2
		if off+int(vl) > PageSize {
			break
		}
		val := make([]byte, vl)
		copy(val, page.Data[off:off+int(vl)])
		off += int(vl)
		out = append(out, BTreeEntry{Key: key, Value: val})
	}
	return out
}

// Put inserts or updates a key-value pair.
func (bt *BTree) Put(key, value []byte) error {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	promotedKey, newPageID, err := bt.insert(bt.rootID, key, value)
	if err != nil {
		return err
	}
	if newPageID != 0 {
		newRoot, err := bt.pool.NewPage(PageTypeBranch)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint64(newRoot.Data[PageHeaderSize:PageHeaderSize+8], uint64(bt.rootID))
		newRoot.SetKeyCount(1)
		off := internalHeaderSize
		binary.LittleEndian.PutUint16(newRoot.Data[off:off+2], uint16(len(promotedKey)))
		off += 2
		copy(newRoot.Data[off:off+len(promotedKey)], promotedKey)
		off += len(promotedKey)
		binary.LittleEndian.PutUint16(newRoot.Data[off:off+2], 8)
		off += 2
		binary.LittleEndian.PutUint64(newRoot.Data[off:off+8], uint64(newPageID))
		newRoot.SetFreeSpace(uint16(off + 8))
		bt.rootID = newRoot.ID
		if bt.onRoot != nil {
			bt.onRoot(bt.rootID)
		}
		bt.pool.UnpinPage(newRoot.ID, true)
	}
	return nil
}

func (bt *BTree) insert(pageID PageID, key, value []byte) ([]byte, PageID, error) {
	page, err := bt.pool.FetchPage(pageID)
	if err != nil {
		return nil, 0, err
	}
	defer bt.pool.UnpinPage(pageID, true)
	if page.GetPageType() == PageTypeLeaf {
		return bt.insertLeaf(page, key, value)
	}
	childID, err := bt.searchBranch(page, key)
	if err != nil {
		return nil, 0, err
	}
	promotedKey, newID, err := bt.insert(childID, key, value)
	if err != nil {
		return nil, 0, err
	}
	if newID == 0 {
		return nil, 0, nil
	}
	return bt.insertBranch(page, promotedKey, newID)
}

func (bt *BTree) insertLeaf(page *Page, key, value []byte) ([]byte, PageID, error) {
	entries := bt.getLeafEntries(page)
	// Update or insert
	for i := range entries {
		if bytes.Equal(entries[i].Key, key) {
			entries[i].Value = value
			return nil, 0, bt.writeLeafEntries(page, entries)
		}
		if bytes.Compare(entries[i].Key, key) > 0 {
			entries = append(entries[:i], append([]BTreeEntry{{Key: key, Value: value}}, entries[i:]...)...)
			return nil, 0, bt.writeLeafEntries(page, entries)
		}
	}
	entries = append(entries, BTreeEntry{Key: key, Value: value})
	if len(entries) <= bt.order && bt.entrySize(entries) <= PageSize-PageHeaderSize-16 {
		return nil, 0, bt.writeLeafEntries(page, entries)
	}
	// Split
	mid := len(entries) / 2
	leftEntries := entries[:mid]
	rightEntries := entries[mid:]
	promotedKey := rightEntries[0].Key
	rightPage, err := bt.pool.NewPage(PageTypeLeaf)
	if err != nil {
		return nil, 0, err
	}
	if err := bt.writeLeafEntries(page, leftEntries); err != nil {
		return nil, 0, err
	}
	if err := bt.writeLeafEntries(rightPage, rightEntries); err != nil {
		return nil, 0, err
	}
	page.SetNextPage(rightPage.ID)
	rightPage.SetPrevPage(page.ID)
	bt.pool.UnpinPage(rightPage.ID, true)
	return promotedKey, rightPage.ID, nil
}

func (bt *BTree) entrySize(entries []BTreeEntry) int {
	n := 0
	for _, e := range entries {
		n += 2 + len(e.Key) + 2 + len(e.Value)
	}
	return n
}

func (bt *BTree) writeLeafEntries(page *Page, entries []BTreeEntry) error {
	off := PageHeaderSize
	for _, e := range entries {
		need := 2 + len(e.Key) + 2 + len(e.Value)
		if off+need > PageSize {
			return ErrPageFull
		}
		binary.LittleEndian.PutUint16(page.Data[off:off+2], uint16(len(e.Key)))
		off += 2
		copy(page.Data[off:off+len(e.Key)], e.Key)
		off += len(e.Key)
		binary.LittleEndian.PutUint16(page.Data[off:off+2], uint16(len(e.Value)))
		off += 2
		copy(page.Data[off:off+len(e.Value)], e.Value)
		off += len(e.Value)
	}
	page.SetKeyCount(uint16(len(entries)))
	page.SetFreeSpace(uint16(off))
	page.MarkDirty()
	return nil
}

func (bt *BTree) insertBranch(page *Page, key []byte, childID PageID) ([]byte, PageID, error) {
	leftPtr, entries := bt.getBranchEntries(page)
	// Insert (key, childID) in order
	inserted := false
	for i := range entries {
		if bytes.Compare(key, entries[i].key) < 0 {
			entries = append(entries[:i], append([]branchEntry{{key: key, pid: childID}}, entries[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		entries = append(entries, branchEntry{key: key, pid: childID})
	}
	if len(entries) <= bt.order {
		bt.writeBranchEntries(page, leftPtr, entries)
		return nil, 0, nil
	}
	mid := len(entries) / 2
	promotedKey := entries[mid].key
	rightPage, err := bt.pool.NewPage(PageTypeBranch)
	if err != nil {
		return nil, 0, err
	}
	bt.writeBranchEntries(page, leftPtr, entries[:mid])
	bt.writeBranchEntries(rightPage, entries[mid].pid, entries[mid+1:])
	bt.pool.UnpinPage(rightPage.ID, true)
	return promotedKey, rightPage.ID, nil
}

func (bt *BTree) writeBranchEntries(page *Page, leftPtr PageID, entries []branchEntry) {
	binary.LittleEndian.PutUint64(page.Data[PageHeaderSize:PageHeaderSize+8], uint64(leftPtr))
	off := internalHeaderSize
	for _, e := range entries {
		binary.LittleEndian.PutUint16(page.Data[off:off+2], uint16(len(e.key)))
		off += 2
		copy(page.Data[off:off+len(e.key)], e.key)
		off += len(e.key)
		binary.LittleEndian.PutUint16(page.Data[off:off+2], 8)
		off += 2
		binary.LittleEndian.PutUint64(page.Data[off:off+8], uint64(e.pid))
		off += 8
	}
	page.SetKeyCount(uint16(len(entries)))
	page.SetFreeSpace(uint16(off))
	page.MarkDirty()
}

// Delete removes a key from the tree (optional; not implemented for MVP).
func (bt *BTree) Delete(key []byte) error {
	// TODO: implement delete with coalesce
	return nil
}

// RangeScan returns entries in [start, end] (inclusive).
func (bt *BTree) RangeScan(start, end []byte) ([]BTreeEntry, error) {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	root, err := bt.pool.FetchPage(bt.rootID)
	if err != nil {
		return nil, err
	}
	defer bt.pool.UnpinPage(root.ID, false)
	leaf, err := bt.findLeaf(root, start)
	if err != nil {
		return nil, err
	}
	if leaf.ID != root.ID {
		defer bt.pool.UnpinPage(leaf.ID, false)
	}
	var result []BTreeEntry
	for {
		for _, e := range bt.getLeafEntries(leaf) {
			if bytes.Compare(e.Key, start) >= 0 && bytes.Compare(e.Key, end) <= 0 {
				result = append(result, e)
			}
			if bytes.Compare(e.Key, end) > 0 {
				return result, nil
			}
		}
		nextID := leaf.GetNextPage()
		if nextID == 0 {
			break
		}
		if leaf.ID != root.ID {
			bt.pool.UnpinPage(leaf.ID, false)
		}
		leaf, err = bt.pool.FetchPage(nextID)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

package storage

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/kartikbazzad/bunbase/bundoc/internal/util"
)

// LoadBPlusTree restores an existing B+Tree from a known Root Page ID.
// This is used during database recovery to reconstruct indexes from the System Catalog.
func LoadBPlusTree(bp *BufferPool, rootID PageID) (*BPlusTree, error) {
	// Verify root page exists
	page, err := bp.FetchPage(rootID)
	if err != nil {
		return nil, err
	}
	defer bp.UnpinPage(rootID, false)

	if page.GetPageType() != PageTypeLeaf && page.GetPageType() != PageTypeIndex {
		return nil, fmt.Errorf("invalid page key type for root: %d", page.GetPageType())
	}

	return &BPlusTree{
		bp:     bp,
		rootID: rootID,
		order:  64,
	}, nil
}

// BPlusTree implements a durable B+Tree data structure.
//
// Properties:
// - **Order**: Max keys per node (currently fixed at 64).
// - **Persistence**: Backed by disk pages via BufferPool.
// - **Consistency**: Uses copy-on-write or WAL (indirectly) for crash safety.
// - **Root Persistence**: Notifies listeners when the root page splits/changes (critical for updating metadata).
type BPlusTree struct {
	bp           *BufferPool
	rootID       PageID
	mu           sync.RWMutex
	order        int // Maximum number of keys per node
	onRootChange func(PageID)
}

// SetOnRootChange registers a callback to be invoked whenever the root page ID changes.
// This typically happens during a root split operation.
// The callback is crucial for updating the System Catalog (Metadata).
func (t *BPlusTree) SetOnRootChange(callback func(PageID)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onRootChange = callback
}

// Entry represents a key-value pair in the B+ tree
type Entry struct {
	Key   []byte
	Value []byte
}

// NewBPlusTree creates a new B+ tree with the given buffer pool
func NewBPlusTree(bp *BufferPool) (*BPlusTree, error) {
	// Create root page (starts as Leaf)
	rootPage, err := bp.NewPage(PageTypeLeaf)
	if err != nil {
		return nil, err
	}

	tree := &BPlusTree{
		bp:     bp,
		rootID: rootPage.ID,
		order:  64, // max keys per node
	}

	// Root is pinned by NewPage, but NewBPlusTree returns ownership?
	// NewPaged pins it.
	// We want to unpin it so it's not permanently pinned?
	// But usually structure doesn't hold pin.
	bp.UnpinPage(rootPage.ID, true)
	return tree, nil
}

// GetRootID returns the root page ID
func (t *BPlusTree) GetRootID() PageID {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.rootID
}

// Insert adds or updates a key-value pair in the B+Tree.
//
// Process:
// 1. **Traverse**: Recurse down to the appropriate leaf node.
// 2. **Insert**: Add entry to leaf.
// 3. **Split**: If node overflows, split into two and bubbles up the median key.
// 4. **Root Split**: If the root splits, a new root is created pointing to the two halves.
func (t *BPlusTree) Insert(key, value []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Recursive insert
	splitKey, splitPageID, err := t.insertRecursive(t.rootID, key, value)
	if err != nil {
		return err
	}

	// Check if root split
	if splitPageID != 0 {
		// Create new root
		newRoot, err := t.bp.NewPage(PageTypeIndex)
		if err != nil {
			return err
		}

		// New Root Points to: P0=OldRoot, K1=SplitKey, P1=SplitNode
		childBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(childBytes, uint64(splitPageID))

		entry := Entry{Key: splitKey, Value: childBytes}

		t.setLeftPtr(newRoot, t.rootID)
		if err := t.writeInternalEntries(newRoot, t.rootID, []Entry{entry}); err != nil {
			return err
		}

		// Update tree root
		t.rootID = newRoot.ID
		if t.onRootChange != nil {
			t.onRootChange(t.rootID)
		}
		t.bp.UnpinPage(newRoot.ID, true)
	}

	return nil
}

// insertRecursive descends the tree, inserts entry, and handles splits on the way up.
// Returns: (Promoted Key, New Sibling PageID) if a split occurred, else (nil, 0).
func (t *BPlusTree) insertRecursive(pageID PageID, key, value []byte) ([]byte, PageID, error) {
	page, err := t.bp.FetchPage(pageID)
	if err != nil {
		return nil, 0, err
	}
	defer t.bp.UnpinPage(pageID, true)

	pageType := page.GetPageType()

	if pageType == PageTypeLeaf {
		return t.insertIntoLeafRecursive(page, key, value)
	} else if pageType == PageTypeIndex {
		// Internal Node
		childID, err := t.searchInternal(page, key)
		if err != nil {
			return nil, 0, err
		}

		promoteKey, splitChildID, err := t.insertRecursive(childID, key, value)
		if err != nil {
			return nil, 0, err
		}

		if splitChildID == 0 {
			return nil, 0, nil
		}

		return t.insertIntoInternalRecursive(page, promoteKey, splitChildID)
	} else {
		return nil, 0, fmt.Errorf("invalid page type %d encountered", pageType)
	}
}

// insertIntoLeafRecursive handles insertion into leaf and potential splitting
func (t *BPlusTree) insertIntoLeafRecursive(page *Page, key, value []byte) ([]byte, PageID, error) {
	entries := t.getLeafEntries(page)

	// Check update
	for i, entry := range entries {
		if bytes.Equal(key, entry.Key) {
			entries[i].Value = value
			return nil, 0, t.writeLeafEntries(page, entries)
		}
	}

	// Insert sorted
	newEntry := Entry{Key: key, Value: value}
	insertPos := 0
	for i, entry := range entries {
		if bytes.Compare(key, entry.Key) < 0 {
			break
		}
		insertPos = i + 1
	}

	newEntries := make([]Entry, 0, len(entries)+1)
	newEntries = append(newEntries, entries[:insertPos]...)
	newEntries = append(newEntries, newEntry)
	newEntries = append(newEntries, entries[insertPos:]...)

	// Check if we need to split
	// Split if count > order OR size > PageSize
	// We scan size to be safe
	currentSize := PageHeaderSize
	for _, e := range newEntries {
		currentSize += 2 + len(e.Key) + 2 + len(e.Value)
	}

	if len(newEntries) > t.order || currentSize > PageSize-16 { // Safety margin
		// Leaf Split
		mid := len(newEntries) / 2
		rightEntries := newEntries[mid:]
		leftEntries := newEntries[:mid]

		newPage, err := t.bp.NewPage(PageTypeLeaf)
		if err != nil {
			return nil, 0, err
		}
		defer t.bp.UnpinPage(newPage.ID, true)

		// Link Leafs
		oldNext := page.GetNextPage()
		page.SetNextPage(newPage.ID)
		newPage.SetNextPage(oldNext)
		newPage.SetPrevPage(page.ID)

		if oldNext != 0 {
			oldNextPage, err := t.bp.FetchPage(oldNext)
			if err == nil {
				oldNextPage.SetPrevPage(newPage.ID)
				t.bp.UnpinPage(oldNext, true)
			}
		}

		if err := t.writeLeafEntries(page, leftEntries); err != nil {
			return nil, 0, err
		}
		if err := t.writeLeafEntries(newPage, rightEntries); err != nil {
			return nil, 0, err
		}

		// Promote Key (Copy up) - First key of Right Node
		promoteKey := rightEntries[0].Key
		return promoteKey, newPage.ID, nil
	}

	return nil, 0, t.writeLeafEntries(page, newEntries)
}

// insertIntoInternalRecursive inserts key/childID into internal node and splits if needed
func (t *BPlusTree) insertIntoInternalRecursive(page *Page, key []byte, childID PageID) ([]byte, PageID, error) {
	entries := t.getInternalEntries(page)
	leftPtr := t.getLeftPtr(page)

	childBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(childBytes, uint64(childID))

	newEntry := Entry{Key: key, Value: childBytes}

	insertPos := 0
	for i, entry := range entries {
		if bytes.Compare(key, entry.Key) < 0 {
			break
		}
		insertPos = i + 1
	}

	newEntries := make([]Entry, 0, len(entries)+1)
	newEntries = append(newEntries, entries[:insertPos]...)
	newEntries = append(newEntries, newEntry)
	newEntries = append(newEntries, entries[insertPos:]...)

	if len(newEntries) > t.order {
		return t.actualSplitInternal(page, leftPtr, newEntries)
	}

	return nil, 0, t.writeInternalEntries(page, leftPtr, newEntries)
}

// Delete removes a key from the B+ tree
func (t *BPlusTree) Delete(key []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	rootPage, err := t.bp.FetchPage(t.rootID)
	if err != nil {
		return err
	}
	defer t.bp.UnpinPage(rootPage.ID, false)

	// Traverse to leaf
	leafPage, err := t.findLeafPage(rootPage, key)
	if err != nil {
		return err
	}
	if leafPage.ID != rootPage.ID {
		defer t.bp.UnpinPage(leafPage.ID, false)
	}

	// Remove from leaf
	return t.deleteFromLeaf(leafPage, key)
}

// deleteFromLeaf removes a key from a leaf page
func (t *BPlusTree) deleteFromLeaf(leafPage *Page, key []byte) error {
	entries := t.getLeafEntries(leafPage)

	newEntries := make([]Entry, 0, len(entries))
	found := false

	for _, entry := range entries {
		if bytes.Equal(entry.Key, key) {
			found = true
			continue
		}
		newEntries = append(newEntries, entry)
	}

	if !found {
		return util.ErrDocumentNotFound
	}

	// Write back (this updates KeyCount and effectively removes the item)
	// Note: We are NOT handling underflow/merging here.
	// Pages might become empty or sparse. This is a "lazy" deletion strategy
	// common in MVCC or simplified storage engines.
	return t.writeLeafEntries(leafPage, newEntries)
}

// Search searches for a key in the B+ tree
func (t *BPlusTree) Search(key []byte) ([]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	rootPage, err := t.bp.FetchPage(t.rootID)
	if err != nil {
		return nil, err
	}
	defer t.bp.UnpinPage(rootPage.ID, false)

	// Traverse to leaf
	// If root is leaf, use it directly
	// If root is index, use findLeafPage
	// findLeafPage handles descent

	leafPage, err := t.findLeafPage(rootPage, key)
	if err != nil {
		return nil, err
	}

	// If leafPage != rootPage, we need to defer unpin
	if leafPage.ID != rootPage.ID {
		defer t.bp.UnpinPage(leafPage.ID, false)
	}

	return t.searchInLeaf(leafPage, key)
}

// RangeScan returns all key-value entries falling within [startKey, endKey] (inclusive).
//
// Logic:
// 1. **Find Start**: Traverses to the leaf page containing startKey.
// 2. **Link Traversal**: Iterates linearly through leaf pages using `NextPageID` pointers.
// 3. **Filter**: Collects items where key is in range. stops if key > endKey.
//
// This method handles page pinning/unpinning carefully across page boundaries.
func (t *BPlusTree) RangeScan(startKey, endKey []byte) ([]Entry, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var results []Entry

	rootPage, err := t.bp.FetchPage(t.rootID)
	if err != nil {
		return nil, err
	}
	defer t.bp.UnpinPage(rootPage.ID, false)

	leafPage, err := t.findLeafPage(rootPage, startKey)
	if err != nil {
		return nil, err
	}

	// Scan logic
	// We need to loop leafPage -> Next
	// leafPage is Pinned by findLeafPage (or it is rootPage)

	// We need manual unpin logic for the loop
	// If leafPage == rootPage, it is pinned by defer above.
	// If different, we must unpin it when moving or finishing.

	currentID := leafPage.ID
	isRoot := (currentID == rootPage.ID)

	for {
		entries := t.getLeafEntries(leafPage)
		for _, entry := range entries {
			// Check if within range
			if bytes.Compare(entry.Key, startKey) >= 0 && bytes.Compare(entry.Key, endKey) <= 0 {
				results = append(results, entry)
			}
			// Stop if we've passed endKey
			if bytes.Compare(entry.Key, endKey) > 0 {
				if !isRoot {
					t.bp.UnpinPage(currentID, false)
				}
				return results, nil
			}
		}

		nextID := leafPage.GetNextPage()
		if nextID == 0 {
			break
		}

		if !isRoot {
			t.bp.UnpinPage(currentID, false)
		}

		// Move to next
		leafPage, err = t.bp.FetchPage(nextID)
		if err != nil {
			return results, nil
		}
		currentID = leafPage.ID
		isRoot = false // Any next page is not root (if root was leaf)
	}

	if !isRoot {
		t.bp.UnpinPage(currentID, false)
	}

	return results, nil
}

// findLeafPage navigates from an index page to the appropriate leaf page
func (t *BPlusTree) findLeafPage(indexPage *Page, key []byte) (*Page, error) {
	currentPage := indexPage
	// While current page is internal, descend
	for currentPage.GetPageType() == PageTypeIndex {
		childID, err := t.searchInternal(currentPage, key)
		if err != nil {
			return nil, err
		}

		nextPage, err := t.bp.FetchPage(childID)
		if err != nil {
			return nil, err
		}

		if currentPage.ID != indexPage.ID {
			t.bp.UnpinPage(currentPage.ID, false)
		}

		currentPage = nextPage
	}

	return currentPage, nil
}

// searchInLeaf searches for a key within a leaf page
func (t *BPlusTree) searchInLeaf(leafPage *Page, key []byte) ([]byte, error) {
	entries := t.getLeafEntries(leafPage)
	// Binary search
	left, right := 0, len(entries)-1
	for left <= right {
		mid := (left + right) / 2
		cmp := bytes.Compare(key, entries[mid].Key)
		if cmp == 0 {
			return entries[mid].Value, nil
		} else if cmp < 0 {
			right = mid - 1
		} else {
			left = mid + 1
		}
	}
	return nil, util.ErrDocumentNotFound
}

// getLeafEntries reads all entries from a leaf page
func (t *BPlusTree) getLeafEntries(leafPage *Page) []Entry {
	var entries []Entry
	leafPage.mu.RLock()
	defer leafPage.mu.RUnlock()
	keyCount := int(binary.LittleEndian.Uint16(leafPage.Data[2:4]))
	if keyCount == 0 {
		return entries
	}
	offset := PageHeaderSize
	for i := 0; i < keyCount && offset < PageSize-8; i++ {
		if offset+2 > PageSize {
			break
		}
		keyLen := int(binary.LittleEndian.Uint16(leafPage.Data[offset : offset+2]))
		offset += 2
		if offset+keyLen > PageSize {
			break
		}
		key := make([]byte, keyLen)
		copy(key, leafPage.Data[offset:offset+keyLen])
		offset += keyLen
		if offset+2 > PageSize {
			break
		}
		valueLen := int(binary.LittleEndian.Uint16(leafPage.Data[offset : offset+2]))
		offset += 2
		if offset+valueLen > PageSize {
			break
		}
		value := make([]byte, valueLen)
		copy(value, leafPage.Data[offset:offset+valueLen])
		offset += valueLen
		entries = append(entries, Entry{Key: key, Value: value})
	}
	return entries
}

// writeLeafEntries writes entries to a leaf page
func (t *BPlusTree) writeLeafEntries(leafPage *Page, entries []Entry) error {
	leafPage.mu.Lock()
	defer leafPage.mu.Unlock()

	for i := PageHeaderSize; i < PageSize; i++ {
		leafPage.Data[i] = 0
	}

	offset := PageHeaderSize
	for i, entry := range entries {
		needed := 2 + len(entry.Key) + 2 + len(entry.Value)
		if offset+needed > PageSize {
			return fmt.Errorf("%w: cannot fit entry %d", util.ErrPageFull, i)
		}
		binary.LittleEndian.PutUint16(leafPage.Data[offset:offset+2], uint16(len(entry.Key)))
		offset += 2
		copy(leafPage.Data[offset:offset+len(entry.Key)], entry.Key)
		offset += len(entry.Key)
		binary.LittleEndian.PutUint16(leafPage.Data[offset:offset+2], uint16(len(entry.Value)))
		offset += 2
		copy(leafPage.Data[offset:offset+len(entry.Value)], entry.Value)
		offset += len(entry.Value)
	}
	binary.LittleEndian.PutUint16(leafPage.Data[2:4], uint16(len(entries)))
	binary.LittleEndian.PutUint16(leafPage.Data[4:6], uint16(offset))
	leafPage.IsDirty = true
	return nil
}

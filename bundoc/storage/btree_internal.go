package storage

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/kartikbazzad/bunbase/bundoc/internal/util"
)

// Internal Node Layout:
// Header (30 bytes)
// LeftPtr (8 bytes) - P0
// Entries (Key, Value=PageID)

const InternalHeaderSize = PageHeaderSize + 8

// getLeftPtr returns the left-most child pointer (P0)
func (t *BPlusTree) getLeftPtr(page *Page) PageID {
	page.mu.RLock()
	defer page.mu.RUnlock()
	return PageID(binary.LittleEndian.Uint64(page.Data[PageHeaderSize : PageHeaderSize+8]))
}

// setLeftPtr sets the left-most child pointer (P0)
func (t *BPlusTree) setLeftPtr(page *Page, ptr PageID) {
	page.mu.Lock()
	defer page.mu.Unlock()
	binary.LittleEndian.PutUint64(page.Data[PageHeaderSize:PageHeaderSize+8], uint64(ptr))
	page.IsDirty = true
}

// getInternalEntries reads all entries from an internal page
func (t *BPlusTree) getInternalEntries(page *Page) []Entry {
	var entries []Entry

	page.mu.RLock()
	defer page.mu.RUnlock()

	keyCount := int(binary.LittleEndian.Uint16(page.Data[2:4]))
	if keyCount == 0 {
		return entries
	}

	offset := InternalHeaderSize
	for i := 0; i < keyCount && offset < PageSize-8; i++ {
		// Read key length
		if offset+2 > PageSize {
			break
		}
		keyLen := int(binary.LittleEndian.Uint16(page.Data[offset : offset+2]))
		offset += 2

		// Read key
		if offset+keyLen > PageSize {
			break
		}
		key := make([]byte, keyLen)
		copy(key, page.Data[offset:offset+keyLen])
		offset += keyLen

		// Read value (PageID - 8 bytes)
		// Internal nodes store PageIDs as values. We treat them as bytes in Entry
		// but we know they are 8 bytes.
		// Standard simple serialization for 'Value' field:
		// We'll trust standard logic: (ValLen=8) + (8 bytes)

		if offset+2 > PageSize {
			break
		}
		valLen := int(binary.LittleEndian.Uint16(page.Data[offset : offset+2]))
		offset += 2

		if offset+valLen > PageSize {
			break
		}
		value := make([]byte, valLen)
		copy(value, page.Data[offset:offset+valLen])
		offset += valLen

		entries = append(entries, Entry{Key: key, Value: value})
	}

	return entries
}

// writeInternalEntries writes entries to an internal page
func (t *BPlusTree) writeInternalEntries(page *Page, leftPtr PageID, entries []Entry) error {
	page.mu.Lock()
	defer page.mu.Unlock()

	// Write Left Pointer
	binary.LittleEndian.PutUint64(page.Data[PageHeaderSize:PageHeaderSize+8], uint64(leftPtr))

	// Clear remaining data
	for i := InternalHeaderSize; i < PageSize; i++ {
		page.Data[i] = 0
	}

	offset := InternalHeaderSize
	for i, entry := range entries {
		// Check space
		// KeyLen(2) + Key + ValLen(2) + Val
		needed := 2 + len(entry.Key) + 2 + len(entry.Value)
		if offset+needed > PageSize {
			return fmt.Errorf("%w: cannot fit internal entry %d", util.ErrPageFull, i)
		}

		// Write Key
		binary.LittleEndian.PutUint16(page.Data[offset:offset+2], uint16(len(entry.Key)))
		offset += 2
		copy(page.Data[offset:offset+len(entry.Key)], entry.Key)
		offset += len(entry.Key)

		// Write Value (PageID)
		binary.LittleEndian.PutUint16(page.Data[offset:offset+2], uint16(len(entry.Value)))
		offset += 2
		copy(page.Data[offset:offset+len(entry.Value)], entry.Value)
		offset += len(entry.Value)
	}

	// Update header
	binary.LittleEndian.PutUint16(page.Data[2:4], uint16(len(entries))) // KeyCount
	binary.LittleEndian.PutUint16(page.Data[4:6], uint16(offset))       // FreeSpace
	page.IsDirty = true

	return nil
}

// searchInternal finds the child page ID that might contain the key
func (t *BPlusTree) searchInternal(page *Page, key []byte) (PageID, error) {
	leftPtr := t.getLeftPtr(page)
	entries := t.getInternalEntries(page)

	// Iterate entries to find the separator
	// Entries: (K1, P1), (K2, P2)...
	// P0 is leftPtr.
	// Logic:
	// if key < K1: return P0
	// if key < K2: return P1
	// ...
	// else return Pn

	currPtr := leftPtr
	for _, entry := range entries {
		if bytes.Compare(key, entry.Key) < 0 {
			return currPtr, nil
		}
		// Decode PageID from entry.Value
		if len(entry.Value) != 8 {
			return 0, fmt.Errorf("invalid internal node value length")
		}
		currPtr = PageID(binary.LittleEndian.Uint64(entry.Value))
	}

	return currPtr, nil
}

// insertIntoInternal inserts a (Key, ChildID) pair into an internal node
func (t *BPlusTree) insertIntoInternal(page *Page, key []byte, childID PageID) error {
	entries := t.getInternalEntries(page)
	leftPtr := t.getLeftPtr(page)

	// Encode ChildID
	childBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(childBytes, uint64(childID))

	// Find insertion position
	insertPos := 0
	for i, entry := range entries {
		if bytes.Compare(key, entry.Key) < 0 {
			break
		}
		insertPos = i + 1
	}

	newEntry := Entry{Key: key, Value: childBytes}
	newEntries := make([]Entry, 0, len(entries)+1)
	newEntries = append(newEntries, entries[:insertPos]...)
	newEntries = append(newEntries, newEntry)
	newEntries = append(newEntries, entries[insertPos:]...)

	if len(newEntries) > t.order {
		return t.splitInternal(page, leftPtr, newEntries)
	}

	return t.writeInternalEntries(page, leftPtr, newEntries)
}

// splitInternal splits an internal node
func (t *BPlusTree) splitInternal(page *Page, leftPtr PageID, entries []Entry) error {
	return fmt.Errorf("splitInternal needs refactoring to return promote Key")
}

// actualSplitInternal performs the split and returns promoted info
func (t *BPlusTree) actualSplitInternal(page *Page, leftPtr PageID, entries []Entry) ([]byte, PageID, error) {
	newPage, err := t.bp.NewPage(PageTypeIndex)
	if err != nil {
		return nil, 0, err
	}
	defer t.bp.UnpinPage(newPage.ID, true) // Unpin when done writing

	// Split logic for Internal Nodes:
	// Entries: [0..mid-1], [mid], [mid+1..end]
	// [0..mid-1] stay in Old Page.
	// [mid] is Promoted Key.
	// [mid+1..end] go to New Page.
	// AND: The LeftPtr of New Page becomes the Value (PageID) associated with [mid].
	// Wait?
	// Standard:
	// Old: P0 K1 P1 ... Kmid Pmid ... Kn Pn
	// Promote Kmid.
	// Left: P0 ... Pmid-1 (Entries < mid)
	// Right: Pmid ... Pn (Entries > mid) (Pmid becomes LeftPtr of Right Node)

	mid := len(entries) / 2
	promoteEntry := entries[mid]
	promoteKey := promoteEntry.Key

	// Right Node Left Pointer comes from the promote entry's value (pointer)
	rightLeftPtr := PageID(binary.LittleEndian.Uint64(promoteEntry.Value))

	leftEntries := entries[:mid]
	rightEntries := entries[mid+1:]

	// Write Old Page (Left)
	if err := t.writeInternalEntries(page, leftPtr, leftEntries); err != nil {
		return nil, 0, err
	}

	// Write New Page (Right)
	if err := t.writeInternalEntries(newPage, rightLeftPtr, rightEntries); err != nil {
		return nil, 0, err
	}

	return promoteKey, newPage.ID, nil
}

package bundoc

import (
	"fmt"
	"sort"

	"github.com/kartikbazzad/bunbase/bundoc/internal/query"
	"github.com/kartikbazzad/bunbase/bundoc/internal/transaction"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

// Iterator defines the interface for iterating over document results.
// It follows the standard Cursor pattern: Next() advances, Value() retrieves.
type Iterator interface {
	Next() bool                       // Advances to the next document. Returns false if exhausted.
	Value() (storage.Document, error) // Returns the current document.
	Close() error                     // Releases resources (e.g., unpins pages).
}

// TableScanIterator iterates over all documents in a collection.
// It essentially performs a full scan of the Primary Index (_id).
type TableScanIterator struct {
	collection   *Collection
	txn          *transaction.Transaction
	docIDs       []string // Snapshot of IDs to iterate
	currentIndex int
}

func NewTableScanIterator(c *Collection, txn *transaction.Transaction) (*TableScanIterator, error) {
	// For MVP TableScan, we just grab all IDs from primary index?
	// Or we scan the B+Tree.
	// B+Tree RangeScan(nil, nil) to get all keys.
	startKey := []byte{0x00}
	endKey := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	c.mu.RLock()
	entries, err := c.indexes["_id"].RangeScan(startKey, endKey)
	c.mu.RUnlock()

	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		// Key is "collectionName:ID"
		// We need to extract ID? Or just keep key?
		// FindByID expects ID.
		// Key format: name + ":" + id
		// But wait, the RangeScan returns keys.
		// Actually, FindByID does the lookup.
		// Let's just trust we can derive ID.
		// But wait, RangeScan returns raw bytes.
		// We can deserialize the Value directly?
		// Yes, Primary Index stores Document Data in Value!
		// But we need to check MVCC Visibility via FindByID (or similar logic).
		// Re-using FindByID is safest for MVCC.
		// So we extract ID from the key.
		// The key stored in B+Tree is "collection:id".
		// We can strip prefix.
		fullKey := string(entry.Key)
		// prefix is c.name + ":"
		prefixLen := len(c.name) + 1
		if len(fullKey) > prefixLen {
			ids = append(ids, fullKey[prefixLen:])
		}
	}

	return &TableScanIterator{
		collection:   c,
		txn:          txn,
		docIDs:       ids,
		currentIndex: -1,
	}, nil
}

func (it *TableScanIterator) Next() bool {
	it.currentIndex++
	return it.currentIndex < len(it.docIDs)
}

func (it *TableScanIterator) Value() (storage.Document, error) {
	if it.currentIndex < 0 || it.currentIndex >= len(it.docIDs) {
		return nil, fmt.Errorf("iterator out of bounds")
	}
	// Fetch document using standard FindByID to ensure MVCC visibility rules
	return it.collection.FindByID(nil, it.txn, it.docIDs[it.currentIndex])
}

func (it *TableScanIterator) Close() error {
	return nil
}

// IndexScanIterator leverages a secondary B+Tree index to find documents.
// It iterates over the index to find Document IDs, then fetches the full document
// from the Primary Index.
type IndexScanIterator struct {
	collection   *Collection
	txn          *transaction.Transaction
	docIDs       []string
	currentIndex int
}

func NewIndexScanIterator(c *Collection, txn *transaction.Transaction, field string, startKey, endKey []byte) (*IndexScanIterator, error) {
	c.mu.RLock()
	index, ok := c.indexes[field]
	// RangeScan needs to be atomic/locked regarding BMS structure?
	// Assuming RangeScan is read-safe with RLock on collection (which protects index map)
	// But we also need RLock on the Tree itself? Or is the tree thread-safe?
	// The Tree implementation uses BufferPool.
	// We should hold Collection RLock to ensure Index doesn't disappear.
	// Real implementation would just lock the index.

	if !ok {
		c.mu.RUnlock()
		return nil, fmt.Errorf("index not found for field: %s", field)
	}

	entries, err := index.RangeScan(startKey, endKey)
	c.mu.RUnlock()

	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		// Secondary Index Value IS the DocID
		// Entry.Key = FieldVal + \0 + DocID
		// Entry.Value = DocID (we store it there for easy retrieval?)
		// Let's check collection.go Insert logic.
		// "index.Insert(compKey, []byte(string(id)))"
		// Yes, Value is DocID.
		ids = append(ids, string(entry.Value))
	}

	return &IndexScanIterator{
		collection:   c,
		txn:          txn,
		docIDs:       ids,
		currentIndex: -1,
	}, nil
}

func (it *IndexScanIterator) Next() bool {
	it.currentIndex++
	return it.currentIndex < len(it.docIDs)
}

func (it *IndexScanIterator) Value() (storage.Document, error) {
	if it.currentIndex < 0 || it.currentIndex >= len(it.docIDs) {
		return nil, fmt.Errorf("iterator out of bounds")
	}
	// Retrieve Doc by ID (Visibility check via FindByID)
	return it.collection.FindByID(nil, it.txn, it.docIDs[it.currentIndex])
}

func (it *IndexScanIterator) Close() error {
	return nil
}

// FilterIterator filters documents based on AST
type FilterIterator struct {
	source  Iterator
	matcher query.Matcher
	current storage.Document
}

func NewFilterIterator(source Iterator, matcher query.Matcher) *FilterIterator {
	return &FilterIterator{
		source:  source,
		matcher: matcher,
	}
}

func (it *FilterIterator) Next() bool {
	for it.source.Next() {
		doc, err := it.source.Value()
		if err != nil {
			// Skip deleted/invisible docs (standard FindByID behavior might return err not found)
			continue
		}

		// Convert doc to map for Matcher?
		// storage.Document IS map[string]interface{}
		if it.matcher.Matches(doc) {
			it.current = doc
			return true
		}
	}
	return false
}

func (it *FilterIterator) Value() (storage.Document, error) {
	return it.current, nil
}

func (it *FilterIterator) Close() error {
	return it.source.Close()
}

// LimitIterator limits the number of results
type LimitIterator struct {
	source Iterator
	limit  int
	count  int
}

func NewLimitIterator(source Iterator, limit int) *LimitIterator {
	return &LimitIterator{
		source: source,
		limit:  limit,
	}
}

func (it *LimitIterator) Next() bool {
	if it.count >= it.limit {
		return false
	}
	if it.source.Next() {
		it.count++
		return true
	}
	return false
}

func (it *LimitIterator) Value() (storage.Document, error) {
	return it.source.Value()
}

func (it *LimitIterator) Close() error {
	return it.source.Close()
}

// SkipIterator skips the first N results
type SkipIterator struct {
	source  Iterator
	skip    int
	skipped bool
}

func NewSkipIterator(source Iterator, skip int) *SkipIterator {
	return &SkipIterator{
		source: source,
		skip:   skip,
	}
}

func (it *SkipIterator) Next() bool {
	if !it.skipped {
		// Skip first N items
		for i := 0; i < it.skip; i++ {
			if !it.source.Next() {
				return false // Source exhausted before skip finished
			}
		}
		it.skipped = true
	}
	return it.source.Next()
}

func (it *SkipIterator) Value() (storage.Document, error) {
	return it.source.Value()
}

func (it *SkipIterator) Close() error {
	return it.source.Close()
}

// SortIterator buffers all results, sorts them, and iterates
type SortIterator struct {
	source    Iterator
	sortField string
	desc      bool
	docs      []storage.Document
	index     int
	prepared  bool
}

func NewSortIterator(source Iterator, field string, desc bool) *SortIterator {
	return &SortIterator{
		source:    source,
		sortField: field,
		desc:      desc,
		index:     -1,
	}
}

func (it *SortIterator) Next() bool {
	if !it.prepared {
		// Buffer all docs
		for it.source.Next() {
			doc, err := it.source.Value()
			if err == nil {
				it.docs = append(it.docs, doc)
			}
		}
		it.source.Close() // Close source as we consumed it all

		// Sort docs
		// We use standard sort.Slice
		if it.sortField != "" {
			sort.Slice(it.docs, func(i, j int) bool {
				valA := it.docs[i][it.sortField]
				valB := it.docs[j][it.sortField]
				// Use query.CompareValues
				result := query.CompareValues(valA, valB)
				if it.desc {
					return result > 0 // Descending
				}
				return result < 0 // Ascending
			})
		}
		it.prepared = true
	}

	it.index++
	return it.index < len(it.docs)
}

func (it *SortIterator) Value() (storage.Document, error) {
	if it.index < 0 || it.index >= len(it.docs) {
		return nil, fmt.Errorf("iterator out of bounds")
	}
	return it.docs[it.index], nil
}

func (it *SortIterator) Close() error {
	it.docs = nil // Release memory
	return nil
}

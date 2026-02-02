package bundoc

import (
	"fmt"
	"sync"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc/internal/query"
	"github.com/kartikbazzad/bunbase/bundoc/internal/transaction"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

// Collection represents a collection of documents
// Collection represents a logical grouping of documents (similar to a table in SQL).
// It manages the primary storage (Transaction Write Set / WAL) and all associated
// B+Tree indexes.
type Collection struct {
	name    string
	db      *Database
	indexes map[string]*storage.BPlusTree // Map of field name -> B+Tree Index
	mu      sync.RWMutex                  // Protects concurrent access to indexes map
}

// Name returns the collection name
func (c *Collection) Name() string {
	return c.name
}

// Insert inserts a new document into the collection.
//
// The operation follows these steps:
// 1. Storage: Writes the document data to the transaction's Write Set (and eventually WAL).
// 2. Indexing: Inserts an entry into the Primary Index (_id).
// 3. Secondary Indexes: Updates all secondary indexes with composite keys.
//
// This operation is atomic within the context of the transaction.
func (c *Collection) Insert(txn *transaction.Transaction, doc storage.Document) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get or generate document ID
	id, hasID := doc.GetID()
	if !hasID || id == "" {
		// Auto-generate ID if not provided
		id = storage.DocumentID(generateID())
		doc.SetID(id)
	}

	// Serialize document
	data, err := doc.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize document: %w", err)
	}

	// Write to transaction's write set
	key := c.name + ":" + string(id)
	if err := c.db.txnMgr.Write(txn, key, data); err != nil {
		return fmt.Errorf("failed to write document: %w", err)
	}

	// Insert into index (will be committed on transaction commit)
	// Primary index _id
	if err := c.indexes["_id"].Insert([]byte(key), data); err != nil {
		return fmt.Errorf("failed to insert into primary index: %w", err)
	}

	// Insert into secondary indexes
	for field, index := range c.indexes {
		if field == "_id" {
			continue
		}

		// Extract field value from document
		if val, ok := doc[field]; ok {
			// Create composite key: value + \0 + docID
			// We need to handle value types. For now assuming string or convertible to bytes.
			// Simple serialization: fmt.Sprint(val)
			valStr := fmt.Sprintf("%v", val)
			compKey := []byte(valStr + "\x00" + string(id))

			if err := index.Insert(compKey, []byte(string(id))); err != nil {
				return fmt.Errorf("failed to insert into index %s: %w", field, err)
			}
		}
	}

	return nil
}

// FindByID retrieves a document by its unique ID.
// It leverages MVCC to ensure that the returned document version is visible
// to the current transaction's snapshot.
func (c *Collection) FindByID(txn *transaction.Transaction, id string) (storage.Document, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.findByIDLocked(txn, id)
}

// findByIDLocked finds a document by ID without locking (callers must hold lock)
func (c *Collection) findByIDLocked(txn *transaction.Transaction, id string) (storage.Document, error) {
	key := c.name + ":" + id

	// Try reading from transaction's write set first
	data, err := c.db.txnMgr.Read(txn, key)
	if err == nil && data != nil {
		// Found in write set
		doc, err := storage.DeserializeDocument(data)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize document: %w", err)
		}
		return doc, nil
	}

	// Search in index
	data, err = c.indexes["_id"].Search([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("document not found: %s", id)
	}

	// Deserialize
	doc, err := storage.DeserializeDocument(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize document: %w", err)
	}

	return doc, nil
}

// Update modifies an existing document.
//
// This method handles full Index Maintenance:
// 1. Fetches the old document to identify changed fields.
// 2. Writes the new document version to the transaction log.
// 3. Updates the Primary Index.
// 4. Updates all affected Secondary Indexes (deleting old keys, inserting new ones).
func (c *Collection) Update(txn *transaction.Transaction, id string, doc storage.Document) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.name + ":" + id

	// Ensure ID matches
	doc.SetID(storage.DocumentID(id))

	// Serialize
	data, err := doc.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize document: %w", err)
	}

	// 1. Fetch old document state for index maintenance
	// We need to know old values to delete them from secondary indexes
	// 1. Fetch old document state for index maintenance
	// We need to know old values to delete them from secondary indexes
	oldDoc, err := c.findByIDLocked(txn, id)
	// FindByID returns error if not found, usually.
	// FindByID returns error if not found, usually.
	if err == nil {
		// fmt.Printf("DEBUG: Update Found Old Doc! Fields: %v\n", oldDoc)
	} else {
		// fmt.Printf("DEBUG: Update Old Doc NOT FOUND via FindByID. Err: %v\n", err)
	}

	// 2. Write new document data to transaction (Primary Store)
	if err := c.db.txnMgr.Write(txn, key, data); err != nil {
		return fmt.Errorf("failed to write document: %w", err)
	}

	// 3. Update Primary Index
	if err := c.indexes["_id"].Insert([]byte(key), data); err != nil {
		return fmt.Errorf("failed to update index: %w", err)
	}

	// 4. Maintenance of Secondary Indexes
	for field, index := range c.indexes {
		if field == "_id" {
			continue
		}

		var oldVal interface{}
		var newVal interface{}
		hasOld := false
		hasNew := false

		if oldDoc != nil {
			oldVal, hasOld = oldDoc[field]
		}
		newVal, hasNew = doc[field]

		// Case A: Field was present, now changing or removed -> Delete Old Entry
		if hasOld {
			// Check if value actually changed or if it's being removed
			valChanged := !hasNew || fmt.Sprintf("%v", oldVal) != fmt.Sprintf("%v", newVal)

			if valChanged {
				valStr := fmt.Sprintf("%v", oldVal)
				oldCompKey := []byte(valStr + "\x00" + string(id))

				// Attempt delete. If not found (e.g. index created after doc), ignore error.
				_ = index.Delete(oldCompKey)
			}
		}

		// Case B: Field is present in new doc -> Insert New Entry
		// (We Insert if it's new OR if it changed. If it didn't change, we didn't delete, so no need to insert?
		//  Wait, if we didn't delete, we don't need to insert only if the key is exactly identical.
		//  But strict correctness safe way: Always Insert if hasNew. B+Tree Insert overwrites/duplicates handling.)
		//  Optimization: Only Insert if `!hasOld` or `valChanged`.

		shouldInsert := hasNew
		if hasOld && hasNew && fmt.Sprintf("%v", oldVal) == fmt.Sprintf("%v", newVal) {
			shouldInsert = false // Value unchanged, index entry remains valid
		}

		if shouldInsert {
			valStr := fmt.Sprintf("%v", newVal)
			compKey := []byte(valStr + "\x00" + string(id))

			if err := index.Insert(compKey, []byte(string(id))); err != nil {
				return fmt.Errorf("failed to update index %s: %w", field, err)
			}
		}
	}

	return nil
}

// InsertBatch inserts multiple documents into the collection
func (c *Collection) InsertBatch(txn *transaction.Transaction, docs []storage.Document) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, doc := range docs {
		// Get or generate document ID
		id, hasID := doc.GetID()
		if !hasID || id == "" {
			// Auto-generate ID if not provided
			id = storage.DocumentID(generateID())
			doc.SetID(id)
		}

		// Serialize document
		data, err := doc.Serialize()
		if err != nil {
			return fmt.Errorf("failed to serialize document: %w", err)
		}

		// Write to transaction's write set
		key := c.name + ":" + string(id)
		if err := c.db.txnMgr.Write(txn, key, data); err != nil {
			return fmt.Errorf("failed to write document: %w", err)
		}

		// Insert into index
		// Primary index _id
		if err := c.indexes["_id"].Insert([]byte(key), data); err != nil {
			return fmt.Errorf("failed to insert into primary index: %w", err)
		}

		// Insert into secondary indexes
		for field, index := range c.indexes {
			if field == "_id" {
				continue
			}

			// Extract field value from document
			if val, ok := doc[field]; ok {
				valStr := fmt.Sprintf("%v", val)
				compKey := []byte(valStr + "\x00" + string(id))

				if err := index.Insert(compKey, []byte(string(id))); err != nil {
					return fmt.Errorf("failed to insert into index %s: %w", field, err)
				}
			}
		}
	}

	return nil
}

// UpdateBatch updates multiple documents in the collection
func (c *Collection) UpdateBatch(txn *transaction.Transaction, docs []storage.Document) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, doc := range docs {
		id, hasID := doc.GetID()
		if !hasID || id == "" {
			return fmt.Errorf("document must have an ID for update")
		}

		key := c.name + ":" + string(id)

		// Serialize
		data, err := doc.Serialize()
		if err != nil {
			return fmt.Errorf("failed to serialize document: %w", err)
		}

		// Write to transaction
		if err := c.db.txnMgr.Write(txn, key, data); err != nil {
			return fmt.Errorf("failed to write document: %w", err)
		}

		// Update index
		if err := c.indexes["_id"].Insert([]byte(key), data); err != nil {
			return fmt.Errorf("failed to update index: %w", err)
		}

		// Update secondary indexes
		for field, index := range c.indexes {
			if field == "_id" {
				continue
			}
			if val, ok := doc[field]; ok {
				valStr := fmt.Sprintf("%v", val)
				compKey := []byte(valStr + "\x00" + string(id))
				if err := index.Insert(compKey, []byte(string(id))); err != nil {
					return fmt.Errorf("failed to update index %s: %w", field, err)
				}
			}
		}
	}

	return nil
}

// Delete deletes a document
func (c *Collection) Delete(txn *transaction.Transaction, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.name + ":" + id

	// 1. Fetch document to clean up secondary indexes
	// 1. Fetch document to clean up secondary indexes
	doc, err := c.findByIDLocked(txn, id)
	if err == nil {
		for field, index := range c.indexes {
			if field == "_id" {
				continue
			}
			if val, ok := doc[field]; ok {
				valStr := fmt.Sprintf("%v", val)
				compKey := []byte(valStr + "\x00" + string(id))
				_ = index.Delete(compKey)
			}
		}
	}

	// 2. Write tombstone (Primary Store Deletion)
	if err := c.db.txnMgr.Write(txn, key, []byte{}); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	// 3. Delete from Primary Index
	// Note: We might want to keep it if we support "Soft Deletes" or Time Travel heavily,
	// but standard delete should remove it to keep index clean.
	// Primary index uses just Key.
	if err := c.indexes["_id"].Delete([]byte(key)); err != nil {
		// Log warning? Or ignore deeply if not found.
		// return fmt.Errorf("failed to delete from primary index: %w", err)
	}

	return nil
}

// DeleteBatch deletes multiple documents by ID
func (c *Collection) DeleteBatch(txn *transaction.Transaction, ids []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, id := range ids {
		key := c.name + ":" + id

		// Write tombstone
		if err := c.db.txnMgr.Write(txn, key, []byte{}); err != nil {
			return fmt.Errorf("failed to delete document: %w", err)
		}
	}

	return nil
}

// Count returns an approximate count of documents (simplified)
func (c *Collection) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// In a full implementation, would scan index
	// For now, return 0 as placeholder
	return 0
}

// EnsureIndex creates a secondary index for the given field if it doesn't already exist.
//
// Mechanism:
//  1. Checks if the index exists.
//  2. Creates a new B+Tree for the index.
//  3. Performs a Backfill operation by scanning the Primary Index and populating the new index
//     with existing documents.
//  4. Registers a persistence listener to save the index root ID on split.
//  5. Persists the new index metadata to the system catalog.
func (c *Collection) EnsureIndex(field string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if field == "_id" {
		return nil // Always exists
	}

	if _, exists := c.indexes[field]; exists {
		return nil // Already exists
	}

	fmt.Printf("[INFO] Auto-creating index for field '%s'...\n", field)

	// Create new index
	index, err := storage.NewBPlusTree(c.db.bufferPool)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// Backfill data from primary index
	// We need to scan all documents.
	// RangeScan with empty start/end scans everything?
	// Standard bytes.Compare: nil is less than everything? No, usually length check.
	// Let's assume RangeScan needs actual boundaries or we add a ScanAll.
	// For this implementation, we'll try RangeScan with empty byte slice as start and high byte as end.
	startKey := []byte{0x00}
	endKey := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF} // Max key?
	// A better way for BPlusTree might be exposing a Walk/Iterate method.
	// But let's use RangeScan for now. If keys are UUID/timestamps, they are ASCII/bytes.

	// Better: Use a dedicated ScanAll method on BPlusTree if available, or hack RangeScan.
	// Assuming keys are printable strings mostly, or standard binary.
	// Let's rely on the fact that we can just Iterate the leaf pages directly if we had access.
	// Using RangeScan with nil, nil is not supported by current implementation?
	// The current RangeScan checks: bytes.Compare(entry.Key, startKey) >= 0.
	// If startKey is empty []byte{}, Compare depends on impl.
	// Let's assume ScanAll logic here:

	// We will iterate through all entries in the primary index
	// (This is a simplified scan - in production we'd use a cursor)
	scanResults, err := c.indexes["_id"].RangeScan(startKey, endKey)
	if err != nil {
		return fmt.Errorf("failed to scan primary index: %w", err)
	}

	for _, entry := range scanResults {
		doc, err := storage.DeserializeDocument(entry.Value)
		if err != nil {
			continue // Skip corrupted docs
		}

		id, _ := doc.GetID()
		if val, ok := doc[field]; ok {
			valStr := fmt.Sprintf("%v", val)
			compKey := []byte(valStr + "\x00" + string(id))
			if err := index.Insert(compKey, []byte(string(id))); err != nil {
				return fmt.Errorf("failed to insert index entry: %w", err)
			}
		}
	}

	// Register listener for persistence
	index.SetOnRootChange(func(newRootID storage.PageID) {
		currentMeta, _ := c.db.metadataMgr.GetCollection(c.name)
		if currentMeta.Indexes == nil {
			currentMeta.Indexes = make(map[string]uint64)
		}
		currentMeta.Indexes[field] = uint64(newRootID)

		saveIdx := make(map[string]storage.PageID)
		for k, v := range currentMeta.Indexes {
			saveIdx[k] = storage.PageID(v)
		}
		c.db.metadataMgr.UpdateCollection(c.name, saveIdx)
	})

	c.indexes[field] = index

	// Persist New Index Metadata immediately
	currentMeta, _ := c.db.metadataMgr.GetCollection(c.name)
	if currentMeta.Indexes == nil {
		currentMeta.Indexes = make(map[string]uint64)
	}
	currentMeta.Indexes[field] = uint64(index.GetRootID())

	saveIdx := make(map[string]storage.PageID)
	for k, v := range currentMeta.Indexes {
		saveIdx[k] = storage.PageID(v)
	}
	if err := c.db.metadataMgr.UpdateCollection(c.name, saveIdx); err != nil {
		return fmt.Errorf("failed to persist index metadata: %w", err)
	}

	return nil
}

// Find searches for documents matching the given field and value
func (c *Collection) Find(txn *transaction.Transaction, field string, value interface{}) ([]storage.Document, error) {
	// Optimization: If field is _id, use FindByID
	if field == "_id" {
		idStr := fmt.Sprintf("%v", value)
		doc, err := c.FindByID(txn, idStr)
		if err != nil {
			return nil, err
		}
		return []storage.Document{doc}, nil
	}

	// 1. Lazy Index Creation
	// We need to check existence with Read Lock first, then Upgrade to Write Lock if needed.
	c.mu.RLock()
	_, exists := c.indexes[field]
	c.mu.RUnlock()

	if !exists {
		// Upgrade to write lock happens inside EnsureIndex
		if err := c.EnsureIndex(field); err != nil {
			return nil, err
		}
	}

	c.mu.RLock()
	index := c.indexes[field]
	c.mu.RUnlock()

	// 2. Range Scan on Index
	valStr := fmt.Sprintf("%v", value)
	startKey := []byte(valStr + "\x00")
	endKey := []byte(valStr + "\x00" + "\xFF")

	entries, err := index.RangeScan(startKey, endKey)
	if err != nil {
		return nil, fmt.Errorf("index scan failed: %w", err)
	}

	var docs []storage.Document
	for _, entry := range entries {
		// Value in secondary index is the DocID (primary key)
		docID := string(entry.Value)

		// 3. Fetch full document
		doc, err := c.FindByID(txn, docID)
		if err != nil {
			// Document might have been deleted but index update lagged?
			// Or just transactional visibility. FindByID handles visibility.
			// If FindByID errors (not found), we just skip.
			continue
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

// FindQuery executes a complex query against the collection
func (c *Collection) FindQuery(txn *transaction.Transaction, queryMap map[string]interface{}, opts ...QueryOptions) ([]storage.Document, error) {
	// 1. Parse Query
	node, err := query.Parse(queryMap)
	if err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	matcher, ok := node.(query.Matcher)
	if !ok {
		return nil, fmt.Errorf("parsed node does not implement Matcher")
	}

	// 2. Plan Query execution strategy
	var iter Iterator

	// Attempt to find an index usage strategy
	// We look for a top-level FieldNode or top-level logical AND with a FieldNode child
	// targeting an indexed field.
	// Simple Planner MVP: Inspect root node only.

	usedIndex := false

	// Case 1: Simple Field Node
	if fNode, ok := node.(*query.FieldNode); ok {
		// Check if index exists
		c.mu.RLock()
		_, hasIndex := c.indexes[fNode.Field]
		c.mu.RUnlock()

		if hasIndex {
			// Convert value to bytes for RangeScan
			// $eq: start=val, end=val
			// $gt: start=val+\0, end=max
			// $lt: start=min, end=val

			valStr := fmt.Sprintf("%v", fNode.Value)
			var startKey, endKey []byte

			switch fNode.Operator {
			case query.OpEq:
				// Exact match
				// RangeScan(valStr\0, valStr\0\xFF) covers all with that prefix?
				// Index entries are: valStr + \0 + DocID
				// So we want everything starting with valStr + \0
				startKey = []byte(valStr + "\x00")
				endKey = []byte(valStr + "\x00" + "\xFF")

			case query.OpGt, query.OpGte:
				startKey = []byte(valStr + "\x00")
				endKey = []byte{0xFF, 0xFF, 0xFF, 0xFF}
				if fNode.Operator == query.OpGt {
					// We might include valStr if we just use startKey.
					// FilterIterator will remove exact matches if needed,
					// or we adjust startKey slightly.
					// For MVP, relying on FilterIterator to cleanup boundary conditions is safe.
					// We just want to narrow the scan.
				}

			case query.OpLt, query.OpLte:
				startKey = []byte{0x00} // Min key
				endKey = []byte(valStr + "\x00")
			}

			if startKey != nil && endKey != nil {
				fmt.Printf("[PLANNER] Using Index Scan on field: %s\n", fNode.Field)
				idxIter, err := NewIndexScanIterator(c, txn, fNode.Field, startKey, endKey)
				if err == nil {
					iter = idxIter
					usedIndex = true
				}
			}
		}
	}

	// Default to TableScan if no index used
	if !usedIndex {
		// fmt.Println("[PLANNER] Using Table Scan") // Debug logging
		tsIter, err := NewTableScanIterator(c, txn)
		if err != nil {
			return nil, fmt.Errorf("failed to create iterator: %w", err)
		}
		iter = tsIter
	}

	// Chain iterators
	var currentIter Iterator = iter

	// 3. Apply Filter
	currentIter = NewFilterIterator(currentIter, matcher)

	// Apply Options
	if len(opts) > 0 {
		opt := opts[0]

		// 4. Sort
		if opt.SortField != "" {
			currentIter = NewSortIterator(currentIter, opt.SortField, opt.SortDesc)
		}

		// 5. Skip
		if opt.Skip > 0 {
			currentIter = NewSkipIterator(currentIter, opt.Skip)
		}

		// 6. Limit
		if opt.Limit > 0 {
			currentIter = NewLimitIterator(currentIter, opt.Limit)
		}
	}

	defer currentIter.Close()

	// 7. Collect Results
	var results []storage.Document
	for currentIter.Next() {
		doc, err := currentIter.Value()
		if err == nil {
			results = append(results, doc)
		}
	}

	return results, nil
}

// generateID generates a unique document ID
func generateID() string {
	// Simple implementation using timestamp
	// In production, would use something like UUID or ULID
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

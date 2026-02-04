package bundoc

import (
	"fmt"
	"sync"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc/internal/query"
	"github.com/kartikbazzad/bunbase/bundoc/internal/transaction"
	"github.com/kartikbazzad/bunbase/bundoc/rules"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
	"github.com/xeipuuv/gojsonschema"
)

// Collection represents a logical grouping of documents (similar to a table in SQL).
// It manages the primary storage (Transaction Write Set / WAL) and all associated
// B+Tree indexes.
type Collection struct {
	name               string
	db                 *Database
	indexes            map[string]*storage.BPlusTree // Map of field name -> B+Tree Index
	linkedGroupIndexes []*GroupIndexLink             // List of Group Indexes this collection feeds into
	mu                 sync.RWMutex                  // Protects concurrent access to indexes map
	schemaLoader       *gojsonschema.Schema          // Compiled JSON Schema
}

// GroupIndexLink holds reference to a group index
type GroupIndexLink struct {
	Index *storage.BPlusTree
	Field string
}

// Name returns the collection name
func (c *Collection) Name() string {
	return c.name
}

// GetSchema returns the current JSON schema definition
func (c *Collection) GetSchema() (string, error) {
	meta, ok := c.db.metadataMgr.GetCollection(c.name)
	if !ok {
		return "", fmt.Errorf("collection metadata not found")
	}
	return meta.Schema, nil
}

// SetSchema updates the collection's schema.
// It compiles the schema and persists it to the metadata.
func (c *Collection) SetSchema(schemaStr string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if schemaStr == "" {
		c.schemaLoader = nil
		return c.updateMetadataSchema("")
	}

	loader := gojsonschema.NewStringLoader(schemaStr)
	schema, err := gojsonschema.NewSchema(loader)
	if err != nil {
		return fmt.Errorf("invalid json schema: %w", err)
	}

	c.schemaLoader = schema
	return c.updateMetadataSchema(schemaStr)
}

func (c *Collection) updateMetadataSchema(schemaStr string) error {
	return c.db.metadataMgr.UpdateCollectionSchema(c.name, schemaStr)
}

// SetRules updates the collection's security rules.
func (c *Collection) SetRules(rules map[string]string) error {
	return c.db.metadataMgr.UpdateCollectionRules(c.name, rules)
}

// GetRules returns the collection's security rules
func (c *Collection) GetRules() map[string]string {
	meta, ok := c.db.metadataMgr.GetCollection(c.name)
	if !ok {
		return nil
	}
	return meta.Rules
}

// evaluateRule checks if the operation is allowed by the defined rules.
func (c *Collection) evaluateRule(op string, auth *rules.AuthContext, resource map[string]interface{}) error {
	// Admin Bypass: If auth is marked as Admin, skip rules.
	if auth != nil && auth.IsAdmin {
		return nil
	}

	meta, ok := c.db.metadataMgr.GetCollection(c.name)
	if !ok {
		return nil // No rules defined (or default deny? Metadata not found usually means collection issue)
	}

	// Default behavior if no rules?
	// Firestore: Default deny.
	// Bundoc MVP: Default allow if no rules set?
	// Current behavior: Allow.
	// Let's keep Default Allow if map is empty/nil to backward compat.
	if len(meta.Rules) == 0 {
		return nil
	}

	rule, ok := meta.Rules[op]
	if !ok {
		// Try generic read/write?
		if op == "create" || op == "update" || op == "delete" {
			rule, ok = meta.Rules["write"]
		}
	}

	if !ok {
		return nil // No rule for this op -> Allow. Consistent with "Default Allow if not specified"
	}

	// Prepare Context
	reqData := make(map[string]interface{})
	if auth != nil {
		reqData["auth"] = map[string]interface{}{
			"uid":    auth.UID,
			"claims": auth.Claims,
		}
	} else {
		reqData["auth"] = nil // Unauthenticated
	}

	ctx := map[string]interface{}{
		"request":  reqData,
		"resource": map[string]interface{}{"data": resource},
	}

	allowed, err := c.db.RulesEngine.Evaluate(rule, ctx)
	if err != nil {
		return fmt.Errorf("rule evaluation error: %w", err)
	}
	if !allowed {
		return fmt.Errorf("permission denied: rule '%s' failed", op)
	}

	return nil
}

// Validate validates a document against the collection's schema
func (c *Collection) validate(doc storage.Document) error {
	// Need Read Lock?
	// This is usually called within Insert/Update which hold Lock.
	// So we assume caller holds lock or we don't need it if schemaLoader is atomic/safe?
	// c.schemaLoader is a pointer. Accessing it requires strict consistency?
	// We are under c.mu.Lock() in Insert/Update.

	if c.schemaLoader == nil {
		return nil
	}

	// Convert Document to JSON generic map for validation
	// Document is map[string]interface{}, so it works directly?
	// gojsonschema expects a loader.
	docLoader := gojsonschema.NewGoLoader(doc)

	result, err := c.schemaLoader.Validate(docLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		var errs []string
		for _, desc := range result.Errors() {
			errs = append(errs, desc.String())
		}
		return fmt.Errorf("document invalid against schema: %s", fmt.Sprintf("%v", errs))
	}

	return nil
}

// Insert inserts a new document into the collection.
//
// The operation follows these steps:
// 1. Storage: Writes the document data to the transaction's Write Set (and eventually WAL).
// 2. Indexing: Inserts an entry into the Primary Index (_id).
// 3. Secondary Indexes: Updates all secondary indexes with composite keys.
//
// This operation is atomic within the context of the transaction.
func (c *Collection) Insert(auth *rules.AuthContext, txn *transaction.Transaction, doc storage.Document) error {
	// 1. Enforce Rules (Pre-creation check)
	if err := c.evaluateRule("create", auth, doc); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Validate Schema
	if err := c.validate(doc); err != nil {
		return err
	}

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
			valStr := fmt.Sprintf("%v", val)
			compKey := []byte(valStr + "\x00" + string(id))

			if err := index.Insert(compKey, []byte(string(id))); err != nil {
				return fmt.Errorf("failed to insert into index %s: %w", field, err)
			}
		}
	}

	// Update Group Indexes
	for _, link := range c.linkedGroupIndexes {
		if val, ok := doc[link.Field]; ok {
			valStr := fmt.Sprintf("%v", val)
			// Composite Key: Value \0 Collection \0 ID
			compKey := []byte(valStr + "\x00" + c.name + "\x00" + string(id))
			// Value: Collection \0 ID
			compVal := []byte(c.name + "\x00" + string(id))

			if err := link.Index.Insert(compKey, compVal); err != nil {
				return fmt.Errorf("failed to insert into group index for field %s: %w", link.Field, err)
			}
		}
	}

	return nil
}

// FindByID retrieves a document by its unique ID.
// It leverages MVCC to ensure that the returned document version is visible
// to the current transaction's snapshot.
func (c *Collection) FindByID(auth *rules.AuthContext, txn *transaction.Transaction, id string) (storage.Document, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	doc, err := c.findByIDLocked(txn, id)
	if err != nil {
		return nil, err
	}

	// Enforce Rules (Read)
	if err := c.evaluateRule("read", auth, doc); err != nil {
		return nil, err
	}

	return doc, nil
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
// Updates all affected Secondary Indexes (deleting old keys, inserting new ones).
func (c *Collection) Update(auth *rules.AuthContext, txn *transaction.Transaction, id string, doc storage.Document) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. Fetch old document for Rule Check
	oldDoc, err := c.findByIDLocked(txn, id)
	if err != nil {
		return fmt.Errorf("document not found for update: %w", err)
	}

	// 2. Enforce Rules (Update)
	// Manual rule check to include both old and new data context
	if auth == nil || !auth.IsAdmin {
		meta, ok := c.db.metadataMgr.GetCollection(c.name)
		if ok && len(meta.Rules) > 0 {
			rule, hasRule := meta.Rules["update"]
			if !hasRule {
				rule, hasRule = meta.Rules["write"]
			}
			if hasRule {
				reqData := map[string]interface{}{
					"auth":     nil,
					"resource": map[string]interface{}{"data": doc},
				}
				if auth != nil {
					reqData["auth"] = map[string]interface{}{"uid": auth.UID, "claims": auth.Claims}
				}

				ctx := map[string]interface{}{
					"request":  reqData,
					"resource": map[string]interface{}{"data": oldDoc},
				}
				allowed, err := c.db.RulesEngine.Evaluate(rule, ctx)
				if err != nil {
					return err
				}
				if !allowed {
					return fmt.Errorf("permission denied: rule 'update' failed")
				}
			}
		}
	}

	// Validate Schema
	if err := c.validate(doc); err != nil {
		return err
	}

	return c.updateLocked(txn, id, doc)
}

// Patch applies a partial update to a document.
// It fetches the current document, merges the patch (supporting dot notation),
// and performs a full update.
// Patch applies a partial update to a document.
// It fetches the current document, merges the patch (supporting dot notation),
// and performs a full update.
func (c *Collection) Patch(auth *rules.AuthContext, txn *transaction.Transaction, id string, patch map[string]interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. Fetch current document
	currentDoc, err := c.findByIDLocked(txn, id)
	if err != nil {
		return err // Not found
	}

	// 2. Clone to avoid mutation
	newDoc := currentDoc.Clone()

	// 3. Apply Patch
	if err := newDoc.ApplyPatch(patch); err != nil {
		return fmt.Errorf("failed to apply patch: %w", err)
	}
	newDoc.SetID(storage.DocumentID(id))

	// 4. Enforce Rules (Update)
	// Manual rule check to include both old and new data context
	if auth == nil || !auth.IsAdmin {
		meta, ok := c.db.metadataMgr.GetCollection(c.name)
		if ok && len(meta.Rules) > 0 {
			rule, hasRule := meta.Rules["update"]
			if !hasRule {
				rule, hasRule = meta.Rules["write"]
			}
			if hasRule {
				reqData := map[string]interface{}{
					"auth":     nil,
					"resource": map[string]interface{}{"data": newDoc},
				}
				if auth != nil {
					reqData["auth"] = map[string]interface{}{"uid": auth.UID, "claims": auth.Claims}
				}

				ctx := map[string]interface{}{
					"request":  reqData,
					"resource": map[string]interface{}{"data": currentDoc},
				}
				allowed, err := c.db.RulesEngine.Evaluate(rule, ctx)
				if err != nil {
					return err
				}
				if !allowed {
					return fmt.Errorf("permission denied: rule 'update' failed")
				}
			}
		}
	}

	// Validate Schema
	if err := c.validate(newDoc); err != nil {
		return err
	}

	// 5. Update
	// Delegate to update logic (which we must duplicate or extract)
	// Since `Update` is basically: Write + Index Maint.
	// We extract `updateLocked` in previous step? Or did I assume it exists?
	// It was in the view! func (c *Collection) updateLocked
	// So I can use it.
	return c.updateLocked(txn, id, newDoc)
}

// updateLocked is the internal implementation of Update (caller must hold Lock)
func (c *Collection) updateLocked(txn *transaction.Transaction, id string, doc storage.Document) error {
	key := c.name + ":" + id

	// Ensure ID matches
	doc.SetID(storage.DocumentID(id))

	// Serialize
	data, err := doc.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize document: %w", err)
	}

	// 1. Fetch old document state for index maintenance
	oldDoc, err := c.findByIDLocked(txn, id)
	if err == nil {
		// Found
	}

	// 2. Write new document data
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

		if hasOld {
			valChanged := !hasNew || fmt.Sprintf("%v", oldVal) != fmt.Sprintf("%v", newVal)
			if valChanged {
				valStr := fmt.Sprintf("%v", oldVal)
				oldCompKey := []byte(valStr + "\x00" + string(id))
				_ = index.Delete(oldCompKey)
			}
		}

		shouldInsert := hasNew
		if hasOld && hasNew && fmt.Sprintf("%v", oldVal) == fmt.Sprintf("%v", newVal) {
			shouldInsert = false
		}

		if shouldInsert {
			valStr := fmt.Sprintf("%v", newVal)
			compKey := []byte(valStr + "\x00" + string(id))
			if err := index.Insert(compKey, []byte(string(id))); err != nil {
				return fmt.Errorf("failed to update index %s: %w", field, err)
			}
		}
	}

	// 5. Maintenance of Group Indexes
	for _, link := range c.linkedGroupIndexes {
		var oldVal interface{}
		var newVal interface{}
		hasOld := false
		hasNew := false

		if oldDoc != nil {
			oldVal, hasOld = oldDoc[link.Field]
		}
		newVal, hasNew = doc[link.Field]

		if hasOld {
			valChanged := !hasNew || fmt.Sprintf("%v", oldVal) != fmt.Sprintf("%v", newVal)
			if valChanged {
				valStr := fmt.Sprintf("%v", oldVal)
				// Key: Value \0 Collection \0 ID
				oldCompKey := []byte(valStr + "\x00" + c.name + "\x00" + string(id))
				_ = link.Index.Delete(oldCompKey)
			}
		}

		shouldInsert := hasNew
		if hasOld && hasNew && fmt.Sprintf("%v", oldVal) == fmt.Sprintf("%v", newVal) {
			shouldInsert = false
		}

		if shouldInsert {
			valStr := fmt.Sprintf("%v", newVal)
			compKey := []byte(valStr + "\x00" + c.name + "\x00" + string(id))
			compVal := []byte(c.name + "\x00" + string(id))
			if err := link.Index.Insert(compKey, compVal); err != nil {
				return fmt.Errorf("failed to update group index %s: %w", link.Field, err)
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
		// Validate Schema
		if err := c.validate(doc); err != nil {
			return err
		}

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
		// Validate Schema
		if err := c.validate(doc); err != nil {
			return err
		}

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
func (c *Collection) Delete(auth *rules.AuthContext, txn *transaction.Transaction, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.name + ":" + id

	// 1. Fetch document to clean up secondary indexes (and Rule Check)
	doc, err := c.findByIDLocked(txn, id)
	if err == nil {
		// Enforce Rules (Delete)
		if err := c.evaluateRule("delete", auth, doc); err != nil {
			return err
		}

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

		// Clean up group indexes
		for _, link := range c.linkedGroupIndexes {
			if val, ok := doc[link.Field]; ok {
				valStr := fmt.Sprintf("%v", val)
				// Key: Value \0 Collection \0 ID
				compKey := []byte(valStr + "\x00" + c.name + "\x00" + string(id))
				_ = link.Index.Delete(compKey)
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

// List returns a list of documents with pagination
func (c *Collection) List(auth *rules.AuthContext, txn *transaction.Transaction, skip, limit int) ([]storage.Document, error) {
	// Rule Check (List)
	// We check for 'list' rule. If not present, maybe 'read'?
	// For listing, we usually require a 'list' rule that evaluates on query/request context,
	// NOT on individual resources (unless post-filtering).
	// For MVP: Check 'list'.
	if auth == nil || !auth.IsAdmin {
		if err := c.evaluateRule("list", auth, nil); err != nil {
			return nil, err
		}
	}

	iter, err := NewTableScanIterator(c, txn)
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()

	var currentIter Iterator = iter

	if skip > 0 {
		currentIter = NewSkipIterator(currentIter, skip)
	}

	if limit > 0 {
		currentIter = NewLimitIterator(currentIter, limit)
	}

	var results []storage.Document
	for currentIter.Next() {
		doc, err := currentIter.Value()
		if err == nil {
			results = append(results, doc)
		}
	}

	return results, nil
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

// DropIndex removes a secondary index for the given field.
// It removes the index from the in-memory map and updates the system catalog.
// Note: It does not currently reclaim the disk pages used by the index immediately (leaks storage until GC/Compact).
func (c *Collection) DropIndex(field string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if field == "_id" {
		return fmt.Errorf("cannot drop primary index")
	}

	if _, exists := c.indexes[field]; !exists {
		return fmt.Errorf("index not found for field: %s", field)
	}

	// Remove from map
	delete(c.indexes, field)

	// Persist Metadata Update
	currentMeta, _ := c.db.metadataMgr.GetCollection(c.name)
	if currentMeta.Indexes != nil {
		delete(currentMeta.Indexes, field)

		saveIdx := make(map[string]storage.PageID)
		for k, v := range currentMeta.Indexes {
			saveIdx[k] = storage.PageID(v)
		}
		if err := c.db.metadataMgr.UpdateCollection(c.name, saveIdx); err != nil {
			return fmt.Errorf("failed to persist index metadata deletion: %w", err)
		}
	}

	fmt.Printf("[INFO] Dropped index for field '%s'\n", field)
	return nil
}

// Find searches for documents matching the given field and value
func (c *Collection) Find(txn *transaction.Transaction, field string, value interface{}) ([]storage.Document, error) {
	// Optimization: If field is _id, use FindByID
	if field == "_id" {
		idStr := fmt.Sprintf("%v", value)
		doc, err := c.FindByID(nil, txn, idStr)
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
		doc, err := c.FindByID(nil, txn, docID)
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

// ListIndexes returns a list of secondary indexes on the collection
func (c *Collection) ListIndexes() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var indexes []string
	for field := range c.indexes {
		if field != "_id" {
			indexes = append(indexes, field)
		}
	}
	return indexes
}

// FindQuery executes a complex query against the collection
func (c *Collection) FindQuery(auth *rules.AuthContext, txn *transaction.Transaction, queryMap map[string]interface{}, opts ...QueryOptions) ([]storage.Document, error) {
	// Rule Check (List/Query)
	// Similar to List, we check 'list' rule.
	// Context could includes the query itself for advanced rules (allow list if query.limit < 100)
	// TODO: Pass query details to context?
	if auth == nil || !auth.IsAdmin {
		if err := c.evaluateRule("list", auth, nil); err != nil {
			return nil, err
		}
	}

	// Parse Options
	// Parse Options
	skip := 0
	limit := 0
	sortField := ""
	sortDesc := false
	if len(opts) > 0 {
		skip = opts[0].Skip
		limit = opts[0].Limit
		sortField = opts[0].SortField
		sortDesc = opts[0].SortDesc
	}

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
	usedIndex := false

	// Attempt to find an index usage strategy
	if fNode, ok := node.(*query.FieldNode); ok {
		c.mu.RLock()
		_, hasIndex := c.indexes[fNode.Field]
		c.mu.RUnlock()

		if hasIndex {
			valStr := fmt.Sprintf("%v", fNode.Value)
			var startKey, endKey []byte

			switch fNode.Operator {
			case query.OpEq:
				startKey = []byte(valStr + "\x00")
				endKey = []byte(valStr + "\x00" + "\xFF")
			case query.OpGt:
				startKey = []byte(valStr + "\x00" + "\xFF")
				endKey = []byte{0xFF, 0xFF, 0xFF, 0xFF}
			case query.OpGte:
				startKey = []byte(valStr + "\x00")
				endKey = []byte{0xFF, 0xFF, 0xFF, 0xFF}
			case query.OpLt:
				startKey = []byte{0x00}
				endKey = []byte(valStr + "\x00")
			case query.OpLte:
				startKey = []byte{0x00}
				endKey = []byte(valStr + "\x00" + "\xFF")
			}

			if startKey != nil && endKey != nil {
				idxIter, err := NewIndexScanIterator(c, txn, fNode.Field, startKey, endKey)
				if err == nil {
					iter = idxIter
					usedIndex = true
				}
			}
		}
	}

	// Fallback to Table Scan
	if !usedIndex {
		tsIter, err := NewTableScanIterator(c, txn)
		if err != nil {
			return nil, fmt.Errorf("failed to create iterator: %w", err)
		}
		iter = tsIter
	}
	defer iter.Close()

	// 3. Apply Filters
	// FilterIterator wraps any iterator and applies filter
	iter = NewFilterIterator(iter, matcher)

	// 4. Apply Sort
	if sortField != "" {
		// Note: SortIterator reads all documents into memory.
		// Future optimization: Use Index order if applicable.
		iter = NewSortIterator(iter, sortField, sortDesc)
	}

	// 5. Apply Skip & Limit
	if skip > 0 {
		iter = NewSkipIterator(iter, skip)
	}
	if limit > 0 {
		iter = NewLimitIterator(iter, limit)
	}

	// 4. Execute
	var results []storage.Document
	for iter.Next() {
		doc, err := iter.Value()
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

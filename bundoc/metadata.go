package bundoc

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

// SystemMetadata holds the persistent schema of the database
type SystemMetadata struct {
	Collections  map[string]CollectionMeta `json:"collections"`
	GroupIndexes map[string]GroupIndexMeta `json:"group_indexes"` // Key is "pattern:field" or just unique ID? Let's use "pattern" + "field"
}

// CollectionMeta holds metadata for a single collection
type CollectionMeta struct {
	Name    string            `json:"name"`
	Indexes map[string]uint64 `json:"indexes"` // Field -> RootPageID
	Schema  string            `json:"schema,omitempty"`
	Rules   map[string]string `json:"rules,omitempty"` // Operation -> Expression (e.g. "read": "true")
}

// GroupIndexMeta holds metadata for a collection group index
type GroupIndexMeta struct {
	Pattern string `json:"pattern"` // e.g., "users/*/posts"
	Field   string `json:"field"`   // e.g., "created_at"
	RootID  uint64 `json:"root_id"` // RootPageID of the B+Tree
}

// MetadataManager handles the persistence of the database schema (System Catalog).
//
// It stores the mapping of Collection Names -> Index Fields -> Root Page IDs.
// This allows the database to restore the exact state of B+Tree indexes after a restart.
// The data is stored in a JSON file (e.g., system_catalog.json).
type MetadataManager struct {
	path     string
	metadata SystemMetadata
	mu       sync.RWMutex
}

// NewMetadataManager creates a new metadata manager
func NewMetadataManager(path string) (*MetadataManager, error) {
	mm := &MetadataManager{
		path: path,
		metadata: SystemMetadata{
			Collections:  make(map[string]CollectionMeta),
			GroupIndexes: make(map[string]GroupIndexMeta),
		},
	}

	if err := mm.load(); err != nil {
		if os.IsNotExist(err) {
			// Initialize empty
			return mm, nil
		}
		return nil, err
	}

	// Ensure maps are initialized if loaded nil
	if mm.metadata.Collections == nil {
		mm.metadata.Collections = make(map[string]CollectionMeta)
	}
	if mm.metadata.GroupIndexes == nil {
		mm.metadata.GroupIndexes = make(map[string]GroupIndexMeta)
	}

	return mm, nil
}

// load reads metadata from disk
func (mm *MetadataManager) load() error {
	data, err := os.ReadFile(mm.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &mm.metadata)
}

// Save writes metadata to disk
func (mm *MetadataManager) Save() error {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	return mm.saveLocked()
}

func (mm *MetadataManager) saveLocked() error {
	data, err := json.MarshalIndent(mm.metadata, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(mm.path, data, 0644)
}

// UpdateCollection updates metadata for a collection (indexes only, preserves schema)
func (mm *MetadataManager) UpdateCollection(name string, indexes map[string]storage.PageID) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	idxMap := make(map[string]uint64)
	for k, v := range indexes {
		idxMap[k] = uint64(v)
	}

	// Get existing to preserve other fields like Schema
	meta, exists := mm.metadata.Collections[name]
	if !exists {
		// New collection
		meta = CollectionMeta{Name: name}
	}

	meta.Indexes = idxMap
	mm.metadata.Collections[name] = meta

	return mm.saveLocked()
}

// UpdateCollectionSchema updates schema for a collection
func (mm *MetadataManager) UpdateCollectionSchema(name string, schema string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	meta, exists := mm.metadata.Collections[name]
	if !exists {
		return fmt.Errorf("collection %s does not exist", name)
	}

	meta.Schema = schema
	mm.metadata.Collections[name] = meta

	return mm.saveLocked()
}

// GetCollection returns metadata for a collection
func (mm *MetadataManager) GetCollection(name string) (CollectionMeta, bool) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	meta, ok := mm.metadata.Collections[name]
	return meta, ok
}

// DeleteCollection removes a collection from metadata
func (mm *MetadataManager) DeleteCollection(name string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	delete(mm.metadata.Collections, name)
	return mm.saveLocked()
}

// ListCollections returns all collection names
func (mm *MetadataManager) ListCollections() []string {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	names := make([]string, 0, len(mm.metadata.Collections))
	for name := range mm.metadata.Collections {
		names = append(names, name)
	}
	return names
}

// ListCollectionsWithPrefix returns collection names matching the prefix
func (mm *MetadataManager) ListCollectionsWithPrefix(prefix string) []string {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	names := make([]string, 0)
	for name := range mm.metadata.Collections {
		// If prefix is empty, return all
		if prefix == "" {
			names = append(names, name)
			continue
		}

		// Check prefix
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			names = append(names, name)
		}
	}
	return names
}

// UpdateGroupIndex updates metadata for a group index
func (mm *MetadataManager) UpdateGroupIndex(pattern, field string, rootID storage.PageID) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	key := pattern + "::" + field
	mm.metadata.GroupIndexes[key] = GroupIndexMeta{
		Pattern: pattern,
		Field:   field,
		RootID:  uint64(rootID),
	}

	return mm.saveLocked()
}

// GetGroupIndex returns metadata for a group index
func (mm *MetadataManager) GetGroupIndex(pattern, field string) (GroupIndexMeta, bool) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	key := pattern + "::" + field
	meta, ok := mm.metadata.GroupIndexes[key]
	return meta, ok
}

// ListGroupIndexes returns all group indexes
func (mm *MetadataManager) ListGroupIndexes() []GroupIndexMeta {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	indexes := make([]GroupIndexMeta, 0, len(mm.metadata.GroupIndexes))
	for _, meta := range mm.metadata.GroupIndexes {
		indexes = append(indexes, meta)
	}
	return indexes
}

// DeleteGroupIndex removes a group index from metadata
func (mm *MetadataManager) DeleteGroupIndex(pattern, field string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	key := pattern + "::" + field
	delete(mm.metadata.GroupIndexes, key)
	return mm.saveLocked()
}

// UpdateCollectionRules updates the rules for a collection
func (mm *MetadataManager) UpdateCollectionRules(name string, rules map[string]string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	meta, ok := mm.metadata.Collections[name]
	if !ok {
		return fmt.Errorf("collection not found: %s", name)
	}

	meta.Rules = rules
	mm.metadata.Collections[name] = meta

	return mm.saveLocked()
}

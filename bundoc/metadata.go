package bundoc

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

// SystemMetadata holds the persistent schema of the database
type SystemMetadata struct {
	Collections map[string]CollectionMeta `json:"collections"`
}

// CollectionMeta holds metadata for a single collection
type CollectionMeta struct {
	Name    string            `json:"name"`
	Indexes map[string]uint64 `json:"indexes"` // Field -> RootPageID
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
			Collections: make(map[string]CollectionMeta),
		},
	}

	if err := mm.load(); err != nil {
		if os.IsNotExist(err) {
			// Initialize empty
			return mm, nil
		}
		return nil, err
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

// UpdateCollection updates metadata for a collection
func (mm *MetadataManager) UpdateCollection(name string, indexes map[string]storage.PageID) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	idxMap := make(map[string]uint64)
	for k, v := range indexes {
		idxMap[k] = uint64(v)
	}

	mm.metadata.Collections[name] = CollectionMeta{
		Name:    name,
		Indexes: idxMap,
	}

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

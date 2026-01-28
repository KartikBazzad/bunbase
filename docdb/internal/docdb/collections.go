package docdb

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/kartikbazzad/docdb/internal/errors"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/types"
)

const (
	DefaultCollection    = "_default"
	MaxCollectionNameLen = 64
)

// CollectionRegistry manages collections within a database.
type CollectionRegistry struct {
	mu          sync.RWMutex
	collections map[string]*types.CollectionMetadata
	defaultColl string
	logger      *logger.Logger
}

// NewCollectionRegistry creates a new collection registry.
func NewCollectionRegistry(log *logger.Logger) *CollectionRegistry {
	reg := &CollectionRegistry{
		collections: make(map[string]*types.CollectionMetadata),
		defaultColl: DefaultCollection,
		logger:      log,
	}

	// Create default collection
	reg.collections[DefaultCollection] = &types.CollectionMetadata{
		Name:      DefaultCollection,
		CreatedAt: time.Now(),
		DocCount:  0,
	}

	return reg
}

// ValidateName validates a collection name according to v0.2 rules.
func ValidateCollectionName(name string) error {
	if name == "" {
		return fmt.Errorf("collection name cannot be empty")
	}

	if !utf8.ValidString(name) {
		return fmt.Errorf("collection name must be valid UTF-8")
	}

	if len(name) > MaxCollectionNameLen {
		return fmt.Errorf("collection name exceeds maximum length of %d bytes", MaxCollectionNameLen)
	}

	// Check for forbidden characters
	if strings.Contains(name, "/") {
		return fmt.Errorf("collection name cannot contain '/'")
	}

	if strings.Contains(name, ".") {
		return fmt.Errorf("collection name cannot contain '.'")
	}

	// Check for null bytes
	if strings.ContainsRune(name, 0) {
		return fmt.Errorf("collection name cannot contain null bytes")
	}

	return nil
}

// CreateCollection creates a new collection.
func (r *CollectionRegistry) CreateCollection(name string) error {
	if err := ValidateCollectionName(name); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if name == DefaultCollection {
		return errors.ErrCollectionExists
	}

	if _, exists := r.collections[name]; exists {
		return errors.ErrCollectionExists
	}

	r.collections[name] = &types.CollectionMetadata{
		Name:      name,
		CreatedAt: time.Now(),
		DocCount:  0,
	}

	r.logger.Info("Created collection: %s", name)
	return nil
}

// DeleteCollection deletes a collection if it's empty.
func (r *CollectionRegistry) DeleteCollection(name string) error {
	if name == DefaultCollection {
		return fmt.Errorf("cannot delete default collection")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	coll, exists := r.collections[name]
	if !exists {
		return errors.ErrCollectionNotFound
	}

	if coll.DocCount > 0 {
		return errors.ErrCollectionNotEmpty
	}

	delete(r.collections, name)
	r.logger.Info("Deleted collection: %s", name)
	return nil
}

// Exists checks if a collection exists.
func (r *CollectionRegistry) Exists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.collections[name]
	return exists
}

// Get returns collection metadata.
func (r *CollectionRegistry) Get(name string) (*types.CollectionMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	coll, exists := r.collections[name]
	if !exists {
		return nil, errors.ErrCollectionNotFound
	}

	return coll, nil
}

// ListCollections returns all collection names.
func (r *CollectionRegistry) ListCollections() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.collections))
	for name := range r.collections {
		names = append(names, name)
	}

	return names
}

// IncrementDocCount increments the document count for a collection.
func (r *CollectionRegistry) IncrementDocCount(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if coll, exists := r.collections[name]; exists {
		coll.DocCount++
	}
}

// DecrementDocCount decrements the document count for a collection.
func (r *CollectionRegistry) DecrementDocCount(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if coll, exists := r.collections[name]; exists && coll.DocCount > 0 {
		coll.DocCount--
	}
}

// SetDocCount sets the document count for a collection (used during recovery).
func (r *CollectionRegistry) SetDocCount(name string, count uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if coll, exists := r.collections[name]; exists {
		coll.DocCount = count
	} else {
		// Create collection if it doesn't exist (during recovery)
		r.collections[name] = &types.CollectionMetadata{
			Name:      name,
			CreatedAt: time.Now(),
			DocCount:  count,
		}
	}
}

// EnsureDefault ensures the default collection exists.
func (r *CollectionRegistry) EnsureDefault() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.collections[DefaultCollection]; !exists {
		r.collections[DefaultCollection] = &types.CollectionMetadata{
			Name:      DefaultCollection,
			CreatedAt: time.Now(),
			DocCount:  0,
		}
	}
}

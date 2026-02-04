package storage

import (
	"encoding/json"
	"fmt"
)

// Document represents a JSON document in the database
type Document map[string]interface{}

// DocumentID is a unique identifier for a document
type DocumentID string

// Serialize converts a document to JSON bytes
func (d Document) Serialize() ([]byte, error) {
	buf := GetBuffer()
	defer PutBuffer(buf)

	encoder := json.NewEncoder(buf)
	// encoder.SetEscapeHTML(false) // Optional: usually better for DB storage
	if err := encoder.Encode(d); err != nil {
		return nil, fmt.Errorf("failed to serialize document: %w", err)
	}

	// Make a copy of the bytes because the buffer is returned to the pool
	// We need to trim the trailing newline added by Encode
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}

	// Copy to new slice to be safe (buffer reused)
	result := make([]byte, len(b))
	copy(result, b)

	return result, nil
}

// DeserializeDocument creates a document from JSON bytes
func DeserializeDocument(data []byte) (Document, error) {
	var d Document
	err := json.Unmarshal(data, &d)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize document: %w", err)
	}
	return d, nil
}

// Deserialize converts JSON bytes to a document
func Deserialize(data []byte) (Document, error) {
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to deserialize document: %w", err)
	}
	return doc, nil
}

// GetID returns the document ID if it exists
func (d Document) GetID() (DocumentID, bool) {
	id, exists := d["_id"]
	if !exists {
		return "", false
	}

	idStr, ok := id.(string)
	if !ok {
		return "", false
	}

	return DocumentID(idStr), true
}

// SetID sets the document ID
func (d Document) SetID(id DocumentID) {
	d["_id"] = string(id)
}

// Clone creates a deep copy of the document
func (d Document) Clone() Document {
	clone := make(Document, len(d))
	for k, v := range d {
		clone[k] = deepCopyValue(v)
	}
	return clone
}

// deepCopyValue creates a deep copy of a value
func deepCopyValue(v interface{}) interface{} {
	switch val := v.(type) {
	case Document:
		return val.Clone()
	case map[string]interface{}:
		return Document(val).Clone()
	case []interface{}:
		cp := make([]interface{}, len(val))
		for i, item := range val {
			cp[i] = deepCopyValue(item)
		}
		return cp
	default:
		// Primitives (string, number, bool) are immutable or copied by value
		return val
	}
}

// Size returns the approximate size of the document in bytes
func (d Document) Size() int {
	data, err := json.Marshal(d)
	if err != nil {
		return 0
	}
	return len(data)
}

// ApplyPatch merges a patch document into the target document.
// It supports dot notation for nested updates (e.g., "settings.theme": "dark").
// ApplyPatch merges a patch document into the target document.
// It supports dot notation for nested updates (e.g., "settings.theme": "dark").
// It also supports MongoDB-style "$unset" operator for deletions.
func (d Document) ApplyPatch(patch map[string]interface{}) error {
	// 1. Handle Deletions first ($unset)
	if unset, ok := patch["$unset"]; ok {
		if unsetMap, ok := unset.(map[string]interface{}); ok {
			for path := range unsetMap {
				if err := d.deletePath(path); err != nil {
					return err
				}
			}
		}
	}

	// 2. Handle Sets
	for k, v := range patch {
		if k == "$unset" {
			continue
		}
		if err := d.setPath(k, v); err != nil {
			return err
		}
	}
	return nil
}

func (d Document) deletePath(path string) error {
	keys := splitPath(path)
	if len(keys) == 0 {
		return nil
	}

	current := d
	// Navigate to parent of leaf
	for i := 0; i < len(keys)-1; i++ {
		key := keys[i]
		val, exists := current[key]
		if !exists {
			return nil // Already gone
		}

		if nextMap, ok := val.(map[string]interface{}); ok {
			current = nextMap
		} else if nextDoc, ok := val.(Document); ok {
			current = nextDoc
		} else {
			return nil // Unreachable path
		}
	}

	delete(current, keys[len(keys)-1])
	return nil
}

// setPath sets a value at the given dot-notation path
func (d Document) setPath(path string, value interface{}) error {
	// Simple split by dot
	// TODO: Handle escaped dots? For now assume simple keys.
	keys := splitPath(path)

	current := d
	for i := 0; i < len(keys)-1; i++ {
		key := keys[i]

		// Get or create nested map
		val, exists := current[key]
		if !exists {
			newMap := make(map[string]interface{})
			current[key] = newMap
			current = newMap
			continue
		}

		// Check if it is a map
		if nextMap, ok := val.(map[string]interface{}); ok {
			current = nextMap
		} else if nextDoc, ok := val.(Document); ok {
			// Convert Document to map[string]interface{} for consistency if needed,
			// or just cast. Document IS map[string]interface{}
			current = nextDoc
		} else {
			// Conflict: trying to traverse into a non-map
			// Force overwrite? Or Error?
			// MongoDB overwrites. Let's overwrite.
			newMap := make(map[string]interface{})
			current[key] = newMap
			current = newMap
		}
	}

	// Set leaf value
	lastKey := keys[len(keys)-1]
	current[lastKey] = value
	return nil
}

func splitPath(path string) []string {
	// Simple implementation
	// "a.b.c" -> ["a", "b", "c"]
	// This does not handle escaped dots.
	var parts []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '.' {
			parts = append(parts, path[start:i])
			start = i + 1
		}
	}
	parts = append(parts, path[start:])
	return parts
}

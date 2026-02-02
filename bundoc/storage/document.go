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

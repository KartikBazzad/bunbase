package docdb

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kartikbazzad/docdb/internal/errors"
)

// ParsePath parses a JSON Pointer-like path into a slice of path segments.
// Path format: /name, /address/city, /tags/0
// Returns error if path is invalid.
func ParsePath(path string) ([]string, error) {
	if path == "" {
		return []string{}, nil
	}

	if !strings.HasPrefix(path, "/") {
		return nil, errors.ErrInvalidPath
	}

	// Remove leading slash and split
	path = path[1:]
	if path == "" {
		return []string{}, nil
	}

	segments := strings.Split(path, "/")

	// Validate each segment
	for i, seg := range segments {
		// Handle escaped slashes and tildes
		seg = strings.ReplaceAll(seg, "~1", "/")
		seg = strings.ReplaceAll(seg, "~0", "~")
		segments[i] = seg
	}

	return segments, nil
}

// GetValue retrieves a value from a JSON document at the specified path.
func GetValue(doc interface{}, path []string) (interface{}, error) {
	current := doc

	for i, segment := range path {
		switch v := current.(type) {
		case map[string]interface{}:
			val, exists := v[segment]
			if !exists {
				return nil, fmt.Errorf("path segment '%s' not found at index %d: %w", segment, i, errors.ErrInvalidPath)
			}
			current = val

		case []interface{}:
			index, err := strconv.Atoi(segment)
			if err != nil {
				return nil, fmt.Errorf("invalid array index '%s' at path segment %d: %w", segment, i, errors.ErrInvalidPath)
			}
			if index < 0 || index >= len(v) {
				return nil, fmt.Errorf("array index %d out of bounds (length %d) at path segment %d: %w", index, len(v), i, errors.ErrInvalidPath)
			}
			current = v[index]

		default:
			return nil, fmt.Errorf("cannot traverse path segment '%s' at index %d: value is not an object or array: %w", segment, i, errors.ErrInvalidPath)
		}
	}

	return current, nil
}

// SetValue sets a value in a JSON document at the specified path.
// Creates intermediate objects/arrays as needed.
func SetValue(doc interface{}, path []string, value interface{}) error {
	if len(path) == 0 {
		return errors.ErrInvalidPath
	}

	// Ensure doc is a map
	docMap, ok := doc.(map[string]interface{})
	if !ok {
		return errors.ErrNotJSONObject
	}

	current := docMap

	// Navigate to parent of target
	for i := 0; i < len(path)-1; i++ {
		segment := path[i]
		val, exists := current[segment]

		if !exists {
			// Create intermediate object
			current[segment] = make(map[string]interface{})
			current = current[segment].(map[string]interface{})
		} else {
			switch v := val.(type) {
			case map[string]interface{}:
				current = v
			case []interface{}:
				// Can't set a key on an array
				return fmt.Errorf("cannot set key '%s' on array at path segment %d: %w", segment, i, errors.ErrInvalidPath)
			default:
				// Replace primitive with object
				current[segment] = make(map[string]interface{})
				current = current[segment].(map[string]interface{})
			}
		}
	}

	// Set the final value
	finalSegment := path[len(path)-1]
	current[finalSegment] = value

	return nil
}

// DeleteValue removes a value from a JSON document at the specified path.
func DeleteValue(doc interface{}, path []string) error {
	if len(path) == 0 {
		return errors.ErrInvalidPath
	}

	docMap, ok := doc.(map[string]interface{})
	if !ok {
		return errors.ErrNotJSONObject
	}

	current := docMap

	// Navigate to parent of target
	for i := 0; i < len(path)-1; i++ {
		segment := path[i]
		val, exists := current[segment]
		if !exists {
			return fmt.Errorf("path segment '%s' not found at index %d: %w", segment, i, errors.ErrInvalidPath)
		}

		switch v := val.(type) {
		case map[string]interface{}:
			current = v
		case []interface{}:
			index, err := strconv.Atoi(segment)
			if err != nil {
				return fmt.Errorf("invalid array index '%s' at path segment %d: %w", segment, i, errors.ErrInvalidPath)
			}
			if index < 0 || index >= len(v) {
				return fmt.Errorf("array index %d out of bounds at path segment %d: %w", index, i, errors.ErrInvalidPath)
			}
			// For arrays, we'd need to handle deletion differently
			// For now, only support object deletion
			return fmt.Errorf("array deletion not supported at path segment %d: %w", i, errors.ErrInvalidPath)
		default:
			return fmt.Errorf("cannot traverse path segment '%s' at index %d: %w", segment, i, errors.ErrInvalidPath)
		}
	}

	// Delete the final value
	finalSegment := path[len(path)-1]
	if _, exists := current[finalSegment]; !exists {
		return fmt.Errorf("path segment '%s' not found: %w", finalSegment, errors.ErrInvalidPath)
	}

	delete(current, finalSegment)
	return nil
}

// InsertValue inserts a value into an array at the specified path and index.
func InsertValue(doc interface{}, path []string, index int, value interface{}) error {
	if len(path) == 0 {
		return errors.ErrInvalidPath
	}

	docMap, ok := doc.(map[string]interface{})
	if !ok {
		return errors.ErrNotJSONObject
	}

	current := docMap

	// Navigate to parent of target array
	for i := 0; i < len(path)-1; i++ {
		segment := path[i]
		val, exists := current[segment]
		if !exists {
			return fmt.Errorf("path segment '%s' not found at index %d: %w", segment, i, errors.ErrInvalidPath)
		}

		switch v := val.(type) {
		case map[string]interface{}:
			current = v
		case []interface{}:
			arrIndex, err := strconv.Atoi(segment)
			if err != nil {
				return fmt.Errorf("invalid array index '%s' at path segment %d: %w", segment, i, errors.ErrInvalidPath)
			}
			if arrIndex < 0 || arrIndex >= len(v) {
				return fmt.Errorf("array index %d out of bounds at path segment %d: %w", arrIndex, i, errors.ErrInvalidPath)
			}
			current = v[arrIndex].(map[string]interface{})
		default:
			return fmt.Errorf("cannot traverse path segment '%s' at index %d: %w", segment, i, errors.ErrInvalidPath)
		}
	}

	// Get the array
	finalSegment := path[len(path)-1]
	val, exists := current[finalSegment]
	if !exists {
		return fmt.Errorf("path segment '%s' not found: %w", finalSegment, errors.ErrInvalidPath)
	}

	arr, ok := val.([]interface{})
	if !ok {
		return fmt.Errorf("path segment '%s' is not an array: %w", finalSegment, errors.ErrInvalidPath)
	}

	if index < 0 || index > len(arr) {
		return fmt.Errorf("insert index %d out of bounds (array length %d): %w", index, len(arr), errors.ErrInvalidPath)
	}

	// Insert value at index
	arr = append(arr[:index], append([]interface{}{value}, arr[index:]...)...)
	current[finalSegment] = arr

	return nil
}

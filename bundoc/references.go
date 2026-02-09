package bundoc

import (
	"encoding/json"
	"errors"
	"fmt"
)

const (
	onDeleteRestrict = "restrict"
	onDeleteSetNull  = "set_null"
	onDeleteCascade  = "cascade"
)

// ReferenceRule defines a schema-level reference from a source collection field
// to a target collection field.
type ReferenceRule struct {
	SourceCollection string
	SourceField      string
	TargetCollection string
	TargetField      string
	OnDelete         string
}

func parseReferenceRules(sourceCollection, schemaStr string) ([]ReferenceRule, error) {
	if schemaStr == "" {
		return nil, nil
	}

	var root map[string]interface{}
	if err := json.Unmarshal([]byte(schemaStr), &root); err != nil {
		return nil, fmt.Errorf("%w: schema is not valid JSON: %v", ErrInvalidReferenceSchema, err)
	}

	propsRaw, ok := root["properties"]
	if !ok {
		return nil, nil
	}

	props, ok := propsRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%w: schema.properties must be an object", ErrInvalidReferenceSchema)
	}

	rules := make([]ReferenceRule, 0)
	for fieldName, defRaw := range props {
		defMap, ok := defRaw.(map[string]interface{})
		if !ok {
			continue
		}
		refRaw, hasRef := defMap["x-bundoc-ref"]
		if !hasRef {
			continue
		}

		refMap, ok := refRaw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("%w: x-bundoc-ref for field %s must be an object", ErrInvalidReferenceSchema, fieldName)
		}

		targetCollection, ok := refMap["collection"].(string)
		if !ok || targetCollection == "" {
			return nil, fmt.Errorf("%w: x-bundoc-ref.collection is required for field %s", ErrInvalidReferenceSchema, fieldName)
		}

		targetField := "_id"
		if v, ok := refMap["field"].(string); ok && v != "" {
			targetField = v
		}

		// v1 supports target _id lookups only.
		if targetField != "_id" {
			return nil, fmt.Errorf("%w: x-bundoc-ref.field for field %s must be _id in v1", ErrInvalidReferenceSchema, fieldName)
		}

		onDelete := onDeleteSetNull
		if v, ok := refMap["on_delete"].(string); ok && v != "" {
			onDelete = v
		}
		if !isValidOnDelete(onDelete) {
			return nil, fmt.Errorf("%w: invalid on_delete %q for field %s", ErrInvalidReferenceSchema, onDelete, fieldName)
		}

		rules = append(rules, ReferenceRule{
			SourceCollection: sourceCollection,
			SourceField:      fieldName,
			TargetCollection: targetCollection,
			TargetField:      targetField,
			OnDelete:         onDelete,
		})
	}

	return rules, nil
}

func isValidOnDelete(v string) bool {
	switch v {
	case onDeleteRestrict, onDeleteSetNull, onDeleteCascade:
		return true
	default:
		return false
	}
}

func normalizeReferenceValue(v interface{}) (string, error) {
	switch typed := v.(type) {
	case string:
		if typed == "" {
			return "", errors.New("empty reference value")
		}
		return typed, nil
	case float64, float32, int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8, bool:
		return fmt.Sprintf("%v", typed), nil
	case nil:
		return "", nil
	default:
		return "", fmt.Errorf("reference field must be a scalar")
	}
}

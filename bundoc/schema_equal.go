package bundoc

import (
	"encoding/json"
	"reflect"
)

// SchemaEqual returns true if the two schema JSON strings are equivalent for
// the purpose of override checks. Key order and whitespace differences are
// ignored by unmarshaling and comparing with reflect.DeepEqual.
func SchemaEqual(a, b string) (bool, error) {
	if a == b {
		return true, nil
	}
	var va, vb interface{}
	if err := json.Unmarshal([]byte(a), &va); err != nil {
		return false, err
	}
	if err := json.Unmarshal([]byte(b), &vb); err != nil {
		return false, err
	}
	return reflect.DeepEqual(va, vb), nil
}

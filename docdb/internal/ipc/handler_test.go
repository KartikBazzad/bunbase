package ipc

import (
	"strings"
	"testing"

	"github.com/kartikbazzad/docdb/internal/pool"
	"github.com/kartikbazzad/docdb/internal/types"
)

func TestValidateJSONPayload_Valid(t *testing.T) {
	cases := [][]byte{
		[]byte(`{"key":"value"}`),
		[]byte(`[1,2,3]`),
		[]byte(`"string"`),
		[]byte(`42`),
		[]byte(`true`),
		[]byte(`null`),
		[]byte(`{"_type":"bytes","encoding":"base64","data":"SGVsbG8="}`),
	}

	for _, payload := range cases {
		err := validateJSONPayload(payload)
		if err != nil {
			t.Errorf("valid JSON should pass: %s, error: %v", payload, err)
		}
	}
}

func TestValidateJSONPayload_Invalid(t *testing.T) {
	cases := [][]byte{
		[]byte(`{invalid}`),
		[]byte(`"unclosed`),
		[]byte{0xFF, 0xFE},
		[]byte{},
	}

	for _, payload := range cases {
		err := validateJSONPayload(payload)
		if err == nil {
			t.Errorf("invalid payload should fail: %s", payload)
		}
		if err != types.ErrInvalidJSON &&
			!strings.Contains(err.Error(), "valid JSON") {
			t.Errorf("error should be explicit: %v", err)
		}
	}
}

func TestValidateJSONPayload_ReadOperations(t *testing.T) {
	req := &pool.Request{
		DBID:     1,
		DocID:    1,
		OpType:   types.OpRead,
		Payload:  nil,
		Response: make(chan pool.Response, 1),
	}

	err := validateJSONPayload(req.Payload)
	if err != nil {
		t.Errorf("read operations with nil payload should pass: %v", err)
	}
}

func TestValidateJSONPayload_DeleteOperations(t *testing.T) {
	req := &pool.Request{
		DBID:     1,
		DocID:    1,
		OpType:   types.OpDelete,
		Payload:  nil,
		Response: make(chan pool.Response, 1),
	}

	err := validateJSONPayload(req.Payload)
	if err != nil {
		t.Errorf("delete operations with nil payload should pass: %v", err)
	}
}

func TestValidateJSONPayload_CreateRequiresJSON(t *testing.T) {
	validPayload := []byte(`{"key":"value"}`)
	invalidPayload := []byte(`{invalid}`)

	err := validateJSONPayload(validPayload)
	if err != nil {
		t.Errorf("valid JSON should pass for create: %v", err)
	}

	err = validateJSONPayload(invalidPayload)
	if err == nil {
		t.Error("invalid JSON should fail for create")
	}
}

func TestValidateJSONPayload_UpdateRequiresJSON(t *testing.T) {
	validPayload := []byte(`{"key":"value"}`)
	invalidPayload := []byte(`{invalid}`)

	err := validateJSONPayload(validPayload)
	if err != nil {
		t.Errorf("valid JSON should pass for update: %v", err)
	}

	err = validateJSONPayload(invalidPayload)
	if err == nil {
		t.Error("invalid JSON should fail for update")
	}
}

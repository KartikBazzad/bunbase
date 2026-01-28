package parser

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodePayload_ValidJSON(t *testing.T) {
	cases := []string{
		`{"name":"Alice","age":30}`,
		`[1,2,3]`,
		`"hello world"`,
		`42`,
		`true`,
		`null`,
	}

	for _, input := range cases {
		result, err := DecodePayload(input)
		if err != nil {
			t.Errorf("valid JSON should pass: %s, error: %v", input, err)
			continue
		}

		var v interface{}
		if err := json.Unmarshal(result, &v); err != nil {
			t.Errorf("result should be valid JSON: %s", result)
		}
	}
}

func TestDecodePayload_InvalidJSON(t *testing.T) {
	cases := []string{
		`{invalid}`,
		`"unclosed string`,
		`[1,2`,
		`undefined`,
	}

	for _, input := range cases {
		_, err := DecodePayload(input)
		if err == nil {
			t.Errorf("invalid JSON should fail: %s", input)
		}
		if err != nil && !strings.Contains(err.Error(), "valid JSON") {
			t.Errorf("error message should mention JSON: %v", err)
		}
	}
}

func TestDecodePayload_ForbiddenPrefixes(t *testing.T) {
	cases := []string{
		`raw:hello`,
		`hex:48656c6c6f`,
		`raw:"quoted"`,
		`hex:00FF`,
	}

	for _, input := range cases {
		_, err := DecodePayload(input)
		if err == nil {
			t.Errorf("forbidden prefix should fail: %s", input)
		}
		expectedPrefix := input[:4]
		if err != nil && !strings.Contains(err.Error(), expectedPrefix) {
			t.Errorf("error should mention forbidden prefix %s: %v", expectedPrefix, err)
		}
	}
}

func TestDecodePayload_BinaryEncoding(t *testing.T) {
	input := `{"_type":"bytes","encoding":"base64","data":"SGVsbG8gd29ybGQ="}`
	result, err := DecodePayload(input)
	if err != nil {
		t.Fatalf("base64 wrapper should be valid JSON: %v", err)
	}

	var v interface{}
	if err := json.Unmarshal(result, &v); err != nil {
		t.Errorf("result should be valid JSON: %v", err)
	}
}

func TestDecodePayload_WithJSONPrefix(t *testing.T) {
	input := `json:{"key":"value"}`
	result, err := DecodePayload(input)
	if err != nil {
		t.Fatalf("json: prefix should work: %v", err)
	}

	var v interface{}
	if err := json.Unmarshal(result, &v); err != nil {
		t.Errorf("result should be valid JSON: %v", err)
	}
}

func TestDecodePayload_Empty(t *testing.T) {
	_, err := DecodePayload("")
	if err == nil {
		t.Error("empty payload should fail")
	}
}

func TestIsJSON(t *testing.T) {
	cases := []struct {
		input  string
		expect bool
	}{
		{`{"key":"value"}`, true},
		{`[1,2,3]`, true},
		{`"string"`, true},
		{`42`, true},
		{`true`, true},
		{`null`, true},
		{`{invalid}`, false},
		{`"unclosed`, false},
		{`[1,2`, false},
	}

	for _, tc := range cases {
		result := IsJSON(tc.input)
		if result != tc.expect {
			t.Errorf("IsJSON(%s) = %v, want %v", tc.input, result, tc.expect)
		}
	}
}

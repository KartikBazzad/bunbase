package parser

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

func DecodePayload(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("payload cannot be empty")
	}

	if strings.HasPrefix(s, "raw:") {
		return decodeRaw(s[4:])
	}

	if strings.HasPrefix(s, "hex:") {
		return decodeHex(s[4:])
	}

	if strings.HasPrefix(s, "json:") {
		return decodeJSON(s[5:])
	}

	return nil, fmt.Errorf("payload must have prefix: raw:, hex:, or json:")
}

func decodeRaw(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	return []byte(s), nil
}

func decodeHex(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if len(s)%2 != 0 {
		return nil, fmt.Errorf("hex string must have even length")
	}

	data, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid hex: %w", err)
	}

	return data, nil
}

func decodeJSON(s string) ([]byte, error) {
	s = strings.TrimSpace(s)

	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil, fmt.Errorf("invalid json: %w", err)
	}

	return json.Marshal(v)
}

func IsJSONPayload(s string) bool {
	return strings.HasPrefix(strings.TrimSpace(s), "json:")
}

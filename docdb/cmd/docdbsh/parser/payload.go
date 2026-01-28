package parser

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/kartikbazzad/docdb/internal/errors"
)

func DecodePayload(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.ErrInvalidJSON
	}

	if strings.HasPrefix(s, "raw:") {
		return nil, errors.ErrInvalidJSON
	}

	if strings.HasPrefix(s, "hex:") {
		return nil, errors.ErrInvalidJSON
	}

	if strings.HasPrefix(s, "json:") {
		s = s[5:]
		s = strings.TrimSpace(s)
	}

	if !utf8.ValidString(s) {
		return nil, errors.ErrInvalidJSON
	}

	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil, errors.ErrInvalidJSON
	}

	return json.Marshal(v)
}

func IsJSON(s string) bool {
	var v interface{}
	return json.Unmarshal([]byte(s), &v) == nil
}

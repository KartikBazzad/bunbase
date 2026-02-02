package core

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
)

func ParseMasterKey(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("BUNKMS_MASTER_KEY is required")
	}

	if strings.HasPrefix(value, "base64:") {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, "base64:"))
		if err != nil {
			return nil, errors.New("invalid base64 master key")
		}
		if len(decoded) != 32 {
			return nil, errors.New("master key must be 32 bytes")
		}
		return decoded, nil
	}

	if isHex(value) {
		decoded, err := hex.DecodeString(value)
		if err != nil {
			return nil, errors.New("invalid hex master key")
		}
		if len(decoded) != 32 {
			return nil, errors.New("master key must be 32 bytes")
		}
		return decoded, nil
	}

	if looksBase64(value) {
		decoded, err := base64.StdEncoding.DecodeString(value)
		if err == nil && len(decoded) == 32 {
			return decoded, nil
		}
	}

	if len(value) == 32 {
		return []byte(value), nil
	}

	return nil, errors.New("master key must be 32 bytes")
}

func isHex(value string) bool {
	if len(value)%2 != 0 {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}

func looksBase64(value string) bool {
	for _, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '+' || r == '/' || r == '=' {
			continue
		}
		return false
	}
	return len(value) >= 44
}

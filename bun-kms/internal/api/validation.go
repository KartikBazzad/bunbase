package api

import (
	"encoding/base64"
	"errors"
	"regexp"
	"unicode"
)

const (
	MaxKeyNameLen    = 256
	MaxPayloadLen    = 512 * 1024 // 512KB
	MaxSecretNameLen = 256
)

var keyNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// ValidateKeyName checks key name format and length.
func ValidateKeyName(name string) error {
	if name == "" {
		return errors.New("name required")
	}
	if len(name) > MaxKeyNameLen {
		return errors.New("name too long")
	}
	if !keyNameRe.MatchString(name) {
		return errors.New("name must start with alphanumeric and contain only letters, numbers, . _ -")
	}
	return nil
}

// ValidateSecretName checks secret name format and length.
func ValidateSecretName(name string) error {
	if name == "" {
		return errors.New("name required")
	}
	if len(name) > MaxSecretNameLen {
		return errors.New("name too long")
	}
	if !keyNameRe.MatchString(name) {
		return errors.New("name must start with alphanumeric and contain only letters, numbers, . _ -")
	}
	return nil
}

// ValidatePayloadSize returns an error if the payload exceeds MaxPayloadLen.
func ValidatePayloadSize(data []byte) error {
	if len(data) > MaxPayloadLen {
		return errors.New("payload too large")
	}
	return nil
}

// DecodeBase64Payload decodes base64 and validates size. Prefer over raw base64 when accepting user input.
func DecodeBase64Payload(b64 string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, errors.New("invalid base64")
	}
	if err := ValidatePayloadSize(decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

// SanitizeKeyName trims and normalizes a key name (no control chars).
func SanitizeKeyName(name string) string {
	var out []rune
	for _, r := range name {
		if unicode.IsControl(r) {
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateSessionToken generates a cryptographically secure random token
func GenerateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// SessionDuration is the default session duration (30 days)
const SessionDuration = 30 * 24 * time.Hour

// CalculateExpiry calculates the expiry time for a session
func CalculateExpiry() time.Time {
	return time.Now().Add(SessionDuration)
}

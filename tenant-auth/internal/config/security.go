package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
)

// ValidateJWTSecret validates the JWT secret and generates one if missing in development mode.
// Returns the secret and an error if validation fails.
func ValidateJWTSecret() (string, error) {
	secret := os.Getenv("TENANTAUTH_JWT_SECRET")
	environment := os.Getenv("PLATFORM_ENVIRONMENT")
	if environment == "" {
		environment = os.Getenv("TENANTAUTH_ENVIRONMENT")
	}
	if environment == "" {
		environment = "development" // Default to development
	}

	if secret == "" {
		if environment == "production" {
			return "", fmt.Errorf("TENANTAUTH_JWT_SECRET must be set in production")
		}
		// Generate strong secret for dev only
		secret = generateStrongSecret(32)
		os.Setenv("TENANTAUTH_JWT_SECRET", secret)
		fmt.Printf("WARNING: Generated JWT secret for development. Set TENANTAUTH_JWT_SECRET in production!\n")
	}

	if len(secret) < 32 {
		return "", fmt.Errorf("JWT secret must be at least 32 bytes (got %d bytes). Set TENANTAUTH_JWT_SECRET to a secure value", len(secret))
	}

	return secret, nil
}

// generateStrongSecret generates a cryptographically secure random secret of the specified length.
func generateStrongSecret(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("failed to generate secret: %v", err))
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

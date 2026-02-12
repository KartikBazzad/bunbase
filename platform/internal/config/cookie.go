package config

import "os"

// GetCookieSecure returns whether cookies should use the Secure flag.
// Defaults to false for development, true for production.
func GetCookieSecure() bool {
	if val := os.Getenv("PLATFORM_COOKIE_SECURE"); val != "" {
		return val == "true"
	}
	// Default to false for development, true for production
	return os.Getenv("PLATFORM_ENVIRONMENT") == "production"
}

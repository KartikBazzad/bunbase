package models

import "time"

// APIToken represents a personal access token for a user (e.g. CLI or integrations)
type APIToken struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Name       string    `json:"name"`
	Scopes     string    `json:"scopes,omitempty"` // comma-separated or JSON-encoded scopes
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}


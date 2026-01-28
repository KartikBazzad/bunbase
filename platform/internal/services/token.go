package services

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kartikbazzad/bunbase/platform/internal/models"
)

// TokenService manages API tokens for users.
type TokenService struct {
	db *sql.DB
}

// NewTokenService creates a new TokenService.
func NewTokenService(db *sql.DB) *TokenService {
	return &TokenService{db: db}
}

// hashToken returns a hex-encoded SHA-256 hash of the token.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// GenerateTokenValue generates a new random token string (hex-encoded).
func GenerateTokenValue() (string, error) {
	raw, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return raw.String(), nil
}

// CreateToken creates a new API token for the given user and returns the
// token record and the raw token value (only returned once).
func (s *TokenService) CreateToken(userID, name, scopes string, ttl time.Duration) (*models.APIToken, string, error) {
	if name == "" {
		return nil, "", fmt.Errorf("token name is required")
	}

	raw, err := GenerateTokenValue()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	now := time.Now()
	var expiresAt *time.Time
	if ttl > 0 {
		e := now.Add(ttl)
		expiresAt = &e
	}

	tokenID := uuid.New().String()
	hash := hashToken(raw)

	_, err = s.db.Exec(
		"INSERT INTO api_tokens (id, user_id, name, token_hash, scopes, expires_at, last_used_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		tokenID,
		userID,
		name,
		hash,
		scopes,
		func() interface{} {
			if expiresAt != nil {
				return expiresAt.Unix()
			}
			return nil
		}(),
		nil,
		now.Unix(),
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create token: %w", err)
	}

	token := &models.APIToken{
		ID:        tokenID,
		UserID:    userID,
		Name:      name,
		Scopes:    scopes,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}

	return token, raw, nil
}

// GetTokenByValue resolves a raw token string to an APIToken.
func (s *TokenService) GetTokenByValue(raw string) (*models.APIToken, error) {
	if raw == "" {
		return nil, fmt.Errorf("token is required")
	}

	h := hashToken(raw)
	var t models.APIToken
	var createdAt int64
	var expiresAt sql.NullInt64
	var lastUsedAt sql.NullInt64

	err := s.db.QueryRow(
		"SELECT id, user_id, name, scopes, expires_at, last_used_at, created_at FROM api_tokens WHERE token_hash = ?",
		h,
	).Scan(&t.ID, &t.UserID, &t.Name, &t.Scopes, &expiresAt, &lastUsedAt, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("token not found")
		}
		return nil, fmt.Errorf("failed to load token: %w", err)
	}

	t.CreatedAt = time.Unix(createdAt, 0)
	if expiresAt.Valid {
		ts := time.Unix(expiresAt.Int64, 0)
		t.ExpiresAt = &ts
	}
	if lastUsedAt.Valid {
		ts := time.Unix(lastUsedAt.Int64, 0)
		t.LastUsedAt = &ts
	}

	return &t, nil
}

// ListTokensForUser returns all tokens for a given user.
func (s *TokenService) ListTokensForUser(userID string) ([]*models.APIToken, error) {
	rows, err := s.db.Query(
		"SELECT id, user_id, name, scopes, expires_at, last_used_at, created_at FROM api_tokens WHERE user_id = ? ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*models.APIToken
	for rows.Next() {
		var t models.APIToken
		var createdAt int64
		var expiresAt sql.NullInt64
		var lastUsedAt sql.NullInt64

		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Scopes, &expiresAt, &lastUsedAt, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan token: %w", err)
		}

		t.CreatedAt = time.Unix(createdAt, 0)
		if expiresAt.Valid {
			ts := time.Unix(expiresAt.Int64, 0)
			t.ExpiresAt = &ts
		}
		if lastUsedAt.Valid {
			ts := time.Unix(lastUsedAt.Int64, 0)
			t.LastUsedAt = &ts
		}

		tokens = append(tokens, &t)
	}

	return tokens, nil
}

// MarkTokenUsed updates the last_used_at for the given token ID.
func (s *TokenService) MarkTokenUsed(id string) error {
	_, err := s.db.Exec(
		"UPDATE api_tokens SET last_used_at = ? WHERE id = ?",
		time.Now().Unix(),
		id,
	)
	return err
}

// RevokeToken deletes a single token by ID.
func (s *TokenService) RevokeToken(id string) error {
	_, err := s.db.Exec("DELETE FROM api_tokens WHERE id = ?", id)
	return err
}

// RevokeAllForUser deletes all tokens for a user.
func (s *TokenService) RevokeAllForUser(userID string) error {
	_, err := s.db.Exec("DELETE FROM api_tokens WHERE user_id = ?", userID)
	return err
}


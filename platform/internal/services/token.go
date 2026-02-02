package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kartikbazzad/bunbase/platform/internal/models"
)

// TokenService manages API tokens for users.
type TokenService struct {
	db *pgxpool.Pool
}

// NewTokenService creates a new TokenService.
func NewTokenService(db *pgxpool.Pool) *TokenService {
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

	_, err = s.db.Exec(context.Background(),
		"INSERT INTO api_tokens (id, user_id, name, token_hash, scopes, expires_at, last_used_at, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		tokenID,
		userID,
		name,
		hash,
		scopes,
		expiresAt,
		nil,
		now,
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

	err := s.db.QueryRow(context.Background(),
		"SELECT id, user_id, name, scopes, expires_at, last_used_at, created_at FROM api_tokens WHERE token_hash = $1",
		h,
	).Scan(&t.ID, &t.UserID, &t.Name, &t.Scopes, &t.ExpiresAt, &t.LastUsedAt, &t.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("token not found")
		}
		return nil, fmt.Errorf("failed to load token: %w", err)
	}

	return &t, nil
}

// ListTokensForUser returns all tokens for a given user.
func (s *TokenService) ListTokensForUser(userID string) ([]*models.APIToken, error) {
	rows, err := s.db.Query(context.Background(),
		"SELECT id, user_id, name, scopes, expires_at, last_used_at, created_at FROM api_tokens WHERE user_id = $1 ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*models.APIToken
	for rows.Next() {
		var t models.APIToken

		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Scopes, &t.ExpiresAt, &t.LastUsedAt, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan token: %w", err)
		}

		tokens = append(tokens, &t)
	}

	return tokens, nil
}

// MarkTokenUsed updates the last_used_at for the given token ID.
func (s *TokenService) MarkTokenUsed(id string) error {
	_, err := s.db.Exec(context.Background(),
		"UPDATE api_tokens SET last_used_at = $1 WHERE id = $2",
		time.Now(),
		id,
	)
	return err
}

// RevokeToken deletes a single token by ID.
func (s *TokenService) RevokeToken(id string) error {
	_, err := s.db.Exec(context.Background(), "DELETE FROM api_tokens WHERE id = $1", id)
	return err
}

// RevokeAllForUser deletes all tokens for a user.
func (s *TokenService) RevokeAllForUser(userID string) error {
	_, err := s.db.Exec(context.Background(), "DELETE FROM api_tokens WHERE user_id = $1", userID)
	return err
}

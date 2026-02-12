package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kartikbazzad/bunbase/pkg/bunauth"
	"github.com/kartikbazzad/bunbase/platform/internal/auth"
	"github.com/kartikbazzad/bunbase/platform/internal/models"
)

// SessionType represents the type of session
type SessionType string

const (
	SessionTypePlatform SessionType = "platform" // Validated via bun-auth
	SessionTypeTenant   SessionType = "tenant"   // Validated via tenant-auth
)

// Session represents a stored session
type Session struct {
	ID             uuid.UUID
	SessionToken   string
	JWTToken       string
	SessionType    SessionType
	UserID         *uuid.UUID
	ProjectID      *uuid.UUID
	ExpiresAt      time.Time
	CreatedAt      time.Time
	LastAccessedAt time.Time
}

// SessionService manages platform sessions
type SessionService struct {
	db              *pgxpool.Pool
	bunAuthClient   *bunauth.Client      // bun-auth client (for platform JWTs)
	tenantAuthClient *auth.TenantClient // tenant-auth client (for tenant JWTs)
}

// NewSessionService creates a new SessionService
func NewSessionService(db *pgxpool.Pool, bunAuthClient *bunauth.Client, tenantAuthClient *auth.TenantClient) *SessionService {
	return &SessionService{
		db:              db,
		bunAuthClient:   bunAuthClient,
		tenantAuthClient: tenantAuthClient,
	}
}

// generateSessionToken generates a cryptographically secure random session token
func generateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// CreateSession creates a new session and stores the JWT token
func (s *SessionService) CreateSession(ctx context.Context, jwtToken string, sessionType SessionType, userID *uuid.UUID, projectID *uuid.UUID, expiresAt time.Time) (string, error) {
	sessionToken, err := generateSessionToken()
	if err != nil {
		return "", err
	}

	sessionID := uuid.New()
	now := time.Now()

	_, err = s.db.Exec(ctx,
		`INSERT INTO sessions (id, session_token, jwt_token, session_type, user_id, project_id, expires_at, created_at, last_accessed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		sessionID, sessionToken, jwtToken, string(sessionType), userID, projectID, expiresAt, now, now,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return sessionToken, nil
}

// GetSession retrieves a session by session token
func (s *SessionService) GetSession(ctx context.Context, sessionToken string) (*Session, error) {
	var session Session
	var userID, projectID *uuid.UUID

	err := s.db.QueryRow(ctx,
		`SELECT id, session_token, jwt_token, session_type, user_id, project_id, expires_at, created_at, last_accessed_at
		 FROM sessions WHERE session_token = $1`,
		sessionToken,
	).Scan(
		&session.ID, &session.SessionToken, &session.JWTToken, &session.SessionType,
		&userID, &projectID, &session.ExpiresAt, &session.CreatedAt, &session.LastAccessedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	session.UserID = userID
	session.ProjectID = projectID

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		// Delete expired session
		s.DeleteSession(ctx, sessionToken)
		return nil, fmt.Errorf("session expired")
	}

	// Update last accessed time
	_, _ = s.db.Exec(ctx,
		`UPDATE sessions SET last_accessed_at = $1 WHERE session_token = $2`,
		time.Now(), sessionToken,
	)

	return &session, nil
}

// ValidateSession validates a session token and returns the user
func (s *SessionService) ValidateSession(ctx context.Context, sessionToken string) (*models.User, error) {
	session, err := s.GetSession(ctx, sessionToken)
	if err != nil {
		return nil, err
	}

	// Route validation to appropriate service based on session type
	if session.SessionType == SessionTypeTenant {
		// Validate tenant JWT via tenant-auth service
		resp, err := s.tenantAuthClient.Verify(session.JWTToken)
		if err != nil || !resp.Valid {
			// Delete invalid session
			s.DeleteSession(ctx, sessionToken)
			return nil, fmt.Errorf("invalid session")
		}

		// Extract user ID from claims (tenant-auth returns claims in VerifyResponse)
		userIDStr, ok := resp.Claims["sub"].(string)
		if !ok || userIDStr == "" {
			s.DeleteSession(ctx, sessionToken)
			return nil, fmt.Errorf("invalid user id in claims")
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid user id: %w", err)
		}

		user := &models.User{ID: userID}
		if email, ok := resp.Claims["email"].(string); ok && email != "" {
			user.Email = email
		}

		return user, nil
	} else {
		// Validate platform JWT via bun-auth service
		resp, err := s.bunAuthClient.Verify(session.JWTToken)
		if err != nil || !resp.Valid {
			// Delete invalid session
			s.DeleteSession(ctx, sessionToken)
			return nil, fmt.Errorf("invalid session")
		}

		// Parse user ID from response
		userID, err := uuid.Parse(resp.UserID)
		if err != nil {
			return nil, fmt.Errorf("invalid user id: %w", err)
		}

		user := &models.User{ID: userID}
		if resp.Email != "" {
			user.Email = resp.Email
		}
		if resp.Name != "" {
			user.Name = resp.Name
		}
		if resp.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, resp.CreatedAt); err == nil {
				user.CreatedAt = t
				user.UpdatedAt = t
			}
		}

		return user, nil
	}
}

// DeleteSession deletes a session
func (s *SessionService) DeleteSession(ctx context.Context, sessionToken string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM sessions WHERE session_token = $1`, sessionToken)
	return err
}

// CleanupExpiredSessions removes expired sessions
func (s *SessionService) CleanupExpiredSessions(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `DELETE FROM sessions WHERE expires_at < NOW()`)
	return err
}

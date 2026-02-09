package auth

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kartikbazzad/bunbase/pkg/bunauth"
	"github.com/kartikbazzad/bunbase/platform/internal/models"
)

// Auth handles authentication operations via BunAuth Service
type Auth struct {
	client *bunauth.Client
}

// NewAuth creates a new Auth instance
func NewAuth(client *bunauth.Client) *Auth {
	return &Auth{client: client}
}

// RegisterUser Delegates to BunAuth
func (a *Auth) RegisterUser(email, password, name string) (*models.User, error) {
	resp, err := a.client.Register(email, password, name)
	if err != nil {
		return nil, fmt.Errorf("bunauth register failed: %w", err)
	}

	// Construct local user model from response (minimal)
	// In Phase 2, platform typically blindly trusts auth service for Identity.
	// If we need the full user object, we might need a GetUser endpoint in BunAuth.
	id, err := uuid.Parse(resp.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id from auth: %w", err)
	}
	return &models.User{
		ID:    id,
		Email: email,
		Name:  name,
	}, nil
}

func (a *Auth) LoginUser(email, password string) (*models.User, string, error) {
	resp, err := a.client.Login(email, password)
	if err != nil {
		return nil, "", fmt.Errorf("bunauth login failed: %w", err)
	}

	// We don't have user details like 'name' in Login response yet (only token/id),
	// but the caller might need it.
	id, err := uuid.Parse(resp.UserID)
	if err != nil {
		return nil, "", fmt.Errorf("invalid user id from auth: %w", err)
	}
	return &models.User{
		ID:    id,
		Email: email,
	}, resp.AccessToken, nil
}

// ValidateSession verifies JWT token and returns user with profile (email, name, created_at)
func (a *Auth) ValidateSession(token string) (*models.User, error) {
	resp, err := a.client.Verify(token)
	if err != nil {
		return nil, fmt.Errorf("bunauth verify failed: %w", err)
	}

	if !resp.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	id, err := uuid.Parse(resp.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id from auth: %w", err)
	}
	user := &models.User{ID: id}
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

// LogoutUser is no-op for stateless JWT (client drops token)
// Or we could implement blacklist in BunAuth.
func (a *Auth) LogoutUser(token string) error {
	return nil
}

// CleanupExpiredSessions is no-op for JWT
func (a *Auth) CleanupExpiredSessions() error {
	return nil
}

// GetUserByID is used by other services.
// We should fetch this from BunAuth service ideally.
// For now, parse the id and return a stub user.
func (a *Auth) GetUserByID(id string) (*models.User, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}
	return &models.User{ID: parsed}, nil
}

package auth

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kartikbazzad/bunbase/platform/internal/models"
)

// Auth handles authentication operations
type Auth struct {
	db *sql.DB
}

// NewAuth creates a new Auth instance
func NewAuth(db *sql.DB) *Auth {
	return &Auth{db: db}
}

// RegisterUser creates a new user account
func (a *Auth) RegisterUser(email, password, name string) (*models.User, error) {
	// Check if user already exists
	var existingID string
	err := a.db.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&existingID)
	if err == nil {
		return nil, fmt.Errorf("user with email %s already exists", email)
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	// Hash password
	passwordHash, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	userID := uuid.New().String()
	now := time.Now().Unix()

	_, err = a.db.Exec(
		"INSERT INTO users (id, email, password_hash, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		userID, email, passwordHash, name, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return a.GetUserByID(userID)
}

// GetUserByID retrieves a user by ID
func (a *Auth) GetUserByID(id string) (*models.User, error) {
	var user models.User
	var createdAt, updatedAt int64

	err := a.db.QueryRow(
		"SELECT id, email, password_hash, name, created_at, updated_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Name, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.CreatedAt = time.Unix(createdAt, 0)
	user.UpdatedAt = time.Unix(updatedAt, 0)
	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (a *Auth) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	var createdAt, updatedAt int64

	err := a.db.QueryRow(
		"SELECT id, email, password_hash, name, created_at, updated_at FROM users WHERE email = ?",
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Name, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.CreatedAt = time.Unix(createdAt, 0)
	user.UpdatedAt = time.Unix(updatedAt, 0)
	return &user, nil
}

// LoginUser authenticates a user and creates a session
func (a *Auth) LoginUser(email, password string) (*models.User, string, error) {
	user, err := a.GetUserByEmail(email)
	if err != nil {
		return nil, "", fmt.Errorf("invalid email or password")
	}

	if !CheckPassword(password, user.PasswordHash) {
		return nil, "", fmt.Errorf("invalid email or password")
	}

	// Create session
	sessionToken, err := GenerateSessionToken()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate session token: %w", err)
	}

	sessionID := uuid.New().String()
	expiresAt := CalculateExpiry()
	now := time.Now().Unix()

	_, err = a.db.Exec(
		"INSERT INTO sessions (id, user_id, token, expires_at, created_at) VALUES (?, ?, ?, ?, ?)",
		sessionID, user.ID, sessionToken, expiresAt.Unix(), now,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create session: %w", err)
	}

	return user, sessionToken, nil
}

// ValidateSession validates a session token and returns the user
func (a *Auth) ValidateSession(token string) (*models.User, error) {
	var userID string
	var expiresAt int64

	err := a.db.QueryRow(
		"SELECT user_id, expires_at FROM sessions WHERE token = ?",
		token,
	).Scan(&userID, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid session")
		}
		return nil, fmt.Errorf("failed to validate session: %w", err)
	}

	// Check if session is expired
	if time.Now().Unix() > expiresAt {
		// Delete expired session
		a.db.Exec("DELETE FROM sessions WHERE token = ?", token)
		return nil, fmt.Errorf("session expired")
	}

	return a.GetUserByID(userID)
}

// LogoutUser deletes a session
func (a *Auth) LogoutUser(token string) error {
	_, err := a.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	if err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}
	return nil
}

// CleanupExpiredSessions removes expired sessions
func (a *Auth) CleanupExpiredSessions() error {
	now := time.Now().Unix()
	_, err := a.db.Exec("DELETE FROM sessions WHERE expires_at < ?", now)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}
	return nil
}

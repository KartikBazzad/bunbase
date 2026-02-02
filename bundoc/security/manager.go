package security

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// UserStore defines the storage interface for users
// This allows the main package to implement storage using internal Collections
type UserStore interface {
	GetUser(username string) (*User, error)
	SaveUser(user *User) error
	DeleteUser(username string) error
	ListUsers() ([]*User, error)
}

// UserManager handles user administration and credential management
type UserManager struct {
	store UserStore
}

// NewUserManager creates a new user manager
func NewUserManager(store UserStore) *UserManager {
	return &UserManager{
		store: store,
	}
}

// CreateUser creates a new user with the given password and roles
func (m *UserManager) CreateUser(username, password string, roles []Role) error {
	// Check if exists
	if _, err := m.store.GetUser(username); err == nil {
		return fmt.Errorf("user %s already exists", username)
	}

	// Generate Salt
	salt, err := GenerateSalt()
	if err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Generate Credentials
	creds, err := GenerateCredentials(password, salt, ScramIterCount)
	if err != nil {
		return fmt.Errorf("failed to generate credentials: %w", err)
	}

	// Create User
	user := &User{
		Username:       username,
		HashedPassword: creds.StoredKey + ":" + creds.ServerKey + ":" + strconv.Itoa(creds.Iterations), // Compact format
		Salt:           salt,
		Roles:          roles,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return m.store.SaveUser(user)
}

// GetUser retrieves a user
func (m *UserManager) GetUser(username string) (*User, error) {
	return m.store.GetUser(username)
}

// UpdateUserRoles updates a user's roles
func (m *UserManager) UpdateUserRoles(username string, roles []Role) error {
	user, err := m.store.GetUser(username)
	if err != nil {
		return err
	}

	user.Roles = roles
	user.UpdatedAt = time.Now()
	return m.store.SaveUser(user)
}

// GetSCRAMCredentials extracts the stored SCRAM data for a user
func (m *UserManager) GetSCRAMCredentials(username string) (ScramCredentials, error) {
	user, err := m.store.GetUser(username)
	if err != nil {
		return ScramCredentials{}, err
	}

	// Parse stored credential format "StoredKey:ServerKey:Iterations"
	parts := strings.Split(user.HashedPassword, ":")
	if len(parts) != 3 {
		return ScramCredentials{}, errors.New("invalid stored credential format")
	}

	iters, _ := strconv.Atoi(parts[2])

	return ScramCredentials{
		Salt:       user.Salt,
		StoredKey:  parts[0],
		ServerKey:  parts[1],
		Iterations: iters,
	}, nil
}

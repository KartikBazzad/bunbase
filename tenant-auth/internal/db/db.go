package db

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/kartikbazzad/bunbase/pkg/errors"
)

// DBClient interface abstracts the storage
type DBClient interface {
	CreateUser(ctx context.Context, projectID uuid.UUID, email, passwordHash string) (*TenantUser, error)
	GetUserByEmail(ctx context.Context, projectID uuid.UUID, email string) (*TenantUser, error)
	Close()
}

// BundocDB implements DBClient using Bundoc Server
type BundocDB struct {
	baseURL    string
	httpClient *http.Client
}

// TenantUser model
type TenantUser struct {
	ID           string    `json:"_id"`     // Stored as uuid or base64 email?
	UserID       uuid.UUID `json:"user_id"` // Actual UUID
	ProjectID    uuid.UUID `json:"project_id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
}

// Config holds database configuration
type Config struct {
	// We reuse DB struct naming from main config for convenience/parsing,
	// but in reality we just need BundocURL if we are strictly using Bundoc.
	// However, the config loader might expect structure.
	// Let's rely on injecting URL from main.go
}

// NewBundocDB creates a new Bundoc client
func NewBundocDB(url string) *BundocDB {
	return &BundocDB{
		baseURL:    url, // e.g., http://bundoc-server:8080
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (db *BundocDB) Close() {
	// No-op for HTTP client
}

// emailToDocID converts email to a base64 string safely usable as DocID
func emailToDocID(email string) string {
	return base64.URLEncoding.EncodeToString([]byte(email))
}

// CreateUser creates a new tenant user in Bundoc
func (db *BundocDB) CreateUser(ctx context.Context, projectID uuid.UUID, email, passwordHash string) (*TenantUser, error) {
	userID := uuid.New()
	docID := emailToDocID(email)

	user := &TenantUser{
		ID:           docID, // Use email as ID to enforce uniqueness
		UserID:       userID,
		ProjectID:    projectID,
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}

	jsonBody, err := json.Marshal(user)
	if err != nil {
		return nil, fmt.Errorf("serialize error: %w", err)
	}

	// Path: /v1/projects/{projectID}/databases/(default)/documents/users
	url := fmt.Sprintf("%s/v1/projects/%s/databases/(default)/documents/users", db.baseURL, projectID.String())

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := db.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bundoc request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		// If 409 Conflict (already exists), Bundoc might return 400 or 500 depending on impl.
		// Assuming optimistic strategy.
		return nil, fmt.Errorf("failed to create user in bundoc: status %d body %s", resp.StatusCode, string(body))
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email and projectID (using email as DocID)
func (db *BundocDB) GetUserByEmail(ctx context.Context, projectID uuid.UUID, email string) (*TenantUser, error) {
	docID := emailToDocID(email)

	// Path: /v1/projects/{projectID}/databases/(default)/documents/users/{docID}
	url := fmt.Sprintf("%s/v1/projects/%s/databases/(default)/documents/users/%s", db.baseURL, projectID.String(), docID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := db.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bundoc request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.NotFound("user not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bundoc error: status %d", resp.StatusCode)
	}

	var user TenantUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user: %w", err)
	}

	// Ensure it's the right project (should be enforced by URL but sanity check)
	if user.ProjectID != projectID {
		// Should ideally not happen if data isolation is correct in Bundoc
		return nil, fmt.Errorf("project mismatch")
	}

	return &user, nil
}

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
	ListUsers(ctx context.Context, projectID uuid.UUID) ([]TenantUser, error)
	GetAuthConfig(ctx context.Context, projectID uuid.UUID) (*AuthConfig, error)
	UpdateAuthConfig(ctx context.Context, projectID uuid.UUID, config *AuthConfig) error
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

// AuthConfig holds configuration for providers and rate limiting
type AuthConfig struct {
	ID        string                 `json:"_id,omitempty"` // "auth_config"
	Providers map[string]interface{} `json:"providers"`     // e.g. {"google": {"client_id": "...", ...}}
	RateLimit map[string]interface{} `json:"rate_limit"`    // e.g. {"window": 60, "max": 100}
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

// ListUsers retrieves all users for a project
func (db *BundocDB) ListUsers(ctx context.Context, projectID uuid.UUID) ([]TenantUser, error) {
	// Path: /v1/projects/{projectID}/databases/(default)/documents/users
	url := fmt.Sprintf("%s/v1/projects/%s/databases/(default)/documents/users", db.baseURL, projectID.String())

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
		// Collection might not exist yet -> empty list
		return []TenantUser{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bundoc error: status %d", resp.StatusCode)
	}

	var result struct {
		Documents []TenantUser `json:"documents"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode users listing: %w", err)
	}

	return result.Documents, nil
}

// GetAuthConfig retrieves auth configuration
func (db *BundocDB) GetAuthConfig(ctx context.Context, projectID uuid.UUID) (*AuthConfig, error) {
	// Path: .../databases/(default)/documents/settings/auth_config
	url := fmt.Sprintf("%s/v1/projects/%s/databases/(default)/documents/settings/auth_config", db.baseURL, projectID.String())

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
		// Return empty default config
		return &AuthConfig{
			ID:        "auth_config",
			Providers: make(map[string]interface{}),
			RateLimit: make(map[string]interface{}),
		}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bundoc error: status %d", resp.StatusCode)
	}

	var config AuthConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode auth config: %w", err)
	}
	return &config, nil
}

// UpdateAuthConfig updates auth configuration
func (db *BundocDB) UpdateAuthConfig(ctx context.Context, projectID uuid.UUID, config *AuthConfig) error {
	config.ID = "auth_config"
	// Path: .../databases/(default)/documents/settings/auth_config
	// Usage: PUT (Replace) or PATCH? Let's use PUT to ensure schema compliance if needed, or just simpler.
	// Actually handling "create or update" aka Upsert.
	// Bundoc CreateDocument uses POST .../documents/{collection}
	// Bundoc UpdateDocument uses PATCH .../documents/{collection}/{id}
	// But Bundoc doesn't support UPSERT on Update?
	// Check Get first, if not found Create, else Patch/Update.
	// OR use Create with specific ID?
	// CreateDocument doesn't let you specify ID inside body?
	// Wait, Bundoc `storage.Document` has `_id`. If passed in body, does Create respect it?
	// Looking at `HandleCreateDocument` in bundoc-server:
	// It calls `coll.Insert`. `Insert` usually generates ID if missing. If present, valid?
	// Let's check `coll.Insert`.

	// Strategy: Try Get via GetAuthConfig logic (which we just wrote, but need raw existence check).
	// Actually, easier to try Update (PATCH), if 404 then Create (POST).

	urlUpdate := fmt.Sprintf("%s/v1/projects/%s/databases/(default)/documents/settings/auth_config", db.baseURL, projectID.String())

	jsonBody, err := json.Marshal(config)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", urlUpdate, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := db.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	if resp.StatusCode == http.StatusNotFound {
		// Create it
		urlCreate := fmt.Sprintf("%s/v1/projects/%s/databases/(default)/documents/settings", db.baseURL, projectID.String())
		// Ensure body has _id
		config.ID = "auth_config"
		createBody, _ := json.Marshal(config)

		reqCreate, _ := http.NewRequestWithContext(ctx, "POST", urlCreate, bytes.NewBuffer(createBody))
		reqCreate.Header.Set("Content-Type", "application/json")
		respCreate, err := db.httpClient.Do(reqCreate)
		if err != nil {
			return err
		}
		defer respCreate.Body.Close()

		if respCreate.StatusCode != http.StatusCreated && respCreate.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(respCreate.Body)
			return fmt.Errorf("failed to create config: %d %s", respCreate.StatusCode, string(body))
		}
		return nil
	}

	return fmt.Errorf("failed to update config: %d", resp.StatusCode)
}

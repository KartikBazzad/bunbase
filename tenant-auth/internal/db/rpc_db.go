package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kartikbazzad/bunbase/pkg/bundocrpc"
	"github.com/kartikbazzad/bunbase/pkg/errors"
)

// BundocRPCDB implements DBClient using the bundoc TCP RPC (faster than HTTP).
type BundocRPCDB struct {
	client *bundocrpc.Client
}

// NewBundocRPCDB creates a DB client that uses bundoc RPC. addr is TCP (e.g. "bundoc-auth:9091").
func NewBundocRPCDB(addr string) *BundocRPCDB {
	return &BundocRPCDB{client: bundocrpc.New(addr)}
}

// Close closes the RPC connection.
func (db *BundocRPCDB) Close() {
	if db.client != nil {
		db.client.Close()
	}
}

const dbPathPrefix = "/databases/(default)"

func (db *BundocRPCDB) CreateUser(ctx context.Context, projectID uuid.UUID, email, passwordHash string) (*TenantUser, error) {
	userID := uuid.New()
	docID := emailToDocID(email)
	user := &TenantUser{
		ID:           docID,
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
	path := dbPathPrefix + "/documents/users"
	status, body, err := db.client.ProxyRequest("POST", projectID.String(), path, jsonBody)
	if err != nil {
		return nil, fmt.Errorf("bundoc RPC failed: %w", err)
	}
	if status != 200 && status != 201 {
		return nil, fmt.Errorf("failed to create user in bundoc: status %d body %s", status, string(body))
	}
	return user, nil
}

func (db *BundocRPCDB) GetUserByEmail(ctx context.Context, projectID uuid.UUID, email string) (*TenantUser, error) {
	docID := emailToDocID(email)
	path := dbPathPrefix + "/documents/users/" + docID
	status, body, err := db.client.ProxyRequest("GET", projectID.String(), path, nil)
	if err != nil {
		return nil, fmt.Errorf("bundoc RPC failed: %w", err)
	}
	if status == 404 {
		return nil, errors.NotFound("user not found")
	}
	if status != 200 {
		return nil, fmt.Errorf("bundoc error: status %d", status)
	}
	var user TenantUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("failed to decode user: %w", err)
	}
	if user.ProjectID != projectID {
		return nil, fmt.Errorf("project mismatch")
	}
	return &user, nil
}

func (db *BundocRPCDB) ListUsers(ctx context.Context, projectID uuid.UUID) ([]TenantUser, error) {
	path := dbPathPrefix + "/documents/users"
	status, body, err := db.client.ProxyRequest("GET", projectID.String(), path, nil)
	if err != nil {
		return nil, fmt.Errorf("bundoc RPC failed: %w", err)
	}
	if status == 404 {
		return []TenantUser{}, nil
	}
	if status != 200 {
		return nil, fmt.Errorf("bundoc error: status %d", status)
	}
	var result struct {
		Documents []TenantUser `json:"documents"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode users listing: %w", err)
	}
	return result.Documents, nil
}

func (db *BundocRPCDB) GetAuthConfig(ctx context.Context, projectID uuid.UUID) (*AuthConfig, error) {
	path := dbPathPrefix + "/documents/settings/auth_config"
	status, body, err := db.client.ProxyRequest("GET", projectID.String(), path, nil)
	if err != nil {
		return nil, fmt.Errorf("bundoc RPC failed: %w", err)
	}
	if status == 404 {
		return &AuthConfig{
			ID:        "auth_config",
			Providers: make(map[string]interface{}),
			RateLimit: make(map[string]interface{}),
		}, nil
	}
	if status != 200 {
		return nil, fmt.Errorf("bundoc error: status %d", status)
	}
	var config AuthConfig
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("failed to decode auth config: %w", err)
	}
	return &config, nil
}

func (db *BundocRPCDB) UpdateAuthConfig(ctx context.Context, projectID uuid.UUID, config *AuthConfig) error {
	config.ID = "auth_config"
	path := dbPathPrefix + "/documents/settings/auth_config"
	jsonBody, err := json.Marshal(config)
	if err != nil {
		return err
	}
	status, body, err := db.client.ProxyRequest("PATCH", projectID.String(), path, jsonBody)
	if err != nil {
		return err
	}
	if status == 200 {
		return nil
	}
	if status == 404 {
		pathCreate := dbPathPrefix + "/collections/settings/documents"
		createBody, _ := json.Marshal(config)
		statusCreate, bodyCreate, errCreate := db.client.ProxyRequest("POST", projectID.String(), pathCreate, createBody)
		if errCreate != nil {
			return errCreate
		}
		if statusCreate != 201 && statusCreate != 200 {
			return fmt.Errorf("failed to create config: %d %s", statusCreate, string(bodyCreate))
		}
		return nil
	}
	return fmt.Errorf("failed to update config: %d %s", status, string(body))
}

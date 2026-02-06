package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kartikbazzad/bunbase/pkg/logger"
	"github.com/kartikbazzad/bunbase/tenant-auth/internal/db"
	"github.com/kartikbazzad/bunbase/tenant-auth/internal/kms"
	"golang.org/x/crypto/bcrypt"
)

// Handler handles authentication requests
type Handler struct {
	db        db.DBClient
	log       *slog.Logger
	jwtSecret []byte
	kmsClient kms.ClientInterface // optional; when set, provider client_secret is stored in KMS and only ref in Bundoc
}

// NewHandler creates a new handler. kmsClient may be nil; then provider secrets are not persisted to KMS.
func NewHandler(database db.DBClient, jwtSecret string, kmsClient kms.ClientInterface) *Handler {
	return &Handler{
		db:        database,
		log:       logger.Get(),
		jwtSecret: []byte(jwtSecret),
		kmsClient: kmsClient,
	}
}

// ServeHTTP implements http.Handler
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CORS? For now assume it's behind gateway or internal.

	if r.URL.Path == "/health" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
		return
	}

	switch {
	case r.URL.Path == "/register" && r.Method == http.MethodPost:
		h.handleRegister(w, r)
	case r.URL.Path == "/login" && r.Method == http.MethodPost:
		h.handleLogin(w, r)
	case r.URL.Path == "/verify" && r.Method == http.MethodPost:
		h.handleVerify(w, r)
	case r.URL.Path == "/health":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))

	// "Admin" routes for Platform Console
	// /projects/{projectID}/users
	case matchProjectRoute(r.URL.Path, "users"):
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.handleListUsers(w, r)

	// /projects/{projectID}/config
	case matchProjectRoute(r.URL.Path, "config"):
		if r.Method == http.MethodGet {
			h.handleGetConfig(w, r)
		} else if r.Method == http.MethodPut {
			h.handleUpdateConfig(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}

	default:
		http.NotFound(w, r)
	}
}

// matchProjectRoute helper checks if path matches /projects/{id}/suffix
// Handles: /projects/{id}/suffix (4 parts), projects/{id}/suffix (3 parts), and trailing slash.
func matchProjectRoute(path, suffix string) bool {
	path = strings.TrimSuffix(strings.Trim(path, "/"), "/")
	parts := strings.Split(path, "/")
	// /projects/id/suffix -> ["", "projects", "id", "suffix"] (4). projects/id/suffix -> ["projects", "id", "suffix"] (3)
	if len(parts) == 4 {
		return parts[1] == "projects" && parts[3] == suffix
	}
	if len(parts) == 3 {
		return parts[0] == "projects" && parts[2] == suffix
	}
	return false
}

type registerRequest struct {
	ProjectID string `json:"project_id"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		http.Error(w, "Invalid project_id", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.log.Error("Failed to hash password", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.db.CreateUser(r.Context(), projectID, req.Email, string(hashedPassword))
	if err != nil {
		h.log.Error("Failed to create user", "error", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Generate token immediately after registration? Or require login?
	// Let's return the user for now.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         user.ID,
		"project_id": user.ProjectID,
		"email":      user.Email,
	})
}

type loginRequest struct {
	ProjectID string `json:"project_id"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		http.Error(w, "Invalid project_id", http.StatusBadRequest)
		return
	}

	user, err := h.db.GetUserByEmail(r.Context(), projectID, req.Email)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := h.generateToken(user)
	if err != nil {
		h.log.Error("Failed to generate token", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":         user.ID,
			"email":      user.Email,
			"project_id": user.ProjectID,
		},
	})
}

type verifyRequest struct {
	Token string `json:"token"`
}

func (h *Handler) handleVerify(w http.ResponseWriter, r *http.Request) {
	// Check Authorization header first
	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		// Fallback to body
		var req verifyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			tokenString = req.Token
		}
	}

	// Remove Bearer prefix if present
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	if tokenString == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, http.ErrAbortHandler // Unexpected signing method
		}
		return h.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(w, "Invalid token claims", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":  true,
		"claims": claims,
	})
}

// tenantUserPublic is the sanitized user shape returned by list users (no password_hash).
type tenantUserPublic struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ProjectID string    `json:"project_id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *Handler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.extractProjectID(r.URL.Path)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	users, err := h.db.ListUsers(r.Context(), projectID)
	if err != nil {
		h.log.Error("Failed to list users", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	out := make([]tenantUserPublic, 0, len(users))
	for _, u := range users {
		out = append(out, tenantUserPublic{
			ID:        u.ID,
			UserID:    u.UserID.String(),
			ProjectID: u.ProjectID.String(),
			Email:     u.Email,
			CreatedAt: u.CreatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": out,
	})
}

func (h *Handler) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.extractProjectID(r.URL.Path)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	config, err := h.db.GetAuthConfig(r.Context(), projectID)
	if err != nil {
		h.log.Error("Failed to get config", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Never return secret values to the client; strip client_secret from every provider (legacy or accidental)
	sanitized := sanitizeConfigForResponse(config)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sanitized)
}

// providerSecretKeys are keys that must be stored in KMS and replaced with a ref in Bundoc.
var providerSecretKeys = []string{"client_secret"}

// sanitizeConfigForResponse returns a copy of config with client_secret (and any secret key) removed from every provider so we never send secrets to the client.
func sanitizeConfigForResponse(config *db.AuthConfig) *db.AuthConfig {
	if config == nil || config.Providers == nil {
		return config
	}
	out := &db.AuthConfig{
		ID:        config.ID,
		Providers: make(map[string]interface{}),
		RateLimit: config.RateLimit,
	}
	for k, v := range config.Providers {
		prov, ok := v.(map[string]interface{})
		if !ok {
			out.Providers[k] = v
			continue
		}
		clone := make(map[string]interface{})
		for kk, vv := range prov {
			isSecret := false
			for _, sk := range providerSecretKeys {
				if kk == sk {
					isSecret = true
					break
				}
			}
			if !isSecret {
				clone[kk] = vv
			}
		}
		out.Providers[k] = clone
	}
	return out
}

func (h *Handler) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.extractProjectID(r.URL.Path)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	var config db.AuthConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	projectIDStr := projectID.String()
	// Process providers: store secrets in KMS, keep only refs in config
	if config.Providers != nil {
		for providerKey, v := range config.Providers {
			prov, ok := v.(map[string]interface{})
			if !ok {
				continue
			}
			for _, secretKey := range providerSecretKeys {
				raw, ok := prov[secretKey]
				if !ok {
					continue
				}
				plaintext, _ := raw.(string)
				if plaintext == "" {
					continue
				}
				// KMS name for client_secret: tenant_auth.projects.<id>.providers.<key>.client_secret
				kmsName := kms.SecretNameForProvider(projectIDStr, providerKey)
				if h.kmsClient != nil {
					if err := h.kmsClient.PutSecret(kmsName, plaintext); err != nil {
						h.log.Error("KMS put secret failed", "provider", providerKey, "error", err)
						http.Error(w, "Failed to store provider secret", http.StatusInternalServerError)
						return
					}
					prov["client_secret_ref"] = kmsName
				}
				delete(prov, secretKey)
			}
			config.Providers[providerKey] = prov
		}
	}

	if err := h.db.UpdateAuthConfig(r.Context(), projectID, &config); err != nil {
		h.log.Error("Failed to update config", "error", err)
		http.Error(w, "Failed to update config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"updated"}`))
}

// GetProviderClientSecret returns the plaintext client_secret for a provider, resolving from KMS when client_secret_ref is set.
// Used at runtime (e.g. OAuth token exchange). Returns empty string and nil error if no secret is configured.
func (h *Handler) GetProviderClientSecret(ctx context.Context, projectID uuid.UUID, providerKey string) (string, error) {
	config, err := h.db.GetAuthConfig(ctx, projectID)
	if err != nil || config == nil || config.Providers == nil {
		return "", err
	}
	prov, _ := config.Providers[providerKey].(map[string]interface{})
	if prov == nil {
		return "", nil
	}
	ref, _ := prov["client_secret_ref"].(string)
	if ref == "" {
		return "", nil
	}
	if h.kmsClient == nil {
		return "", nil
	}
	return h.kmsClient.GetSecret(ref)
}

func (h *Handler) extractProjectID(path string) (uuid.UUID, error) {
	// Path may be /projects/{id}/suffix (3 parts) or /v1/projects/{id}/suffix (4+ parts).
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, p := range parts {
		if p == "projects" && i+1 < len(parts) {
			return uuid.Parse(parts[i+1])
		}
	}
	return uuid.Nil, fmt.Errorf("path does not contain projects segment: %s", path)
}

func (h *Handler) generateToken(user *db.TenantUser) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":        user.UserID.String(), // Use UUID
		"project_id": user.ProjectID.String(),
		"email":      user.Email,
		"type":       "tenant_user",
		"iss":        "bun-tenant-auth",
		"iat":        time.Now().Unix(),
		"exp":        time.Now().Add(24 * time.Hour).Unix(),
	})

	return token.SignedString(h.jwtSecret)
}

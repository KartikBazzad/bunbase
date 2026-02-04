package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kartikbazzad/bunbase/pkg/logger"
	"github.com/kartikbazzad/bunbase/tenant-auth/internal/db"
	"golang.org/x/crypto/bcrypt"
)

// Handler handles authentication requests
type Handler struct {
	db  db.DBClient
	log *slog.Logger
	// TODO: support per-project keys. For now, using a global secret for PoC phase.
	jwtSecret []byte
}

// NewHandler creates a new handler
func NewHandler(database db.DBClient, jwtSecret string) *Handler {
	return &Handler{
		db:        database,
		log:       logger.Get(),
		jwtSecret: []byte(jwtSecret),
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

	if r.Method != http.MethodPost {
		http.NotFound(w, r)
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
func matchProjectRoute(path, suffix string) bool {
	// pattern: /projects/{id}/suffix
	// len parts: 0:"", 1:"projects", 2:"{id}", 3:"suffix"
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 {
		return false
	}
	return parts[1] == "projects" && parts[3] == suffix
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": users,
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
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

	if err := h.db.UpdateAuthConfig(r.Context(), projectID, &config); err != nil {
		h.log.Error("Failed to update config", "error", err)
		http.Error(w, "Failed to update config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"updated"}`))
}

func (h *Handler) extractProjectID(path string) (uuid.UUID, error) {
	// Assumes path matches /projects/{id}/...
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 3 || parts[1] != "projects" {
		return uuid.Parse("invalid-path-structure")
	}
	return uuid.Parse(parts[2])
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

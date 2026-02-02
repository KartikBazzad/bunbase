package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
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

	switch r.URL.Path {
	case "/register":
		h.handleRegister(w, r)
	case "/login":
		h.handleLogin(w, r)
	case "/verify":
		h.handleVerify(w, r)
	case "/health":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	default:
		http.NotFound(w, r)
	}
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

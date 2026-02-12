package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/auth"
	"github.com/kartikbazzad/bunbase/platform/internal/config"
	"github.com/kartikbazzad/bunbase/platform/internal/middleware"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	auth            *auth.Auth
	instanceService *services.InstanceService
	sessionService  *services.SessionService
}

// NewAuthHandler creates a new AuthHandler. instanceService may be nil (cloud-only); when set, Register is gated for self-hosted.
func NewAuthHandler(authService *auth.Auth, instanceService *services.InstanceService, sessionService *services.SessionService) *AuthHandler {
	return &AuthHandler{auth: authService, instanceService: instanceService, sessionService: sessionService}
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	if h.instanceService != nil && h.instanceService.DeploymentMode() == "self_hosted" {
		complete, err := h.instanceService.SetupComplete(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if complete {
			c.JSON(http.StatusForbidden, gin.H{"error": "Sign up is disabled on this instance. Contact your administrator."})
			return
		}
	}

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email, password, and name are required"})
		return
	}

	if len(req.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
		return
	}

	// Register user
	user, err := h.auth.RegisterUser(req.Email, req.Password, req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Login to get JWT token from bun-auth
	_, jwtToken, err := h.auth.LoginUser(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	// Create session using SessionService (stores JWT, returns session token)
	userID := &user.ID
	expiresAt := time.Now().Add(30 * 24 * time.Hour) // 30 days
	sessionToken, err := h.sessionService.CreateSession(c.Request.Context(), jwtToken, services.SessionTypePlatform, userID, nil, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	// Set session cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   config.GetCookieSecure(),
	})

	c.JSON(http.StatusOK, user.ToResponse())
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password are required"})
		return
	}

	// Login user to get JWT token from bun-auth
	user, jwtToken, err := h.auth.LoginUser(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// Create session using SessionService (stores JWT, returns session token)
	userID := &user.ID
	expiresAt := time.Now().Add(30 * 24 * time.Hour) // 30 days
	sessionToken, err := h.sessionService.CreateSession(c.Request.Context(), jwtToken, services.SessionTypePlatform, userID, nil, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	// Set session cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   config.GetCookieSecure(),
	})

	c.JSON(http.StatusOK, user.ToResponse())
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	sessionToken := middleware.GetSessionTokenFromContext(c)
	if sessionToken != "" {
		// Delete session from SessionService
		_ = h.sessionService.DeleteSession(c.Request.Context(), sessionToken)
	}

	// Clear cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// Me returns the current authenticated user
func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := middleware.RequireAuth(c)
	if !ok {
		return
	}

	resp := user.ToResponse()
	if h.instanceService != nil && h.instanceService.DeploymentMode() == "self_hosted" {
		admin, _ := h.instanceService.IsInstanceAdmin(c.Request.Context(), user.ID.String())
		resp.IsInstanceAdmin = &admin
	}
	c.JSON(http.StatusOK, resp)
}


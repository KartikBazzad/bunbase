package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/auth"
	"github.com/kartikbazzad/bunbase/platform/internal/middleware"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	auth            *auth.Auth
	instanceService *services.InstanceService
}

// NewAuthHandler creates a new AuthHandler. instanceService may be nil (cloud-only); when set, Register is gated for self-hosted.
func NewAuthHandler(authService *auth.Auth, instanceService *services.InstanceService) *AuthHandler {
	return &AuthHandler{auth: authService, instanceService: instanceService}
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

	// Create session (user already created, but we need to create session)
	_, sessionToken, err := h.auth.LoginUser(req.Email, req.Password)
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
		Secure:   false, // Set to true in production with HTTPS
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

	// Login user
	user, sessionToken, err := h.auth.LoginUser(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
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
		Secure:   false, // Set to true in production with HTTPS
	})

	c.JSON(http.StatusOK, user.ToResponse())
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	token := middleware.GetSessionTokenFromContext(c)
	if token != "" {
		h.auth.LogoutUser(token)
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


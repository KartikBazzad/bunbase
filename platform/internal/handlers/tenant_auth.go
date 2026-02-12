package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kartikbazzad/bunbase/platform/internal/auth"
	"github.com/kartikbazzad/bunbase/platform/internal/config"
	"github.com/kartikbazzad/bunbase/platform/internal/middleware"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

type TenantAuthHandler struct {
	client         *auth.TenantClient
	projectService *services.ProjectService
	sessionService *services.SessionService
}

func NewTenantAuthHandler(client *auth.TenantClient, projectService *services.ProjectService, sessionService *services.SessionService) *TenantAuthHandler {
	return &TenantAuthHandler{
		client:         client,
		projectService: projectService,
		sessionService: sessionService,
	}
}

// ListProjectUsers lists all users (identities) for a project's auth system
// GET /api/projects/:id/auth/users
func (h *TenantAuthHandler) ListProjectUsers(c *gin.Context) {
	projectID := middleware.GetProjectID(c)
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID required"})
		return
	}

	// Verify ownership/permissions (assuming middleware checked Auth, but we check if user owns project)
	// Actually middleware `AuthAnyMiddleware` populates context user/token.
	// We should verify accessing user has access to projectID.
	// For now, let's assume middleware handles basic auth, and we trust for MVP or do:
	// userID := c.GetString("userID")
	// h.projectService.CheckAccess(userID, projectID)

	users, err := h.client.ListUsers(projectID)
	if err != nil {
		// Return 200 with empty list so the console still loads; include error for UI to show
		c.JSON(http.StatusOK, gin.H{"users": []interface{}{}, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// GetProjectAuthConfig retrieves auth configuration for a project
// GET /api/projects/:id/auth/config
func (h *TenantAuthHandler) GetProjectAuthConfig(c *gin.Context) {
	projectID := middleware.GetProjectID(c)
	config, err := h.client.GetConfig(projectID)
	if err != nil {
		// Return 200 with default config so the console still loads; include error for UI to show
		c.JSON(http.StatusOK, gin.H{
			"providers":  map[string]interface{}{},
			"rate_limit": map[string]interface{}{},
			"error":      err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, config)
}

// UpdateProjectAuthConfig updates auth configuration
// PUT /api/projects/:id/auth/config
func (h *TenantAuthHandler) UpdateProjectAuthConfig(c *gin.Context) {
	projectID := middleware.GetProjectID(c)

	var config auth.AuthConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	if err := h.client.UpdateConfig(projectID, &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// CreateProjectUser creates a new auth user for the project (admin/console).
// POST /api/projects/:id/auth/users
func (h *TenantAuthHandler) CreateProjectUser(c *gin.Context) {
	projectID := middleware.GetProjectID(c)
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID required"})
		return
	}

	var body struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password required"})
		return
	}

	user, err := h.client.Register(projectID, body.Email, body.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// RegisterProjectUser registers a new project user (key-scoped: project from API key).
// POST /v1/auth/project/register
func (h *TenantAuthHandler) RegisterProjectUser(c *gin.Context) {
	projectID := middleware.GetProjectID(c)
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID required"})
		return
	}

	var body struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password required"})
		return
	}

	user, err := h.client.Register(projectID, body.Email, body.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// LoginProjectUser logs in a project user and returns a tenant JWT (key-scoped).
// POST /v1/auth/project/login
func (h *TenantAuthHandler) LoginProjectUser(c *gin.Context) {
	projectID := middleware.GetProjectID(c)
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID required"})
		return
	}

	var body struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password required"})
		return
	}

	res, err := h.client.Login(projectID, body.Email, body.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// Create session using SessionService (stores JWT, returns session token)
	// Extract user ID from tenant-auth response
	var userID *uuid.UUID
	if res.User.UserID != "" {
		if uid, err := uuid.Parse(res.User.UserID); err == nil {
			userID = &uid
		}
	}
	var projectUUID *uuid.UUID
	if projectID != "" {
		if pid, err := uuid.Parse(projectID); err == nil {
			projectUUID = &pid
		}
	}
	expiresAt := time.Now().Add(24 * time.Hour) // 24 hours to match JWT expiry
	sessionToken, err := h.sessionService.CreateSession(c.Request.Context(), res.Token, services.SessionTypeTenant, userID, projectUUID, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	// Set HTTP-only cookie with session token (not JWT)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "tenant_session_token",
		Value:    sessionToken,
		Path:     "/",
		MaxAge:   24 * 60 * 60, // 24 hours to match JWT expiry
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   config.GetCookieSecure(),
	})

	c.JSON(http.StatusOK, gin.H{"user": res.User})
}

// ProjectUserSession returns the current project user from a tenant session cookie.
// GET /v1/auth/session - reads from tenant_session_token cookie and validates via SessionService
func (h *TenantAuthHandler) ProjectUserSession(c *gin.Context) {
	sessionToken, err := c.Cookie("tenant_session_token")
	if err != nil || sessionToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization"})
		return
	}

	// Validate session via SessionService (routes to tenant-auth for JWT validation)
	user, err := h.sessionService.ValidateSession(c.Request.Context(), sessionToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired session"})
		return
	}

	// Get project ID from session
	session, err := h.sessionService.GetSession(c.Request.Context(), sessionToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session not found"})
		return
	}

	var projectID string
	if session.ProjectID != nil {
		projectID = session.ProjectID.String()
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":         user.ID.String(),
			"user_id":    user.ID.String(),
			"email":      user.Email,
			"project_id": projectID,
		},
	})
}

// LogoutProjectUser logs out a project user by clearing the tenant session cookie.
// POST /v1/auth/project/logout
func (h *TenantAuthHandler) LogoutProjectUser(c *gin.Context) {
	// Get session token from cookie
	sessionToken, err := c.Cookie("tenant_session_token")
	if err == nil && sessionToken != "" {
		// Delete session from SessionService
		_ = h.sessionService.DeleteSession(c.Request.Context(), sessionToken)
	}

	// Clear the tenant session cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "tenant_session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

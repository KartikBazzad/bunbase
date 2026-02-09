package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/auth"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

// SetupHandler handles one-time bootstrap for self-hosted instances.
type SetupHandler struct {
	auth            *auth.Auth
	instanceService *services.InstanceService
}

// NewSetupHandler creates a new SetupHandler.
func NewSetupHandler(authService *auth.Auth, instanceService *services.InstanceService) *SetupHandler {
	return &SetupHandler{auth: authService, instanceService: instanceService}
}

// SetupRequest is the body for POST /api/setup (same as register).
type SetupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// Setup creates the first instance admin (self-hosted only). Unauthenticated.
func (h *SetupHandler) Setup(c *gin.Context) {
	if h.instanceService.DeploymentMode() != "self_hosted" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Setup is not available in cloud mode"})
		return
	}

	complete, err := h.instanceService.SetupComplete(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if complete {
		c.JSON(http.StatusForbidden, gin.H{"error": "Setup already completed"})
		return
	}

	var req SetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.Email == "" || req.Password == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email, password, and name are required"})
		return
	}
	if len(req.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
		return
	}

	user, err := h.auth.RegisterUser(req.Email, req.Password, req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.instanceService.BootstrapAdmin(c.Request.Context(), user.ID.String()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete setup"})
		return
	}

	_, sessionToken, err := h.auth.LoginUser(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusOK, user.ToResponse())
		return
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	})

	c.JSON(http.StatusCreated, user.ToResponse())
}

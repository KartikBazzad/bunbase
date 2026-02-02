package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/auth"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

type TenantAuthHandler struct {
	authClient     *auth.TenantClient
	projectService *services.ProjectService
}

func NewTenantAuthHandler(client *auth.TenantClient, projectService *services.ProjectService) *TenantAuthHandler {
	return &TenantAuthHandler{
		authClient:     client,
		projectService: projectService,
	}
}

// extractAPIKey gets the key from header X-Bunbase-Client-Key or Bearer (optional)
func (h *TenantAuthHandler) getProjectID(c *gin.Context) (string, error) {
	key := c.GetHeader("X-Bunbase-Client-Key")
	if key == "" {
		// Fallback: Check Query param ?key=...
		key = c.Query("key")
	}

	if key == "" {
		// Fallback: Authorization: Bearer pk_... (sometimes easy for integrations)
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer pk_") {
			key = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if key == "" {
		return "", nil // No key found
	}

	return h.projectService.GetProjectIDByPublicKey(key)
}

type authRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name"` // Optional for login
}

func (h *TenantAuthHandler) Register(c *gin.Context) {
	projectID, err := h.getProjectID(c)
	if err != nil || projectID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing Public API Key"})
		return
	}

	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Name is optional in auth service currently, but good to have
	user, err := h.authClient.Register(projectID, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *TenantAuthHandler) Login(c *gin.Context) {
	projectID, err := h.getProjectID(c)
	if err != nil || projectID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing Public API Key"})
		return
	}

	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.authClient.Login(projectID, req.Email, req.Password)
	if err != nil {
		// Determine status code based on error message or type (ideally client returns typed errors)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Login failed"})
		return
	}

	c.JSON(http.StatusOK, res)
}

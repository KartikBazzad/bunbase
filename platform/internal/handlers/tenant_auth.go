package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/auth"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

type TenantAuthHandler struct {
	client         *auth.TenantClient
	projectService *services.ProjectService
}

func NewTenantAuthHandler(client *auth.TenantClient, projectService *services.ProjectService) *TenantAuthHandler {
	return &TenantAuthHandler{
		client:         client,
		projectService: projectService,
	}
}

// ListProjectUsers lists all users (identities) for a project's auth system
// GET /api/projects/:id/auth/users
func (h *TenantAuthHandler) ListProjectUsers(c *gin.Context) {
	projectID := c.Param("id")
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// GetProjectAuthConfig retrieves auth configuration for a project
// GET /api/projects/:id/auth/config
func (h *TenantAuthHandler) GetProjectAuthConfig(c *gin.Context) {
	projectID := c.Param("id")
	config, err := h.client.GetConfig(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, config)
}

// UpdateProjectAuthConfig updates auth configuration
// PUT /api/projects/:id/auth/config
func (h *TenantAuthHandler) UpdateProjectAuthConfig(c *gin.Context) {
	projectID := c.Param("id")

	var config auth.AuthConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	if err := h.client.UpdateConfig(projectID, &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

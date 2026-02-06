package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/middleware"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

// ProjectConfigHandler handles project config (services) endpoint.
type ProjectConfigHandler struct {
	projectService       *services.ProjectService
	projectConfigService *services.ProjectConfigService
}

// NewProjectConfigHandler creates a new ProjectConfigHandler.
func NewProjectConfigHandler(projectService *services.ProjectService, projectConfigService *services.ProjectConfigService) *ProjectConfigHandler {
	return &ProjectConfigHandler{
		projectService:       projectService,
		projectConfigService: projectConfigService,
	}
}

// GetCurrentProject returns the current project and config when authorized by API key (key-scoped routes).
// GET /v1/project. Project ID is read from context (set by RequireProjectKeyMiddleware). No user auth.
func (h *ProjectConfigHandler) GetCurrentProject(c *gin.Context) {
	projectID := middleware.GetProjectID(c)
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project not found"})
		return
	}
	project, err := h.projectService.GetProjectByID(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}
	config := h.projectConfigService.GetConfig(project)
	c.JSON(http.StatusOK, gin.H{
		"project": gin.H{
			"id":    project.ID,
			"name":  project.Name,
			"slug":  project.Slug,
			"owner_id": project.OwnerID,
		},
		"config": config,
	})
}

// GetProjectConfig returns client-ready config for the project (gateway URL + KV, Bundoc, Buncast, Functions paths).
// GET /api/projects/:id/config (or /api/projects/:id/services).
// Requires project membership.
func (h *ProjectConfigHandler) GetProjectConfig(c *gin.Context) {
	user, ok := middleware.RequireAuth(c)
	if !ok {
		return
	}

	projectID := middleware.GetProjectID(c)
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ID required"})
		return
	}

	project, err := h.projectService.GetProjectByID(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	isMember, _, err := h.projectService.IsProjectMember(projectID, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !isMember && project.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	config := h.projectConfigService.GetConfig(project)
	c.JSON(http.StatusOK, config)
}

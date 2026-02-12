package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/authz"
	"github.com/kartikbazzad/bunbase/platform/internal/middleware"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

// ProjectHandler handles project endpoints
type ProjectHandler struct {
	projectService  *services.ProjectService
	instanceService *services.InstanceService
	enforcer        *authz.Enforcer
	limitService    *services.LimitService
}

// NewProjectHandler creates a new ProjectHandler.
func NewProjectHandler(projectService *services.ProjectService, instanceService *services.InstanceService, enforcer *authz.Enforcer, limitService *services.LimitService) *ProjectHandler {
	return &ProjectHandler{projectService: projectService, instanceService: instanceService, enforcer: enforcer, limitService: limitService}
}

// CreateProjectRequest represents a project creation request
type CreateProjectRequest struct {
	Name string `json:"name"`
}

// UpdateProjectRequest represents a project update request
type UpdateProjectRequest struct {
	Name string `json:"name"`
}

// ListProjects lists all projects for the authenticated user
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	user, ok := middleware.RequireAuth(c)
	if !ok {
		return
	}

	projects, err := h.projectService.ListProjectsByUser(user.ID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, projects)
}

// CreateProject creates a new project
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	user, ok := middleware.RequireAuth(c)
	if !ok {
		return
	}

	if h.enforcer != nil {
		deploymentMode := h.instanceService.DeploymentMode()
		allowed, err := h.enforcer.InstanceEnforce(user.ID.String(), "create_project", deploymentMode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to create projects"})
			return
		}
	}

	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	if h.limitService != nil {
		if err := h.limitService.CheckProjectLimit(c.Request.Context(), user.ID.String()); err != nil {
			if errors.Is(err, services.ErrProjectLimitReached) {
				c.JSON(http.StatusForbidden, gin.H{"error": h.limitService.LimitMessage(err)})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	project, err := h.projectService.CreateProject(req.Name, user.ID.String())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, project)
}

// GetProject retrieves a project by ID
func (h *ProjectHandler) GetProject(c *gin.Context) {
	user, ok := middleware.RequireAuth(c)
	if !ok {
		return
	}

	projectID := c.Param("id")

	project, err := h.projectService.GetProjectByID(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	if h.enforcer != nil {
		allowed, err := h.enforcer.ProjectEnforce(user.ID.String(), projectID, "project", "read")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !allowed {
			role, found, roleErr := h.projectService.GetRoleInProject(c.Request.Context(), projectID, user.ID.String())
			log.Printf("[authz] GetProject 403: userID=%s projectID=%s GetRoleInProject role=%q found=%v err=%v", user.ID.String(), projectID, role, found, roleErr)
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	} else {
		isMember, _, err := h.projectService.IsProjectMember(projectID, user.ID.String())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !isMember && project.OwnerID != user.ID.String() {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	}

	c.JSON(http.StatusOK, project)
}

// UpdateProject updates a project
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	user, ok := middleware.RequireAuth(c)
	if !ok {
		return
	}

	projectID := c.Param("id")

	if h.enforcer != nil {
		allowed, err := h.enforcer.ProjectEnforce(user.ID.String(), projectID, "project", "update")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	} else {
		isOwner, err := h.projectService.IsProjectOwner(projectID, user.ID.String())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !isOwner {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	}

	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	project, err := h.projectService.UpdateProject(projectID, req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, project)
}

// DeleteProject deletes a project
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	user, ok := middleware.RequireAuth(c)
	if !ok {
		return
	}

	projectID := c.Param("id")

	if h.enforcer != nil {
		allowed, err := h.enforcer.ProjectEnforce(user.ID.String(), projectID, "project", "delete")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	} else {
		isOwner, err := h.projectService.IsProjectOwner(projectID, user.ID.String())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !isOwner {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	}

	if err := h.projectService.DeleteProject(projectID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "project deleted"})
}

// RegenerateProjectAPIKey generates a new project API key. Only owners can regenerate.
// Returns the project with the new public_api_key (shown once).
func (h *ProjectHandler) RegenerateProjectAPIKey(c *gin.Context) {
	user, ok := middleware.RequireAuth(c)
	if !ok {
		return
	}

	projectID := c.Param("id")

	if h.enforcer != nil {
		allowed, err := h.enforcer.ProjectEnforce(user.ID.String(), projectID, "project", "regenerate_key")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	} else {
		isOwner, err := h.projectService.IsProjectOwner(projectID, user.ID.String())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !isOwner {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	}

	project, newKey, err := h.projectService.RegenerateProjectAPIKey(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Return project with new key; also return api_key for one-time display
	c.JSON(http.StatusOK, gin.H{"project": project, "api_key": newKey})
}

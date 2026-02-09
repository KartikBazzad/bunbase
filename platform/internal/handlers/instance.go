package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

// InstanceHandler handles instance-level endpoints (e.g. status for dashboard).
type InstanceHandler struct {
	instanceService *services.InstanceService
}

// NewInstanceHandler creates a new InstanceHandler.
func NewInstanceHandler(instanceService *services.InstanceService) *InstanceHandler {
	return &InstanceHandler{instanceService: instanceService}
}

// StatusResponse is the response for GET /api/instance/status.
type StatusResponse struct {
	DeploymentMode string `json:"deployment_mode"`
	SetupComplete  bool   `json:"setup_complete"`
}

// Status returns deployment_mode and setup_complete. Unauthenticated.
func (h *InstanceHandler) Status(c *gin.Context) {
	complete, err := h.instanceService.SetupComplete(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, StatusResponse{
		DeploymentMode: h.instanceService.DeploymentMode(),
		SetupComplete:  complete,
	})
}

package handlers

import (
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/middleware"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

// FunctionHandler handles function endpoints
type FunctionHandler struct {
	functionService *services.FunctionService
	projectService  *services.ProjectService
}

// NewFunctionHandler creates a new FunctionHandler
func NewFunctionHandler(functionService *services.FunctionService, projectService *services.ProjectService) *FunctionHandler {
	return &FunctionHandler{
		functionService: functionService,
		projectService:  projectService,
	}
}

// DeployFunctionRequest represents a function deployment request
type DeployFunctionRequest struct {
	Name    string `json:"name"`
	Runtime string `json:"runtime"`
	Handler string `json:"handler"`
	Version string `json:"version"`
	Bundle  string `json:"bundle"` // Base64 encoded bundle
}

// ListFunctions lists all functions for a project
func (h *FunctionHandler) ListFunctions(c *gin.Context) {
	user, ok := middleware.RequireAuth(c)
	if !ok {
		return
	}

	projectID := c.Param("id")

	// Check if user has access to this project
	isMember, _, err := h.projectService.IsProjectMember(projectID, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	isOwner, err := h.projectService.IsProjectOwner(projectID, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !isMember && !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	functions, err := h.functionService.ListFunctionsByProject(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, functions)
}

// DeployFunction deploys a function to a project
func (h *FunctionHandler) DeployFunction(c *gin.Context) {
	user, ok := middleware.RequireAuth(c)
	if !ok {
		return
	}

	projectID := c.Param("id")

	// Check if user has access to this project
	isMember, _, err := h.projectService.IsProjectMember(projectID, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	isOwner, err := h.projectService.IsProjectOwner(projectID, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !isMember && !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var req DeployFunctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Validate input
	if req.Name == "" || req.Runtime == "" || req.Handler == "" || req.Version == "" || req.Bundle == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, runtime, handler, version, and bundle are required"})
		return
	}

	// Decode base64 bundle
	bundleData, err := base64.StdEncoding.DecodeString(req.Bundle)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 bundle"})
		return
	}

	function, err := h.functionService.DeployFunction(projectID, req.Name, req.Runtime, req.Handler, req.Version, bundleData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, function)
}

// DeleteFunction deletes a function
func (h *FunctionHandler) DeleteFunction(c *gin.Context) {
	user, ok := middleware.RequireAuth(c)
	if !ok {
		return
	}

	projectID := c.Param("id")
	functionID := c.Param("functionId")

	// Check if user has access to this project
	isOwner, err := h.projectService.IsProjectOwner(projectID, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	if err := h.functionService.DeleteFunction(functionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "function deleted"})
}


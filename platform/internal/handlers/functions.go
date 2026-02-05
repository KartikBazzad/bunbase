package handlers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"sort"
	"time"

	"io"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/middleware"
	"github.com/kartikbazzad/bunbase/platform/internal/models"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
	"github.com/kartikbazzad/bunbase/platform/pkg/functions"
)

// FunctionHandler handles function endpoints
type FunctionHandler struct {
	functionService *services.FunctionService
	projectService  *services.ProjectService
	functionsURL    string
}

// NewFunctionHandler creates a new FunctionHandler
func NewFunctionHandler(functionService *services.FunctionService, projectService *services.ProjectService, functionsURL string) *FunctionHandler {
	return &FunctionHandler{
		functionService: functionService,
		projectService:  projectService,
		functionsURL:    functionsURL,
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

// InvokeFunction handles function invocation
func (h *FunctionHandler) InvokeFunction(c *gin.Context) {
	// 1. Get Project ID (from Client Key)
	key := c.GetHeader("X-Bunbase-Client-Key")
	var projectID string

	if key != "" {
		id, err := h.projectService.GetProjectIDByPublicKey(key)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API Key"})
			return
		}
		projectID = id
	} else {
		// Fallback to User Auth if needed, but for now strict Key requirement for SDK
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API Key required"})
		return
	}

	functionName := c.Param("name")
	if functionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Function name required"})
		return
	}

	// 2. Get Function by Name and Project
	function, err := h.functionService.GetFunctionByName(projectID, functionName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Function not found"})
		return
	}

	// 3. Proxy to Functions Service (HTTP Gateway)
	targetURL := fmt.Sprintf("%s/functions/%s", h.functionsURL, function.FunctionServiceID)

	proxyReq, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create proxy request"})
		return
	}

	for k, v := range c.Request.Header {
		proxyReq.Header[k] = v
	}

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Failed to invoke function: %v", err)})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	for k, v := range resp.Header {
		c.Writer.Header()[k] = v
	}

	c.Writer.WriteHeader(resp.StatusCode)
	c.Writer.Write(body)
}

// InvokeProjectFunction handles function invocation (via Project ID; user or project API key).
func (h *FunctionHandler) InvokeProjectFunction(c *gin.Context) {
	projectID := c.Param("id")
	functionName := c.Param("name")

	// Allow if authorized by project API key
	if middleware.GetProjectKeyProjectID(c) != projectID {
		user, ok := middleware.RequireAuth(c)
		if !ok {
			return
		}
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
	}

	if functionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Function name required"})
		return
	}

	// Get Function by Name and Project
	function, err := h.functionService.GetFunctionByName(projectID, functionName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Function not found"})
		return
	}

	// Proxy to Functions Service (HTTP Gateway)
	targetURL := fmt.Sprintf("%s/functions/%s", h.functionsURL, function.FunctionServiceID)

	proxyReq, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create proxy request"})
		return
	}

	// Copy headers, but filter out sensitive ones if needed
	// For console invocation, we might want to inject a special header indicating it's a test
	for k, v := range c.Request.Header {
		// Skip Host header, let Go set it
		if k == "Host" {
			continue
		}
		proxyReq.Header[k] = v
	}

	// Ensure we pass the Authorization header if the function needs it (though console usually uses cookies)
	// If the user provided custom headers in the UI, they should be in c.Request.Header

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Failed to invoke function: %v", err)})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	for k, v := range resp.Header {
		c.Writer.Header()[k] = v
	}

	c.Writer.WriteHeader(resp.StatusCode)
	c.Writer.Write(body)
}

// GetProjectFunctionLogs returns logs for the project's functions (optionally filtered by function).
func (h *FunctionHandler) GetProjectFunctionLogs(c *gin.Context) {
	user, ok := middleware.RequireAuth(c)
	if !ok {
		return
	}
	projectID := c.Param("id")
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
	functionIDOrName := c.Query("function_id")
	sinceStr := c.Query("since")
	limit := 100
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
		if limit <= 0 {
			limit = 100
		}
		if limit > 1000 {
			limit = 1000
		}
	}
	var since *time.Time
	if sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = &t
		}
	}
	if since == nil {
		t := time.Now().Add(-24 * time.Hour)
		since = &t
	}

	type logRow struct {
		functions.LogEntry
		FunctionName string `json:"function_name,omitempty"`
	}

	var all []logRow
	if functionIDOrName != "" {
		var fn *models.Function
		fn, err = h.functionService.GetFunctionByName(projectID, functionIDOrName)
		if err != nil {
			fn, err = h.functionService.GetFunctionByID(functionIDOrName)
			if err != nil || fn == nil || fn.ProjectID != projectID {
				c.JSON(http.StatusNotFound, gin.H{"error": "Function not found"})
				return
			}
		}
		entries, err := h.functionService.GetLogs(fn.FunctionServiceID, since, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for _, e := range entries {
			all = append(all, logRow{LogEntry: e, FunctionName: fn.Name})
		}
	} else {
		fns, err := h.functionService.ListFunctionsByProject(projectID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		nameByID := make(map[string]string)
		for _, fn := range fns {
			nameByID[fn.FunctionServiceID] = fn.Name
		}
		for _, fn := range fns {
			entries, err := h.functionService.GetLogs(fn.FunctionServiceID, since, limit*2)
			if err != nil {
				continue
			}
			for _, e := range entries {
				all = append(all, logRow{LogEntry: e, FunctionName: nameByID[e.FunctionID]})
			}
		}
		sort.Slice(all, func(i, j int) bool {
			return all[i].CreatedAt.After(all[j].CreatedAt)
		})
		if len(all) > limit {
			all = all[:limit]
		}
	}

	c.JSON(http.StatusOK, all)
}

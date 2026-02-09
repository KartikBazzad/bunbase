package handlers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/functions/pkg/client"
	"github.com/kartikbazzad/bunbase/platform/internal/authz"
	"github.com/kartikbazzad/bunbase/platform/internal/middleware"
	"github.com/kartikbazzad/bunbase/platform/internal/models"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
	"github.com/kartikbazzad/bunbase/platform/pkg/functions"
)

// FunctionHandler handles function endpoints
type FunctionHandler struct {
	functionService      *services.FunctionService
	projectService       *services.ProjectService
	projectConfigService *services.ProjectConfigService
	functionsURL         string
	functionsRPC         *client.Client
	enforcer             *authz.Enforcer
}

// NewFunctionHandler creates a new FunctionHandler. functionsRPC is optional; when set, invoke uses it instead of HTTP.
func NewFunctionHandler(functionService *services.FunctionService, projectService *services.ProjectService, projectConfigService *services.ProjectConfigService, functionsURL string, functionsRPC *client.Client, enforcer *authz.Enforcer) *FunctionHandler {
	return &FunctionHandler{
		functionService:      functionService,
		projectService:       projectService,
		projectConfigService: projectConfigService,
		functionsURL:         functionsURL,
		functionsRPC:         functionsRPC,
		enforcer:             enforcer,
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

// FunctionResponse represents the enriched function response
type FunctionResponse struct {
	ID                string `json:"id"`
	ProjectID         string `json:"project_id"`
	FunctionServiceID string `json:"function_service_id"`
	Name              string `json:"name"`
	Runtime           string `json:"runtime"`
	Trigger           string `json:"trigger"`    // "http" (default for all functions)
	Status            string `json:"status"`      // "active" (default, can be enhanced later)
	PathOrCron        string `json:"path_or_cron"` // Function name or invoke path
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

// ListFunctions lists all functions for a project (key-scoped GET /v1/functions or user-scoped GET /projects/:id/functions).
func (h *FunctionHandler) ListFunctions(c *gin.Context) {
	projectID := middleware.GetProjectID(c)
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ID required"})
		return
	}

	if middleware.GetProjectKeyProjectID(c) == "" {
		user, ok := middleware.RequireAuth(c)
		if !ok {
			return
		}
		if h.enforcer != nil {
			allowed, err := h.enforcer.ProjectEnforce(user.ID.String(), projectID, "function", "read")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if !allowed {
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
		} else {
			isMember, _, err := h.projectService.IsProjectMember(projectID, user.ID.String())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			isOwner, err := h.projectService.IsProjectOwner(projectID, user.ID.String())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if !isMember && !isOwner {
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
		}
	}

	functions, err := h.functionService.ListFunctionsByProject(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Transform to enriched response
	responses := make([]FunctionResponse, len(functions))
	for i, fn := range functions {
		responses[i] = FunctionResponse{
			ID:                fn.ID,
			ProjectID:         fn.ProjectID,
			FunctionServiceID: fn.FunctionServiceID,
			Name:              fn.Name,
			Runtime:           fn.Runtime,
			Trigger:           "http", // All functions are HTTP-triggered
			Status:            "active", // Default status, can be enhanced later with functions service query
			PathOrCron:        fn.Name, // Use function name as path (can be enhanced to actual invoke path)
			CreatedAt:         fn.CreatedAt.Format(time.RFC3339),
			UpdatedAt:         fn.UpdatedAt.Format(time.RFC3339),
		}
	}

	c.JSON(http.StatusOK, responses)
}

// DeployFunction deploys a function to a project
func (h *FunctionHandler) DeployFunction(c *gin.Context) {
	user, ok := middleware.RequireAuth(c)
	if !ok {
		return
	}

	projectID := middleware.GetProjectID(c)

	if h.enforcer != nil {
		allowed, err := h.enforcer.ProjectEnforce(user.ID.String(), projectID, "function", "deploy")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	} else {
		isMember, _, err := h.projectService.IsProjectMember(projectID, user.ID.String())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		isOwner, err := h.projectService.IsProjectOwner(projectID, user.ID.String())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !isMember && !isOwner {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
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

	projectID := middleware.GetProjectID(c)
	functionID := c.Param("functionId")

	if h.enforcer != nil {
		allowed, err := h.enforcer.ProjectEnforce(user.ID.String(), projectID, "function", "delete")
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
	var project *models.Project

	if key != "" {
		id, err := h.projectService.GetProjectIDByPublicKey(key)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API Key"})
			return
		}
		projectID = id
		// Load full project (includes public_api_key) for context injection
		project, err = h.projectService.GetProjectByID(projectID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load project"})
			return
		}
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

	// 2b. Build project config (for gateway URL) and inject context headers for Functions service
	if project != nil && h.projectConfigService != nil {
		cfg := h.projectConfigService.GetConfig(project)
		if cfg != nil {
			if project.PublicAPIKey != nil {
				c.Request.Header.Set("X-Bunbase-API-Key", *project.PublicAPIKey)
			}
			c.Request.Header.Set("X-Bunbase-Project-ID", projectID)
			c.Request.Header.Set("X-Bunbase-Gateway-URL", cfg.GatewayURL)
		}
	}

	h.doInvoke(c, function.FunctionServiceID)
}

// doInvoke performs the actual invoke via RPC (if configured) or HTTP proxy. Call after project context headers are set on c.Request.
func (h *FunctionHandler) doInvoke(c *gin.Context, functionServiceID string) {
	var body []byte
	if c.Request.Body != nil {
		body, _ = io.ReadAll(c.Request.Body)
	}
	path := c.Request.URL.Path
	if c.Request.URL.RawQuery != "" {
		path += "?" + c.Request.URL.RawQuery
	}
	headers := make(map[string]string)
	for k, v := range c.Request.Header {
		if len(v) > 0 && k != "Host" {
			headers[k] = v[0]
		}
	}
	query := make(map[string]string)
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}

	if h.functionsRPC != nil {
		resp, err := h.functionsRPC.Invoke(&client.InvokeRequest{
			FunctionID: functionServiceID,
			Method:     c.Request.Method,
			Path:       path,
			Headers:    headers,
			Query:      query,
			Body:       body,
		})
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Failed to invoke function: %v", err)})
			return
		}
		if !resp.Success {
			code := resp.Status
			if code == 0 {
				code = http.StatusInternalServerError
			}
			c.Data(code, "application/json", resp.Body)
			return
		}
		for k, v := range resp.Headers {
			c.Writer.Header().Set(k, v)
		}
		c.Data(resp.Status, "application/json", resp.Body)
		return
	}

	// HTTP proxy
	targetURL := fmt.Sprintf("%s/functions/%s", h.functionsURL, functionServiceID)
	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}
	proxyReq, err := http.NewRequest(c.Request.Method, targetURL, io.NopCloser(bytes.NewReader(body)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create proxy request"})
		return
	}
	for k, v := range c.Request.Header {
		if k != "Host" {
			proxyReq.Header[k] = v
		}
	}
	httpClient := &http.Client{}
	resp, err := httpClient.Do(proxyReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Failed to invoke function: %v", err)})
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	for k, v := range resp.Header {
		c.Writer.Header()[k] = v
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// InvokeProjectFunction handles function invocation (key-scoped /v1/functions/:name/invoke or user-scoped /projects/:id/functions/:name/invoke).
func (h *FunctionHandler) InvokeProjectFunction(c *gin.Context) {
	projectID := middleware.GetProjectID(c)
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ID required"})
		return
	}
	functionName := c.Param("name")
	var project *models.Project

	if middleware.GetProjectKeyProjectID(c) == "" {
		user, ok := middleware.RequireAuth(c)
		if !ok {
			return
		}
		if h.enforcer != nil {
			allowed, err := h.enforcer.ProjectEnforce(user.ID.String(), projectID, "function", "read")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if !allowed {
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
		} else {
			isMember, _, err := h.projectService.IsProjectMember(projectID, user.ID.String())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			isOwner, err := h.projectService.IsProjectOwner(projectID, user.ID.String())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if !isMember && !isOwner {
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
		}
	}

	if h.projectService != nil {
		var err error
		project, err = h.projectService.GetProjectByID(projectID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load project"})
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

	// Inject project context headers for Functions service
	if project != nil && h.projectConfigService != nil {
		cfg := h.projectConfigService.GetConfig(project)
		if cfg != nil {
			if project.PublicAPIKey != nil {
				c.Request.Header.Set("X-Bunbase-API-Key", *project.PublicAPIKey)
			}
			c.Request.Header.Set("X-Bunbase-Project-ID", projectID)
			c.Request.Header.Set("X-Bunbase-Gateway-URL", cfg.GatewayURL)
		}
	}

	h.doInvoke(c, function.FunctionServiceID)
}

// GetProjectFunctionLogs returns logs for the project's functions (optionally filtered by function).
func (h *FunctionHandler) GetProjectFunctionLogs(c *gin.Context) {
	user, ok := middleware.RequireAuth(c)
	if !ok {
		return
	}
	projectID := middleware.GetProjectID(c)
	if h.enforcer != nil {
		allowed, err := h.enforcer.ProjectEnforce(user.ID.String(), projectID, "function", "logs")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	} else {
		isMember, _, err := h.projectService.IsProjectMember(projectID, user.ID.String())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		isOwner, err := h.projectService.IsProjectOwner(projectID, user.ID.String())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !isMember && !isOwner {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
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
		fn, err := h.functionService.GetFunctionByName(projectID, functionIDOrName)
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

package handlers

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/bunder"
	"github.com/kartikbazzad/bunbase/platform/internal/middleware"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

// KVHandler proxies authenticated KV requests to bunder-manager (one Bunder instance per project).
type KVHandler struct {
	projectService *services.ProjectService
	proxy          bunder.Proxy
}

// NewKVHandler creates a KVHandler with a Proxy (HTTP or RPC).
func NewKVHandler(projectService *services.ProjectService, proxy bunder.Proxy) *KVHandler {
	return &KVHandler{
		projectService: projectService,
		proxy:          proxy,
	}
}


// DeveloperProxyHandler handles KV operations (key-scoped /v1/kv/... or user-scoped /projects/:id/kv/...).
// Project ID comes from context (key-scoped) or route param (user-scoped).
func (h *KVHandler) DeveloperProxyHandler(c *gin.Context) {
	if h.proxy == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "KV service not configured"})
		return
	}

	projectID := middleware.GetProjectID(c)
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	// Build backend path based on route params and FullPath
	// Platform routes: /v1/kv/keys, /v1/kv/kv/:key, /v1/kv/health
	// Backend path (suffix after /kv/{projectID}): /keys, /kv/{key}, /health
	fullPath := c.FullPath()
	var backendPath string
	
	if strings.Contains(fullPath, "/kv/:key") {
		// GET/PUT/DELETE /kv/:key -> /kv/{key}
		key := c.Param("key")
		backendPath = "/kv/" + key
	} else if strings.HasSuffix(fullPath, "/keys") {
		// GET /keys -> /keys
		backendPath = "/keys"
	} else if strings.HasSuffix(fullPath, "/health") {
		// GET /health -> /health
		backendPath = "/health"
	} else {
		// Fallback: extract path after /kv
		path := strings.TrimPrefix(c.Request.URL.Path, "/v1/kv")
		if strings.HasPrefix(c.Request.URL.Path, "/api/projects/") {
			path = strings.TrimPrefix(c.Request.URL.Path, "/api/projects/"+projectID+"/kv")
		} else if strings.HasPrefix(c.Request.URL.Path, "/v1/projects/") {
			path = strings.TrimPrefix(c.Request.URL.Path, "/v1/projects/"+projectID+"/kv")
		}
		if path == "" {
			path = "/"
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		backendPath = path
	}

	// Add query params to path if present
	if c.Request.URL.RawQuery != "" {
		backendPath += "?" + c.Request.URL.RawQuery
	}

	// Read request body
	var body []byte
	if c.Request.Body != nil {
		var err error
		body, err = io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
			return
		}
	}

	// Call proxy (HTTP or RPC)
	status, respBody, err := h.proxy.ProxyRequest(c.Request.Method, projectID, backendPath, body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// Set response headers
	c.Header("Content-Type", "application/octet-stream")
	if status == http.StatusOK && len(respBody) > 0 {
		// Try to detect content type for JSON responses (like /keys)
		if respBody[0] == '[' || respBody[0] == '{' {
			c.Header("Content-Type", "application/json")
		}
	}

	c.Data(status, "", respBody)
}


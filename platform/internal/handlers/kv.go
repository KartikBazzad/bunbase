package handlers

import (
	"encoding/json"
	"fmt"
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
	projectService       *services.ProjectService
	proxy                 bunder.Proxy
	subscriptionManager   *services.SubscriptionManager
}

// NewKVHandler creates a KVHandler with a Proxy (HTTP or RPC). subscriptionManager may be nil (no SSE subscribe).
func NewKVHandler(projectService *services.ProjectService, proxy bunder.Proxy, subscriptionManager *services.SubscriptionManager) *KVHandler {
	return &KVHandler{
		projectService:     projectService,
		proxy:              proxy,
		subscriptionManager: subscriptionManager,
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

// HandleKVSubscribe streams KV change events over SSE (key-scoped or user-scoped).
func (h *KVHandler) HandleKVSubscribe(c *gin.Context) {
	projectID := middleware.GetProjectID(c)
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	if h.subscriptionManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "KV realtime subscriptions not available"})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
		return
	}

	events, cancel := h.subscriptionManager.SubscribeKV(c.Request.Context(), projectID)
	defer cancel()

	if _, err := c.Writer.Write([]byte("event: connected\n")); err != nil {
		return
	}
	if _, err := c.Writer.Write([]byte(fmt.Sprintf("data: {\"projectId\":\"%s\"}\n\n", projectID))); err != nil {
		return
	}
	flusher.Flush()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			if _, err := c.Writer.Write([]byte("event: change\n")); err != nil {
				return
			}
			if _, err := c.Writer.Write([]byte("data: " + string(data) + "\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

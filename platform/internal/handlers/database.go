package handlers

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/bundoc"
	"github.com/kartikbazzad/bunbase/platform/internal/middleware"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

type DatabaseHandler struct {
	bundoc         *bundoc.Client
	projectService *services.ProjectService
}

func NewDatabaseHandler(bundoc *bundoc.Client, projectService *services.ProjectService) *DatabaseHandler {
	return &DatabaseHandler{
		bundoc:         bundoc,
		projectService: projectService,
	}
}

// Helper to resolve Project ID from API Key
func (h *DatabaseHandler) getProjectID(c *gin.Context) (string, error) {
	key := c.GetHeader("X-Bunbase-Client-Key")
	if key == "" {
		key = c.Query("key")
	}
	if key == "" {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer pk_") {
			key = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}
	if key == "" {
		return "", nil // Not found
	}
	return h.projectService.GetProjectIDByPublicKey(key)
}

// ProxyHandler handles all database operations by forwarding them to Bundoc.
func (h *DatabaseHandler) ProxyHandler(c *gin.Context) {
	projectID, err := h.getProjectID(c)
	if err != nil || projectID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing Public API Key"})
		return
	}

	// Read Body
	var body []byte
	if c.Request.Body != nil {
		body, _ = io.ReadAll(c.Request.Body)
	}

	// Construct path relative to project
	// Platform path: /v1/databases/...
	// We want to pass: /databases/...
	// c.FullPath() gives the registered path, typically /v1/databases/:dbName/...
	// but we want the actual values.

	// c.FullPath() gives the registered path, typically /v1/databases/:dbName/...
	// but we want the actual values.

	// INFO: Enforce dbName to be the ProjectID associated with the API Key for isolation.
	// The URL param :dbName is ignored/checked.
	dbName := projectID
	collection := c.Param("collection")
	docID := c.Param("docID")

	// Base path: /databases/{dbName}/documents/{collection}
	upstreamPath := "/databases/" + dbName + "/documents/" + collection
	if docID != "" {
		upstreamPath += "/" + docID
	} else if c.Request.Method == http.MethodGet {
		// List documents? Query params might be needed.
		// bundoc client ProxyRequest doesn't handle query params yet in the `path` argument logic
		// if we blindly append.
		// But let's check ProxyRequest implementation:
		// url := fmt.Sprintf("%s/v1/projects/%s%s", c.BaseURL, projectID, path)
		// So if path contains query string, it works?
	}

	// Handle Query String
	if c.Request.URL.RawQuery != "" {
		upstreamPath += "?" + c.Request.URL.RawQuery
	}

	status, respBody, err := h.bundoc.ProxyRequest(c.Request.Method, projectID, upstreamPath, body)
	if err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// Forward response
	c.Data(status, "application/json", respBody)
}

// DeveloperProxyHandler handles database operations for authenticated developers via Console.
func (h *DatabaseHandler) DeveloperProxyHandler(c *gin.Context) {
	// 1. Get Project ID from URL
	projectID := c.Param("id") // /projects/:id/database/...
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	// 2. Get User ID from Context (set by auth middleware)
	user, ok := middleware.RequireAuth(c)
	if !ok {
		return
	}

	// 3. Check Membership
	isMember, _, err := h.projectService.IsProjectMember(projectID, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check project membership"})
		return
	}
	if !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have access to this project"})
		return
	}

	// 4. Construct Upstream Path
	// /api/projects/:id/database/collections/:collection/documents/:docID
	// We want to forward to: /databases/{dbName}/documents/{collection}/{docID}
	// Bundoc uses "dbName" = projectID usually.
	dbName := projectID

	// Extract path after /database
	// c.FullPath() includes route pattern.
	// c.Request.URL.Path includes full path.
	// Let's assume the router group is /api/projects/:id/database
	// and we strip the prefix manually?
	// Or we use named params.

	// Router setup:
	// /collections -> List collections
	// /collections/:collection/documents -> List docs
	// /collections/:collection/documents/:docID -> Get doc

	// We'll trust the params.
	collection := c.Param("collection")
	docID := c.Param("docID")

	// Determine operation based on path params available
	var upstreamPath string

	if collection == "" {
		// List Collections (GET /collections) or Create Collection (POST /collections)
		upstreamPath = "/databases/" + dbName + "/collections"
	} else if docID == "" {
		// List Documents or Create Document
		upstreamPath = "/databases/" + dbName + "/documents/" + collection
	} else {
		// Document Operations
		upstreamPath = "/databases/" + dbName + "/documents/" + collection + "/" + docID
	}

	// Handle Query String
	if c.Request.URL.RawQuery != "" {
		upstreamPath += "?" + c.Request.URL.RawQuery
	}

	// Read Body
	var body []byte
	if c.Request.Body != nil {
		body, _ = io.ReadAll(c.Request.Body)
	}

	// 5. Proxy Request
	status, respBody, err := h.bundoc.ProxyRequest(c.Request.Method, projectID, upstreamPath, body)
	if err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.Data(status, "application/json", respBody)
}

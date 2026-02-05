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
	bundoc             *bundoc.Client
	projectService     *services.ProjectService
	subscriptionManager *services.SubscriptionManager
}

func NewDatabaseHandler(bundoc *bundoc.Client, projectService *services.ProjectService, subscriptionManager *services.SubscriptionManager) *DatabaseHandler {
	return &DatabaseHandler{
		bundoc:             bundoc,
		projectService:     projectService,
		subscriptionManager: subscriptionManager,
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

// ProxyHandler handles database operations for project API key (SDK / external clients).
// Uses /databases/{projectID}/documents/{collection}/{docID} — do not change; bundoc and API-key DB depend on this shape.
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

	dbName := projectID // Enforce project isolation; :dbName param ignored
	collection := c.Param("collection")
	docID := c.Param("docID")

	// API-key path: /databases/{projectID}/documents/{collection}[/{docID}]. Do not use BundocDBPath here.
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

// DeveloperProxyHandler handles database operations for the dashboard (projects/:id/database/...).
// Uses bundoc.BundocDBPath; tenant-auth talks to bundoc directly and is unaffected.
func (h *DatabaseHandler) DeveloperProxyHandler(c *gin.Context) {
	// 1. Get Project ID from URL
	projectID := c.Param("id") // /projects/:id/database/...
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	// 2. Allow if authorized by project API key (key matched this project)
	if middleware.GetProjectKeyProjectID(c) == projectID {
		// Authorized by key; continue to proxy
	} else {
		// 3. Otherwise require user and membership
		user, ok := middleware.RequireAuth(c)
		if !ok {
			return
		}
		isMember, _, err := h.projectService.IsProjectMember(projectID, user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check project membership"})
			return
		}
		if !isMember {
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not have access to this project"})
			return
		}
	}

	// 4. Construct Upstream Path (suffix only - ProxyRequest prepends /v1/projects/{projectID})
	// Bundoc client: url = BaseURL + "/v1/projects/" + projectID + path
	// So path must be e.g. /databases/default/collections/users/documents (no /v1/projects/...)
	collection := c.Param("collection")
	docID := c.Param("docID")
	field := c.Param("field")

	var upstreamPath string
	fullPath := c.FullPath()

	// Path suffix only (client adds /v1/projects/{id}); see bundoc.BundocDBPath
	pathSuffix := bundoc.BundocDBPath

	if strings.Contains(fullPath, "/indexes") {
		upstreamPath = pathSuffix + "/indexes?collection=" + collection
		if field != "" {
			upstreamPath += "&field=" + field
		}
	} else if strings.HasSuffix(fullPath, "/query") {
		upstreamPath = pathSuffix + "/documents/query"
	} else if strings.Contains(fullPath, "/rules") {
		upstreamPath = pathSuffix + "/collections/" + collection + "/rules"
	} else if collection == "" {
		upstreamPath = pathSuffix + "/collections"
	} else if strings.Contains(fullPath, "/documents") {
		if docID != "" {
			// Get/Update/Delete: bundoc expects .../documents/{collection}/{docId} (parseProjectAndPathSuffix)
			upstreamPath = pathSuffix + "/documents/" + collection + "/" + docID
		} else {
			// List/Create: bundoc routes on .../collections/{collection}/documents
			upstreamPath = pathSuffix + "/collections/" + collection + "/documents"
		}
	} else {
		upstreamPath = pathSuffix + "/collections/" + collection
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

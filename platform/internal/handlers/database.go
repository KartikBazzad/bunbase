package handlers

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/authz"
	"github.com/kartikbazzad/bunbase/platform/internal/bundoc"
	"github.com/kartikbazzad/bunbase/platform/internal/middleware"
	"github.com/kartikbazzad/bunbase/platform/internal/services"
)

type DatabaseHandler struct {
	bundoc              bundoc.Proxy
	projectService      *services.ProjectService
	subscriptionManager *services.SubscriptionManager
	enforcer            *authz.Enforcer
}

func NewDatabaseHandler(bundoc bundoc.Proxy, projectService *services.ProjectService, subscriptionManager *services.SubscriptionManager, enforcer *authz.Enforcer) *DatabaseHandler {
	return &DatabaseHandler{
		bundoc:              bundoc,
		projectService:      projectService,
		subscriptionManager: subscriptionManager,
		enforcer:            enforcer,
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

// DeveloperProxyHandler handles database operations (key-scoped /v1/database/... or user-scoped /projects/:id/database/...).
// Uses bundoc.BundocDBPath. Project ID comes from context (key-scoped) or route param (user-scoped).
func (h *DatabaseHandler) DeveloperProxyHandler(c *gin.Context) {
	projectID := middleware.GetProjectID(c)
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	if middleware.GetProjectKeyProjectID(c) != "" {
		// Authorized by key; continue to proxy
	} else {
		user, ok := middleware.RequireAuth(c)
		if !ok {
			return
		}
		if h.enforcer != nil {
			allowed, err := h.enforcer.ProjectEnforce(user.ID.String(), projectID, "database", "read")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if !allowed {
				c.JSON(http.StatusForbidden, gin.H{"error": "You do not have access to this project"})
				return
			}
		} else {
			isMember, _, err := h.projectService.IsProjectMember(projectID, user.ID.String())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check project membership"})
				return
			}
			if !isMember {
				c.JSON(http.StatusForbidden, gin.H{"error": "You do not have access to this project"})
				return
			}
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

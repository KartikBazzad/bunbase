package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/auth"
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
	tenantClient        *auth.TenantClient
}

func NewDatabaseHandler(bundoc bundoc.Proxy, projectService *services.ProjectService, subscriptionManager *services.SubscriptionManager, enforcer *authz.Enforcer, tenantClient *auth.TenantClient) *DatabaseHandler {
	return &DatabaseHandler{
		bundoc:              bundoc,
		projectService:      projectService,
		subscriptionManager: subscriptionManager,
		enforcer:            enforcer,
		tenantClient:        tenantClient,
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

// getTenantUserFromCookie extracts tenant user information from the tenant_session_token cookie.
// Returns user ID and claims if cookie is valid, nil otherwise.
func (h *DatabaseHandler) getTenantUserFromCookie(c *gin.Context, projectID string) (string, map[string]interface{}) {
	if h.tenantClient == nil {
		return "", nil
	}

	token, err := c.Cookie("tenant_session_token")
	if err != nil || token == "" {
		return "", nil
	}

	verifyRes, err := h.tenantClient.Verify(token)
	if err != nil || !verifyRes.Valid {
		log.Printf("Failed to verify tenant session token: %v", err)
		return "", nil
	}

	// Verify project ID matches
	claimsProjectID, _ := verifyRes.Claims["project_id"].(string)
	if claimsProjectID != projectID {
		log.Printf("Tenant session project_id mismatch: expected %s, got %s", projectID, claimsProjectID)
		return "", nil
	}

	// Extract user ID from sub claim
	userID, _ := verifyRes.Claims["sub"].(string)
	if userID == "" {
		return "", nil
	}

	return userID, verifyRes.Claims
}

// formatBundocAuthHeader formats tenant user info for bundoc's X-Bundoc-Auth header.
// Format: "uid:claims_json"
func (h *DatabaseHandler) formatBundocAuthHeader(uid string, claims map[string]interface{}) string {
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		log.Printf("Failed to marshal claims: %v", err)
		return uid + ":{}"
	}
	return uid + ":" + string(claimsJSON)
}

// ProxyHandler handles database operations for project API key (SDK / external clients).
// Uses /databases/{projectID}/documents/{collection}/{docID} — do not change; bundoc and API-key DB depend on this shape.
// Validates API key, extracts tenant cookie if present, and forwards to bundoc for rule evaluation.
// Prevents admin access - never forwards X-Bunbase-Client-Key to bundoc.
func (h *DatabaseHandler) ProxyHandler(c *gin.Context) {
	projectID, err := h.getProjectID(c)
	if err != nil || projectID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing Public API Key"})
		return
	}

	// Extract tenant session cookie if present (optional - bundoc will evaluate rules)
	tenantUID, tenantClaims := h.getTenantUserFromCookie(c, projectID)

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

	// Format auth header with tenant user info if cookie is present
	// If no cookie, bundoc will receive unauthenticated context and evaluate rules accordingly
	headers := make(map[string]string)
	if tenantUID != "" && tenantClaims != nil {
		authHeader := h.formatBundocAuthHeader(tenantUID, tenantClaims)
		headers["X-Bundoc-Auth"] = authHeader
	}
	// Note: X-Bunbase-Client-Key is NOT forwarded to bundoc (prevents admin access)

	status, respBody, err := h.bundoc.ProxyRequest(c.Request.Method, projectID, upstreamPath, body, headers)
	if err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// Forward response
	c.Data(status, "application/json", respBody)
}

// DeveloperProxyHandler handles database operations (key-scoped /v1/database/... or user-scoped /projects/:id/database/...).
// Uses bundoc.BundocDBPath. Project ID comes from context (key-scoped) or route param (user-scoped).
// Validates API key, extracts tenant cookie if present, and forwards to bundoc for rule evaluation.
// Prevents admin access - never forwards X-Bunbase-Client-Key to bundoc.
func (h *DatabaseHandler) DeveloperProxyHandler(c *gin.Context) {
	projectID := middleware.GetProjectID(c)
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	// Differentiate between admin requests (platform web console) and SDK requests
	isAdminRequest := middleware.GetProjectKeyProjectID(c) == ""
	
	var headers map[string]string
	
	if isAdminRequest {
		// Admin request: Platform web console with session_token cookie
		// Verify platform user authentication and project access
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
		
		// Set admin header for bundoc (admin access)
		headers = make(map[string]string)
		headers["X-Bundoc-Admin"] = "true"
	} else {
		// SDK request: API key authenticated, extract tenant cookie if present
		tenantUID, tenantClaims := h.getTenantUserFromCookie(c, projectID)
		
		headers = make(map[string]string)
		if tenantUID != "" && tenantClaims != nil {
			// Pass tenant user context to bundoc
			authHeader := h.formatBundocAuthHeader(tenantUID, tenantClaims)
			headers["X-Bundoc-Auth"] = authHeader
		}
		// Note: X-Bunbase-Client-Key is NOT forwarded to bundoc (prevents SDK admin access)
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

	// Headers are already set above based on request type (admin vs SDK)

	// 5. Proxy Request
	status, respBody, err := h.bundoc.ProxyRequest(c.Request.Method, projectID, upstreamPath, body, headers)
	if err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.Data(status, "application/json", respBody)
}

package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kartikbazzad/bunbase/platform/internal/middleware"
)

// HandleCustomDomainInvoke routes function invocations coming from custom
// domains or generated function URLs. It expects Traefik to route all such
// requests to the Platform service.
//
// Supported host patterns:
//   - {slug}.bunbase.com -> resolve project by slug
//   - {projectID}.functions.bunbase.com -> resolve project by ID
//
// Path format:
//   - /{functionName}[/*extraPath] (we use the first segment as function name)
func (h *FunctionHandler) HandleCustomDomainInvoke(c *gin.Context) {
	host := c.Request.Host
	if host == "" {
		c.Next()
		return
	}

	// Strip port if present (e.g. localhost:8080)
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	var projectID string

	// Generated URL: {projectID}.functions.bunbase.com
	if strings.Contains(host, ".functions.bunbase.com") {
		projectID = strings.TrimSuffix(host, ".functions.bunbase.com")
	} else if strings.HasSuffix(host, ".bunbase.com") {
		// Subdomain: {slug}.bunbase.com
		slug := strings.TrimSuffix(host, ".bunbase.com")
		if slug != "" && h.projectService != nil {
			project, err := h.projectService.GetProjectBySlug(slug)
			if err == nil && project != nil {
				projectID = project.ID
			}
		}
	}

	// If we couldn't resolve a project, let other handlers try.
	if projectID == "" {
		c.Next()
		return
	}

	// Path is passed by Traefik as /_/invoke/{name}[/*]; param "path" = name[/rest]
	path := strings.TrimPrefix(c.Param("path"), "/")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Function name required"})
		return
	}
	segments := strings.SplitN(path, "/", 2)
	functionName := segments[0]
	if functionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Function name required"})
		return
	}

	// Ensure SDK-style project key checks still work if present.
	if middleware.GetProjectKeyProjectID(c) != "" &&
		middleware.GetProjectKeyProjectID(c) != projectID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	// Reuse existing invocation logic via internal helper.
	h.invokeForProjectAndFunction(c, projectID, functionName)
}

// invokeForProjectAndFunction is a small helper that reuses the existing
// InvokeProjectFunction logic but allows us to call it directly with
// a known projectID + functionName (e.g. from custom domain routing).
func (h *FunctionHandler) invokeForProjectAndFunction(c *gin.Context, projectID, functionName string) {
	if functionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Function name required"})
		return
	}

	// Lookup function by name and project
	function, err := h.functionService.GetFunctionByName(projectID, functionName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Function not found"})
		return
	}

	// Load project for context injection
	project, err := h.projectService.GetProjectByID(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load project"})
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

package http

import (
	"net/http"
	"strings"

	"github.com/kartikbazzad/bunbase/bunder-manager/internal/manager"
)

// ProxyHandler is an HTTP handler that routes /kv/{project_id}/... to the project's embedded Bunder instance.
type ProxyHandler struct {
	manager *manager.InstanceManager
}

// NewProxyHandler creates a new ProxyHandler.
func NewProxyHandler(m *manager.InstanceManager) *ProxyHandler {
	return &ProxyHandler{
		manager: m,
	}
}

// ServeHTTP parses /kv/{project_id}/... and routes to the embedded Bunder instance for that project.
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	// Expect: kv/{project_id}/... (rest is backend path)
	if len(parts) < 3 || parts[0] != "kv" {
		http.NotFound(w, r)
		return
	}
	projectID := parts[1]

	// Basic validation for projectID
	if projectID == "" || projectID == "." || projectID == ".." {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	// Acquire the embedded Bunder store for this project
	kvStore, release, err := h.manager.Acquire(projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer release()

	// Create HTTP handler wrapper for the Bunder store
	kvHandler := NewKVHandler(kvStore)

	// Rewrite the request path: remove /kv/{project_id} prefix
	// Original: /kv/{project_id}/kv/{key} -> /kv/{key}
	// Original: /kv/{project_id}/keys -> /keys
	// Original: /kv/{project_id}/health -> /health
	targetPath := "/" + strings.Join(parts[2:], "/")
	r.URL.Path = targetPath

	// Serve the request using the embedded Bunder HTTP handler
	kvHandler.ServeHTTP(w, r)
}

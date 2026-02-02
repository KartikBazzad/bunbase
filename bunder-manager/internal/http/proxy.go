package http

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/kartikbazzad/bunbase/bunder-manager/internal/manager"
)

// ProxyHandler is an HTTP handler that routes /kv/{project_id}/... to the project's Bunder instance.
type ProxyHandler struct {
	manager   *manager.InstanceManager
	transport *http.Transport
}

// NewProxyHandler creates a new ProxyHandler.
func NewProxyHandler(m *manager.InstanceManager) *ProxyHandler {
	return &ProxyHandler{
		manager: m,
		transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

// ServeHTTP parses /kv/{project_id}/... and proxies to the Bunder instance for that project.
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	// Expect: kv/{project_id}/... (rest is backend path)
	if len(parts) < 3 || parts[0] != "kv" {
		http.NotFound(w, r)
		return
	}
	projectID := parts[1]

	// Basic validation for projectID to prevent directory traversal attacks if used in file paths somewhere
	// (Though manager handles path building safely usually, good practice here too)
	// For now, manager.Acquire already handles it, but let's be safe.
	if projectID == "" || projectID == "." || projectID == ".." {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	baseURL, release, err := h.manager.Acquire(projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer release()

	// Create reverse proxy
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// Rewrite the URL path: remove /kv/{project_id} prefix
			// baseURL is like http://127.0.0.1:9000
			// req.URL.Path is like /kv/proj1/key/val -> /key/val

			targetPath := "/" + strings.Join(parts[2:], "/")
			req.URL.Scheme = "http"
			req.URL.Host = baseURL[7:] // Strip http://
			req.URL.Path = targetPath
			if _, ok := req.Header["User-Agent"]; !ok {
				// explicitly disable User-Agent so it's not set to default value
				req.Header.Set("User-Agent", "")
			}
		},
		Transport: h.transport,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if err != nil {
				http.Error(w, fmt.Sprintf("Proxy error: %v", err), http.StatusBadGateway)
			}
		},
	}

	proxy.ServeHTTP(w, r)
}

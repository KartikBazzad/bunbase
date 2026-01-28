package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kartikbazzad/bunbase/functions/internal/config"
	"github.com/kartikbazzad/bunbase/functions/internal/logger"
	"github.com/kartikbazzad/bunbase/functions/internal/router"
	"github.com/kartikbazzad/bunbase/functions/internal/scheduler"
)

// Gateway provides HTTP endpoints for function invocations
type Gateway struct {
	router    *router.Router
	scheduler *scheduler.Scheduler
	logger    *logger.Logger
	server    *http.Server
}

// NewGateway creates a new HTTP gateway
func NewGateway(r *router.Router, s *scheduler.Scheduler, cfg *config.GatewayConfig, log *logger.Logger) *Gateway {
	g := &Gateway{
		router:    r,
		scheduler: s,
		logger:    log,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/functions/", g.handleInvoke)
	mux.HandleFunc("/health", g.handleHealth)

	g.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: mux,
	}

	return g
}

// Start starts the HTTP server
func (g *Gateway) Start() error {
	g.logger.Info("Starting HTTP gateway on %s", g.server.Addr)
	return g.server.ListenAndServe()
}

// Stop stops the HTTP server
func (g *Gateway) Stop() error {
	if g.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Shutdown gracefully
	err := g.server.Shutdown(ctx)
	if err != nil {
		// If shutdown times out, force close
		if err == context.DeadlineExceeded {
			g.logger.Warn("Gateway shutdown timeout, forcing close")
			closeErr := g.server.Close()
			// Ignore "Server closed" error (it's expected)
			if closeErr != nil && closeErr.Error() != "http: Server closed" {
				return closeErr
			}
			return nil
		}
		// Ignore "Server closed" error (it's expected when server wasn't started)
		if err.Error() == "http: Server closed" {
			return nil
		}
		return err
	}
	return nil
}

// handleHealth handles health check requests
func (g *Gateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleInvoke handles function invocation requests
func (g *Gateway) handleInvoke(w http.ResponseWriter, r *http.Request) {
	// Extract function name from path
	// Path format: /functions/:name
	path := r.URL.Path
	if len(path) < 11 { // "/functions/" is 11 chars
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	functionName := path[11:] // Skip "/functions/"
	if functionName == "" {
		http.Error(w, "Function name required", http.StatusBadRequest)
		return
	}

	// Route to function
	fn, _, err := g.router.Route(functionName)
	if err != nil {
		if err == router.ErrFunctionNotFound {
			http.Error(w, "Function not found", http.StatusNotFound)
			return
		}
		if err == router.ErrNotDeployed {
			http.Error(w, "Function not deployed", http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("Routing error: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse request
	req, err := g.parseRequest(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse request: %v", err), http.StatusBadRequest)
		return
	}

	// Set deadline (default 30 seconds)
	deadlineMS := int64(30000)
	if req.DeadlineMS > 0 {
		deadlineMS = req.DeadlineMS
	}

	// Update request with the deadline so it gets passed to the worker
	req.DeadlineMS = deadlineMS

	// Create context with deadline
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(deadlineMS)*time.Millisecond)
	defer cancel()

	// Log invocation attempt
	g.logger.Debug("Invoking function %s (method: %s, path: %s)", fn.ID, req.Method, req.Path)

	// Schedule invocation
	result, err := g.scheduler.Schedule(ctx, fn.ID, req)
	if err != nil {
		g.logger.Error("Invocation failed for function %s: %v", fn.ID, err)
		http.Error(w, fmt.Sprintf("Invocation failed: %v", err), http.StatusInternalServerError)
		return
	}

	g.logger.Debug("Invocation completed for function %s (success: %v, duration: %v)", fn.ID, result.Success, result.ExecutionTime)

	// Write response
	if !result.Success {
		http.Error(w, result.Error, http.StatusInternalServerError)
		return
	}

	// Set response headers
	for k, v := range result.Headers {
		w.Header().Set(k, v)
	}
	w.WriteHeader(result.Status)

	// Write response body
	if len(result.Body) > 0 {
		w.Write(result.Body)
	}
}

// parseRequest parses an HTTP request into an InvokeRequest
func (g *Gateway) parseRequest(r *http.Request) (*scheduler.InvokeRequest, error) {
	// Parse query parameters
	query := make(map[string]string)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}

	// Parse headers
	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			// Skip certain headers
			if k == "Host" || k == "Content-Length" {
				continue
			}
			headers[k] = v[0]
		}
	}

	// Read body
	var body []byte
	if r.Body != nil {
		var err error
		body, err = io.ReadAll(r.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read body: %w", err)
		}
		r.Body.Close()
	}

	return &scheduler.InvokeRequest{
		Method:     r.Method,
		Path:       r.URL.Path,
		Headers:    headers,
		Query:      query,
		Body:       body,
		DeadlineMS: 0, // Will be set by gateway
	}, nil
}

// handleLogs handles log retrieval requests (future)
func (g *Gateway) handleLogs(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Not implemented"})
}

// handleMetrics handles metrics retrieval requests (future)
func (g *Gateway) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Not implemented"})
}

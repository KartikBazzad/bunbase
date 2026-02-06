package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kartikbazzad/bunbase/functions/internal/capabilities"
	"github.com/kartikbazzad/bunbase/functions/internal/config"
	"github.com/kartikbazzad/bunbase/functions/internal/logstore"
	"github.com/kartikbazzad/bunbase/functions/internal/logger"
	"github.com/kartikbazzad/bunbase/functions/internal/metadata"
	"github.com/kartikbazzad/bunbase/functions/internal/pool"
	"github.com/kartikbazzad/bunbase/functions/internal/prometrics"
	"github.com/kartikbazzad/bunbase/functions/internal/router"
	"github.com/kartikbazzad/bunbase/functions/internal/scheduler"
)

// Gateway provides HTTP endpoints for function invocations and management
type Gateway struct {
	router       *router.Router
	scheduler    *scheduler.Scheduler
	logger       *logger.Logger
	metadata     *metadata.Store
	cfg          *config.Config
	workerScript string
	initScript   string
	server       *http.Server
	logStore     logstore.Store
}

// NewGateway creates a new HTTP gateway
func NewGateway(r *router.Router, s *scheduler.Scheduler, meta *metadata.Store, cfg *config.Config, workerScript, initScript string, log *logger.Logger) *Gateway {
	g := &Gateway{
		router:       r,
		scheduler:    s,
		metadata:     meta,
		cfg:          cfg,
		workerScript: workerScript,
		initScript:   initScript,
		logger:       log,
	}
	lokiURL := ""
	if cfg != nil && cfg.Logs.LokiURL != "" {
		lokiURL = cfg.Logs.LokiURL
	}
	if lokiURL == "" {
		lokiURL = os.Getenv("LOKI_URL")
	}
	if lokiURL != "" {
		g.logStore = logstore.NewLokiStore(lokiURL)
	} else {
		g.logStore = &logstore.NoopStore{}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/functions/", g.handleFunctions)
	mux.HandleFunc("/v1/functions/register", g.handleRegister)
	mux.HandleFunc("/v1/functions/deploy", g.handleDeploy)
	mux.HandleFunc("/health", g.handleHealth)
	mux.Handle("/metrics", prometrics.Handler())

	g.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Gateway.HTTPPort),
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

// handleLogs handles GET /functions/:id/logs
func (g *Gateway) handleLogs(w http.ResponseWriter, r *http.Request, functionNameOrID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fn, _, err := g.router.Route(functionNameOrID)
	if err != nil {
		if err == router.ErrFunctionNotFound {
			http.Error(w, "Function not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Routing error: %v", err), http.StatusInternalServerError)
		return
	}
	since := time.Now().Add(-24 * time.Hour)
	if s := r.URL.Query().Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			since = t
		}
	}
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := fmt.Sscanf(l, "%d", &limit); n == 1 && err == nil && limit > 0 {
			if limit > 1000 {
				limit = 1000
			}
		}
	}
	entries, err := g.logStore.GetLogs(fn.ID, since, limit)
	if err != nil {
		g.logger.Error("GetLogs failed for %s: %v", fn.ID, err)
		http.Error(w, fmt.Sprintf("Failed to get logs: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

// handleFunctions routes /functions/... to either logs or invoke
func (g *Gateway) handleFunctions(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if len(path) < 11 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	suffix := path[11:]
	if strings.HasSuffix(suffix, "/logs") {
		// GET /functions/:id/logs
		funcPart := strings.TrimSuffix(suffix, "/logs")
		funcPart = strings.TrimSuffix(funcPart, "/")
		if funcPart == "" {
			http.Error(w, "Function name required", http.StatusBadRequest)
			return
		}
		g.handleLogs(w, r, funcPart)
		return
	}
	g.handleInvoke(w, r)
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

	// Check if pool is missing (lazy load)
	pool, err := g.router.GetPool(fn.ID)
	if err != nil || pool == nil {
		// Pool missing, try to create it
		g.logger.Info("Lazy loading pool for function %s", fn.ID)

		if fn.ActiveVersionID == "" {
			http.Error(w, "Function has no active version", http.StatusInternalServerError)
			return
		}

		version, err := g.metadata.GetVersionByID(fn.ActiveVersionID)
		if err != nil {
			g.logger.Error("Failed to get version %s: %v", fn.ActiveVersionID, err)
			http.Error(w, "Failed to load function version", http.StatusInternalServerError)
			return
		}

		if err := g.createPoolForFunction(fn, version); err != nil {
			g.logger.Error("Failed to create pool: %v", err)
			http.Error(w, "Failed to initialize function worker", http.StatusInternalServerError)
			return
		}
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

	// Extract optional project context headers (set by Platform)
	projectID := r.Header.Get("X-Bunbase-Project-ID")
	projectAPIKey := r.Header.Get("X-Bunbase-API-Key")
	gatewayURL := r.Header.Get("X-Bunbase-Gateway-URL")

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
		Method:        r.Method,
		Path:          r.URL.Path,
		Headers:       headers,
		Query:         query,
		Body:          body,
		DeadlineMS:    0, // Will be set by gateway
		ProjectID:     projectID,
		ProjectAPIKey: projectAPIKey,
		GatewayURL:    gatewayURL,
	}, nil
}

// RegisterFunctionRequest represents a function registration request
type RegisterFunctionRequest struct {
	Name    string `json:"name"`
	Runtime string `json:"runtime"`
	Handler string `json:"handler"`
}

// RegisterFunctionResponse represents a function registration response
type RegisterFunctionResponse struct {
	FunctionID string `json:"function_id"`
	Name       string `json:"name"`
	Runtime    string `json:"runtime"`
	Handler    string `json:"handler"`
	Status     string `json:"status"`
}

func (g *Gateway) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if g.metadata == nil {
		http.Error(w, "Metadata store not available", http.StatusInternalServerError)
		return
	}

	var req RegisterFunctionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	if req.Runtime == "" {
		req.Runtime = "bun" // Default runtime
	}

	if req.Handler == "" {
		req.Handler = "default" // Default handler
	}

	// Use name as function ID (already unique from platform)
	functionID := req.Name

	// Check if function already exists
	existingFn, err := g.metadata.GetFunctionByID(functionID)
	if err == nil && existingFn != nil {
		// Function already exists, return it
		resp := RegisterFunctionResponse{
			FunctionID: existingFn.ID,
			Name:       existingFn.Name,
			Runtime:    existingFn.Runtime,
			Handler:    existingFn.Handler,
			Status:     string(existingFn.Status),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Create default capabilities (strict profile)
	// Extract project ID from function ID if possible (func-{project-slug}-{name})
	projectID := ""
	if len(functionID) > 5 && functionID[:5] == "func-" {
		// Try to extract project slug
		parts := functionID[5:]
		// This is a simplified extraction - in production, you might want more robust parsing
		projectID = parts
	}
	caps := capabilities.DefaultProfile(projectID)

	// Register function
	fn, err := g.metadata.RegisterFunction(functionID, req.Name, req.Runtime, req.Handler, caps)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to register function: %v", err), http.StatusInternalServerError)
		return
	}

	resp := RegisterFunctionResponse{
		FunctionID: fn.ID,
		Name:       fn.Name,
		Runtime:    fn.Runtime,
		Handler:    fn.Handler,
		Status:     string(fn.Status),
	}

	g.logger.Info("Registered function: %s (runtime: %s)", fn.ID, fn.Runtime)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// DeployFunctionRequest represents a function deployment request
type DeployFunctionRequest struct {
	FunctionID string `json:"function_id"`
	Version    string `json:"version"`
	BundlePath string `json:"bundle_path"`
}

// DeployFunctionResponse represents a function deployment response
type DeployFunctionResponse struct {
	DeploymentID string `json:"deployment_id"`
	FunctionID   string `json:"function_id"`
	Version      string `json:"version"`
	Status       string `json:"status"`
}

func (g *Gateway) handleDeploy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if g.metadata == nil || g.cfg == nil {
		http.Error(w, "Dependencies not available", http.StatusInternalServerError)
		return
	}

	var req DeployFunctionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate input
	if req.FunctionID == "" {
		http.Error(w, "function_id is required", http.StatusBadRequest)
		return
	}

	if req.Version == "" {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}

	if req.BundlePath == "" {
		http.Error(w, "bundle_path is required", http.StatusBadRequest)
		return
	}

	// Check if bundle file exists
	if _, err := os.Stat(req.BundlePath); err != nil {
		http.Error(w, fmt.Sprintf("Bundle file not found: %v", err), http.StatusBadRequest)
		return
	}

	// Get function
	fn, err := g.metadata.GetFunctionByID(req.FunctionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Function not found: %v", err), http.StatusNotFound) // Warning: typo in statusNotFound, let's fix it manually in thought or careful edit
		return
	}

	// Check if version already exists
	versions, err := g.metadata.GetVersionsByFunctionID(req.FunctionID)
	if err == nil {
		for _, v := range versions {
			if v.Version == req.Version {
				// Version exists, use it
				versionID := v.ID
				deploymentID := uuid.New().String()

				// Deploy the existing version
				if err := g.metadata.DeployFunction(deploymentID, req.FunctionID, versionID); err != nil {
					http.Error(w, fmt.Sprintf("Failed to deploy: %v", err), http.StatusInternalServerError)
					return
				}

				// Create/update worker pool
				if err := g.createPoolForFunction(fn, v); err != nil {
					g.logger.Warn("Failed to create pool for function %s: %v", fn.ID, err)
					// Don't fail deployment if pool creation fails
				}

				resp := DeployFunctionResponse{
					DeploymentID: deploymentID,
					FunctionID:   req.FunctionID,
					Version:      req.Version,
					Status:       "deployed",
				}

				g.logger.Info("Deployed function %s version %s", req.FunctionID, req.Version)

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
				return
			}
		}
	}

	// Create new version
	versionID := uuid.New().String()
	version, err := g.metadata.CreateVersion(versionID, req.FunctionID, req.Version, req.BundlePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create version: %v", err), http.StatusInternalServerError)
		return
	}

	// Deploy version
	deploymentID := uuid.New().String()
	if err := g.metadata.DeployFunction(deploymentID, req.FunctionID, versionID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to deploy: %v", err), http.StatusInternalServerError)
		return
	}

	// Create/update worker pool
	if err := g.createPoolForFunction(fn, version); err != nil {
		g.logger.Warn("Failed to create pool for function %s: %v", fn.ID, err)
		// Don't fail deployment if pool creation fails
	}

	resp := DeployFunctionResponse{
		DeploymentID: deploymentID,
		FunctionID:   req.FunctionID,
		Version:      req.Version,
		Status:       "deployed",
	}

	g.logger.Info("Deployed function %s version %s", req.FunctionID, req.Version)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// createPoolForFunction creates or updates a worker pool for a function
func (g *Gateway) createPoolForFunction(fn *metadata.Function, version *metadata.FunctionVersion) error {
	if g.router == nil || g.cfg == nil {
		return fmt.Errorf("router or config not available")
	}

	// Get function capabilities
	caps := fn.Capabilities
	if caps == nil {
		// Use default capabilities
		projectID := ""
		if len(fn.ID) > 5 && fn.ID[:5] == "func-" {
			projectID = fn.ID[5:]
		}
		caps = capabilities.DefaultProfile(projectID)
	}

	// Create pool configuration
	poolCfg := g.cfg.Worker
	poolCfg.Runtime = fn.Runtime
	poolCfg.Capabilities = caps

	// Determine worker script path based on runtime
	runtimeWorkerScript := g.workerScript
	if poolCfg.Runtime == "quickjs-ng" || poolCfg.Runtime == "quickjs" {
		// For QuickJS, worker script is not used
		runtimeWorkerScript = ""
	}

	// Check if pool already exists
	existingPool, err := g.router.GetPool(fn.ID)
	if err == nil && existingPool != nil {
		// Unregister existing pool
		g.router.UnregisterPool(fn.ID)
		// Stop pool gracefully
		existingPool.Stop()
	}

	// Create new pool
	p := pool.NewPool(
		fn.ID,
		version.Version,
		version.BundlePath,
		&poolCfg,
		runtimeWorkerScript,
		g.initScript,
		map[string]string{}, // TODO: Load env vars from database
		g.logger,
	)
	if g.logStore != nil {
		p.SetLogStore(g.logStore)
	}

	// Register pool
	g.router.RegisterPool(fn.ID, p)

	g.logger.Info("Created pool for function %s (version %s, runtime %s)", fn.ID, version.Version, fn.Runtime)

	return nil
}

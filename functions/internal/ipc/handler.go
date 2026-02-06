package ipc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/kartikbazzad/bunbase/functions/internal/capabilities"
	"github.com/kartikbazzad/bunbase/functions/internal/config"
	"github.com/kartikbazzad/bunbase/functions/internal/logstore"
	"github.com/kartikbazzad/bunbase/functions/internal/logger"
	"github.com/kartikbazzad/bunbase/functions/internal/metadata"
	"github.com/kartikbazzad/bunbase/functions/internal/pool"
	"github.com/kartikbazzad/bunbase/functions/internal/router"
	"github.com/kartikbazzad/bunbase/functions/internal/scheduler"
)

// Handler handles IPC requests
type Handler struct {
	router       *router.Router
	scheduler    *scheduler.Scheduler
	logger       *logger.Logger
	metadata     *metadata.Store
	cfg          *config.Config
	workerScript string
	initScript   string
	logStore     logstore.Store
}

// NewHandler creates a new IPC handler
func NewHandler(r *router.Router, s *scheduler.Scheduler, log *logger.Logger) *Handler {
	return &Handler{
		router:    r,
		scheduler: s,
		logger:    log,
	}
}

// SetDependencies sets additional dependencies needed for function registration/deployment
func (h *Handler) SetDependencies(meta *metadata.Store, cfg *config.Config, workerScript string, initScript string) {
	h.metadata = meta
	h.cfg = cfg
	h.workerScript = workerScript
	h.initScript = initScript
	lokiURL := ""
	if cfg != nil && cfg.Logs.LokiURL != "" {
		lokiURL = cfg.Logs.LokiURL
	}
	if lokiURL == "" {
		lokiURL = os.Getenv("LOKI_URL")
	}
	if lokiURL != "" {
		h.logStore = logstore.NewLokiStore(lokiURL)
	} else {
		h.logStore = &logstore.NoopStore{}
	}
}

// Handle handles an IPC request
func (h *Handler) Handle(frame *RequestFrame) *ResponseFrame {
	response := &ResponseFrame{
		RequestID: frame.RequestID,
		Status:    StatusOK,
	}

	switch frame.Command {
	case CmdInvoke:
		response = h.handleInvoke(frame)
	case CmdGetLogs:
		response = h.handleGetLogs(frame)
	case CmdGetMetrics:
		response = h.handleGetMetrics(frame)
	case CmdRegisterFunction:
		response = h.handleRegisterFunction(frame)
	case CmdDeployFunction:
		response = h.handleDeployFunction(frame)
	default:
		response.Status = StatusError
		response.Payload = []byte(fmt.Sprintf(`{"error":"unknown command: %d"}`, frame.Command))
	}

	return response
}

// InvokeRequest represents an invoke request payload
type InvokeRequestPayload struct {
	FunctionID string            `json:"function_id"`
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	Headers    map[string]string `json:"headers"`
	Query      map[string]string `json:"query"`
	Body       string            `json:"body"` // base64
}

// InvokeResponsePayload represents an invoke response payload
type InvokeResponsePayload struct {
	Success       bool              `json:"success"`
	Status        int               `json:"status,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	Body          string            `json:"body,omitempty"` // base64
	Error         string            `json:"error,omitempty"`
	ExecutionTime int64             `json:"execution_time_ms,omitempty"`
	ExecutionID   string            `json:"execution_id,omitempty"`
}

func (h *Handler) handleInvoke(frame *RequestFrame) *ResponseFrame {
	response := &ResponseFrame{
		RequestID: frame.RequestID,
		Status:    StatusOK,
	}

	var req InvokeRequestPayload
	if err := json.Unmarshal(frame.Payload, &req); err != nil {
		response.Status = StatusError
		response.Payload = []byte(fmt.Sprintf(`{"error":"invalid request: %v"}`, err))
		return response
	}

	// Route to function
	fn, _, err := h.router.Route(req.FunctionID)
	if err != nil {
		response.Status = StatusError
		response.Payload = []byte(fmt.Sprintf(`{"error":"%v"}`, err))
		return response
	}

	// Decode body (base64)
	body, _ := base64.StdEncoding.DecodeString(req.Body)
	invokeReq := &scheduler.InvokeRequest{
		Method:        req.Method,
		Path:          req.Path,
		Headers:       req.Headers,
		Query:         req.Query,
		Body:          body,
		DeadlineMS:    30000,
		ProjectID:     req.Headers["X-Bunbase-Project-ID"],
		ProjectAPIKey: req.Headers["X-Bunbase-API-Key"],
		GatewayURL:    req.Headers["X-Bunbase-Gateway-URL"],
	}

	// Schedule invocation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := h.scheduler.Schedule(ctx, fn.ID, invokeReq)
	if err != nil {
		response.Status = StatusError
		response.Payload = []byte(fmt.Sprintf(`{"error":"%v"}`, err))
		return response
	}

	// Build response
	respPayload := InvokeResponsePayload{
		Success:       result.Success,
		ExecutionTime: int64(result.ExecutionTime / time.Millisecond),
		ExecutionID:   fmt.Sprintf("exec-%d", time.Now().UnixNano()),
	}

	if result.Success {
		respPayload.Status = result.Status
		respPayload.Headers = result.Headers
		if len(result.Body) > 0 {
			respPayload.Body = base64.StdEncoding.EncodeToString(result.Body)
		}
	} else {
		respPayload.Error = result.Error
	}

	payloadJSON, _ := json.Marshal(respPayload)
	response.Payload = payloadJSON

	return response
}

func (h *Handler) handleGetLogs(frame *RequestFrame) *ResponseFrame {
	response := &ResponseFrame{
		RequestID: frame.RequestID,
		Status:    StatusOK,
	}
	var req struct {
		FunctionID string `json:"function_id"`
		Since      string `json:"since"`
		Limit      int    `json:"limit"`
	}
	if err := json.Unmarshal(frame.Payload, &req); err != nil {
		response.Status = StatusError
		response.Payload = []byte(fmt.Sprintf(`{"error":"invalid request: %v"}`, err))
		return response
	}
	if req.FunctionID == "" {
		response.Status = StatusError
		response.Payload = []byte(`{"error":"function_id required"}`)
		return response
	}
	if h.logStore == nil {
		response.Payload = []byte(`{"logs":[]}`)
		return response
	}
	since := time.Now().Add(-24 * time.Hour)
	if req.Since != "" {
		if t, err := time.Parse(time.RFC3339, req.Since); err == nil {
			since = t
		}
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	entries, err := h.logStore.GetLogs(req.FunctionID, since, limit)
	if err != nil {
		response.Status = StatusError
		response.Payload = []byte(fmt.Sprintf(`{"error":"%v"}`, err))
		return response
	}
	payload, _ := json.Marshal(map[string]interface{}{"logs": entries})
	response.Payload = payload
	return response
}

func (h *Handler) handleGetMetrics(frame *RequestFrame) *ResponseFrame {
	response := &ResponseFrame{
		RequestID: frame.RequestID,
		Status:    StatusError,
		Payload:   []byte(`{"error":"not implemented"}`),
	}
	return response
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

func (h *Handler) handleRegisterFunction(frame *RequestFrame) *ResponseFrame {
	response := &ResponseFrame{
		RequestID: frame.RequestID,
		Status:    StatusOK,
	}

	if h.metadata == nil {
		response.Status = StatusError
		response.Payload = []byte(`{"error":"metadata store not available"}`)
		return response
	}

	var req RegisterFunctionRequest
	if err := json.Unmarshal(frame.Payload, &req); err != nil {
		response.Status = StatusError
		response.Payload = []byte(fmt.Sprintf(`{"error":"invalid request: %v"}`, err))
		return response
	}

	// Validate input
	if req.Name == "" {
		response.Status = StatusError
		response.Payload = []byte(`{"error":"name is required"}`)
		return response
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
	existingFn, err := h.metadata.GetFunctionByID(functionID)
	if err == nil && existingFn != nil {
		// Function already exists, return it
		resp := RegisterFunctionResponse{
			FunctionID: existingFn.ID,
			Name:       existingFn.Name,
			Runtime:    existingFn.Runtime,
			Handler:    existingFn.Handler,
			Status:     string(existingFn.Status),
		}
		payloadJSON, _ := json.Marshal(resp)
		response.Payload = payloadJSON
		return response
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
	fn, err := h.metadata.RegisterFunction(functionID, req.Name, req.Runtime, req.Handler, caps)
	if err != nil {
		response.Status = StatusError
		response.Payload = []byte(fmt.Sprintf(`{"error":"failed to register function: %v"}`, err))
		return response
	}

	resp := RegisterFunctionResponse{
		FunctionID: fn.ID,
		Name:       fn.Name,
		Runtime:    fn.Runtime,
		Handler:    fn.Handler,
		Status:     string(fn.Status),
	}

	payloadJSON, _ := json.Marshal(resp)
	response.Payload = payloadJSON

	h.logger.Info("Registered function: %s (runtime: %s)", fn.ID, fn.Runtime)

	return response
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

func (h *Handler) handleDeployFunction(frame *RequestFrame) *ResponseFrame {
	response := &ResponseFrame{
		RequestID: frame.RequestID,
		Status:    StatusOK,
	}

	if h.metadata == nil || h.cfg == nil {
		response.Status = StatusError
		response.Payload = []byte(`{"error":"dependencies not available"}`)
		return response
	}

	var req DeployFunctionRequest
	if err := json.Unmarshal(frame.Payload, &req); err != nil {
		response.Status = StatusError
		response.Payload = []byte(fmt.Sprintf(`{"error":"invalid request: %v"}`, err))
		return response
	}

	// Validate input
	if req.FunctionID == "" {
		response.Status = StatusError
		response.Payload = []byte(`{"error":"function_id is required"}`)
		return response
	}

	if req.Version == "" {
		response.Status = StatusError
		response.Payload = []byte(`{"error":"version is required"}`)
		return response
	}

	if req.BundlePath == "" {
		response.Status = StatusError
		response.Payload = []byte(`{"error":"bundle_path is required"}`)
		return response
	}

	// Check if bundle file exists
	if _, err := os.Stat(req.BundlePath); err != nil {
		response.Status = StatusError
		response.Payload = []byte(fmt.Sprintf(`{"error":"bundle file not found: %v"}`, err))
		return response
	}

	// Get function
	fn, err := h.metadata.GetFunctionByID(req.FunctionID)
	if err != nil {
		response.Status = StatusError
		response.Payload = []byte(fmt.Sprintf(`{"error":"function not found: %v"}`, err))
		return response
	}

	// Check if version already exists
	versions, err := h.metadata.GetVersionsByFunctionID(req.FunctionID)
	if err == nil {
		for _, v := range versions {
			if v.Version == req.Version {
				// Version exists, use it
				versionID := v.ID
				deploymentID := uuid.New().String()

				// Deploy the existing version
				if err := h.metadata.DeployFunction(deploymentID, req.FunctionID, versionID); err != nil {
					response.Status = StatusError
					response.Payload = []byte(fmt.Sprintf(`{"error":"failed to deploy: %v"}`, err))
					return response
				}

				// Create/update worker pool
				if err := h.createPoolForFunction(fn, v); err != nil {
					h.logger.Warn("Failed to create pool for function %s: %v", fn.ID, err)
					// Don't fail deployment if pool creation fails
				}

				resp := DeployFunctionResponse{
					DeploymentID: deploymentID,
					FunctionID:   req.FunctionID,
					Version:      req.Version,
					Status:       "deployed",
				}

				payloadJSON, _ := json.Marshal(resp)
				response.Payload = payloadJSON

				h.logger.Info("Deployed function %s version %s", req.FunctionID, req.Version)
				return response
			}
		}
	}

	// Create new version
	versionID := uuid.New().String()
	version, err := h.metadata.CreateVersion(versionID, req.FunctionID, req.Version, req.BundlePath)
	if err != nil {
		response.Status = StatusError
		response.Payload = []byte(fmt.Sprintf(`{"error":"failed to create version: %v"}`, err))
		return response
	}

	// Deploy version
	deploymentID := uuid.New().String()
	if err := h.metadata.DeployFunction(deploymentID, req.FunctionID, versionID); err != nil {
		response.Status = StatusError
		response.Payload = []byte(fmt.Sprintf(`{"error":"failed to deploy: %v"}`, err))
		return response
	}

	// Create/update worker pool
	if err := h.createPoolForFunction(fn, version); err != nil {
		h.logger.Warn("Failed to create pool for function %s: %v", fn.ID, err)
		// Don't fail deployment if pool creation fails
	}

	resp := DeployFunctionResponse{
		DeploymentID: deploymentID,
		FunctionID:   req.FunctionID,
		Version:      req.Version,
		Status:       "deployed",
	}

	payloadJSON, _ := json.Marshal(resp)
	response.Payload = payloadJSON

	h.logger.Info("Deployed function %s version %s", req.FunctionID, req.Version)

	return response
}

// createPoolForFunction creates or updates a worker pool for a function
func (h *Handler) createPoolForFunction(fn *metadata.Function, version *metadata.FunctionVersion) error {
	if h.router == nil || h.cfg == nil {
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
	poolCfg := h.cfg.Worker
	poolCfg.Runtime = fn.Runtime
	poolCfg.Capabilities = caps

	// Determine worker script path based on runtime
	runtimeWorkerScript := h.workerScript
	if poolCfg.Runtime == "quickjs-ng" || poolCfg.Runtime == "quickjs" {
		// For QuickJS, worker script is not used
		runtimeWorkerScript = ""
	}

	// Check if pool already exists
	existingPool, err := h.router.GetPool(fn.ID)
	if err == nil && existingPool != nil {
		// Unregister existing pool
		h.router.UnregisterPool(fn.ID)
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
		h.initScript,
		map[string]string{}, // TODO: Load env vars from database
		h.logger,
	)
	if h.logStore != nil {
		p.SetLogStore(h.logStore)
	}

	// Register pool
	h.router.RegisterPool(fn.ID, p)

	h.logger.Info("Created pool for function %s (version %s, runtime %s)", fn.ID, version.Version, fn.Runtime)

	return nil
}

package functions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client communicates with the functions service via HTTP
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new functions service client
func NewClient(baseURL string) (*Client, error) {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Close closes the client (no-op for HTTP client)
func (c *Client) Close() error {
	return nil
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

// RegisterFunction registers a function in the functions service
func (c *Client) RegisterFunction(name, runtime, handler string) (*RegisterFunctionResponse, error) {
	reqBody := RegisterFunctionRequest{
		Name:    name,
		Runtime: runtime,
		Handler: handler,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/v1/functions/register", "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("functions service error (status %d): %s", resp.StatusCode, string(body))
	}

	var result RegisterFunctionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// DeployFunction deploys a function version
func (c *Client) DeployFunction(functionID, version, bundlePath string) (*DeployFunctionResponse, error) {
	reqBody := DeployFunctionRequest{
		FunctionID: functionID,
		Version:    version,
		BundlePath: bundlePath,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/v1/functions/deploy", "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("functions service error (status %d): %s", resp.StatusCode, string(body))
	}

	var result DeployFunctionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// LogEntry represents a single function log line from the functions service.
type LogEntry struct {
	FunctionID   string    `json:"function_id"`
	InvocationID string    `json:"invocation_id"`
	Level        string    `json:"level"`
	Message      string    `json:"message"`
	CreatedAt    time.Time `json:"created_at"`
}

// GetLogs fetches logs for a function from the functions service.
func (c *Client) GetLogs(functionServiceID string, since *time.Time, limit int) ([]LogEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	sinceVal := time.Now().Add(-24 * time.Hour)
	if since != nil {
		sinceVal = *since
	}
	u := fmt.Sprintf("%s/functions/%s/logs?limit=%d&since=%s",
		c.baseURL, functionServiceID, limit, sinceVal.UTC().Format(time.RFC3339))
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("functions service error (status %d): %s", resp.StatusCode, string(body))
	}
	var entries []LogEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return entries, nil
}

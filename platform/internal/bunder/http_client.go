package bunder

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Ensure Client implements Proxy.
var _ Proxy = (*Client)(nil)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ProxyRequest forwards a request to the Bunder backend.
// path must be the suffix after /kv/{projectID}, e.g. /kv/{key} or /keys or /health.
func (c *Client) ProxyRequest(method, projectID, path string, body []byte) (int, []byte, error) {
	// Normalize path (ensure leading slash)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	url := fmt.Sprintf("%s/kv/%s%s", c.BaseURL, projectID, path)

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to create request: %w", err)
	}

	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/octet-stream")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return http.StatusBadGateway, nil, fmt.Errorf("failed to call bunder-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to read response: %w", err)
	}

	return resp.StatusCode, respBody, nil
}

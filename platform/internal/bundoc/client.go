package bundoc

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

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

// ProxyRequest forwards a request to the Bundoc backend.
// pathComponent: /databases/{db}/documents/{coll}
func (c *Client) ProxyRequest(method, projectID, path string, body []byte) (int, []byte, error) {
	// Construct backend URL: /v1/projects/{projectID}{path}
	// e.g. /v1/projects/PROJ_ID/databases/default/documents/users

	// Normalize path (ensure leading slash)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	url := fmt.Sprintf("%s/v1/projects/%s%s", c.BaseURL, projectID, path)

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return http.StatusBadGateway, nil, fmt.Errorf("failed to call bundoc: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to read response: %w", err)
	}

	return resp.StatusCode, respBody, nil
}

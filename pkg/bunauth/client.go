package bunauth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kartikbazzad/bunbase/pkg/logger"
)

// Client handles communication with the Auth service
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new BunAuth client
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// AuthResponse matches the service struct
type AuthResponse struct {
	AccessToken string `json:"access_token"`
	UserID      string `json:"user_id"`
}

type VerifyResponse struct {
	Valid     bool   `json:"valid"`
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// Login calls POST /login
func (c *Client) Login(email, password string) (*AuthResponse, error) {
	reqBody := map[string]string{
		"email":    email,
		"password": password,
	}
	var resp AuthResponse
	if err := c.doRequest("POST", "/login", reqBody, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Register calls POST /register
func (c *Client) Register(email, password, name string) (*AuthResponse, error) {
	reqBody := map[string]string{
		"email":    email,
		"password": password,
		"name":     name,
	}
	var resp AuthResponse
	if err := c.doRequest("POST", "/register", reqBody, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Verify calls GET /verify via Auth Header
func (c *Client) Verify(token string) (*VerifyResponse, error) {
	// Manually construct request for GET with header
	url := c.BaseURL + "/verify"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	r, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d", r.StatusCode)
	}

	var resp VerifyResponse
	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decoding error: %w", err)
	}

	return &resp, nil
}

func (c *Client) doRequest(method, path string, body interface{}, result interface{}) error {
	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequest(method, c.BaseURL+path, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		// Try to read error message?
		logger.Error("Auth request failed", "status", resp.StatusCode, "path", path)
		return fmt.Errorf("api error: %d", resp.StatusCode)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

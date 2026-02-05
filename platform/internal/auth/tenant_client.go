package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TenantClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewTenantClient(url string) *TenantClient {
	return &TenantClient{
		baseURL:    url,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type User struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	ProjectID string `json:"project_id"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at,omitempty"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

func (c *TenantClient) Register(projectID, email, password string) (*User, error) {
	body := map[string]string{
		"project_id": projectID,
		"email":      email,
		"password":   password,
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := c.httpClient.Post(c.baseURL+"/register", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to register: status %d", resp.StatusCode)
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *TenantClient) Login(projectID, email, password string) (*LoginResponse, error) {
	body := map[string]string{
		"project_id": projectID,
		"email":      email,
		"password":   password,
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := c.httpClient.Post(c.baseURL+"/login", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to login: status %d", resp.StatusCode)
	}

	var res LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

type AuthConfig struct {
	ID        string                 `json:"_id,omitempty"`
	Providers map[string]interface{} `json:"providers"`
	RateLimit map[string]interface{} `json:"rate_limit"`
}

func (c *TenantClient) ListUsers(projectID string) ([]User, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/projects/%s/users", c.baseURL, projectID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list users: status %d", resp.StatusCode)
	}

	var result struct {
		Users []User `json:"users"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Users, nil
}

func (c *TenantClient) GetConfig(projectID string) (*AuthConfig, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/projects/%s/config", c.baseURL, projectID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get config: status %d", resp.StatusCode)
	}

	var config AuthConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *TenantClient) UpdateConfig(projectID string, config *AuthConfig) error {
	jsonBody, _ := json.Marshal(config)
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/projects/%s/config", c.baseURL, projectID), bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update config: status %d", resp.StatusCode)
	}
	return nil
}

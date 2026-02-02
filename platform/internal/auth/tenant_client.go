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
	ProjectID string `json:"project_id"`
	Email     string `json:"email"`
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

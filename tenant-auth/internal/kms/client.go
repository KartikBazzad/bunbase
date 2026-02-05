package kms

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Client calls Bun-KMS to store and retrieve secrets.
// If BaseURL is empty, all operations are no-ops (caller should skip KMS).
type Client struct {
	BaseURL    string
	Token      string // optional JWT for Authorization: Bearer
	httpClient *http.Client
}

// NewClient creates a KMS client. If baseURL is empty, PutSecret and GetSecret
// will return ErrKMSDisabled; the handler can treat this as "KMS not configured".
func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ErrKMSDisabled is returned when BaseURL is not set.
var ErrKMSDisabled = fmt.Errorf("KMS not configured")

// SecretNameForProvider returns the KMS secret name for a provider's client_secret.
// Format: tenant_auth.projects.<projectID>.providers.<providerKey>.client_secret
// KMS allows [a-zA-Z0-9][a-zA-Z0-9._-]*; projectID UUID with hyphens is allowed.
func SecretNameForProvider(projectID, providerKey string) string {
	return fmt.Sprintf("tenant_auth.projects.%s.providers.%s.client_secret", projectID, providerKey)
}

// PutSecret stores a secret in KMS. Name must match KMS secret name rules.
func (c *Client) PutSecret(name, value string) error {
	if c.BaseURL == "" {
		return ErrKMSDisabled
	}
	body := map[string]string{"name": name, "value": value}
	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/v1/secrets", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KMS put secret: status %d", resp.StatusCode)
	}
	return nil
}

// GetSecret retrieves a secret from KMS by name.
func (c *Client) GetSecret(name string) (string, error) {
	if c.BaseURL == "" {
		return "", ErrKMSDisabled
	}
	path := "/v1/secrets/" + url.PathEscape(name)
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return "", err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("KMS get secret: status %d", resp.StatusCode)
	}
	var out struct {
		Value   string `json:"value"`
		ValueB64 string `json:"value_b64"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Value != "" {
		return out.Value, nil
	}
	if out.ValueB64 != "" {
		dec, err := base64.StdEncoding.DecodeString(out.ValueB64)
		if err != nil {
			return "", err
		}
		return string(dec), nil
	}
	return "", fmt.Errorf("KMS get secret: empty response")
}

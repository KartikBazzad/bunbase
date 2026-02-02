package loadtest

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client calls the BunKMS HTTP API.
type Client struct {
	baseURL string
	token   string
	hc      *http.Client
}

// NewClient creates a client for the given base URL (e.g. "http://localhost:8080").
func NewClient(baseURL string, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		hc:      &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) do(method, path string, body []byte) ([]byte, int, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, c.baseURL+path, r)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		return out, resp.StatusCode, fmt.Errorf("http %d: %s", resp.StatusCode, string(out))
	}
	return out, resp.StatusCode, nil
}

// CreateKey creates an AES key with the given name. Returns nil on 201 or 400 (already exists).
func (c *Client) CreateKey(name string) error {
	body, _ := json.Marshal(map[string]string{"name": name, "type": "aes-256"})
	_, code, err := c.do("POST", "/v1/keys", body)
	if err != nil {
		if code == 400 {
			return nil
		}
		return err
	}
	return nil
}

// Encrypt encrypts plaintext with the named key. Returns ciphertext base64 or error.
func (c *Client) Encrypt(keyName string, plaintext []byte) (ciphertextB64 string, err error) {
	body, _ := json.Marshal(map[string]string{
		"plaintext_b64": base64.StdEncoding.EncodeToString(plaintext),
	})
	out, _, err := c.do("POST", "/v1/encrypt/"+keyName, body)
	if err != nil {
		return "", err
	}
	var res struct {
		Ciphertext string `json:"ciphertext"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		return "", err
	}
	return res.Ciphertext, nil
}

// Decrypt decrypts ciphertext (base64) with the named key.
func (c *Client) Decrypt(keyName string, ciphertextB64 string) ([]byte, error) {
	body, _ := json.Marshal(map[string]string{"ciphertext": ciphertextB64})
	out, _, err := c.do("POST", "/v1/decrypt/"+keyName, body)
	if err != nil {
		return nil, err
	}
	var res struct {
		PlaintextB64 string `json:"plaintext_b64"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(res.PlaintextB64)
}

// PutSecret stores a secret.
func (c *Client) PutSecret(name string, value []byte) error {
	body, _ := json.Marshal(map[string]string{
		"name":      name,
		"value_b64": base64.StdEncoding.EncodeToString(value),
	})
	_, _, err := c.do("POST", "/v1/secrets", body)
	return err
}

// GetSecret retrieves a secret.
func (c *Client) GetSecret(name string) ([]byte, error) {
	out, _, err := c.do("GET", "/v1/secrets/"+name, nil)
	if err != nil {
		return nil, err
	}
	var res struct {
		ValueB64 string `json:"value_b64"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(res.ValueB64)
}

// GetKey fetches key metadata.
func (c *Client) GetKey(name string) error {
	_, _, err := c.do("GET", "/v1/keys/"+name, nil)
	return err
}

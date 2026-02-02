package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kartikbazzad/bunbase/bun-kms/internal/core"
)

func testServer(t *testing.T) (*Server, *core.Vault, *core.SecretStore) {
	t.Helper()
	logger := log.New(&bytes.Buffer{}, "", 0)
	masterKey, err := core.ParseMasterKey("base64:" + base64.StdEncoding.EncodeToString(core.TestMasterKey))
	if err != nil {
		t.Fatalf("ParseMasterKey: %v", err)
	}
	vault := core.NewVault()
	secrets, err := core.NewSecretStore(masterKey)
	if err != nil {
		t.Fatalf("NewSecretStore: %v", err)
	}
	return NewServer(vault, secrets, logger, nil), vault, secrets
}

func TestServer_NotFound(t *testing.T) {
	s, _, _ := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET / = %d, want 404", rec.Code)
	}
}

func TestServer_CreateKey_GetKey(t *testing.T) {
	s, _, _ := testServer(t)

	body := `{"name":"api-key","type":"aes-256"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/keys", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("POST /v1/keys = %d, want 201: %s", rec.Code, rec.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/v1/keys/api-key", nil)
	rec2 := httptest.NewRecorder()
	s.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("GET /v1/keys/api-key = %d, want 200", rec2.Code)
	}
	var out map[string]interface{}
	if err := json.NewDecoder(rec2.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["name"] != "api-key" {
		t.Errorf("name = %v, want api-key", out["name"])
	}
}

func TestServer_Encrypt_Decrypt(t *testing.T) {
	s, _, _ := testServer(t)
	s.vault.CreateKey("enc-key", core.KeyTypeAES256)

	encBody := `{"plaintext":"hello"}`
	encReq := httptest.NewRequest(http.MethodPost, "/v1/encrypt/enc-key", bytes.NewReader([]byte(encBody)))
	encReq.Header.Set("Content-Type", "application/json")
	encRec := httptest.NewRecorder()
	s.ServeHTTP(encRec, encReq)
	if encRec.Code != http.StatusOK {
		t.Errorf("POST encrypt = %d: %s", encRec.Code, encRec.Body.String())
	}
	var encResp map[string]interface{}
	if err := json.NewDecoder(encRec.Body).Decode(&encResp); err != nil {
		t.Fatalf("decode encrypt resp: %v", err)
	}
	ciphertext, _ := encResp["ciphertext"].(string)
	if ciphertext == "" {
		t.Fatal("ciphertext missing")
	}

	decBody := `{"ciphertext":"` + ciphertext + `"}`
	decReq := httptest.NewRequest(http.MethodPost, "/v1/decrypt/enc-key", bytes.NewReader([]byte(decBody)))
	decReq.Header.Set("Content-Type", "application/json")
	decRec := httptest.NewRecorder()
	s.ServeHTTP(decRec, decReq)
	if decRec.Code != http.StatusOK {
		t.Errorf("POST decrypt = %d: %s", decRec.Code, decRec.Body.String())
	}
	var decResp map[string]interface{}
	if err := json.NewDecoder(decRec.Body).Decode(&decResp); err != nil {
		t.Fatalf("decode decrypt resp: %v", err)
	}
	if decResp["plaintext"] != "hello" {
		t.Errorf("plaintext = %v, want hello", decResp["plaintext"])
	}
}

func TestServer_PutSecret_GetSecret(t *testing.T) {
	s, _, _ := testServer(t)

	putBody := `{"name":"db-password","value":"s3cr3t"}`
	putReq := httptest.NewRequest(http.MethodPost, "/v1/secrets", bytes.NewReader([]byte(putBody)))
	putReq.Header.Set("Content-Type", "application/json")
	putRec := httptest.NewRecorder()
	s.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusCreated {
		t.Errorf("POST /v1/secrets = %d: %s", putRec.Code, putRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/secrets/db-password", nil)
	getRec := httptest.NewRecorder()
	s.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Errorf("GET /v1/secrets/db-password = %d", getRec.Code)
	}
	var out map[string]interface{}
	if err := json.NewDecoder(getRec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["value"] != "s3cr3t" {
		t.Errorf("value = %v, want s3cr3t", out["value"])
	}
}

func TestServer_RotateKey(t *testing.T) {
	s, _, _ := testServer(t)
	s.vault.CreateKey("rot-key", core.KeyTypeAES256)

	req := httptest.NewRequest(http.MethodPost, "/v1/keys/rot-key/rotate", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("POST rotate = %d: %s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["latest_version"].(float64) != 2 {
		t.Errorf("latest_version = %v, want 2", out["latest_version"])
	}
}

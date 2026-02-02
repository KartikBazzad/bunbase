package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/kartikbazzad/bunbase/bun-kms/internal/auth"
	"github.com/kartikbazzad/bunbase/bun-kms/internal/core"
)

type Server struct {
	vault     *core.Vault
	secrets   *core.SecretStore
	logger    *log.Logger
	jwtSecret []byte
}

// NewServer creates an API server. jwtSecret is optional; when non-empty, /v1/* requires Bearer JWT.
func NewServer(vault *core.Vault, secrets *core.SecretStore, logger *log.Logger, jwtSecret []byte) *Server {
	return &Server{
		vault:     vault,
		secrets:   secrets,
		logger:    logger,
		jwtSecret: jwtSecret,
	}
}

// Handler returns the HTTP handler, wrapped with JWT auth when jwtSecret is set.
func (s *Server) Handler() http.Handler {
	h := http.HandlerFunc(s.ServeHTTP)
	return auth.Middleware(s.jwtSecret)(h)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path := strings.Trim(r.URL.Path, "/")
	parts := splitPath(path)
	if len(parts) == 0 || parts[0] != "v1" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	switch {
	case len(parts) == 2 && parts[1] == "keys" && r.Method == http.MethodPost:
		s.handleCreateKey(w, r)
	case len(parts) == 3 && parts[1] == "keys" && r.Method == http.MethodGet:
		s.handleGetKey(w, r, parts[2])
	case len(parts) == 4 && parts[1] == "keys" && parts[3] == "rotate" && r.Method == http.MethodPost:
		s.handleRotateKey(w, r, parts[2])
	case len(parts) == 4 && parts[1] == "keys" && parts[3] == "revoke" && r.Method == http.MethodPost:
		s.handleRevokeKey(w, r, parts[2])
	case len(parts) == 4 && parts[1] == "keys" && parts[3] == "sign" && r.Method == http.MethodPost:
		s.handleSign(w, r, parts[2])
	case len(parts) == 4 && parts[1] == "keys" && parts[3] == "verify" && r.Method == http.MethodPost:
		s.handleVerify(w, r, parts[2])
	case len(parts) == 3 && parts[1] == "encrypt" && r.Method == http.MethodPost:
		s.handleEncrypt(w, r, parts[2])
	case len(parts) == 3 && parts[1] == "decrypt" && r.Method == http.MethodPost:
		s.handleDecrypt(w, r, parts[2])
	case len(parts) == 2 && parts[1] == "secrets" && r.Method == http.MethodPost:
		s.handlePutSecret(w, r)
	case len(parts) == 3 && parts[1] == "secrets" && r.Method == http.MethodGet:
		s.handleGetSecret(w, r, parts[2])
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (s *Server) handleCreateKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string       `json:"name"`
		Type core.KeyType `json:"type"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Name = SanitizeKeyName(req.Name)
	if err := ValidateKeyName(req.Name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	key, err := s.vault.CreateKey(req.Name, req.Type)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, keyMetadataResponse(key))
}

func (s *Server) handleGetKey(w http.ResponseWriter, r *http.Request, name string) {
	if err := ValidateKeyName(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	key, err := s.vault.GetKey(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, keyMetadataResponse(key))
}

func (s *Server) handleRotateKey(w http.ResponseWriter, r *http.Request, name string) {
	if err := ValidateKeyName(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	key, err := s.vault.RotateKey(name)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, keyMetadataResponse(key))
}

func (s *Server) handleRevokeKey(w http.ResponseWriter, r *http.Request, name string) {
	if err := ValidateKeyName(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.vault.RevokeKey(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked", "name": name})
}

func (s *Server) handleSign(w http.ResponseWriter, r *http.Request, name string) {
	if err := ValidateKeyName(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req struct {
		DigestB64 string `json:"digest_b64"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	digest, err := DecodeBase64Payload(req.DigestB64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(digest) != 32 {
		writeError(w, http.StatusBadRequest, "digest must be 32 bytes (SHA-256)")
		return
	}
	signature, err := s.vault.Sign(name, digest)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"signature_b64": base64.StdEncoding.EncodeToString(signature),
	})
}

func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request, name string) {
	if err := ValidateKeyName(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req struct {
		DigestB64    string `json:"digest_b64"`
		SignatureB64 string `json:"signature_b64"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	digest, err := DecodeBase64Payload(req.DigestB64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(digest) != 32 {
		writeError(w, http.StatusBadRequest, "digest must be 32 bytes (SHA-256)")
		return
	}
	signature, err := base64.StdEncoding.DecodeString(req.SignatureB64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid signature_b64")
		return
	}
	err = s.vault.Verify(name, digest, signature)
	writeJSON(w, http.StatusOK, map[string]bool{"valid": err == nil})
}

func (s *Server) handleEncrypt(w http.ResponseWriter, r *http.Request, name string) {
	var req struct {
		Plaintext    string `json:"plaintext"`
		PlaintextB64 string `json:"plaintext_b64"`
		AAD          string `json:"aad"`
		AADB64       string `json:"aad_b64"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	plaintext, err := decodePayload(req.Plaintext, req.PlaintextB64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := ValidatePayloadSize(plaintext); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	aad, err := decodePayload(req.AAD, req.AADB64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := ValidateKeyName(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ciphertext, version, nonce, err := s.vault.Encrypt(name, plaintext, aad)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	combined, err := core.EncodeCiphertext(version, nonce, ciphertext)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ciphertext": base64.StdEncoding.EncodeToString(combined),
		"version":    version,
		"nonce_b64":  base64.StdEncoding.EncodeToString(nonce),
		"data_b64":   base64.StdEncoding.EncodeToString(ciphertext),
	})
}

func (s *Server) handleDecrypt(w http.ResponseWriter, r *http.Request, name string) {
	var req struct {
		Ciphertext string `json:"ciphertext"`
		AAD        string `json:"aad"`
		AADB64     string `json:"aad_b64"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Ciphertext == "" {
		writeError(w, http.StatusBadRequest, "ciphertext required")
		return
	}
	if err := ValidateKeyName(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	aad, err := decodePayload(req.AAD, req.AADB64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	raw, err := DecodeBase64Payload(req.Ciphertext)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	version, nonce, data, err := core.DecodeCiphertext(raw, 12)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	plaintext, err := s.vault.Decrypt(name, version, nonce, data, aad)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp := map[string]any{
		"plaintext_b64": base64.StdEncoding.EncodeToString(plaintext),
	}
	if utf8.Valid(plaintext) {
		resp["plaintext"] = string(plaintext)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handlePutSecret(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Value    string `json:"value"`
		ValueB64 string `json:"value_b64"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	value, err := decodePayload(req.Value, req.ValueB64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := ValidatePayloadSize(value); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Name = SanitizeKeyName(req.Name)
	if err := ValidateSecretName(req.Name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	record, err := s.secrets.Put(req.Name, value)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"name":       record.Name,
		"created_at": record.CreatedAt,
	})
}

func (s *Server) handleGetSecret(w http.ResponseWriter, r *http.Request, name string) {
	if err := ValidateSecretName(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	value, record, err := s.secrets.Get(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	resp := map[string]any{
		"name":       record.Name,
		"created_at": record.CreatedAt,
		"value_b64":  base64.StdEncoding.EncodeToString(value),
	}
	if utf8.Valid(value) {
		resp["value"] = string(value)
	}
	writeJSON(w, http.StatusOK, resp)
}

func keyMetadataResponse(key *core.Key) map[string]any {
	versions := make([]map[string]any, 0, len(key.Versions))
	latest := 0
	for _, v := range key.Versions {
		if v.Version > latest {
			latest = v.Version
		}
		versions = append(versions, map[string]any{
			"version":    v.Version,
			"created_at": v.CreatedAt,
		})
	}
	out := map[string]any{
		"name":           key.Name,
		"type":           key.Type,
		"created_at":     key.CreatedAt,
		"latest_version": latest,
		"versions":       versions,
	}
	if key.RevokedAt != nil {
		out["revoked_at"] = *key.RevokedAt
	}
	return out
}

func decodeJSON(r *http.Request, out any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("invalid json payload")
	}
	return nil
}

func decodePayload(value, valueB64 string) ([]byte, error) {
	if valueB64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(valueB64)
		if err != nil {
			return nil, errors.New("invalid base64 payload")
		}
		return decoded, nil
	}
	return []byte(value), nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil && err != http.ErrHandlerTimeout {
		if status < 500 {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func splitPath(path string) []string {
	if path == "" {
		return nil
	}
	parts := strings.Split(path, "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

package core

import (
	"bytes"
	"testing"
)

func TestVault_CreateKey(t *testing.T) {
	v := NewVault()

	key, err := v.CreateKey("test-key", KeyTypeAES256)
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	if key.Name != "test-key" {
		t.Errorf("key.Name = %q, want test-key", key.Name)
	}
	if key.Type != KeyTypeAES256 {
		t.Errorf("key.Type = %q, want aes-256", key.Type)
	}
	if len(key.Versions) != 1 || key.Versions[0].Version != 1 {
		t.Errorf("key.Versions = %+v, want single version 1", key.Versions)
	}
}

func TestVault_CreateKey_EmptyName(t *testing.T) {
	v := NewVault()
	_, err := v.CreateKey("", KeyTypeAES256)
	if err == nil {
		t.Error("CreateKey with empty name should error")
	}
}

func TestVault_CreateKey_DefaultType(t *testing.T) {
	v := NewVault()
	key, err := v.CreateKey("k", "")
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	if key.Type != KeyTypeAES256 {
		t.Errorf("key.Type = %q, want aes-256", key.Type)
	}
}

func TestVault_CreateKey_UnsupportedType(t *testing.T) {
	v := NewVault()
	_, err := v.CreateKey("k", "des")
	if err == nil {
		t.Error("CreateKey with unsupported type should error")
	}
}

func TestVault_CreateKey_Duplicate(t *testing.T) {
	v := NewVault()
	v.CreateKey("dup", KeyTypeAES256)
	_, err := v.CreateKey("dup", KeyTypeAES256)
	if err == nil {
		t.Error("CreateKey duplicate should error")
	}
}

func TestVault_GetKey_NotFound(t *testing.T) {
	v := NewVault()
	_, err := v.GetKey("missing")
	if err == nil {
		t.Error("GetKey missing should error")
	}
}

func TestVault_GetKey(t *testing.T) {
	v := NewVault()
	v.CreateKey("k1", KeyTypeAES256)
	key, err := v.GetKey("k1")
	if err != nil {
		t.Fatalf("GetKey: %v", err)
	}
	if key.Name != "k1" {
		t.Errorf("key.Name = %q, want k1", key.Name)
	}
}

func TestVault_RotateKey(t *testing.T) {
	v := NewVault()
	v.CreateKey("rot", KeyTypeAES256)
	key, err := v.RotateKey("rot")
	if err != nil {
		t.Fatalf("RotateKey: %v", err)
	}
	if len(key.Versions) != 2 {
		t.Errorf("len(key.Versions) = %d, want 2", len(key.Versions))
	}
	if key.Versions[1].Version != 2 {
		t.Errorf("Versions[1].Version = %d, want 2", key.Versions[1].Version)
	}
}

func TestVault_RotateKey_NotFound(t *testing.T) {
	v := NewVault()
	_, err := v.RotateKey("missing")
	if err == nil {
		t.Error("RotateKey missing should error")
	}
}

func TestVault_Encrypt_Decrypt_RoundTrip(t *testing.T) {
	v := NewVault()
	v.CreateKey("enc-key", KeyTypeAES256)

	plaintext := []byte("hello world")
	aad := []byte("optional aad")

	ciphertext, version, nonce, err := v.Encrypt("enc-key", plaintext, aad)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if version != 1 {
		t.Errorf("version = %d, want 1", version)
	}
	if len(nonce) != 12 {
		t.Errorf("nonce length = %d, want 12", len(nonce))
	}

	decrypted, err := v.Decrypt("enc-key", version, nonce, ciphertext, aad)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestVault_Encrypt_KeyNotFound(t *testing.T) {
	v := NewVault()
	_, _, _, err := v.Encrypt("missing", []byte("x"), nil)
	if err == nil {
		t.Error("Encrypt with missing key should error")
	}
}

func TestVault_Decrypt_KeyNotFound(t *testing.T) {
	v := NewVault()
	_, err := v.Decrypt("missing", 1, make([]byte, 12), []byte("x"), nil)
	if err == nil {
		t.Error("Decrypt with missing key should error")
	}
}

func TestVault_Decrypt_WrongAAD(t *testing.T) {
	v := NewVault()
	v.CreateKey("k", KeyTypeAES256)
	ciphertext, version, nonce, _ := v.Encrypt("k", []byte("secret"), []byte("aad1"))
	_, err := v.Decrypt("k", version, nonce, ciphertext, []byte("aad2"))
	if err == nil {
		t.Error("Decrypt with wrong AAD should error")
	}
}

func TestVault_Decrypt_InvalidVersion(t *testing.T) {
	v := NewVault()
	v.CreateKey("k", KeyTypeAES256)
	ciphertext, _, nonce, _ := v.Encrypt("k", []byte("x"), nil)
	_, err := v.Decrypt("k", 99, nonce, ciphertext, nil)
	if err == nil {
		t.Error("Decrypt with invalid version should error")
	}
}

func TestCloneKeyMetadata_NoMaterial(t *testing.T) {
	v := NewVault()
	v.CreateKey("meta", KeyTypeAES256)
	key, _ := v.GetKey("meta")
	for _, kv := range key.Versions {
		if len(kv.Material) != 0 {
			t.Error("cloneKeyMetadata should not expose Material")
		}
	}
}

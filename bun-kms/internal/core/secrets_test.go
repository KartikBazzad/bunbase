package core

import (
	"bytes"
	"testing"
)

func TestNewSecretStore(t *testing.T) {
	tests := []struct {
		name      string
		masterKey []byte
		wantErr   bool
	}{
		{"valid 32 bytes", make([]byte, 32), false},
		{"nil", nil, true},
		{"short", []byte("short"), true},
		{"long", make([]byte, 64), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSecretStore(tt.masterKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSecretStore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecretStore_Put_Get(t *testing.T) {
	s := MustNewSecretStore()
	defer func() { _ = s }()

	record, err := s.Put("my-secret", []byte("secret-value"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if record.Name != "my-secret" {
		t.Errorf("record.Name = %q, want my-secret", record.Name)
	}

	value, rec, err := s.Get("my-secret")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(value, []byte("secret-value")) {
		t.Errorf("value = %q, want secret-value", value)
	}
	if rec.Name != "my-secret" {
		t.Errorf("rec.Name = %q, want my-secret", rec.Name)
	}
}

func TestSecretStore_Put_EmptyName(t *testing.T) {
	s := MustNewSecretStore()
	_, err := s.Put("", []byte("x"))
	if err == nil {
		t.Error("Put with empty name should error")
	}
}

func TestSecretStore_Get_NotFound(t *testing.T) {
	s := MustNewSecretStore()
	_, _, err := s.Get("nonexistent")
	if err == nil {
		t.Error("Get nonexistent should error")
	}
}

func TestSecretStore_Put_Overwrite(t *testing.T) {
	s := MustNewSecretStore()
	s.Put("key", []byte("v1"))
	_, err := s.Put("key", []byte("v2"))
	if err != nil {
		t.Fatalf("Put overwrite: %v", err)
	}
	value, _, _ := s.Get("key")
	if !bytes.Equal(value, []byte("v2")) {
		t.Errorf("value = %q, want v2", value)
	}
}

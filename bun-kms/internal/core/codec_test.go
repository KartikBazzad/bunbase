package core

import (
	"bytes"
	"testing"
)

func TestEncodeCiphertext(t *testing.T) {
	tests := []struct {
		name    string
		version int
		nonce   []byte
		data    []byte
		wantErr bool
	}{
		{"valid", 1, make([]byte, 12), []byte("hello"), false},
		{"version zero", 0, make([]byte, 12), []byte("x"), true},
		{"negative version", -1, make([]byte, 12), []byte("x"), true},
		{"empty nonce", 1, nil, []byte("x"), true},
		{"large version", 999, make([]byte, 12), []byte("data"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EncodeCiphertext(tt.version, tt.nonce, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncodeCiphertext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != 4+len(tt.nonce)+len(tt.data) {
					t.Errorf("EncodeCiphertext() length = %d, want %d", len(got), 4+len(tt.nonce)+len(tt.data))
				}
			}
		})
	}
}

func TestDecodeCiphertext(t *testing.T) {
	nonceSize := 12
	tests := []struct {
		name      string
		blob      []byte
		nonceSize int
		wantErr   bool
	}{
		{"too short", []byte("abc"), nonceSize, true},
		{"minimal short", make([]byte, 4+nonceSize-1), nonceSize, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := DecodeCiphertext(tt.blob, tt.nonceSize)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeCiphertext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	version := 2
	nonce := []byte("123456789012")
	data := []byte("secret payload")

	encoded, err := EncodeCiphertext(version, nonce, data)
	if err != nil {
		t.Fatalf("EncodeCiphertext: %v", err)
	}

	gotVersion, gotNonce, gotData, err := DecodeCiphertext(encoded, 12)
	if err != nil {
		t.Fatalf("DecodeCiphertext: %v", err)
	}
	if gotVersion != version {
		t.Errorf("version = %d, want %d", gotVersion, version)
	}
	if !bytes.Equal(gotNonce, nonce) {
		t.Errorf("nonce = %q, want %q", gotNonce, nonce)
	}
	if !bytes.Equal(gotData, data) {
		t.Errorf("data = %q, want %q", gotData, data)
	}
}

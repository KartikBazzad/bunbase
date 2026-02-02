package core

// TestMasterKey is a fixed 32-byte key for use in tests only.
// Do not use in production.
var TestMasterKey = []byte("0123456789abcdef0123456789abcdef")

// MustNewSecretStore creates a SecretStore with TestMasterKey, panics on error.
func MustNewSecretStore() *SecretStore {
	s, err := NewSecretStore(TestMasterKey)
	if err != nil {
		panic(err)
	}
	return s
}

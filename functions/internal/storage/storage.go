package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// Storage manages function bundle storage on filesystem
type Storage struct {
	baseDir string
}

// NewStorage creates a new bundle storage
func NewStorage(baseDir string) (*Storage, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &Storage{baseDir: baseDir}, nil
}

// GetBundlePath returns the filesystem path for a function bundle version
func (s *Storage) GetBundlePath(functionID, version string) string {
	return filepath.Join(s.baseDir, functionID, version, "bundle.js")
}

// StoreBundle stores a function bundle
func (s *Storage) StoreBundle(functionID, version string, bundleData []byte) error {
	bundlePath := s.GetBundlePath(functionID, version)
	dir := filepath.Dir(bundlePath)

	// Create directory
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write bundle file
	if err := os.WriteFile(bundlePath, bundleData, 0644); err != nil {
		return fmt.Errorf("failed to write bundle: %w", err)
	}

	return nil
}

// GetBundle reads a function bundle
func (s *Storage) GetBundle(functionID, version string) ([]byte, error) {
	bundlePath := s.GetBundlePath(functionID, version)
	data, err := os.ReadFile(bundlePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("bundle not found: %s/%s", functionID, version)
		}
		return nil, fmt.Errorf("failed to read bundle: %w", err)
	}
	return data, nil
}

// BundleExists checks if a bundle exists
func (s *Storage) BundleExists(functionID, version string) bool {
	bundlePath := s.GetBundlePath(functionID, version)
	_, err := os.Stat(bundlePath)
	return err == nil
}

// DeleteBundle deletes a function bundle version
func (s *Storage) DeleteBundle(functionID, version string) error {
	bundlePath := s.GetBundlePath(functionID, version)
	if err := os.Remove(bundlePath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete bundle: %w", err)
	}

	// Try to remove version directory if empty
	versionDir := filepath.Dir(bundlePath)
	if err := os.Remove(versionDir); err != nil {
		// Ignore error if directory not empty
	}

	// Try to remove function directory if empty
	functionDir := filepath.Dir(versionDir)
	if err := os.Remove(functionDir); err != nil {
		// Ignore error if directory not empty
	}

	return nil
}

// ListVersions lists all versions for a function
func (s *Storage) ListVersions(functionID string) ([]string, error) {
	functionDir := filepath.Join(s.baseDir, functionID)
	entries, err := os.ReadDir(functionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read function directory: %w", err)
	}

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() {
			versions = append(versions, entry.Name())
		}
	}

	return versions, nil
}

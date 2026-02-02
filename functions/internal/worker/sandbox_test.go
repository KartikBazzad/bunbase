package worker

import (
	"context"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kartikbazzad/bunbase/functions/internal/config"
	"github.com/kartikbazzad/bunbase/functions/internal/logger"
)

func TestSandboxIsolation(t *testing.T) {
	// Check if bun is installed
	if _, err := exec.LookPath("bun"); err != nil {
		t.Skip("bun not found in PATH, skipping integration test")
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// Verify we are in the right directory or find the root
	// We assume we are running from functions/internal/worker
	// But if running from root, paths might differ.
	// Let's rely on relative paths from the package directory.

	workerScriptPath := filepath.Join(cwd, "../../worker/worker.ts")
	initScriptPath := filepath.Join(cwd, "../../worker/init.js")
	bundlePath := filepath.Join(cwd, "testdata/malicious.ts")

	// If paths don't exist, search up or assume specific layout
	if _, err := os.Stat(workerScriptPath); err != nil {
		// Maybe running from project root?
		if _, err := os.Stat("functions/worker/worker.ts"); err == nil {
			workerScriptPath, _ = filepath.Abs("functions/worker/worker.ts")
			initScriptPath, _ = filepath.Abs("functions/worker/init.js")
			bundlePath, _ = filepath.Abs("functions/internal/worker/testdata/malicious.ts")
		} else {
			t.Skipf("worker.ts not found at %s", workerScriptPath)
		}
	}

	// Ensure paths are absolute
	workerScriptPath, _ = filepath.Abs(workerScriptPath)
	initScriptPath, _ = filepath.Abs(initScriptPath)
	bundlePath, _ = filepath.Abs(bundlePath)

	log := logger.Default()
	w := NewBunWorker("sandbox-test", "v1", bundlePath, log)

	cfg := &config.WorkerConfig{
		BunPath:        "bun",
		StartupTimeout: 10 * time.Second,
	}

	// Spawn with init script
	err = w.Spawn(cfg, workerScriptPath, initScriptPath, nil)
	if err != nil {
		t.Fatalf("Failed to spawn worker: %v", err)
	}
	defer w.Terminate()

	// Invoke
	ctx := context.Background()
	payload := &InvokePayload{
		Method:     "GET",
		Path:       "/",
		Headers:    make(map[string]string),
		Query:      make(map[string]string),
		Body:       "",
		DeadlineMS: 5000,
	}

	resp, invokeErr, err := w.Invoke(ctx, payload)
	if err != nil {
		t.Fatalf("Invoke failed with error: %v", err)
	}
	if invokeErr != nil {
		t.Fatalf("Invoke returned application error: %v", invokeErr)
	}

	// Check response status
	if resp.Status != 403 {
		t.Errorf("Expected status 403 (Forbidden), got %d. Body: %s", resp.Status, resp.Body)
	}

	// Check response body for the specific error message from the script
	decodedBytes, err := base64.StdEncoding.DecodeString(resp.Body)
	if err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}
	body := string(decodedBytes)

	if !strings.Contains(body, "Caught error") {
		t.Errorf("Expected body to contain 'Caught error', got '%s'", body)
	}

	// Check if the error message confirms it was blocked (optional, depends on implementation)
	// The init.js should probably throw an error saying it's disabled
	if !strings.Contains(body, "Bun.file is disabled") && !strings.Contains(body, "Blocked by BunBase Sandbox") {
		// "is not a function" might happen if we deleted the function entirely
		t.Logf("Received body: %s", body)
	}
}

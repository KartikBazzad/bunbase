package worker

import (
	"context"
	"testing"

	"github.com/kartikbazzad/bunbase/functions/internal/capabilities"
	"github.com/kartikbazzad/bunbase/functions/internal/config"
	"github.com/kartikbazzad/bunbase/functions/internal/logger"
)

func TestWorkerInterface(t *testing.T) {
	// Test that BunWorker implements Worker interface
	var _ Worker = (*BunWorker)(nil)
	
	// Test that QuickJSWorker implements Worker interface
	var _ Worker = (*QuickJSWorker)(nil)
}

func TestBunWorkerCreation(t *testing.T) {
	log := logger.Default()
	w := NewBunWorker("test-func", "v1", "/path/to/bundle.js", log)
	
	if w == nil {
		t.Fatal("NewBunWorker returned nil")
	}
	
	if w.GetID() == "" {
		t.Error("Worker should have an ID")
	}
	
	if w.GetState() != WorkerStateStarting {
		t.Errorf("Expected state Starting, got %s", w.GetState())
	}
	
	if w.GetInvocations() != 0 {
		t.Error("New worker should have 0 invocations")
	}
}

func TestQuickJSWorkerCreation(t *testing.T) {
	log := logger.Default()
	w := NewQuickJSWorker("test-func", "v1", "/path/to/bundle.js", log)
	
	if w == nil {
		t.Fatal("NewQuickJSWorker returned nil")
	}
	
	if w.GetID() == "" {
		t.Error("Worker should have an ID")
	}
	
	if w.GetState() != WorkerStateStarting {
		t.Errorf("Expected state Starting, got %s", w.GetState())
	}
	
	if w.GetInvocations() != 0 {
		t.Error("New worker should have 0 invocations")
	}
}

func TestQuickJSWorkerCapabilities(t *testing.T) {
	log := logger.Default()
	w := NewQuickJSWorker("test-func", "v1", "/path/to/bundle.js", log)
	
	caps := capabilities.StrictProfile("test-project")
	w.SetCapabilities(caps)
	
	// Verify capabilities are set (we can't directly access private field,
	// but we can test that SetCapabilities doesn't panic)
	if caps == nil {
		t.Error("Capabilities should not be nil")
	}
}

func TestWorkerStateTransitions(t *testing.T) {
	log := logger.Default()
	w := NewBunWorker("test-func", "v1", "/path/to/bundle.js", log)
	
	initialState := w.GetState()
	if initialState != WorkerStateStarting {
		t.Errorf("Expected initial state Starting, got %s", initialState)
	}
	
	// Test that we can get state without panicking
	_ = w.GetState()
}

func TestWorkerHealthCheck(t *testing.T) {
	log := logger.Default()
	w := NewBunWorker("test-func", "v1", "/path/to/bundle.js", log)
	
	// New worker should not be healthy (not spawned)
	if w.HealthCheck() {
		t.Error("Unspawned worker should not be healthy")
	}
}

func TestWorkerInvokeNotReady(t *testing.T) {
	log := logger.Default()
	w := NewBunWorker("test-func", "v1", "/path/to/bundle.js", log)
	
	ctx := context.Background()
	payload := &InvokePayload{
		Method:     "GET",
		Path:       "/",
		Headers:    make(map[string]string),
		Query:      make(map[string]string),
		Body:       "",
		DeadlineMS: 5000,
	}
	
	_, _, err := w.Invoke(ctx, payload)
	if err == nil {
		t.Error("Invoke on unready worker should return error")
	}
}

func TestWorkerTerminate(t *testing.T) {
	log := logger.Default()
	w := NewBunWorker("test-func", "v1", "/path/to/bundle.js", log)
	
	// Terminate should not panic even if worker is not spawned
	err := w.Terminate()
	if err != nil {
		t.Errorf("Terminate should not error on unspawned worker: %v", err)
	}
	
	// State should be terminated
	if w.GetState() != WorkerStateTerminated {
		t.Errorf("Expected state Terminated after Terminate(), got %s", w.GetState())
	}
}

func TestWorkerLastUsed(t *testing.T) {
	log := logger.Default()
	w := NewBunWorker("test-func", "v1", "/path/to/bundle.js", log)
	
	lastUsed := w.GetLastUsed()
	if !lastUsed.IsZero() {
		t.Error("New worker should have zero last used time")
	}
}

func TestWorkerConfigRuntime(t *testing.T) {
	cfg := config.DefaultConfig()
	
	// Default should be bun
	if cfg.Worker.Runtime != "bun" {
		t.Errorf("Expected default runtime 'bun', got '%s'", cfg.Worker.Runtime)
	}
	
	// Test setting runtime
	cfg.Worker.Runtime = "quickjs-ng"
	if cfg.Worker.Runtime != "quickjs-ng" {
		t.Error("Failed to set runtime")
	}
}

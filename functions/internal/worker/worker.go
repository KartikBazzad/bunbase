package worker

import (
	"context"
	"time"

	"github.com/kartikbazzad/bunbase/functions/internal/config"
)

// Worker is the interface that all worker implementations must satisfy
type Worker interface {
	// Spawn starts the worker process
	Spawn(cfg *config.WorkerConfig, workerScriptPath string, initScriptPath string, env map[string]string) error

	// Invoke sends an invoke message to the worker and waits for response
	Invoke(ctx context.Context, payload *InvokePayload) (*ResponsePayload, *ErrorPayload, error)

	// Terminate kills the worker process
	Terminate() error

	// HealthCheck checks if the worker process is still alive
	HealthCheck() bool

	// GetState returns the current worker state
	GetState() WorkerState

	// GetID returns the worker ID
	GetID() string

	// GetLastUsed returns when the worker was last used
	GetLastUsed() time.Time

	// GetInvocations returns the number of invocations handled by this worker
	GetInvocations() int64
}

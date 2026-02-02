package worker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kartikbazzad/bunbase/functions/internal/config"
	"github.com/kartikbazzad/bunbase/functions/internal/logger"
)

// WorkerState represents the state of a worker
type WorkerState int

const (
	WorkerStateStarting WorkerState = iota
	WorkerStateReady
	WorkerStateBusy
	WorkerStateIdle
	WorkerStateTerminated
)

func (s WorkerState) String() string {
	switch s {
	case WorkerStateStarting:
		return "starting"
	case WorkerStateReady:
		return "ready"
	case WorkerStateBusy:
		return "busy"
	case WorkerStateIdle:
		return "idle"
	case WorkerStateTerminated:
		return "terminated"
	default:
		return "unknown"
	}
}

// BunWorker represents a Bun worker process
type BunWorker struct {
	id          string
	functionID  string
	version     string
	bundlePath  string
	process     *exec.Cmd
	stdin       io.WriteCloser
	stdout      io.ReadCloser
	state       WorkerState
	lastUsed    time.Time
	invocations int64
	mu          sync.Mutex
	logger      *logger.Logger
	reader      *MessageReader
	writer      *MessageWriter
	ctx         context.Context
	cancel      context.CancelFunc

	// Message routing: invocation ID -> response channel
	pendingInvocations map[string]chan *Message
	invocationMu       sync.RWMutex
}

// NewBunWorker creates a new Bun worker instance (does not spawn process)
func NewBunWorker(functionID, version, bundlePath string, log *logger.Logger) *BunWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &BunWorker{
		id:                 uuid.New().String(),
		functionID:         functionID,
		version:            version,
		bundlePath:         bundlePath,
		state:              WorkerStateStarting,
		logger:             log,
		ctx:                ctx,
		cancel:             cancel,
		pendingInvocations: make(map[string]chan *Message),
	}
}

// Spawn starts the Bun worker process
func (w *BunWorker) Spawn(cfg *config.WorkerConfig, workerScriptPath string, initScriptPath string, env map[string]string) error {
	w.mu.Lock()

	if w.process != nil {
		w.mu.Unlock()
		return fmt.Errorf("worker already spawned")
	}

	// Get absolute path to worker script
	scriptPath, err := filepath.Abs(workerScriptPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path to worker script: %w", err)
	}

	// Verify script exists
	if _, err := os.Stat(scriptPath); err != nil {
		return fmt.Errorf("worker script not found: %s", scriptPath)
	}

	// Verify bundle exists
	if _, err := os.Stat(w.bundlePath); err != nil {
		return fmt.Errorf("bundle not found: %s", w.bundlePath)
	}

	w.logger.Debug("Spawning worker: bun=%s script=%s bundle=%s", cfg.BunPath, scriptPath, w.bundlePath)

	// Prepare arguments
	args := []string{}
	if initScriptPath != "" {
		// Verify init script exists
		if _, err := os.Stat(initScriptPath); err == nil {
			args = append(args, "--preload", initScriptPath)
		} else {
			w.logger.Warn("Init script not found at %s, skipping preload", initScriptPath)
		}
	}
	args = append(args, scriptPath)

	// Create command
	cmd := exec.CommandContext(w.ctx, cfg.BunPath, args...)

	// Set environment variables
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("BUNDLE_PATH=%s", w.bundlePath))
	cmd.Env = append(cmd.Env, fmt.Sprintf("WORKER_ID=%s", w.id))
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set up stdin/stdout/stderr pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	w.stdin = stdin

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	w.stdout = stdout

	// Capture stderr to see Bun errors
	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start goroutine to read stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			// Parse log level from line if it has a prefix like [INFO], [DEBUG], etc.
			if len(line) >= 6 && line[0] == '[' && line[5] == ']' {
				level := line[1:5]
				message := line[6:]
				switch level {
				case "DEBUG":
					w.logger.Debug("Worker %s: %s", w.id, message)
				case "INFO":
					w.logger.Info("Worker %s: %s", w.id, message)
				case "WARN":
					w.logger.Warn("Worker %s: %s", w.id, message)
				case "ERROR":
					w.logger.Error("Worker %s: %s", w.id, message)
				default:
					// Unknown log level, use info
					w.logger.Info("Worker %s stderr: %s", w.id, line)
				}
			} else {
				// No log level prefix, log as debug
				w.logger.Debug("Worker %s: %s", w.id, line)
			}
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			w.logger.Debug("Worker %s stderr scanner error: %v", w.id, err)
		}
	}()

	// Start process
	w.logger.Info("Starting Bun process: %s %s (bundle: %s)", cfg.BunPath, scriptPath, w.bundlePath)
	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		w.mu.Unlock()
		return fmt.Errorf("failed to start process: %w", err)
	}

	w.process = cmd
	w.reader = NewMessageReader(stdout)
	w.writer = NewMessageWriter(stdin)
	w.state = WorkerStateStarting

	w.logger.Info("Worker %s process started (PID: %d), waiting for READY message (timeout: %v)", w.id, cmd.Process.Pid, cfg.StartupTimeout)

	// Unlock mutex before starting goroutines to avoid deadlock
	w.mu.Unlock()

	// Wait for READY message FIRST (before starting readMessages goroutine)
	// This prevents race condition where readMessages consumes the READY message
	deadline := time.Now().Add(cfg.StartupTimeout)
	readyReceived := false

	for !readyReceived {
		// Check timeout
		if time.Now().After(deadline) {
			w.logger.Error("Worker %s startup timeout after %v (no READY message received)", w.id, cfg.StartupTimeout)
			w.Terminate()
			return fmt.Errorf("worker startup timeout after %v", cfg.StartupTimeout)
		}

		// Check if process exited
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			exitCode := cmd.ProcessState.ExitCode()
			w.logger.Error("Worker %s process exited before ready (exit code: %d)", w.id, exitCode)
			w.Terminate()
			return fmt.Errorf("worker process exited before ready (exit code: %d)", exitCode)
		}

		// Read message (blocks until message available or EOF)
		msg, err := w.reader.Read()
		if err != nil {
			if err == io.EOF {
				w.logger.Error("Worker %s stdout closed before ready", w.id)
				w.Terminate()
				return fmt.Errorf("worker process exited before ready")
			}
			w.logger.Debug("Worker %s read error (will retry): %v", w.id, err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		w.logger.Debug("Worker %s received message type: %s", w.id, msg.Type)

		if msg.Type == MessageTypeReady {
			w.logger.Debug("Worker %s received READY, breaking out of startup loop", w.id)
			readyReceived = true
			break
		}

		// Handle error messages during startup
		if msg.Type == MessageTypeError {
			var payload ErrorPayload
			if err := json.Unmarshal(msg.Payload, &payload); err == nil {
				w.logger.Error("Worker %s error during startup: %s", w.id, payload.Message)
				w.Terminate()
				return fmt.Errorf("worker startup error: %s", payload.Message)
			}
		}

		// Handle log messages (but continue waiting for ready)
		if msg.Type == MessageTypeLog {
			var payload LogPayload
			if err := json.Unmarshal(msg.Payload, &payload); err == nil {
				w.logger.Info("Worker %s startup log: %s", w.id, payload.Message)
			}
		}
	}

	w.logger.Debug("Worker %s startup loop completed, starting readMessages goroutine", w.id)

	// Now start the message reader goroutine (for future messages)
	go w.readMessages()

	w.logger.Debug("Worker %s readMessages goroutine started, marking as ready", w.id)

	w.mu.Lock()
	w.state = WorkerStateReady
	w.lastUsed = time.Now()
	state := w.state
	w.mu.Unlock()

	// Force flush logs to ensure they appear
	w.logger.Info("Worker %s ready (state: %s)", w.id, state)
	w.logger.Debug("Worker %s Spawn() returning successfully", w.id)

	return nil
}

// waitForReady is now inlined in Spawn() to avoid race condition
// This function is kept for reference but not used
func (w *BunWorker) waitForReady(timeout time.Duration) error {
	// This is now handled inline in Spawn() to prevent race conditions
	return fmt.Errorf("waitForReady should not be called directly")
}

// readMessages reads messages from the worker and routes them
// This is the ONLY goroutine that reads from w.reader to avoid race conditions
func (w *BunWorker) readMessages() {
	for {
		// Check context cancellation first
		select {
		case <-w.ctx.Done():
			w.logger.Debug("Worker %s readMessages stopping (context cancelled)", w.id)
			return
		default:
		}

		// Read message (this can block, but closing stdout will cause EOF)
		msg, err := w.reader.Read()
		if err != nil {
			if err == io.EOF {
				w.logger.Debug("Worker %s stdout closed", w.id)
				w.mu.Lock()
				if w.state != WorkerStateTerminated {
					w.state = WorkerStateTerminated
					w.logger.Warn("Worker %s process exited unexpectedly", w.id)
				}
				w.mu.Unlock()
				// Notify all pending invocations
				w.invocationMu.Lock()
				for id, ch := range w.pendingInvocations {
					close(ch)
					delete(w.pendingInvocations, id)
				}
				w.invocationMu.Unlock()
				return
			}
			// Check if context was cancelled during read
			select {
			case <-w.ctx.Done():
				w.logger.Debug("Worker %s readMessages stopping (context cancelled after read error)", w.id)
				return
			default:
			}
			w.logger.Error("Worker %s read error: %v", w.id, err)
			// Small delay before retrying to avoid tight loop
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Route message based on type
		switch msg.Type {
		case MessageTypeLog:
			payload, err := ParseLogPayload(msg)
			if err == nil {
				w.logger.Info("Worker %s log [%s]: %s", w.id, payload.Level, payload.Message)
			}
		case MessageTypeResponse, MessageTypeError:
			// Route to pending invocation channel
			w.invocationMu.RLock()
			ch, exists := w.pendingInvocations[msg.ID]
			w.invocationMu.RUnlock()
			if exists {
				w.logger.Debug("Worker %s routing %s message to invocation %s", w.id, msg.Type, msg.ID)
				select {
				case ch <- msg:
					w.logger.Debug("Worker %s successfully delivered %s message to invocation %s", w.id, msg.Type, msg.ID)
					// Message delivered
				default:
					w.logger.Warn("Worker %s invocation channel full for ID %s", w.id, msg.ID)
				}
			} else {
				w.logger.Warn("Worker %s received %s for unknown invocation ID: %s (pending: %v)", w.id, msg.Type, msg.ID, len(w.pendingInvocations))
			}
		default:
			w.logger.Debug("Worker %s received unhandled message type: %s", w.id, msg.Type)
		}
	}
}

// Invoke sends an invoke message to the worker and waits for response
func (w *BunWorker) Invoke(ctx context.Context, payload *InvokePayload) (*ResponsePayload, *ErrorPayload, error) {
	w.mu.Lock()
	if w.state != WorkerStateReady {
		state := w.state
		w.mu.Unlock()
		w.logger.Error("Worker %s not ready (state: %s)", w.id, state)
		return nil, nil, fmt.Errorf("worker not ready (state: %s)", state)
	}
	w.state = WorkerStateBusy
	w.lastUsed = time.Now()
	w.invocations++
	invokeID := uuid.New().String()
	w.mu.Unlock()

	w.logger.Debug("Worker %s starting invocation %s", w.id, invokeID)

	// Register channel for this invocation
	msgCh := make(chan *Message, 1)
	w.invocationMu.Lock()
	w.pendingInvocations[invokeID] = msgCh
	w.invocationMu.Unlock()

	w.logger.Debug("Worker %s registered invocation channel for %s", w.id, invokeID)

	// Clean up channel when done
	defer func() {
		w.invocationMu.Lock()
		delete(w.pendingInvocations, invokeID)
		close(msgCh)
		w.invocationMu.Unlock()
		w.logger.Debug("Worker %s cleaned up invocation channel for %s", w.id, invokeID)
	}()

	// Send invoke message
	w.logger.Debug("Worker %s sending invoke message %s", w.id, invokeID)
	if err := w.writer.WriteInvoke(invokeID, payload); err != nil {
		w.mu.Lock()
		w.state = WorkerStateReady
		w.mu.Unlock()
		w.logger.Error("Worker %s failed to send invoke message: %v", w.id, err)
		return nil, nil, fmt.Errorf("failed to send invoke message: %w", err)
	}
	w.logger.Debug("Worker %s invoke message sent, waiting for response", w.id)

	// Wait for response with deadline
	deadline := time.Now().Add(time.Duration(payload.DeadlineMS) * time.Millisecond)
	responseCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	select {
	case msg := <-msgCh:
		w.logger.Debug("Worker %s received message for invocation %s: type=%s", w.id, invokeID, msg.Type)
		if msg == nil {
			// Channel closed (worker died)
			w.mu.Lock()
			w.state = WorkerStateTerminated
			w.mu.Unlock()
			w.logger.Error("Worker %s channel closed (worker died)", w.id)
			return nil, nil, fmt.Errorf("worker process exited")
		}

		switch msg.Type {
		case MessageTypeResponse:
			resp, err := ParseResponsePayload(msg)
			if err != nil {
				w.mu.Lock()
				w.state = WorkerStateReady
				w.mu.Unlock()
				w.logger.Error("Worker %s failed to parse response: %v", w.id, err)
				return nil, nil, fmt.Errorf("failed to parse response: %w", err)
			}
			w.mu.Lock()
			w.state = WorkerStateReady
			w.mu.Unlock()
			w.logger.Debug("Worker %s invocation %s completed successfully", w.id, invokeID)
			return resp, nil, nil
		case MessageTypeError:
			errPayload, err := ParseErrorPayload(msg)
			if err != nil {
				w.mu.Lock()
				w.state = WorkerStateReady
				w.mu.Unlock()
				w.logger.Error("Worker %s failed to parse error: %v", w.id, err)
				return nil, nil, fmt.Errorf("failed to parse error: %w", err)
			}
			w.mu.Lock()
			w.state = WorkerStateReady
			w.mu.Unlock()
			w.logger.Debug("Worker %s invocation %s returned error: %s", w.id, invokeID, errPayload.Message)
			return nil, errPayload, nil
		default:
			w.mu.Lock()
			w.state = WorkerStateReady
			w.mu.Unlock()
			w.logger.Warn("Worker %s received unexpected message type: %s", w.id, msg.Type)
			return nil, nil, fmt.Errorf("unexpected message type: %s", msg.Type)
		}
	case <-responseCtx.Done():
		w.mu.Lock()
		w.state = WorkerStateReady
		w.mu.Unlock()
		w.logger.Warn("Worker %s invocation %s deadline exceeded: %v", w.id, invokeID, responseCtx.Err())
		return nil, nil, fmt.Errorf("invocation deadline exceeded: %w", responseCtx.Err())
	}
}

// Terminate kills the worker process
func (w *BunWorker) Terminate() error {
	w.mu.Lock()
	// Don't return early if already terminated - still try to kill process
	// to ensure it's really dead
	w.state = WorkerStateTerminated

	process := w.process
	stdin := w.stdin
	stdout := w.stdout
	w.mu.Unlock()

	// Cancel context first (this will signal readMessages to stop)
	w.cancel()

	if process != nil {
		// Close pipes first (this will cause readMessages to get EOF and exit)
		if stdout != nil {
			stdout.Close()
		}
		if stdin != nil {
			stdin.Close()
		}

		// Kill process
		if process.Process != nil {
			pid := process.Process.Pid
			w.logger.Debug("Attempting to kill worker %s (PID: %d)", w.id, pid)
			if err := process.Process.Kill(); err != nil {
				w.logger.Debug("Failed to kill worker %s (PID: %d, may already be dead): %v", w.id, pid, err)
			} else {
				w.logger.Debug("Successfully sent kill signal to worker %s (PID: %d)", w.id, pid)
			}

			// Wait for process to exit (with timeout)
			done := make(chan error, 1)
			go func() {
				done <- process.Wait()
			}()

			select {
			case err := <-done:
				if err != nil {
					w.logger.Debug("Worker %s (PID: %d) exited with error: %v", w.id, pid, err)
				} else {
					w.logger.Debug("Worker %s (PID: %d) exited successfully", w.id, pid)
				}
			case <-time.After(2 * time.Second):
				w.logger.Warn("Worker %s (PID: %d) did not exit within 2 seconds, continuing anyway", w.id, pid)
			}
		} else {
			w.logger.Debug("Worker %s process.Process is nil, cannot kill", w.id)
		}
	}

	w.logger.Debug("Worker %s terminated", w.id)
	return nil
}

// HealthCheck checks if the worker process is still alive
func (w *BunWorker) HealthCheck() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.process == nil || w.state == WorkerStateTerminated {
		return false
	}

	// Check if process is still running
	if w.process.ProcessState != nil && w.process.ProcessState.Exited() {
		w.state = WorkerStateTerminated
		return false
	}

	// Try to send a signal 0 (doesn't kill, just checks if process exists)
	if err := w.process.Process.Signal(os.Signal(nil)); err != nil {
		w.state = WorkerStateTerminated
		return false
	}

	return true
}

// GetState returns the current worker state
func (w *BunWorker) GetState() WorkerState {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.state
}

// GetID returns the worker ID
func (w *BunWorker) GetID() string {
	return w.id
}

// GetLastUsed returns when the worker was last used
func (w *BunWorker) GetLastUsed() time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastUsed
}

// GetInvocations returns the number of invocations handled by this worker
func (w *BunWorker) GetInvocations() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.invocations
}

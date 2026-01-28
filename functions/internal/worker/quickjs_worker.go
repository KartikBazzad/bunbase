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
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/kartikbazzad/bunbase/functions/internal/capabilities"
	"github.com/kartikbazzad/bunbase/functions/internal/config"
	"github.com/kartikbazzad/bunbase/functions/internal/logger"
)

// QuickJSWorker represents a QuickJS-NG worker process
type QuickJSWorker struct {
	id                 string
	functionID         string
	version            string
	bundlePath         string
	process            *exec.Cmd
	stdin              io.WriteCloser
	stdout             io.ReadCloser
	state              WorkerState
	lastUsed           time.Time
	invocations        int64
	mu                 sync.Mutex
	logger             *logger.Logger
	reader             *MessageReader
	writer             *MessageWriter
	ctx                context.Context
	cancel             context.CancelFunc
	capabilities       *capabilities.Capabilities
	pendingInvocations map[string]chan *Message
	invocationMu       sync.RWMutex
}

// NewQuickJSWorker creates a new QuickJS worker instance (does not spawn process)
func NewQuickJSWorker(functionID, version, bundlePath string, log *logger.Logger) *QuickJSWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &QuickJSWorker{
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

// SetCapabilities sets the security capabilities for this worker
func (w *QuickJSWorker) SetCapabilities(caps *capabilities.Capabilities) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.capabilities = caps
}

// Spawn starts the QuickJS worker process
func (w *QuickJSWorker) Spawn(cfg *config.WorkerConfig, workerScriptPath string, env map[string]string) error {
	w.mu.Lock()

	if w.process != nil {
		w.mu.Unlock()
		return fmt.Errorf("worker already spawned")
	}

	// Get QuickJS worker binary path
	quickjsPath := cfg.QuickJSPath
	if quickjsPath == "" {
		quickjsPath = "./cmd/quickjs-worker/quickjs-worker"
	}

	// Get absolute path to QuickJS worker binary
	absQuickJSPath, err := filepath.Abs(quickjsPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path to QuickJS worker: %w", err)
	}

	// Verify QuickJS worker binary exists
	if _, err := os.Stat(absQuickJSPath); err != nil {
		return fmt.Errorf("QuickJS worker binary not found: %s", absQuickJSPath)
	}

	// Verify bundle exists
	if _, err := os.Stat(w.bundlePath); err != nil {
		return fmt.Errorf("bundle not found: %s", w.bundlePath)
	}

	w.logger.Debug("Spawning QuickJS worker: binary=%s bundle=%s", absQuickJSPath, w.bundlePath)

	// Create command
	cmd := exec.CommandContext(w.ctx, absQuickJSPath)

	// Set environment variables
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("BUNDLE_PATH=%s", w.bundlePath))
	cmd.Env = append(cmd.Env, fmt.Sprintf("WORKER_ID=%s", w.id))
	
	// Add capabilities to environment (as JSON)
	if w.capabilities != nil {
		capsJSON, err := json.Marshal(w.capabilities)
		if err == nil {
			cmd.Env = append(cmd.Env, fmt.Sprintf("CAPABILITIES=%s", string(capsJSON)))
		}
	}
	
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set resource limits before starting process
	if w.capabilities != nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
		
		// Set memory limit
		if w.capabilities.MaxMemory > 0 {
			// Note: Setrlimit will be called in the child process
			// We'll pass it via environment and set it in the C wrapper
			cmd.Env = append(cmd.Env, fmt.Sprintf("MAX_MEMORY=%d", w.capabilities.MaxMemory))
		}
		
		// Set file descriptor limit
		if w.capabilities.MaxFileDescriptors > 0 {
			cmd.Env = append(cmd.Env, fmt.Sprintf("MAX_FDS=%d", w.capabilities.MaxFileDescriptors))
		}
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

	// Capture stderr to see QuickJS errors
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
					w.logger.Debug("QuickJS Worker %s: %s", w.id, message)
				case "INFO":
					w.logger.Info("QuickJS Worker %s: %s", w.id, message)
				case "WARN":
					w.logger.Warn("QuickJS Worker %s: %s", w.id, message)
				case "ERROR":
					w.logger.Error("QuickJS Worker %s: %s", w.id, message)
				default:
					w.logger.Info("QuickJS Worker %s stderr: %s", w.id, line)
				}
			} else {
				w.logger.Debug("QuickJS Worker %s: %s", w.id, line)
			}
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			w.logger.Debug("QuickJS Worker %s stderr scanner error: %v", w.id, err)
		}
	}()

	// Start process
	w.logger.Info("Starting QuickJS process: %s (bundle: %s)", absQuickJSPath, w.bundlePath)
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

	w.logger.Info("QuickJS Worker %s process started (PID: %d), waiting for READY message (timeout: %v)", w.id, cmd.Process.Pid, cfg.StartupTimeout)

	// Unlock mutex before starting goroutines to avoid deadlock
	w.mu.Unlock()

	// Wait for READY message FIRST (before starting readMessages goroutine)
	deadline := time.Now().Add(cfg.StartupTimeout)
	readyReceived := false

	for !readyReceived {
		// Check timeout
		if time.Now().After(deadline) {
			w.logger.Error("QuickJS Worker %s startup timeout after %v (no READY message received)", w.id, cfg.StartupTimeout)
			w.Terminate()
			return fmt.Errorf("worker startup timeout after %v", cfg.StartupTimeout)
		}

		// Check if process exited
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			exitCode := cmd.ProcessState.ExitCode()
			w.logger.Error("QuickJS Worker %s process exited before ready (exit code: %d)", w.id, exitCode)
			w.Terminate()
			return fmt.Errorf("worker process exited before ready (exit code: %d)", exitCode)
		}

		// Read message (blocks until message available or EOF)
		msg, err := w.reader.Read()
		if err != nil {
			if err == io.EOF {
				w.logger.Error("QuickJS Worker %s stdout closed before ready", w.id)
				w.Terminate()
				return fmt.Errorf("worker process exited before ready")
			}
			w.logger.Debug("QuickJS Worker %s read error (will retry): %v", w.id, err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		w.logger.Debug("QuickJS Worker %s received message type: %s", w.id, msg.Type)

		if msg.Type == MessageTypeReady {
			w.logger.Debug("QuickJS Worker %s received READY, breaking out of startup loop", w.id)
			readyReceived = true
			break
		}

		// Handle error messages during startup
		if msg.Type == MessageTypeError {
			var payload ErrorPayload
			if err := json.Unmarshal(msg.Payload, &payload); err == nil {
				w.logger.Error("QuickJS Worker %s error during startup: %s", w.id, payload.Message)
				w.Terminate()
				return fmt.Errorf("worker startup error: %s", payload.Message)
			}
		}

		// Handle log messages (but continue waiting for ready)
		if msg.Type == MessageTypeLog {
			var payload LogPayload
			if err := json.Unmarshal(msg.Payload, &payload); err == nil {
				w.logger.Info("QuickJS Worker %s startup log: %s", w.id, payload.Message)
			}
		}
	}

	w.logger.Debug("QuickJS Worker %s startup loop completed, starting readMessages goroutine", w.id)

	// Now start the message reader goroutine (for future messages)
	go w.readMessages()

	w.logger.Debug("QuickJS Worker %s readMessages goroutine started, marking as ready", w.id)

	w.mu.Lock()
	w.state = WorkerStateReady
	w.lastUsed = time.Now()
	state := w.state
	w.mu.Unlock()

	w.logger.Info("QuickJS Worker %s ready (state: %s)", w.id, state)
	w.logger.Debug("QuickJS Worker %s Spawn() returning successfully", w.id)

	return nil
}

// readMessages reads messages from the worker and routes them
func (w *QuickJSWorker) readMessages() {
	for {
		select {
		case <-w.ctx.Done():
			w.logger.Debug("QuickJS Worker %s readMessages stopping (context cancelled)", w.id)
			return
		default:
		}

		msg, err := w.reader.Read()
		if err != nil {
			if err == io.EOF {
				w.logger.Debug("QuickJS Worker %s stdout closed", w.id)
				w.mu.Lock()
				if w.state != WorkerStateTerminated {
					w.state = WorkerStateTerminated
					w.logger.Warn("QuickJS Worker %s process exited unexpectedly", w.id)
				}
				w.mu.Unlock()
				w.invocationMu.Lock()
				for id, ch := range w.pendingInvocations {
					close(ch)
					delete(w.pendingInvocations, id)
				}
				w.invocationMu.Unlock()
				return
			}
			select {
			case <-w.ctx.Done():
				w.logger.Debug("QuickJS Worker %s readMessages stopping (context cancelled after read error)", w.id)
				return
			default:
			}
			w.logger.Error("QuickJS Worker %s read error: %v", w.id, err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		switch msg.Type {
		case MessageTypeLog:
			payload, err := ParseLogPayload(msg)
			if err == nil {
				w.logger.Info("QuickJS Worker %s log [%s]: %s", w.id, payload.Level, payload.Message)
			}
		case MessageTypeResponse, MessageTypeError:
			w.invocationMu.RLock()
			ch, exists := w.pendingInvocations[msg.ID]
			w.invocationMu.RUnlock()
			if exists {
				w.logger.Debug("QuickJS Worker %s routing %s message to invocation %s", w.id, msg.Type, msg.ID)
				select {
				case ch <- msg:
					w.logger.Debug("QuickJS Worker %s successfully delivered %s message to invocation %s", w.id, msg.Type, msg.ID)
				default:
					w.logger.Warn("QuickJS Worker %s invocation channel full for ID %s", w.id, msg.ID)
				}
			} else {
				w.logger.Warn("QuickJS Worker %s received %s for unknown invocation ID: %s", w.id, msg.Type, msg.ID)
			}
		default:
			w.logger.Debug("QuickJS Worker %s received unhandled message type: %s", w.id, msg.Type)
		}
	}
}

// Invoke sends an invoke message to the worker and waits for response
func (w *QuickJSWorker) Invoke(ctx context.Context, payload *InvokePayload) (*ResponsePayload, *ErrorPayload, error) {
	w.mu.Lock()
	if w.state != WorkerStateReady {
		state := w.state
		w.mu.Unlock()
		w.logger.Error("QuickJS Worker %s not ready (state: %s)", w.id, state)
		return nil, nil, fmt.Errorf("worker not ready (state: %s)", state)
	}
	w.state = WorkerStateBusy
	w.lastUsed = time.Now()
	w.invocations++
	invokeID := uuid.New().String()
	w.mu.Unlock()

	w.logger.Debug("QuickJS Worker %s starting invocation %s", w.id, invokeID)

	msgCh := make(chan *Message, 1)
	w.invocationMu.Lock()
	w.pendingInvocations[invokeID] = msgCh
	w.invocationMu.Unlock()

	defer func() {
		w.invocationMu.Lock()
		delete(w.pendingInvocations, invokeID)
		close(msgCh)
		w.invocationMu.Unlock()
		w.logger.Debug("QuickJS Worker %s cleaned up invocation channel for %s", w.id, invokeID)
	}()

	if err := w.writer.WriteInvoke(invokeID, payload); err != nil {
		w.mu.Lock()
		w.state = WorkerStateReady
		w.mu.Unlock()
		w.logger.Error("QuickJS Worker %s failed to send invoke message: %v", w.id, err)
		return nil, nil, fmt.Errorf("failed to send invoke message: %w", err)
	}

	deadline := time.Now().Add(time.Duration(payload.DeadlineMS) * time.Millisecond)
	responseCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	select {
	case msg := <-msgCh:
		if msg == nil {
			w.mu.Lock()
			w.state = WorkerStateTerminated
			w.mu.Unlock()
			w.logger.Error("QuickJS Worker %s channel closed (worker died)", w.id)
			return nil, nil, fmt.Errorf("worker process exited")
		}

		switch msg.Type {
		case MessageTypeResponse:
			resp, err := ParseResponsePayload(msg)
			if err != nil {
				w.mu.Lock()
				w.state = WorkerStateReady
				w.mu.Unlock()
				return nil, nil, fmt.Errorf("failed to parse response: %w", err)
			}
			w.mu.Lock()
			w.state = WorkerStateReady
			w.mu.Unlock()
			return resp, nil, nil
		case MessageTypeError:
			errPayload, err := ParseErrorPayload(msg)
			if err != nil {
				w.mu.Lock()
				w.state = WorkerStateReady
				w.mu.Unlock()
				return nil, nil, fmt.Errorf("failed to parse error: %w", err)
			}
			w.mu.Lock()
			w.state = WorkerStateReady
			w.mu.Unlock()
			return nil, errPayload, nil
		default:
			w.mu.Lock()
			w.state = WorkerStateReady
			w.mu.Unlock()
			return nil, nil, fmt.Errorf("unexpected message type: %s", msg.Type)
		}
	case <-responseCtx.Done():
		w.mu.Lock()
		w.state = WorkerStateReady
		w.mu.Unlock()
		w.logger.Warn("QuickJS Worker %s invocation %s deadline exceeded", w.id, invokeID)
		return nil, nil, fmt.Errorf("invocation deadline exceeded: %w", responseCtx.Err())
	}
}

// Terminate kills the worker process
func (w *QuickJSWorker) Terminate() error {
	w.mu.Lock()
	w.state = WorkerStateTerminated
	process := w.process
	stdin := w.stdin
	stdout := w.stdout
	w.mu.Unlock()

	w.cancel()

	if process != nil {
		if stdout != nil {
			stdout.Close()
		}
		if stdin != nil {
			stdin.Close()
		}

		if process.Process != nil {
			pid := process.Process.Pid
			w.logger.Debug("Attempting to kill QuickJS Worker %s (PID: %d)", w.id, pid)
			if err := process.Process.Kill(); err != nil {
				w.logger.Debug("Failed to kill QuickJS Worker %s (PID: %d): %v", w.id, pid, err)
			}

			done := make(chan error, 1)
			go func() {
				done <- process.Wait()
			}()

			select {
			case <-done:
			case <-time.After(2 * time.Second):
				w.logger.Warn("QuickJS Worker %s (PID: %d) did not exit within 2 seconds", w.id, pid)
			}
		}
	}

	w.logger.Debug("QuickJS Worker %s terminated", w.id)
	return nil
}

// HealthCheck checks if the worker process is still alive
func (w *QuickJSWorker) HealthCheck() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.process == nil || w.state == WorkerStateTerminated {
		return false
	}

	if w.process.ProcessState != nil && w.process.ProcessState.Exited() {
		w.state = WorkerStateTerminated
		return false
	}

	// Check if process is still alive by sending signal 0 (doesn't kill, just checks)
	// Use syscall.SIGCONT (0) which is a no-op signal that just checks if process exists
	if err := w.process.Process.Signal(syscall.Signal(0)); err != nil {
		w.state = WorkerStateTerminated
		return false
	}

	return true
}

// GetState returns the current worker state
func (w *QuickJSWorker) GetState() WorkerState {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.state
}

// GetID returns the worker ID
func (w *QuickJSWorker) GetID() string {
	return w.id
}

// GetLastUsed returns when the worker was last used
func (w *QuickJSWorker) GetLastUsed() time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastUsed
}

// GetInvocations returns the number of invocations handled by this worker
func (w *QuickJSWorker) GetInvocations() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.invocations
}

package manager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// BunderInstance represents a running Bunder process for a project.
type BunderInstance struct {
	ProjectID  string
	Port       int
	BaseURL    string
	Cmd        *exec.Cmd
	refCount   int32
	lastAccess int64
	createdAt  time.Time
}

// InstanceManager manages Bunder processes per project (hot/cold pool).
type InstanceManager struct {
	hot           sync.Map // projectID (string) → *BunderInstance
	dataPath      string
	bunderBin     string
	portPool      *portPool
	maxHot        int
	idleTTL       time.Duration
	evictInterval time.Duration
	stopChan      chan struct{}
	wg            sync.WaitGroup
	closed        atomic.Bool
}

// ManagerOptions configures the instance manager.
type ManagerOptions struct {
	DataPath      string
	BunderBin     string // Path to bunder binary (default: "bunder" in PATH)
	PortBase      int    // First port in pool (default 9000)
	PortCount     int    // Number of ports (default 1000)
	MaxHot        int
	IdleTTL       time.Duration
	EvictInterval time.Duration
}

// DefaultManagerOptions returns default configuration.
func DefaultManagerOptions(dataPath string) *ManagerOptions {
	return &ManagerOptions{
		DataPath:      dataPath,
		BunderBin:     "bunder",
		PortBase:      9000,
		PortCount:     1000,
		MaxHot:        100,
		IdleTTL:       10 * time.Minute,
		EvictInterval: 1 * time.Minute,
	}
}

// NewInstanceManager creates a new instance manager.
func NewInstanceManager(opts *ManagerOptions) (*InstanceManager, error) {
	if opts == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}
	pool, err := newPortPool(opts.PortBase, opts.PortCount)
	if err != nil {
		return nil, err
	}
	m := &InstanceManager{
		dataPath:      opts.DataPath,
		bunderBin:     opts.BunderBin,
		portPool:      pool,
		maxHot:        opts.MaxHot,
		idleTTL:       opts.IdleTTL,
		evictInterval: opts.EvictInterval,
		stopChan:      make(chan struct{}),
	}
	m.wg.Add(1)
	go m.evictionLoop()
	return m, nil
}

// Acquire returns the base URL for the project's Bunder instance and a release callback.
// Spawns a new Bunder process if needed.
func (m *InstanceManager) Acquire(projectID string) (baseURL string, release func(), err error) {
	if m.closed.Load() {
		return "", nil, fmt.Errorf("instance manager is closed")
	}

	// Fast path: already hot
	if val, ok := m.hot.Load(projectID); ok {
		inst := val.(*BunderInstance)
		atomic.AddInt32(&inst.refCount, 1)
		atomic.StoreInt64(&inst.lastAccess, time.Now().UnixNano())
		release := func() { atomic.AddInt32(&inst.refCount, -1) }
		return inst.BaseURL, release, nil
	}

	// Cold path: spawn new process
	var inst *BunderInstance
	inst, err = m.spawn(projectID)
	if err != nil {
		return "", nil, err
	}
	atomic.AddInt32(&inst.refCount, 1)
	release = func() { atomic.AddInt32(&inst.refCount, -1) }
	return inst.BaseURL, release, nil
}

// spawn starts a new Bunder process for the project and stores it in the hot map.
func (m *InstanceManager) spawn(projectID string) (*BunderInstance, error) {
	port, err := m.portPool.Acquire()
	if err != nil {
		return nil, fmt.Errorf("no free port: %w", err)
	}
	dataDir := filepath.Join(m.dataPath, "projects", projectID)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		m.portPool.Release(port)
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	// Bunder requires both TCP (RESP) and HTTP. Use port for HTTP, port+10000 for TCP to avoid collision.
	tcpPort := port + 10000
	cmd := exec.Command(m.bunderBin,
		"-data", dataDir,
		"-http", fmt.Sprintf(":%d", port),
		"-addr", fmt.Sprintf(":%d", tcpPort),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		m.portPool.Release(port)
		return nil, fmt.Errorf("start bunder: %w", err)
	}

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	inst := &BunderInstance{
		ProjectID:  projectID,
		Port:       port,
		BaseURL:    baseURL,
		Cmd:        cmd,
		refCount:   0,
		lastAccess: time.Now().UnixNano(),
		createdAt:  time.Now(),
	}

	val, loaded := m.hot.LoadOrStore(projectID, inst)
	if loaded {
		// Another goroutine won; kill our process and use theirs
		terminateProcess(cmd)
		m.portPool.Release(port)
		return val.(*BunderInstance), nil
	}

	// Give Bunder a moment to bind
	time.Sleep(100 * time.Millisecond)
	return inst, nil
}

func (m *InstanceManager) evictionLoop() {
	defer m.wg.Done()
	ticker := time.NewTicker(m.evictInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.evictIdle()
		case <-m.stopChan:
			return
		}
	}
}

func (m *InstanceManager) evictIdle() {
	now := time.Now()
	var toEvict []string
	m.hot.Range(func(key, value interface{}) bool {
		projectID := key.(string)
		inst := value.(*BunderInstance)
		if atomic.LoadInt32(&inst.refCount) == 0 &&
			now.UnixNano()-atomic.LoadInt64(&inst.lastAccess) > m.idleTTL.Nanoseconds() {
			toEvict = append(toEvict, projectID)
		}
		return true
	})
	for _, projectID := range toEvict {
		m.evictInstance(projectID)
	}
}

func (m *InstanceManager) evictInstance(projectID string) {
	val, ok := m.hot.LoadAndDelete(projectID)
	if !ok {
		return
	}
	inst := val.(*BunderInstance)
	if atomic.LoadInt32(&inst.refCount) > 0 {
		m.hot.Store(projectID, inst)
		return
	}
	if inst.Cmd != nil && inst.Cmd.Process != nil {
		terminateProcess(inst.Cmd)
	}
	m.portPool.Release(inst.Port)
}

// Close shuts down the manager and all Bunder processes.
func (m *InstanceManager) Close() error {
	if !m.closed.CompareAndSwap(false, true) {
		return nil
	}
	close(m.stopChan)
	m.wg.Wait()
	m.hot.Range(func(key, value interface{}) bool {
		inst := value.(*BunderInstance)
		if inst.Cmd != nil && inst.Cmd.Process != nil {
			terminateProcess(inst.Cmd)
		}
		m.portPool.Release(inst.Port)
		return true
	})
	return nil
}

// terminateProcess attempts to gracefully shut down the process with SIGTERM,
// waiting up to 5 seconds before resorting to SIGKILL.
func terminateProcess(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}

	// Try SIGTERM first
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// If process is already gone, or other error, just try kill to be safe or ignore
		cmd.Process.Kill()
		cmd.Wait() // release resources
		return
	}

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		// Use SIGKILL if still running
		cmd.Process.Kill()
		<-done
	case <-done:
		// Process exited gracefully
	}
}

// portPool manages a pool of ports for Bunder instances.
type portPool struct {
	ports chan int
}

func newPortPool(base, count int) (*portPool, error) {
	if count <= 0 {
		return nil, fmt.Errorf("port count must be positive")
	}
	ch := make(chan int, count)
	for i := 0; i < count; i++ {
		ch <- base + i
	}
	return &portPool{ports: ch}, nil
}

func (p *portPool) Acquire() (int, error) {
	select {
	case port := <-p.ports:
		return port, nil
	default:
		return 0, fmt.Errorf("port pool exhausted")
	}
}

func (p *portPool) Release(port int) {
	select {
	case p.ports <- port:
	default:
		// pool full, ignore
	}
}

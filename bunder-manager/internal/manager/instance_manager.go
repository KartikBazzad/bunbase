package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kartikbazzad/bunbase/bunder/pkg/store"
)

// BunderInstance represents an embedded Bunder KV store instance for a project.
type BunderInstance struct {
	ProjectID  string
	Store       store.Store
	refCount    int32
	lastAccess  int64
	createdAt   time.Time
	initDone    chan struct{} // Signals when instance is initialized
	initErr     error         // Initialization error, if any
}

// InstanceManager manages embedded Bunder KV store instances per project (hot/cold pool).
type InstanceManager struct {
	hot           sync.Map // projectID (string) → *BunderInstance
	dataPath      string
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
	MaxHot        int
	IdleTTL       time.Duration
	EvictInterval time.Duration
}

// DefaultManagerOptions returns default configuration.
func DefaultManagerOptions(dataPath string) *ManagerOptions {
	return &ManagerOptions{
		DataPath:      dataPath,
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

	m := &InstanceManager{
		dataPath:      opts.DataPath,
		maxHot:        opts.MaxHot,
		idleTTL:       opts.IdleTTL,
		evictInterval: opts.EvictInterval,
		stopChan:      make(chan struct{}),
	}
	m.wg.Add(1)
	go m.evictionLoop()
	return m, nil
}

// Acquire returns the Store for the project's Bunder instance and a release callback.
// Loads a new instance from disk if needed.
func (m *InstanceManager) Acquire(projectID string) (store.Store, func(), error) {
	if m.closed.Load() {
		return nil, nil, fmt.Errorf("instance manager is closed")
	}

	// Fast path: already hot
	if val, ok := m.hot.Load(projectID); ok {
		inst := val.(*BunderInstance)

		// Wait for initialization to complete
		<-inst.initDone
		if inst.initErr != nil {
			return nil, nil, inst.initErr
		}

		atomic.AddInt32(&inst.refCount, 1)
		atomic.StoreInt64(&inst.lastAccess, time.Now().UnixNano())

		release := func() {
			atomic.AddInt32(&inst.refCount, -1)
		}

		return inst.Store, release, nil
	}

	// Cold path: load from disk
	instance, err := m.loadInstance(projectID)
	if err != nil {
		return nil, nil, err
	}

	atomic.AddInt32(&instance.refCount, 1)
	release := func() {
		atomic.AddInt32(&instance.refCount, -1)
	}

	return instance.Store, release, nil
}

// loadInstance loads a Bunder instance from disk (cold start).
func (m *InstanceManager) loadInstance(projectID string) (*BunderInstance, error) {
	// Try to load or store atomically
	newInstance := &BunderInstance{
		ProjectID:  projectID,
		createdAt:  time.Now(),
		lastAccess: time.Now().UnixNano(),
		initDone:   make(chan struct{}),
	}

	val, loaded := m.hot.LoadOrStore(projectID, newInstance)
	instance := val.(*BunderInstance)

	if loaded {
		// Another goroutine beat us, wait for their initialization
		<-instance.initDone
		if instance.initErr != nil {
			return nil, instance.initErr
		}
		return instance, nil
	}

	// We won the race, actually initialize the KV store
	dataDir := filepath.Join(m.dataPath, "projects", projectID)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		instance.initErr = fmt.Errorf("create data dir: %w", err)
		m.hot.Delete(projectID)
		close(instance.initDone)
		return nil, instance.initErr
	}

	kvStore, err := store.Open(store.DefaultOptions(dataDir))
	if err != nil {
		instance.initErr = fmt.Errorf("failed to open KV store: %w", err)
		m.hot.Delete(projectID)
		close(instance.initDone)
		return nil, instance.initErr
	}

	// Update instance with initialized resources
	instance.Store = kvStore
	close(instance.initDone)

	return instance, nil
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
	if inst.Store != nil {
		inst.Store.Close()
	}
}

// Close shuts down the manager and all Bunder instances.
func (m *InstanceManager) Close() error {
	if !m.closed.CompareAndSwap(false, true) {
		return nil
	}
	close(m.stopChan)
	m.wg.Wait()
	m.hot.Range(func(key, value interface{}) bool {
		inst := value.(*BunderInstance)
		if inst.Store != nil {
			inst.Store.Close()
		}
		return true
	})
	return nil
}

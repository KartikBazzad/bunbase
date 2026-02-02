package manager

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc"
	"github.com/kartikbazzad/bunbase/bundoc/pool"
)

// HotInstance represents a hot (in-memory) database instance
type HotInstance struct {
	ProjectID  string
	DB         *bundoc.Database
	Pool       *pool.Pool
	refCount   int32 // Active requests using this instance
	lastAccess int64 // Unix nano timestamp (atomic)
	createdAt  time.Time
	initDone   chan struct{} // Signals when DB is initialized
	initErr    error         // Initialization error, if any
}

// InstanceManager manages database instances with hot/cold optimization
type InstanceManager struct {
	hot           sync.Map // projectID (string) → *HotInstance
	dataPath      string
	maxHot        int
	idleTTL       time.Duration
	evictInterval time.Duration
	stopChan      chan struct{}
	wg            sync.WaitGroup
	closed        atomic.Bool
}

// ManagerOptions configures the instance manager
type ManagerOptions struct {
	DataPath      string        // Root path for project databases
	MaxHot        int           // Maximum hot instances to keep
	IdleTTL       time.Duration // Time before evicting idle instance
	EvictInterval time.Duration // How often to check for eviction
}

// DefaultManagerOptions returns default configuration
func DefaultManagerOptions(dataPath string) *ManagerOptions {
	return &ManagerOptions{
		DataPath:      dataPath,
		MaxHot:        100,
		IdleTTL:       10 * time.Minute,
		EvictInterval: 1 * time.Minute,
	}
}

// NewInstanceManager creates a new instance manager
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

	// Start background eviction goroutine
	m.wg.Add(1)
	go m.evictionLoop()

	return m, nil
}

// Acquire gets a database instance for the project (lock-free hot path)
func (m *InstanceManager) Acquire(projectID string) (*bundoc.Database, func(), error) {
	if m.closed.Load() {
		return nil, nil, fmt.Errorf("instance manager is closed")
	}

	// Fast path: check if already hot (lock-free!)
	if val, ok := m.hot.Load(projectID); ok {
		instance := val.(*HotInstance)

		// Wait for initialization to complete
		<-instance.initDone
		if instance.initErr != nil {
			return nil, nil, instance.initErr
		}

		atomic.AddInt32(&instance.refCount, 1)
		atomic.StoreInt64(&instance.lastAccess, time.Now().UnixNano())

		release := func() {
			atomic.AddInt32(&instance.refCount, -1)
		}

		return instance.DB, release, nil
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

	return instance.DB, release, nil
}

// loadInstance loads a database instance from disk (cold start)
func (m *InstanceManager) loadInstance(projectID string) (*HotInstance, error) {
	// Try to load or store atomically
	newInstance := &HotInstance{
		ProjectID:  projectID,
		createdAt:  time.Now(),
		lastAccess: time.Now().UnixNano(),
		initDone:   make(chan struct{}),
	}

	val, loaded := m.hot.LoadOrStore(projectID, newInstance)
	instance := val.(*HotInstance)

	if loaded {
		// Another goroutine beat us, wait for their initialization
		<-instance.initDone
		if instance.initErr != nil {
			return nil, instance.initErr
		}
		return instance, nil
	}

	// We won the race, actually initialize the database
	dbPath := fmt.Sprintf("%s/%s", m.dataPath, projectID)
	dbOpts := bundoc.DefaultOptions(dbPath)

	db, err := bundoc.Open(dbOpts)
	if err != nil {
		instance.initErr = fmt.Errorf("failed to open database: %w", err)
		m.hot.Delete(projectID)  // Clean up failed instance
		close(instance.initDone) // Signal completion
		return nil, instance.initErr
	}

	// Create connection pool for this instance
	poolOpts := pool.DefaultPoolOptions()
	poolOpts.MinSize = 5
	poolOpts.MaxSize = 50

	connPool, err := pool.NewPool(dbPath, dbOpts, poolOpts)
	if err != nil {
		db.Close()
		instance.initErr = fmt.Errorf("failed to create pool: %w", err)
		m.hot.Delete(projectID)
		close(instance.initDone) // Signal completion
		return nil, instance.initErr
	}

	// Update instance with initialized resources
	instance.DB = db
	instance.Pool = connPool
	close(instance.initDone) // Signal completion

	return instance, nil
}

// evictionLoop periodically evicts idle instances
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

// evictIdle removes idle database instances
func (m *InstanceManager) evictIdle() {
	now := time.Now()
	var toEvict []string

	// Collect instances to evict
	m.hot.Range(func(key, value interface{}) bool {
		projectID := key.(string)
		instance := value.(*HotInstance)

		refCount := atomic.LoadInt32(&instance.refCount)
		lastAccessNano := atomic.LoadInt64(&instance.lastAccess)
		lastAccess := time.Unix(0, lastAccessNano)

		// Evict if idle and no active references
		if refCount == 0 && now.Sub(lastAccess) > m.idleTTL {
			toEvict = append(toEvict, projectID)
		}

		return true
	})

	// Evict collected instances
	for _, projectID := range toEvict {
		m.evictInstance(projectID)
	}
}

// evictInstance safely closes and removes an instance
func (m *InstanceManager) evictInstance(projectID string) {
	val, ok := m.hot.LoadAndDelete(projectID)
	if !ok {
		return
	}

	instance := val.(*HotInstance)

	// Double-check no active references
	if atomic.LoadInt32(&instance.refCount) > 0 {
		// Someone acquired it, put it back
		m.hot.Store(projectID, instance)
		return
	}

	// Close pool and database
	if instance.Pool != nil {
		instance.Pool.Close()
	}
	if instance.DB != nil {
		instance.DB.Close()
	}
}

// GetStats returns statistics about the instance manager
func (m *InstanceManager) GetStats() ManagerStats {
	stats := ManagerStats{}

	m.hot.Range(func(key, value interface{}) bool {
		stats.TotalInstances++

		instance := value.(*HotInstance)
		refCount := atomic.LoadInt32(&instance.refCount)

		if refCount > 0 {
			stats.ActiveInstances++
		} else {
			stats.IdleInstances++
		}

		return true
	})

	return stats
}

// ManagerStats contains instance manager statistics
type ManagerStats struct {
	TotalInstances  int
	ActiveInstances int
	IdleInstances   int
}

// Close shuts down the instance manager and all instances
func (m *InstanceManager) Close() error {
	if !m.closed.CompareAndSwap(false, true) {
		return fmt.Errorf("already closed")
	}

	// Stop eviction loop
	close(m.stopChan)
	m.wg.Wait()

	// Close all instances
	m.hot.Range(func(key, value interface{}) bool {
		instance := value.(*HotInstance)

		if instance.Pool != nil {
			instance.Pool.Close()
		}
		if instance.DB != nil {
			instance.DB.Close()
		}

		return true
	})

	return nil
}

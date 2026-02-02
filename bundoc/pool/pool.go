package pool

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc"
)

// Connection represents a pooled database connection
type Connection struct {
	DB        *bundoc.Database
	ID        uint64
	lastUsed  time.Time
	InUse     atomic.Bool
	CreatedAt time.Time
	pool      *Pool
	mu        sync.RWMutex
}

// GetLastUsed returns when connection was last used (thread-safe)
func (c *Connection) GetLastUsed() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastUsed
}

// setLastUsed sets last used time (thread-safe)
func (c *Connection) setLastUsed(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastUsed = t
}

// Pool manages a pool of database connections
type Pool struct {
	path           string
	opts           *bundoc.Options
	connections    []*Connection
	mu             sync.RWMutex
	nextID         atomic.Uint64
	minSize        int
	maxSize        int
	idleTimeout    time.Duration
	healthInterval time.Duration
	stopChan       chan struct{}
	running        bool
}

// PoolOptions configures the connection pool
type PoolOptions struct {
	MinSize        int           // Minimum pool size (default: 5)
	MaxSize        int           // Maximum pool size (default: 100)
	IdleTimeout    time.Duration // Idle connection timeout (default: 5min)
	HealthInterval time.Duration // Health check interval (default: 30s)
}

// DefaultPoolOptions returns default pool options
func DefaultPoolOptions() *PoolOptions {
	return &PoolOptions{
		MinSize:        5,
		MaxSize:        100,
		IdleTimeout:    5 * time.Minute,
		HealthInterval: 30 * time.Second,
	}
}

// NewPool creates a new connection pool
func NewPool(path string, dbOpts *bundoc.Options, poolOpts *PoolOptions) (*Pool, error) {
	if poolOpts == nil {
		poolOpts = DefaultPoolOptions()
	}

	if dbOpts == nil {
		dbOpts = bundoc.DefaultOptions(path)
	}

	pool := &Pool{
		path:           path,
		opts:           dbOpts,
		connections:    make([]*Connection, 0, poolOpts.MaxSize),
		minSize:        poolOpts.MinSize,
		maxSize:        poolOpts.MaxSize,
		idleTimeout:    poolOpts.IdleTimeout,
		healthInterval: poolOpts.HealthInterval,
		stopChan:       make(chan struct{}),
		running:        false,
	}

	// Create minimum connections
	for i := 0; i < poolOpts.MinSize; i++ {
		conn, err := pool.createConnection()
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to create initial connection: %w", err)
		}
		pool.connections = append(pool.connections, conn)
	}

	// Start health checker
	pool.running = true
	go pool.healthChecker()

	return pool, nil
}

// createConnection creates a new database connection
func (p *Pool) createConnection() (*Connection, error) {
	db, err := bundoc.Open(p.opts)
	if err != nil {
		return nil, err
	}

	conn := &Connection{
		DB:        db,
		ID:        p.nextID.Add(1),
		CreatedAt: time.Now(),
		pool:      p,
	}
	conn.InUse.Store(false)
	conn.setLastUsed(time.Now())

	return conn, nil
}

// Acquire acquires a connection from the pool
func (p *Pool) Acquire() (*Connection, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil, fmt.Errorf("pool is closed")
	}

	// Try to find an idle connection
	for _, conn := range p.connections {
		if !conn.InUse.Load() && !conn.DB.IsClosed() {
			conn.InUse.Store(true)
			conn.setLastUsed(time.Now())
			return conn, nil
		}
	}

	// Create new connection if under max size
	if len(p.connections) < p.maxSize {
		conn, err := p.createConnection()
		if err != nil {
			return nil, fmt.Errorf("failed to create connection: %w", err)
		}
		conn.InUse.Store(true)
		p.connections = append(p.connections, conn)
		return conn, nil
	}

	// Wait and retry if at max size
	return nil, fmt.Errorf("pool exhausted, max size %d reached", p.maxSize)
}

// Release releases a connection back to the pool
func (p *Pool) Release(conn *Connection) error {
	if conn == nil {
		return fmt.Errorf("cannot release nil connection")
	}

	if conn.pool != p {
		return fmt.Errorf("connection does not belong to this pool")
	}

	conn.InUse.Store(false)
	conn.setLastUsed(time.Now())

	return nil
}

// healthChecker periodically checks connection health
func (p *Pool) healthChecker() {
	ticker := time.NewTicker(p.healthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.checkHealth()
		case <-p.stopChan:
			return
		}
	}
}

// checkHealth checks and removes unhealthy/idle connections
func (p *Pool) checkHealth() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	activeconns := make([]*Connection, 0, len(p.connections))

	for _, conn := range p.connections {
		// Skip connections in use
		if conn.InUse.Load() {
			activeconns = append(activeconns, conn)
			continue
		}

		// Check if connection is closed
		if conn.DB.IsClosed() {
			continue // Remove from pool
		}

		// Check idle timeout
		if now.Sub(conn.GetLastUsed()) > p.idleTimeout && len(activeconns) >= p.minSize {
			conn.DB.Close()
			continue // Remove from pool
		}

		activeconns = append(activeconns, conn)
	}

	p.connections = activeconns

	// Ensure minimum pool size
	for len(p.connections) < p.minSize {
		conn, err := p.createConnection()
		if err != nil {
			break
		}
		p.connections = append(p.connections, conn)
	}
}

// GetStats returns pool statistics
func (p *Pool) GetStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := PoolStats{
		TotalConnections:  len(p.connections),
		IdleConnections:   0,
		ActiveConnections: 0,
		MinSize:           p.minSize,
		MaxSize:           p.maxSize,
	}

	for _, conn := range p.connections {
		if conn.InUse.Load() {
			stats.ActiveConnections++
		} else {
			stats.IdleConnections++
		}
	}

	return stats
}

// Close closes all connections in the pool
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("pool already closed")
	}

	p.running = false
	close(p.stopChan)

	var lastErr error
	for _, conn := range p.connections {
		if err := conn.DB.Close(); err != nil {
			lastErr = err
		}
	}

	p.connections = nil
	return lastErr
}

// PoolStats contains pool statistics
type PoolStats struct {
	TotalConnections  int
	IdleConnections   int
	ActiveConnections int
	MinSize           int
	MaxSize           int
}

// Package pool manages multiple logical databases in a single runtime.
//
// Pool provides:
//   - Database lifecycle management (create, open, close, delete)
//   - Request scheduling (round-robin across databases)
//   - Memory management (global and per-DB limits)
//   - Lazy database opening (opened on first access)
//
// Architecture:
//   - Each database gets its own queue (depth = QueueDepth)
//   - Round-robin scheduler picks next queue to service
//   - Workers execute requests sequentially per-database
//   - Backpressure: QueueFull error when queue at capacity
//
// Thread Safety: Pool operations are thread-safe via scheduler.
package pool

import (
	"github.com/kartikbazzad/docdb/internal/catalog"
	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/docdb"
	"github.com/kartikbazzad/docdb/internal/errors"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
	"github.com/kartikbazzad/docdb/internal/types"
)

var (
	ErrPoolStopped = errors.ErrPoolStopped
	ErrQueueFull   = errors.ErrQueueFull
)

var (
	// Re-export for backward compatibility
	ErrDBNotActive      = errors.ErrDBNotActive
	ErrUnknownOperation = errors.ErrUnknownOperation
)

// Request represents a database operation to be executed.
//
// It contains all information needed to execute the operation
// and a response channel for sending the result.
//
// Lifecycle:
//  1. Created by client (via IPC)
//  2. Enqueued in scheduler (per-DB queue)
//  3. Executed by worker (handleRequest)
//  4. Result sent on Response channel
//  5. Response sent back to client
type Request struct {
	DBID     uint64
	DocID    uint64
	OpType   types.OperationType
	Payload  []byte
	Response chan Response
}

// Response represents result of database operation.
//
// It contains operation status, optional data, and error.
//
// Fields:
//   - Status: Operation result (OK, Error, NotFound, MemoryLimit)
//   - Data: Response data (document payload, DB ID, etc.)
//   - Error: Error if operation failed (nil on success)
//
// Lifecycle: Created by handleRequest, sent on Response channel.
type Response struct {
	Status types.Status // Operation result status
	Data   []byte       // Response data (document payload, DB ID, stats, etc.)
	Error  error        // Error if operation failed, nil on success
}

// Pool manages multiple logical databases.
//
// It provides:
//   - Database lifecycle management (create, open, close, delete)
//   - Catalog persistence (metadata about databases)
//   - Request scheduling (round-robin across databases)
//   - Memory management (global + per-DB limits)
//   - Statistics aggregation
//
// Thread Safety: Pool operations are thread-safe.
// Individual databases have their own locking.
type Pool struct {
	dbs      map[uint64]*docdb.LogicalDB // Open databases by ID
	catalog  *catalog.Catalog            // Database metadata catalog
	sched    *Scheduler                  // Request scheduler
	memory   *memory.Caps                // Memory limit tracker
	pool     *memory.BufferPool          // Buffer allocation pool
	cfg      *config.Config              // Configuration
	logger   *logger.Logger              // Structured logging
	stopped  bool                        // True if pool is shutting down
	shutdown *GracefulShutdown           // Graceful shutdown handler
}

// NewPool creates a new database pool.
//
// It initializes:
//   - Memory caps (global and per-DB limits)
//   - Buffer pool (for efficient allocations)
//   - Catalog (for database metadata)
//   - Scheduler (for request distribution)
//
// Parameters:
//   - cfg: Pool configuration (memory, queue depth, etc.)
//   - log: Logger instance
//
// Returns:
//   - Initialized pool ready for Start()
//
// Note: Pool is not started until Start() is called.
func NewPool(cfg *config.Config, log *logger.Logger) *Pool {
	memCaps := memory.NewCaps(cfg.Memory.GlobalCapacityMB, cfg.Memory.PerDBLimitMB)
	bufferPool := memory.NewBufferPool(cfg.Memory.BufferSizes)
	catalog := catalog.NewCatalog(cfg.DataDir+"/.catalog", log)
	sched := NewScheduler(cfg.Sched.QueueDepth, log)

	return &Pool{
		dbs:     make(map[uint64]*docdb.LogicalDB),
		catalog: catalog,
		sched:   sched,
		memory:  memCaps,
		pool:    bufferPool,
		cfg:     cfg,
		logger:  log,
		stopped: false,
	}
}

func (p *Pool) Start() error {
	if err := p.catalog.Load(); err != nil {
		return err
	}

	p.sched.SetPool(p)
	p.sched.Start()

	// Initialize graceful shutdown handler
	p.shutdown = NewGracefulShutdown(p, p.logger)
	p.shutdown.StartSignalHandling()

	p.logger.Info("Pool started")
	return nil
}

func (p *Pool) Stop() {
	if p.shutdown != nil {
		p.shutdown.Shutdown()
	} else {
		// Fallback to immediate shutdown if graceful shutdown not initialized
		p.stopped = true
		p.sched.Stop()
		for _, db := range p.dbs {
			db.Close()
		}
		p.catalog.Close()
		p.logger.Info("Pool stopped")
	}
}

func (p *Pool) CreateDB(name string) (uint64, error) {
	if p.stopped {
		return 0, ErrPoolStopped
	}

	dbID, err := p.catalog.Create(name)
	if err != nil {
		return 0, err
	}

	p.memory.RegisterDB(dbID, p.cfg.Memory.PerDBLimitMB)

	return dbID, nil
}

func (p *Pool) OpenOrCreateDB(name string) (uint64, error) {
	if p.stopped {
		return 0, ErrPoolStopped
	}

	// Try to create the database
	dbID, err := p.catalog.Create(name)
	if err == nil {
		// Database was created, register it with memory
		p.memory.RegisterDB(dbID, p.cfg.Memory.PerDBLimitMB)
		return dbID, nil
	}

	// If database already exists, get its ID
	if err == catalog.ErrDBExists {
		dbID, err := p.catalog.GetByName(name)
		if err != nil {
			return 0, err
		}
		return dbID, nil
	}

	// Some other error occurred
	return 0, err
}

func (p *Pool) DeleteDB(dbID uint64) error {
	if p.stopped {
		return ErrPoolStopped
	}

	if err := p.catalog.Delete(dbID); err != nil {
		return err
	}

	if db, exists := p.dbs[dbID]; exists {
		db.Close()
		delete(p.dbs, dbID)
	}

	p.memory.UnregisterDB(dbID)

	p.logger.Info("Deleted database: id=%d", dbID)
	return nil
}

func (p *Pool) OpenDB(dbID uint64) (*docdb.LogicalDB, error) {
	if p.stopped {
		return nil, ErrPoolStopped
	}

	entry, err := p.catalog.Get(dbID)
	if err != nil {
		return nil, err
	}

	if entry.Status != types.DBActive {
		return nil, errors.ErrDBNotActive
	}

	if db, exists := p.dbs[dbID]; exists {
		return db, nil
	}

	db := docdb.NewLogicalDB(dbID, entry.DBName, p.cfg, p.memory, p.pool, p.logger)
	if err := db.Open(p.cfg.DataDir, p.cfg.WAL.Dir); err != nil {
		return nil, err
	}

	p.dbs[dbID] = db
	p.logger.Info("Opened database: %s (id=%d)", entry.DBName, dbID)

	return db, nil
}

func (p *Pool) CloseDB(dbID uint64) error {
	if db, exists := p.dbs[dbID]; exists {
		db.Close()
		delete(p.dbs, dbID)
		p.logger.Info("Closed database: id=%d", dbID)
	}

	return nil
}

func (p *Pool) Execute(req *Request) {
	if p.stopped {
		req.Response <- Response{
			Status: types.StatusError,
			Error:  ErrPoolStopped,
		}
		return
	}

	if err := p.sched.Enqueue(req); err != nil {
		req.Response <- Response{
			Status: types.StatusError,
			Error:  err,
		}
		return
	}
}

func (p *Pool) handleRequest(req *Request) {
	db, dbErr := p.OpenDB(req.DBID)
	if dbErr != nil {
		req.Response <- Response{
			Status: types.StatusError,
			Error:  dbErr,
		}
		return
	}

	var data []byte
	var err error

	switch req.OpType {
	case types.OpCreate:
		err = db.Create(req.DocID, req.Payload)
	case types.OpRead:
		data, err = db.Read(req.DocID)
	case types.OpUpdate:
		err = db.Update(req.DocID, req.Payload)
	case types.OpDelete:
		err = db.Delete(req.DocID)
	default:
		err = errors.ErrUnknownOperation
	}

	status := types.StatusOK
	if err != nil {
		status = types.StatusError
		if err == docdb.ErrDocNotFound || err == types.ErrDocNotFound {
			status = types.StatusNotFound
		} else if err == docdb.ErrMemoryLimit || err == types.ErrMemoryLimit {
			status = types.StatusMemoryLimit
		} else if err == docdb.ErrDocExists || err == docdb.ErrDocAlreadyExists || err == types.ErrDocExists {
			status = types.StatusConflict
		}
	}

	req.Response <- Response{
		Status: status,
		Data:   data,
		Error:  err,
	}
}

func (p *Pool) Stats() *types.Stats {
	return &types.Stats{
		TotalDBs:       len(p.dbs),
		ActiveDBs:      len(p.catalog.List()),
		TotalTxns:      0,
		WALSize:        0,
		MemoryUsed:     p.memory.GlobalUsage(),
		MemoryCapacity: p.memory.GlobalCapacity(),
	}
}

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
	"encoding/json"
	"errors"

	"github.com/kartikbazzad/docdb/internal/catalog"
	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/docdb"
	docdberrors "github.com/kartikbazzad/docdb/internal/errors"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
	"github.com/kartikbazzad/docdb/internal/query"
	"github.com/kartikbazzad/docdb/internal/types"
)

var (
	ErrPoolStopped = docdberrors.ErrPoolStopped
	ErrQueueFull   = docdberrors.ErrQueueFull
)

var (
	// Re-export for backward compatibility
	ErrDBNotActive      = docdberrors.ErrDBNotActive
	ErrUnknownOperation = docdberrors.ErrUnknownOperation
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
	DBID       uint64
	Collection string // v0.2: collection name
	DocID      uint64
	OpType     types.OperationType
	Payload    []byte                 // For OpCreate, OpUpdate
	PatchOps   []types.PatchOperation // For OpPatch
	Response   chan Response
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
	dbs          map[uint64]*docdb.LogicalDB // Open databases by ID
	catalog      *catalog.Catalog            // Database metadata catalog
	sched        *Scheduler                  // Request scheduler
	memory       *memory.Caps                // Memory limit tracker
	pool         *memory.BufferPool          // Buffer allocation pool
	cfg          *config.Config              // Configuration
	logger       *logger.Logger              // Structured logging
	stopped      bool                        // True if pool is shutting down
	shutdown     *GracefulShutdown           // Graceful shutdown handler
	classifier   *docdberrors.Classifier     // Error classifier
	errorTracker *docdberrors.ErrorTracker   // Error tracker
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
	// Enforce single-writer-by-default: only allow multiple workers when explicitly enabled.
	workerCount := cfg.Sched.WorkerCount
	maxWorkers := cfg.Sched.MaxWorkers
	if !cfg.Sched.UnsafeMultiWriter {
		if workerCount == 0 || workerCount > 1 {
			workerCount = 1
		}
		if maxWorkers == 0 || maxWorkers > 1 {
			maxWorkers = 1
		}
	}
	sched.SetWorkerConfig(workerCount, maxWorkers)
	sched.SetAntsOptions(cfg.Sched.WorkerExpiry, cfg.Sched.PreAlloc)

	return &Pool{
		dbs:          make(map[uint64]*docdb.LogicalDB),
		catalog:      catalog,
		sched:        sched,
		memory:       memCaps,
		pool:         bufferPool,
		cfg:          cfg,
		logger:       log,
		stopped:      false,
		classifier:   docdberrors.NewClassifier(),
		errorTracker: docdberrors.NewErrorTracker(),
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
		return nil, docdberrors.ErrDBNotActive
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
		category := p.classifier.Classify(dbErr)
		p.errorTracker.RecordError(dbErr, category)
		req.Response <- Response{
			Status: types.StatusError,
			Error:  dbErr,
		}
		return
	}

	// v0.4: If LogicalDB is partitioned, submit task to worker pool and wait for result
	if db.PartitionCount() > 1 {
		partitionID := docdb.RouteToPartition(req.DocID, db.PartitionCount())
		var task *docdb.Task
		switch req.OpType {
		case types.OpCreate:
			task = docdb.NewTaskWithPayload(partitionID, types.OpCreate, req.Collection, req.DocID, req.Payload)
		case types.OpRead:
			task = docdb.NewTask(partitionID, types.OpRead, req.Collection, req.DocID)
		case types.OpUpdate:
			task = docdb.NewTaskWithPayload(partitionID, types.OpUpdate, req.Collection, req.DocID, req.Payload)
		case types.OpDelete:
			task = docdb.NewTask(partitionID, types.OpDelete, req.Collection, req.DocID)
		case types.OpPatch:
			task = docdb.NewTaskWithPatch(partitionID, req.Collection, req.DocID, req.PatchOps)
		default:
			req.Response <- Response{Status: types.StatusError, Error: docdberrors.ErrUnknownOperation}
			return
		}
		result := db.SubmitTaskAndWait(task)
		req.Response <- Response{
			Status: result.Status,
			Data:   result.Data,
			Error:  result.Error,
		}
		return
	}

	// Legacy path (PartitionCount <= 1): direct db operations
	var data []byte
	var err error

	switch req.OpType {
	case types.OpCreate:
		err = db.Create(req.Collection, req.DocID, req.Payload)
	case types.OpRead:
		data, err = db.Read(req.Collection, req.DocID)
	case types.OpUpdate:
		err = db.Update(req.Collection, req.DocID, req.Payload)
	case types.OpDelete:
		err = db.Delete(req.Collection, req.DocID)
	case types.OpPatch:
		err = db.Patch(req.Collection, req.DocID, req.PatchOps)
	default:
		err = docdberrors.ErrUnknownOperation
	}

	status := types.StatusOK
	if err != nil {
		category := p.classifier.Classify(err)
		p.errorTracker.RecordError(err, category)
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

// GetErrorTracker returns the error tracker for metrics.
func (p *Pool) GetErrorTracker() *docdberrors.ErrorTracker {
	return p.errorTracker
}

func (p *Pool) CreateCollection(dbID uint64, name string) error {
	db, err := p.OpenDB(dbID)
	if err != nil {
		return err
	}
	return db.CreateCollection(name)
}

func (p *Pool) DeleteCollection(dbID uint64, name string) error {
	db, err := p.OpenDB(dbID)
	if err != nil {
		return err
	}
	return db.DeleteCollection(name)
}

func (p *Pool) ListCollections(dbID uint64) ([]string, error) {
	db, err := p.OpenDB(dbID)
	if err != nil {
		return nil, err
	}
	return db.ListCollections(), nil
}

func (p *Pool) ListDBs() []*types.DBInfo {
	entries := p.catalog.List()
	infos := make([]*types.DBInfo, 0, len(entries))

	for _, entry := range entries {
		info := &types.DBInfo{
			Name:      entry.DBName,
			ID:        entry.DBID,
			CreatedAt: entry.CreatedAt,
		}

		// Get stats if database is already open
		if db, exists := p.dbs[entry.DBID]; exists {
			stats := db.Stats()
			info.WALSize = stats.WALSize
			info.MemoryUsed = stats.MemoryUsed
			info.DocsLive = stats.DocsLive
		}
		// If database is not open, stats will remain zero (which is fine)

		infos = append(infos, info)
	}

	return infos
}

// GetSchedulerStats returns scheduler queue depth statistics and WAL group-commit metrics.
func (p *Pool) GetSchedulerStats() map[string]interface{} {
	queueStats := p.sched.GetQueueDepthStats()
	avgDepth := p.sched.GetAvgQueueDepth()

	out := map[string]interface{}{
		"queue_depths":    queueStats,
		"avg_queue_depth": avgDepth,
		"worker_count":    p.sched.GetCurrentWorkerCount(),
	}
	if antsStats := p.sched.GetAntsStats(); antsStats != nil {
		out["ants_pool"] = antsStats
	}

	// Aggregate WAL/group-commit stats per DB (when group commit is active)
	walStats := make(map[uint64]map[string]interface{})
	for dbID, db := range p.dbs {
		if st := db.GetWALStats(); st != nil {
			walStats[dbID] = st
		}
	}
	if len(walStats) > 0 {
		out["wal_group_commit"] = walStats
	}

	return out
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

// HealDocument heals a specific document in the specified database.
func (p *Pool) HealDocument(dbID uint64, collection string, docID uint64) error {
	if p.stopped {
		return ErrPoolStopped
	}

	db, err := p.OpenDB(dbID)
	if err != nil {
		return err
	}

	if db.HealingService() == nil {
		return errors.New("healing service not available")
	}

	return db.HealingService().HealDocument(collection, docID)
}

// HealAll triggers a full database healing scan.
func (p *Pool) HealAll(dbID uint64) ([]uint64, error) {
	if p.stopped {
		return nil, ErrPoolStopped
	}

	db, err := p.OpenDB(dbID)
	if err != nil {
		return nil, err
	}

	if db.HealingService() == nil {
		return nil, errors.New("healing service not available")
	}

	return db.HealingService().HealAll()
}

// HealStats returns healing statistics for the specified database.
func (p *Pool) HealStats(dbID uint64) (docdb.HealingStats, error) {
	if p.stopped {
		return docdb.HealingStats{}, ErrPoolStopped
	}

	db, err := p.OpenDB(dbID)
	if err != nil {
		return docdb.HealingStats{}, err
	}

	if db.HealingService() == nil {
		return docdb.HealingStats{}, errors.New("healing service not available")
	}

	stats := db.HealingService().GetStats()
	return stats, nil
}

// querySpec is the JSON wire format for CmdQuery payload.
type querySpec struct {
	Filter *struct {
		Field, Op string
		Value     interface{}
	} `json:"filter"`
	Limit   int `json:"limit"`
	OrderBy *struct {
		Field string
		Asc   bool
	} `json:"orderBy"`
}

// ExecuteQuery runs a server-side query on the database and returns rows as JSON.
// queryPayload is JSON: {"filter":{"field":"x","op":"eq","value":1},"limit":10,"orderBy":{"field":"id","asc":true}}
func (p *Pool) ExecuteQuery(dbID uint64, collection string, queryPayload []byte) ([]byte, error) {
	if p.stopped {
		return nil, ErrPoolStopped
	}

	db, err := p.OpenDB(dbID)
	if err != nil {
		return nil, err
	}

	q := query.Query{}
	if len(queryPayload) > 0 {
		var spec querySpec
		if err := json.Unmarshal(queryPayload, &spec); err != nil {
			return nil, err
		}
		if spec.Filter != nil {
			q.Filter = query.Expression{
				Field: spec.Filter.Field,
				Op:    spec.Filter.Op,
				Value: spec.Filter.Value,
			}
		}
		q.Limit = spec.Limit
		if spec.OrderBy != nil {
			q.OrderBy = &query.OrderSpec{
				Field: spec.OrderBy.Field,
				Asc:   spec.OrderBy.Asc,
			}
		}
	}

	rows, err := db.ExecuteQuery(collection, q)
	if err != nil {
		return nil, err
	}

	// Serialize rows as JSON array of { "docID": N, "payload": <raw JSON bytes as string or base64?> }
	// Use array of objects with docID and payload (payload as JSON-encoded string for simplicity)
	out := make([]map[string]interface{}, len(rows))
	for i, r := range rows {
		out[i] = map[string]interface{}{
			"docID":   r.DocID,
			"payload": json.RawMessage(r.Payload),
		}
	}
	return json.Marshal(out)
}

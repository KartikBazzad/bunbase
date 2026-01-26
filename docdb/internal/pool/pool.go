package pool

import (
	"errors"

	"github.com/kartikbazzad/docdb/internal/catalog"
	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/docdb"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/kartikbazzad/docdb/internal/memory"
	"github.com/kartikbazzad/docdb/internal/types"
)

var (
	ErrPoolStopped = errors.New("pool is stopped")
	ErrQueueFull   = errors.New("request queue is full")
)

type Request struct {
	DBID     uint64
	DocID    uint64
	OpType   types.OperationType
	Payload  []byte
	Response chan Response
}

type Response struct {
	Status types.Status
	Data   []byte
	Error  error
}

type Pool struct {
	dbs     map[uint64]*docdb.LogicalDB
	catalog *catalog.Catalog
	sched   *Scheduler
	memory  *memory.Caps
	pool    *memory.BufferPool
	cfg     *config.Config
	logger  *logger.Logger
	stopped bool
}

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
	p.logger.Info("Pool started")
	return nil
}

func (p *Pool) Stop() {
	p.stopped = true

	p.sched.Stop()

	for _, db := range p.dbs {
		db.Close()
	}

	p.catalog.Close()
	p.logger.Info("Pool stopped")
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
		return nil, errors.New("database is not active")
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

	p.sched.Enqueue(req)
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
		err = errors.New("unknown operation type")
	}

	status := types.StatusOK
	if err != nil {
		status = types.StatusError
		if err == docdb.ErrDocNotFound {
			status = types.StatusNotFound
		} else if err == docdb.ErrMemoryLimit {
			status = types.StatusMemoryLimit
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

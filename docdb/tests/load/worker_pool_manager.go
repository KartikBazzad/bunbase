package load

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kartikbazzad/docdb/pkg/client"
)

// Worker executes operations for assigned databases.
type Worker struct {
	ID           int
	Databases    []string // Databases this worker handles
	Client       *client.Client
	CurrentPhase *WorkloadPhase
	StopCh       chan struct{}
	Config       *MultiDBLoadTestConfig
	DBManager    *DatabaseManager
	ProfileMgr   *WorkloadProfileManager
	RNG          *rand.Rand
	OpsCount     int64
}

// WorkerPoolManager handles dynamic worker allocation.
type WorkerPoolManager struct {
	workers    []*Worker
	dbWorkers  map[string][]*Worker // Database -> workers
	mu         sync.RWMutex
	config     *MultiDBLoadTestConfig
	dbManager  *DatabaseManager
	profileMgr *WorkloadProfileManager
	totalOps   int64
	stopFlag   int32
}

// NewWorkerPoolManager creates a new worker pool manager.
func NewWorkerPoolManager(config *MultiDBLoadTestConfig, dbManager *DatabaseManager, profileMgr *WorkloadProfileManager) *WorkerPoolManager {
	return &WorkerPoolManager{
		workers:    make([]*Worker, 0),
		dbWorkers:  make(map[string][]*Worker),
		config:     config,
		dbManager:  dbManager,
		profileMgr: profileMgr,
		stopFlag:   0,
	}
}

// Start initializes and starts workers based on current phase.
func (wpm *WorkerPoolManager) Start() error {
	wpm.mu.Lock()
	defer wpm.mu.Unlock()

	// Get initial worker count from profile or config
	workerCount := wpm.getWorkerCount()

	// Allocate workers to databases
	if err := wpm.allocateWorkers(workerCount); err != nil {
		return err
	}

	// Start all workers
	for _, worker := range wpm.workers {
		go wpm.runWorker(worker)
	}

	return nil
}

// getWorkerCount returns the current worker count.
func (wpm *WorkerPoolManager) getWorkerCount() int {
	if wpm.profileMgr != nil {
		return wpm.profileMgr.GetWorkerCount()
	}

	// Sum workers from databases
	total := 0
	for _, db := range wpm.config.Databases {
		total += db.Workers
	}
	return total
}

// allocateWorkers allocates workers to databases.
func (wpm *WorkerPoolManager) allocateWorkers(totalWorkers int) error {
	// Clear existing workers
	wpm.stopAllWorkers()
	wpm.workers = make([]*Worker, 0)
	wpm.dbWorkers = make(map[string][]*Worker)

	// If profile specifies workers, distribute across databases
	if wpm.profileMgr != nil {
		workersPerDB := totalWorkers / len(wpm.config.Databases)
		if workersPerDB == 0 {
			workersPerDB = 1
		}

		workerID := 0
		for _, dbConfig := range wpm.config.Databases {
			dbWorkers := make([]*Worker, 0)
			for i := 0; i < workersPerDB; i++ {
				client := client.New(wpm.config.SocketPath)
				if err := client.Connect(); err != nil {
					return fmt.Errorf("failed to connect worker client: %w", err)
				}

				worker := &Worker{
					ID:         workerID,
					Databases:  []string{dbConfig.Name},
					Client:     client,
					StopCh:     make(chan struct{}),
					Config:     wpm.config,
					DBManager:  wpm.dbManager,
					ProfileMgr: wpm.profileMgr,
					RNG:        rand.New(rand.NewSource(wpm.config.Seed + int64(workerID))),
				}

				wpm.workers = append(wpm.workers, worker)
				dbWorkers = append(dbWorkers, worker)
				workerID++
			}
			wpm.dbWorkers[dbConfig.Name] = dbWorkers
		}
	} else {
		// Use per-database worker counts
		workerID := 0
		for _, dbConfig := range wpm.config.Databases {
			dbWorkers := make([]*Worker, 0)
			for i := 0; i < dbConfig.Workers; i++ {
				client := client.New(wpm.config.SocketPath)
				if err := client.Connect(); err != nil {
					return fmt.Errorf("failed to connect worker client: %w", err)
				}

				worker := &Worker{
					ID:         workerID,
					Databases:  []string{dbConfig.Name},
					Client:     client,
					StopCh:     make(chan struct{}),
					Config:     wpm.config,
					DBManager:  wpm.dbManager,
					ProfileMgr: wpm.profileMgr,
					RNG:        rand.New(rand.NewSource(wpm.config.Seed + int64(workerID))),
				}

				wpm.workers = append(wpm.workers, worker)
				dbWorkers = append(dbWorkers, worker)
				workerID++
			}
			wpm.dbWorkers[dbConfig.Name] = dbWorkers
		}
	}

	return nil
}

// ScaleWorkers adjusts worker count based on current phase.
func (wpm *WorkerPoolManager) ScaleWorkers() error {
	wpm.mu.Lock()
	defer wpm.mu.Unlock()

	newWorkerCount := wpm.getWorkerCount()
	currentCount := len(wpm.workers)

	if newWorkerCount == currentCount {
		return nil // No change needed
	}

	// Reallocate workers
	return wpm.allocateWorkers(newWorkerCount)
}

// runWorker runs a worker's main loop.
func (wpm *WorkerPoolManager) runWorker(worker *Worker) {
	for {
		// Check if we should stop
		if atomic.LoadInt32(&wpm.stopFlag) == 1 {
			return
		}

		// Select a database for this operation
		dbName := worker.selectDatabase()
		if dbName == "" {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		// Get database context
		ctx, err := wpm.dbManager.GetDatabase(dbName)
		if err != nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		// Get CRUD percentages (from profile or per-database)
		crudPercent := wpm.getCRUDPercent(dbName, ctx)

		// Get current phase for tracking
		var phaseName string
		if wpm.profileMgr != nil {
			if phaseInfo := wpm.profileMgr.GetPhaseInfo(); phaseInfo != nil {
				phaseName = phaseInfo.Name
			}
		}

		// Generate and execute work item
		item := worker.generateWorkItem(ctx, crudPercent)
		worker.executeOperation(ctx, item)

		atomic.AddInt64(&wpm.totalOps, 1)
		atomic.AddInt64(&worker.OpsCount, 1)

		// Record phase operation if metrics available
		if phaseName != "" && wpm.config != nil {
			// Phase tracking will be handled by metrics collector
		}
	}
}

// selectDatabase selects a database for the worker to operate on.
func (w *Worker) selectDatabase() string {
	if len(w.Databases) == 0 {
		return ""
	}
	if len(w.Databases) == 1 {
		return w.Databases[0]
	}
	// Round-robin or random selection
	return w.Databases[w.RNG.Intn(len(w.Databases))]
}

// getCRUDPercent returns CRUD percentages for a database.
func (wpm *WorkerPoolManager) getCRUDPercent(dbName string, ctx *DatabaseContext) CRUDPercentages {
	// If per-database CRUD is enabled and database has its own CRUD config
	if wpm.config.PerDatabaseCRUD && ctx.Config.CRUDPercent != nil {
		return *ctx.Config.CRUDPercent
	}

	// Use profile CRUD if available
	if wpm.profileMgr != nil {
		return wpm.profileMgr.GetCRUDPercent()
	}

	// Default CRUD mix
	return CRUDPercentages{
		ReadPercent:   40,
		WritePercent:  30,
		UpdatePercent: 20,
		DeletePercent: 10,
	}
}

// generateWorkItem generates a work item for a database.
func (w *Worker) generateWorkItem(ctx *DatabaseContext, crudPercent CRUDPercentages) workItem {
	docID := uint64(w.RNG.Intn(ctx.Config.DocumentCount)) + 1
	payload := ctx.Payloads[w.RNG.Intn(len(ctx.Payloads))]

	// Determine operation type based on percentages
	roll := w.RNG.Intn(100)
	var opType OperationType
	if roll < crudPercent.ReadPercent {
		opType = OpRead
	} else if roll < crudPercent.ReadPercent+crudPercent.WritePercent {
		opType = OpCreate
	} else if roll < crudPercent.ReadPercent+crudPercent.WritePercent+crudPercent.UpdatePercent {
		opType = OpUpdate
	} else {
		opType = OpDelete
	}

	return workItem{
		opType:     opType,
		docID:      docID,
		payload:    payload,
		collection: "_default",
	}
}

// executeOperation executes an operation and records metrics.
func (w *Worker) executeOperation(ctx *DatabaseContext, item workItem) {
	start := time.Now()

	switch item.opType {
	case OpCreate:
		_ = ctx.Client.Create(ctx.DBID, item.collection, item.docID, item.payload)
	case OpRead:
		_, _ = ctx.Client.Read(ctx.DBID, item.collection, item.docID)
	case OpUpdate:
		_ = ctx.Client.Update(ctx.DBID, item.collection, item.docID, item.payload)
	case OpDelete:
		_ = ctx.Client.Delete(ctx.DBID, item.collection, item.docID)
	}

	latency := time.Since(start)

	// Record latency (even for errors)
	ctx.LatencyMetrics.Record(item.opType, latency)
}

// Stop stops all workers.
func (wpm *WorkerPoolManager) Stop() {
	atomic.StoreInt32(&wpm.stopFlag, 1)
	wpm.stopAllWorkers()
}

// stopAllWorkers stops all current workers.
func (wpm *WorkerPoolManager) stopAllWorkers() {
	for _, worker := range wpm.workers {
		close(worker.StopCh)
		if worker.Client != nil {
			worker.Client.Close()
		}
	}
}

// GetTotalOperations returns total operations across all workers.
func (wpm *WorkerPoolManager) GetTotalOperations() int64 {
	return atomic.LoadInt64(&wpm.totalOps)
}

// GetWorkerStats returns statistics about workers.
func (wpm *WorkerPoolManager) GetWorkerStats() map[string]int64 {
	wpm.mu.RLock()
	defer wpm.mu.RUnlock()

	stats := make(map[string]int64)
	for dbName, workers := range wpm.dbWorkers {
		var totalOps int64
		for _, worker := range workers {
			totalOps += atomic.LoadInt64(&worker.OpsCount)
		}
		stats[dbName] = totalOps
	}
	return stats
}

// workItem represents a single operation to execute.
type workItem struct {
	opType     OperationType
	docID      uint64
	payload    []byte
	collection string
}

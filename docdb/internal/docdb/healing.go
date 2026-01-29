package docdb

import (
	"sync"
	"time"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/logger"
	"github.com/panjf2000/ants/v2"
)

// healTask is a single document to heal (used with ants pool).
type healTask struct {
	Collection string
	DocID      uint64
}

// healTaskWithWg carries a task and WaitGroup for parallel HealAll.
type healTaskWithWg struct {
	Collection string
	DocID      uint64
	Wg         *sync.WaitGroup
	Healed     *[]uint64
	HealedMu   *sync.Mutex
}

// HealingService provides automatic document healing capabilities.
type HealingService struct {
	db           *LogicalDB
	cfg          *config.HealingConfig
	logger       *logger.Logger
	healer       *Healer
	validator    *Validator
	stopCh       chan struct{}
	wg           sync.WaitGroup
	mu           sync.Mutex
	healingStats HealingStats
	healingQueue []uint64
	queueMu      sync.Mutex
	healPool     *ants.PoolWithFunc // Pool for parallel healing (nil until first use)
	healPoolMu   sync.Mutex
}

// HealingStats tracks healing statistics.
type HealingStats struct {
	TotalScans         uint64
	DocumentsHealed    uint64
	DocumentsCorrupted uint64
	LastScanTime       time.Time
	LastHealingTime    time.Time
	OnDemandHealings   uint64
	BackgroundHealings uint64
}

// NewHealingService creates a new healing service.
func NewHealingService(db *LogicalDB, cfg *config.HealingConfig, log *logger.Logger) *HealingService {
	return &HealingService{
		db:        db,
		cfg:       cfg,
		logger:    log,
		healer:    NewHealer(db, log),
		validator: NewValidator(db, log),
		stopCh:    make(chan struct{}),
	}
}

// Start starts the background healing service.
func (hs *HealingService) Start() {
	if !hs.cfg.Enabled {
		return
	}

	hs.wg.Add(1)
	go hs.backgroundHealingLoop()
	hs.logger.Info("Healing service started (interval: %v)", hs.cfg.Interval)
}

// Stop stops the background healing service and releases the ants heal pool.
func (hs *HealingService) Stop() {
	if !hs.cfg.Enabled {
		return
	}

	close(hs.stopCh)
	hs.wg.Wait()
	hs.healPoolMu.Lock()
	if hs.healPool != nil {
		hs.healPool.Release()
		hs.healPool = nil
	}
	hs.healPoolMu.Unlock()
	hs.logger.Info("Healing service stopped")
}

// HealOnCorruption triggers healing when corruption is detected during read.
func (hs *HealingService) HealOnCorruption(collection string, docID uint64) {
	if !hs.cfg.OnReadCorruption {
		return
	}

	if collection == "" {
		collection = DefaultCollection
	}

	hs.mu.Lock()
	hs.healingStats.OnDemandHealings++
	hs.mu.Unlock()

	hs.logger.Info("Triggering on-demand healing for document %d in collection %s", docID, collection)

	// For simplicity, heal immediately rather than queueing
	// Queue would need to store (collection, docID) pairs
	if err := hs.healer.HealDocument(collection, docID); err != nil {
		hs.logger.Warn("Failed to heal document %d: %v", docID, err)
	}
}

// HealDocument manually heals a specific document.
func (hs *HealingService) HealDocument(collection string, docID uint64) error {
	if collection == "" {
		collection = DefaultCollection
	}

	hs.mu.Lock()
	hs.healingStats.OnDemandHealings++
	hs.mu.Unlock()

	if err := hs.healer.HealDocument(collection, docID); err != nil {
		hs.logger.Warn("Failed to heal document %d in collection %s: %v", docID, collection, err)
		return err
	}

	hs.mu.Lock()
	hs.healingStats.DocumentsHealed++
	hs.healingStats.LastHealingTime = time.Now()
	hs.mu.Unlock()

	hs.logger.Info("Successfully healed document %d in collection %s", docID, collection)
	return nil
}

// getHealPool returns (or creates) the ants pool for parallel healing.
func (hs *HealingService) getHealPool() *ants.PoolWithFunc {
	hs.healPoolMu.Lock()
	defer hs.healPoolMu.Unlock()
	if hs.healPool != nil {
		return hs.healPool
	}
	capacity := hs.cfg.MaxBatchSize
	if capacity <= 0 {
		capacity = 4
	}
	pool, err := ants.NewPoolWithFunc(capacity, func(arg any) {
		t := arg.(*healTaskWithWg)
		if err := hs.healer.HealDocument(t.Collection, t.DocID); err == nil && t.Healed != nil {
			t.HealedMu.Lock()
			*t.Healed = append(*t.Healed, t.DocID)
			t.HealedMu.Unlock()
		}
		t.Wg.Done()
	}, ants.WithPanicHandler(func(v any) {
		hs.logger.Error("healing worker panic: %v", v)
	}))
	if err != nil {
		return nil
	}
	hs.healPool = pool
	return hs.healPool
}

// HealAll triggers a full database healing scan using parallel workers when ants pool is available.
func (hs *HealingService) HealAll() ([]uint64, error) {
	hs.logger.Info("Starting full database healing scan")

	healthMap, err := hs.validator.ValidateAllDocuments()
	if err != nil {
		hs.logger.Error("Failed to validate documents: %v", err)
		return nil, err
	}

	var tasks []*healTask
	for collection, docs := range healthMap {
		for docID, health := range docs {
			if health == HealthCorrupt {
				tasks = append(tasks, &healTask{Collection: collection, DocID: docID})
			}
		}
	}

	healed := make([]uint64, 0, len(tasks))
	pool := hs.getHealPool()
	if pool == nil || len(tasks) == 0 {
		for _, t := range tasks {
			if err := hs.healer.HealDocument(t.Collection, t.DocID); err != nil {
				hs.logger.Warn("Failed to heal document %d in collection %s: %v", t.DocID, t.Collection, err)
				continue
			}
			healed = append(healed, t.DocID)
		}
	} else {
		var wg sync.WaitGroup
		var healedMu sync.Mutex
		for _, t := range tasks {
			wg.Add(1)
			_ = pool.Invoke(&healTaskWithWg{Collection: t.Collection, DocID: t.DocID, Wg: &wg, Healed: &healed, HealedMu: &healedMu})
		}
		wg.Wait()
	}

	hs.mu.Lock()
	hs.healingStats.DocumentsHealed += uint64(len(healed))
	hs.healingStats.LastHealingTime = time.Now()
	hs.mu.Unlock()

	hs.logger.Info("Healed %d documents", len(healed))
	return healed, nil
}

// GetStats returns healing statistics.
func (hs *HealingService) GetStats() HealingStats {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	return hs.healingStats
}

// backgroundHealingLoop runs periodic health scans and processes the healing queue.
func (hs *HealingService) backgroundHealingLoop() {
	defer hs.wg.Done()

	ticker := time.NewTicker(hs.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-hs.stopCh:
			// Process any remaining items in queue before stopping
			hs.processHealingQueue()
			return
		case <-ticker.C:
			hs.performHealthScan()
			hs.processHealingQueue()
		}
	}
}

// performHealthScan performs a periodic health scan of all documents.
// Uses ants pool for parallel healing when available.
func (hs *HealingService) performHealthScan() {
	hs.logger.Debug("Starting periodic health scan")

	healthMap, err := hs.validator.ValidateAllDocuments()
	if err != nil {
		hs.logger.Error("Failed to perform health scan: %v", err)
		return
	}

	var tasks []*healTask
	for collection, docs := range healthMap {
		for docID, health := range docs {
			if health == HealthCorrupt {
				tasks = append(tasks, &healTask{Collection: collection, DocID: docID})
			}
		}
	}

	pool := hs.getHealPool()
	if pool == nil || len(tasks) == 0 {
		for _, t := range tasks {
			_ = hs.healer.HealDocument(t.Collection, t.DocID)
		}
	} else {
		var wg sync.WaitGroup
		var healedMu sync.Mutex
		healed := make([]uint64, 0, len(tasks))
		for _, t := range tasks {
			wg.Add(1)
			_ = pool.Invoke(&healTaskWithWg{Collection: t.Collection, DocID: t.DocID, Wg: &wg, Healed: &healed, HealedMu: &healedMu})
		}
		wg.Wait()
	}

	corruptedCount := len(tasks)
	hs.mu.Lock()
	hs.healingStats.TotalScans++
	hs.healingStats.DocumentsCorrupted = uint64(corruptedCount)
	hs.healingStats.LastScanTime = time.Now()
	hs.mu.Unlock()

	if corruptedCount > 0 {
		hs.logger.Info("Health scan found %d corrupted documents", corruptedCount)
	} else {
		hs.logger.Debug("Health scan completed - no corruption detected")
	}
}

// processHealingQueue processes documents in the healing queue.
// Note: Queue processing simplified - queue stores docIDs but we need collection.
// For v0.2, we'll heal immediately rather than queueing.
func (hs *HealingService) processHealingQueue() {
	// Queue processing is simplified in v0.2
	// Actual healing happens in performHealthScan()
	hs.queueMu.Lock()
	hs.healingQueue = hs.healingQueue[:0] // Clear queue
	hs.queueMu.Unlock()
}

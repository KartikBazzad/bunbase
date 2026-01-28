package docdb

import (
	"sync"
	"time"

	"github.com/kartikbazzad/docdb/internal/config"
	"github.com/kartikbazzad/docdb/internal/logger"
)

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

// Stop stops the background healing service.
func (hs *HealingService) Stop() {
	if !hs.cfg.Enabled {
		return
	}

	close(hs.stopCh)
	hs.wg.Wait()
	hs.logger.Info("Healing service stopped")
}

// HealOnCorruption triggers healing when corruption is detected during read.
func (hs *HealingService) HealOnCorruption(docID uint64) {
	if !hs.cfg.OnReadCorruption {
		return
	}

	hs.mu.Lock()
	hs.healingStats.OnDemandHealings++
	hs.mu.Unlock()

	hs.logger.Info("Triggering on-demand healing for document %d", docID)

	// Add to queue for batch processing
	hs.queueMu.Lock()
	hs.healingQueue = append(hs.healingQueue, docID)
	queueLen := len(hs.healingQueue)
	hs.queueMu.Unlock()

	// If queue is getting large, process immediately
	if queueLen >= hs.cfg.MaxBatchSize {
		hs.processHealingQueue()
	}
}

// HealDocument manually heals a specific document.
func (hs *HealingService) HealDocument(docID uint64) error {
	hs.mu.Lock()
	hs.healingStats.OnDemandHealings++
	hs.mu.Unlock()

	if err := hs.healer.HealDocument(docID); err != nil {
		hs.logger.Warn("Failed to heal document %d: %v", docID, err)
		return err
	}

	hs.mu.Lock()
	hs.healingStats.DocumentsHealed++
	hs.healingStats.LastHealingTime = time.Now()
	hs.mu.Unlock()

	hs.logger.Info("Successfully healed document %d", docID)
	return nil
}

// HealAll triggers a full database healing scan.
func (hs *HealingService) HealAll() ([]uint64, error) {
	hs.logger.Info("Starting full database healing scan")

	healed, err := hs.healer.HealAllCorruptedDocuments()
	if err != nil {
		hs.logger.Error("Failed to heal all documents: %v", err)
		return nil, err
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
func (hs *HealingService) performHealthScan() {
	hs.logger.Debug("Starting periodic health scan")

	healthMap, err := hs.validator.ValidateAllDocuments()
	if err != nil {
		hs.logger.Error("Failed to perform health scan: %v", err)
		return
	}

	corrupted := make([]uint64, 0)
	for docID, health := range healthMap {
		if health == HealthCorrupt {
			corrupted = append(corrupted, docID)
		}
	}

	hs.mu.Lock()
	hs.healingStats.TotalScans++
	hs.healingStats.DocumentsCorrupted = uint64(len(corrupted))
	hs.healingStats.LastScanTime = time.Now()
	hs.mu.Unlock()

	if len(corrupted) > 0 {
		hs.logger.Info("Health scan found %d corrupted documents", len(corrupted))

		// Add to healing queue
		hs.queueMu.Lock()
		hs.healingQueue = append(hs.healingQueue, corrupted...)
		hs.queueMu.Unlock()
	} else {
		hs.logger.Debug("Health scan completed - no corruption detected")
	}
}

// processHealingQueue processes documents in the healing queue.
func (hs *HealingService) processHealingQueue() {
	hs.queueMu.Lock()
	if len(hs.healingQueue) == 0 {
		hs.queueMu.Unlock()
		return
	}

	// Process up to MaxBatchSize documents
	batchSize := hs.cfg.MaxBatchSize
	if batchSize > len(hs.healingQueue) {
		batchSize = len(hs.healingQueue)
	}

	batch := make([]uint64, batchSize)
	copy(batch, hs.healingQueue[:batchSize])
	hs.healingQueue = hs.healingQueue[batchSize:]
	hs.queueMu.Unlock()

	hs.logger.Debug("Processing healing queue: %d documents", len(batch))

	healed := 0
	for _, docID := range batch {
		if err := hs.healer.HealDocument(docID); err != nil {
			hs.logger.Warn("Failed to heal document %d: %v", docID, err)
			continue
		}
		healed++
	}

	hs.mu.Lock()
	hs.healingStats.DocumentsHealed += uint64(healed)
	hs.healingStats.BackgroundHealings += uint64(healed)
	hs.healingStats.LastHealingTime = time.Now()
	hs.mu.Unlock()

	if healed > 0 {
		hs.logger.Info("Healed %d documents from queue", healed)
	}
}

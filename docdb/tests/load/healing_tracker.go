package load

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kartikbazzad/docdb/internal/ipc"
	"github.com/kartikbazzad/docdb/internal/types"
)

// HealingEvent represents a healing operation event.
type HealingEvent struct {
	Timestamp       time.Time
	Duration        time.Duration
	DocumentsHealed int
	Type            string // "on-demand" or "background"
}

// HealingStats represents healing statistics from DocDB.
type HealingStats struct {
	TotalScans         uint64
	DocumentsHealed    uint64
	DocumentsCorrupted uint64
	OnDemandHealings   uint64
	BackgroundHealings uint64
	LastScanTime       time.Time
	LastHealingTime    time.Time
}

// HealingTracker tracks healing overhead.
type HealingTracker struct {
	mu               sync.RWMutex
	events           []HealingEvent
	initialStats     *HealingStats
	finalStats       *HealingStats
	client           HealingStatsClient
	dbID             uint64
	totalHealingTime time.Duration
}

// HealingStatsClient is an interface for getting healing statistics.
type HealingStatsClient interface {
	GetHealingStats(dbID uint64) (*HealingStats, error)
}

// NewHealingTracker creates a new healing tracker.
func NewHealingTracker(client HealingStatsClient, dbID uint64) *HealingTracker {
	return &HealingTracker{
		events: make([]HealingEvent, 0),
		client: client,
		dbID:   dbID,
	}
}

// Start records initial healing statistics.
func (ht *HealingTracker) Start() error {
	stats, err := ht.client.GetHealingStats(ht.dbID)
	if err != nil {
		return fmt.Errorf("failed to get initial healing stats: %w", err)
	}

	ht.mu.Lock()
	defer ht.mu.Unlock()

	ht.initialStats = stats
	return nil
}

// Stop records final healing statistics.
func (ht *HealingTracker) Stop() error {
	stats, err := ht.client.GetHealingStats(ht.dbID)
	if err != nil {
		return fmt.Errorf("failed to get final healing stats: %w", err)
	}

	ht.mu.Lock()
	defer ht.mu.Unlock()

	ht.finalStats = stats
	return nil
}

// RecordEvent records a healing event.
func (ht *HealingTracker) RecordEvent(event HealingEvent) {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	ht.events = append(ht.events, event)
	ht.totalHealingTime += event.Duration
}

// GetSummary returns summary statistics about healing overhead.
func (ht *HealingTracker) GetSummary(totalDuration time.Duration) HealingSummary {
	ht.mu.RLock()
	defer ht.mu.RUnlock()

	var totalHealings uint64
	var totalDocumentsHealed uint64

	if ht.initialStats != nil && ht.finalStats != nil {
		totalHealings = ht.finalStats.OnDemandHealings + ht.finalStats.BackgroundHealings -
			(ht.initialStats.OnDemandHealings + ht.initialStats.BackgroundHealings)
		totalDocumentsHealed = ht.finalStats.DocumentsHealed - ht.initialStats.DocumentsHealed
	}

	overheadPercent := 0.0
	if totalDuration > 0 {
		overheadPercent = (float64(ht.totalHealingTime) / float64(totalDuration)) * 100.0
	}

	return HealingSummary{
		TotalHealings:        totalHealings,
		TotalDocumentsHealed: totalDocumentsHealed,
		HealingTimeSeconds:   ht.totalHealingTime.Seconds(),
		OverheadPercent:      overheadPercent,
		EventCount:           len(ht.events),
		InitialStats:         ht.initialStats,
		FinalStats:           ht.finalStats,
	}
}

// GetEvents returns all healing events.
func (ht *HealingTracker) GetEvents() []HealingEvent {
	ht.mu.RLock()
	defer ht.mu.RUnlock()

	result := make([]HealingEvent, len(ht.events))
	copy(result, ht.events)
	return result
}

// HealingSummary contains summary statistics about healing overhead.
type HealingSummary struct {
	TotalHealings        uint64
	TotalDocumentsHealed uint64
	HealingTimeSeconds   float64
	OverheadPercent      float64
	EventCount           int
	InitialStats         *HealingStats
	FinalStats           *HealingStats
}

// IPCHealingStatsClient implements HealingStatsClient using IPC protocol.
type IPCHealingStatsClient struct {
	socketPath string
	conn       *ipcConnection
}

// NewIPCHealingStatsClient creates a new IPC-based healing stats client.
func NewIPCHealingStatsClient(socketPath string) *IPCHealingStatsClient {
	return &IPCHealingStatsClient{
		socketPath: socketPath,
	}
}

// GetHealingStats retrieves healing statistics via IPC.
func (c *IPCHealingStatsClient) GetHealingStats(dbID uint64) (*HealingStats, error) {
	if c.conn == nil {
		conn, err := newIPCConnection(c.socketPath)
		if err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
		c.conn = conn
	}

	frame := &ipc.RequestFrame{
		RequestID: c.conn.nextRequestID(),
		Command:   ipc.CmdHealStats,
		DBID:      dbID,
		OpCount:   0,
		Ops:       nil,
	}

	resp, err := c.conn.sendRequest(frame)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.Status != types.StatusOK {
		return nil, fmt.Errorf("healing stats request failed: %s", string(resp.Data))
	}

	var statsMap map[string]interface{}
	if err := json.Unmarshal(resp.Data, &statsMap); err != nil {
		return nil, fmt.Errorf("failed to parse healing stats: %w", err)
	}

	stats := &HealingStats{}
	if v, ok := statsMap["TotalScans"].(float64); ok {
		stats.TotalScans = uint64(v)
	}
	if v, ok := statsMap["DocumentsHealed"].(float64); ok {
		stats.DocumentsHealed = uint64(v)
	}
	if v, ok := statsMap["DocumentsCorrupted"].(float64); ok {
		stats.DocumentsCorrupted = uint64(v)
	}
	if v, ok := statsMap["OnDemandHealings"].(float64); ok {
		stats.OnDemandHealings = uint64(v)
	}
	if v, ok := statsMap["BackgroundHealings"].(float64); ok {
		stats.BackgroundHealings = uint64(v)
	}

	if lastScanStr, ok := statsMap["LastScanTime"].(string); ok && lastScanStr != "" {
		if t, err := time.Parse(time.RFC3339, lastScanStr); err == nil {
			stats.LastScanTime = t
		}
	}

	if lastHealingStr, ok := statsMap["LastHealingTime"].(string); ok && lastHealingStr != "" {
		if t, err := time.Parse(time.RFC3339, lastHealingStr); err == nil {
			stats.LastHealingTime = t
		}
	}

	return stats, nil
}

// Close closes the IPC connection.
func (c *IPCHealingStatsClient) Close() error {
	if c.conn != nil {
		return c.conn.close()
	}
	return nil
}

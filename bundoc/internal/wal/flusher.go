package wal

import (
	"sync"
	"sync/atomic"
	"time"
)

// SharedFlusher is a global singleton that performs all fsync operations
// This minimizes fsync bottlenecks across multiple databases
type SharedFlusher struct {
	requests     chan *FlushRequest
	batchSize    int
	batchTimeout time.Duration
	stopped      atomic.Bool
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

// FlushRequest represents a request to flush a WAL
type FlushRequest struct {
	WAL      *WAL
	Response chan error
}

var (
	globalFlusher     *SharedFlusher
	globalFlusherOnce sync.Once
)

// GetSharedFlusher returns the global shared flusher (singleton)
func GetSharedFlusher() *SharedFlusher {
	globalFlusherOnce.Do(func() {
		globalFlusher = &SharedFlusher{
			requests:     make(chan *FlushRequest, 10000),
			batchSize:    1000,                 // Max 1000 flushes per batch
			batchTimeout: time.Millisecond * 5, // Max 5ms wait
			stopChan:     make(chan struct{}),
		}
		globalFlusher.wg.Add(1)
		go globalFlusher.run()
	})
	return globalFlusher
}

// Flush submits a flush request for a WAL
func (sf *SharedFlusher) Flush(wal *WAL) error {
	if sf.stopped.Load() {
		return ErrFlusherStopped
	}

	req := &FlushRequest{
		WAL:      wal,
		Response: make(chan error, 1),
	}

	// Send request
	select {
	case sf.requests <- req:
	case <-sf.stopChan:
		return ErrFlusherStopped
	}

	// Wait for response
	return <-req.Response
}

// run processes flush requests in batches
func (sf *SharedFlusher) run() {
	defer sf.wg.Done()

	var batch []*FlushRequest
	timer := time.NewTimer(sf.batchTimeout)
	defer timer.Stop()

	for {
		select {
		case req := <-sf.requests:
			batch = append(batch, req)

			// If batch is full, flush immediately
			if len(batch) >= sf.batchSize {
				sf.flushBatch(batch)
				batch = nil
				timer.Reset(sf.batchTimeout)
			}

		case <-timer.C:
			// Timeout - flush whatever we have
			if len(batch) > 0 {
				sf.flushBatch(batch)
				batch = nil
			}
			timer.Reset(sf.batchTimeout)

		case <-sf.stopChan:
			// Flush remaining batch before stopping
			if len(batch) > 0 {
				sf.flushBatch(batch)
			}
			return
		}
	}
}

// flushBatch flushes a batch of WALs
func (sf *SharedFlusher) flushBatch(batch []*FlushRequest) {
	// Group requests by WAL to avoid duplicate syncs
	walMap := make(map[*WAL][]*FlushRequest)
	for _, req := range batch {
		walMap[req.WAL] = append(walMap[req.WAL], req)
	}

	// Flush each unique WAL once
	for wal, requests := range walMap {
		err := wal.Sync()

		// Respond to all requests for this WAL
		for _, req := range requests {
			req.Response <- err
		}
	}
}

// Stop stops the shared flusher
func (sf *SharedFlusher) Stop() {
	if sf.stopped.Swap(true) {
		return // Already stopped
	}

	close(sf.stopChan)
	sf.wg.Wait()
}

// GetStats returns statistics about the shared flusher
func (sf *SharedFlusher) GetStats() Stats {
	return Stats{
		QueueDepth: len(sf.requests),
		BatchSize:  sf.batchSize,
		IsStopped:  sf.stopped.Load(),
	}
}

// Stats contains statistics about the flusher
type Stats struct {
	QueueDepth int
	BatchSize  int
	IsStopped  bool
}

// ErrFlusherStopped is returned when the shared flusher is stopped
var ErrFlusherStopped = &FlusherError{msg: "shared flusher stopped"}

// FlusherError represents a flusher error
type FlusherError struct {
	msg string
}

func (e *FlusherError) Error() string {
	return e.msg
}

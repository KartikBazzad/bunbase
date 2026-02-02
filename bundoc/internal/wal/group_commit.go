package wal

import (
	"sync"
	"time"
)

// CommitRequest represents a request to commit a transaction
type CommitRequest struct {
	LSN      LSN
	Response chan error
}

// GroupCommitter reduces disk I/O overhead by batching multiple commit requests (fsync)
// into a single system call.
//
// How it works:
// 1. Transactions request a commit by sending a request to the channel.
// 2. The background goroutine collects requests into a batch.
// 3. The batch is flushed when:
//   - The batch size limit is reached.
//   - The timeout triggers (latency bound).
//   - The incoming channel is empty (immediate flush for low load).
//
// 4. A single WAL.Sync() is performed.
// 5. All waiting transactions in the batch are notified.
type GroupCommitter struct {
	wal          *WAL
	requests     chan *CommitRequest
	batchSize    int
	batchTimeout time.Duration
	mu           sync.Mutex
	stopped      bool
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

// NewGroupCommitter creates a new group committer
func NewGroupCommitter(wal *WAL) *GroupCommitter {
	gc := &GroupCommitter{
		wal:          wal,
		requests:     make(chan *CommitRequest, 1000),
		batchSize:    100,                   // Max 100 commits per batch
		batchTimeout: time.Millisecond * 10, // Max 10ms wait
		stopChan:     make(chan struct{}),
	}

	gc.wg.Add(1)
	go gc.run()

	return gc
}

// Commit submits a commit request and waits for it to be flushed
func (gc *GroupCommitter) Commit(lsn LSN) error {
	gc.mu.Lock()
	if gc.stopped {
		gc.mu.Unlock()
		return ErrCommitterStopped
	}
	gc.mu.Unlock()

	req := &CommitRequest{
		LSN:      lsn,
		Response: make(chan error, 1),
	}

	// Send request
	select {
	case gc.requests <- req:
	case <-gc.stopChan:
		return ErrCommitterStopped
	}

	// Wait for response
	return <-req.Response
}

// run processes commit requests in batches
func (gc *GroupCommitter) run() {
	defer gc.wg.Done()

	var batch []*CommitRequest
	timer := time.NewTimer(gc.batchTimeout)
	defer timer.Stop()

	for {
		select {
		case req := <-gc.requests:
			batch = append(batch, req)

			// If batch is full OR channel is empty (no immediate followers), flush immediately
			// This optimizes latency for serial/low-throughput workloads while maintaining
			// group commit for high-throughput bursts.
			if len(batch) >= gc.batchSize || len(gc.requests) == 0 {
				gc.flushBatch(batch)
				batch = nil
				timer.Reset(gc.batchTimeout)
			}

		case <-timer.C:
			// Timeout - flush whatever we have
			if len(batch) > 0 {
				gc.flushBatch(batch)
				batch = nil
			}
			timer.Reset(gc.batchTimeout)

		case <-gc.stopChan:
			// Flush remaining batch before stopping
			if len(batch) > 0 {
				gc.flushBatch(batch)
			}
			return
		}
	}
}

// flush Batch flushes a batch of commit requests
func (gc *GroupCommitter) flushBatch(batch []*CommitRequest) {
	// Perform single fsync for entire batch
	err := gc.wal.Sync()

	// Respond to all requests in batch
	for _, req := range batch {
		req.Response <- err
	}
}

// Stop stops the group committer
func (gc *GroupCommitter) Stop() {
	gc.mu.Lock()
	if gc.stopped {
		gc.mu.Unlock()
		return
	}
	gc.stopped = true
	gc.mu.Unlock()

	close(gc.stopChan)
	gc.wg.Wait()
}

// ErrCommitterStopped is returned when the group committer is stopped
var ErrCommitterStopped = &CommitError{msg: "group committer stopped"}

// CommitError represents a commit error
type CommitError struct {
	msg string
}

func (e *CommitError) Error() string {
	return e.msg
}

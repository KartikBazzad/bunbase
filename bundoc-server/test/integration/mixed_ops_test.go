package integration

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

// TestMixedOps_ReadHeavyWorkload simulates read-heavy pattern (80% reads, 20% writes)
func TestMixedOps_ReadHeavyWorkload(t *testing.T) {
	const (
		numProjects  = 5
		numWorkers   = 50
		opsPerWorker = 100
	)

	projectName := "read-heavy-test"

	// Seed initial data
	t.Log("Seeding initial data...")
	for i := 0; i < 100; i++ {
		doc := storage.Document{
			"_id":   fmt.Sprintf("seed-%d", i),
			"value": i,
			"type":  "seed",
		}
		if err := createDocument(projectName, "data", doc); err != nil {
			t.Fatalf("Failed to seed: %v", err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	var wg sync.WaitGroup
	var reads, writes, updates atomic.Int32
	var errors atomic.Int32

	// 80% readers, 20% writers
	readWorkers := int(float64(numWorkers) * 0.8)
	writeWorkers := numWorkers - readWorkers

	// Readers
	for i := 0; i < readWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

			for op := 0; op < opsPerWorker; op++ {
				docID := fmt.Sprintf("seed-%d", r.Intn(100))
				_, err := getDocument(projectName, "data", docID)
				if err == nil {
					reads.Add(1)
				} else {
					errors.Add(1)
				}
			}
		}(i)
	}

	// Writers
	for i := 0; i < writeWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

			for op := 0; op < opsPerWorker; op++ {
				// 50% creates, 50% updates
				if r.Float64() < 0.5 {
					// Create
					doc := storage.Document{
						"_id":    fmt.Sprintf("writer-%d-op-%d", workerID, op),
						"worker": workerID,
						"value":  op,
					}
					if err := createDocument(projectName, "data", doc); err == nil {
						writes.Add(1)
					} else {
						errors.Add(1)
					}
				} else {
					// Update existing
					docID := fmt.Sprintf("seed-%d", r.Intn(100))
					update := storage.Document{
						"_id":     docID,
						"updated": true,
						"ts":      time.Now().Unix(),
					}
					if err := updateDocument(projectName, "data", docID, update); err == nil {
						updates.Add(1)
					}
					// Failures are ok (doc might not exist)
				}
			}
		}(i)
	}

	wg.Wait()

	totalOps := reads.Load() + writes.Load() + updates.Load()
	t.Logf("📊 Read-Heavy Workload:")
	t.Logf("  Reads: %d", reads.Load())
	t.Logf("  Writes: %d", writes.Load())
	t.Logf("  Updates: %d", updates.Load())
	t.Logf("  Total: %d", totalOps)
	t.Logf("  Errors: %d", errors.Load())

	if totalOps == 0 {
		t.Error("No operations completed")
	}
}

// TestMixedOps_WriteHeavyWorkload simulates write-heavy pattern (80% writes, 20% reads)
func TestMixedOps_WriteHeavyWorkload(t *testing.T) {
	const (
		numWorkers   = 50
		opsPerWorker = 100
	)

	projectName := "write-heavy-test"

	var wg sync.WaitGroup
	var creates, updates, deletes, reads atomic.Int32

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

			for op := 0; op < opsPerWorker; op++ {
				roll := r.Float64()

				if roll < 0.4 {
					// 40% Creates
					doc := storage.Document{
						"_id":    fmt.Sprintf("w%d-doc%d", workerID, op),
						"worker": workerID,
						"op":     op,
					}
					if err := createDocument(projectName, "writes", doc); err == nil {
						creates.Add(1)
					}
				} else if roll < 0.7 {
					// 30% Updates
					docID := fmt.Sprintf("w%d-doc%d", workerID, r.Intn(op+1))
					update := storage.Document{
						"_id":     docID,
						"updated": true,
					}
					if err := updateDocument(projectName, "writes", docID, update); err == nil {
						updates.Add(1)
					}
				} else if roll < 0.85 {
					// 15% Deletes
					docID := fmt.Sprintf("w%d-doc%d", workerID, r.Intn(op+1))
					if err := deleteDocument(projectName, "writes", docID); err == nil {
						deletes.Add(1)
					}
				} else {
					// 15% Reads
					docID := fmt.Sprintf("w%d-doc%d", workerID, r.Intn(op+1))
					if _, err := getDocument(projectName, "writes", docID); err == nil {
						reads.Add(1)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	total := creates.Load() + updates.Load() + deletes.Load() + reads.Load()
	t.Logf("📊 Write-Heavy Workload:")
	t.Logf("  Creates: %d (%.1f%%)", creates.Load(), float64(creates.Load())/float64(total)*100)
	t.Logf("  Updates: %d (%.1f%%)", updates.Load(), float64(updates.Load())/float64(total)*100)
	t.Logf("  Deletes: %d (%.1f%%)", deletes.Load(), float64(deletes.Load())/float64(total)*100)
	t.Logf("  Reads: %d (%.1f%%)", reads.Load(), float64(reads.Load())/float64(total)*100)
	t.Logf("  Total: %d", total)

	if total == 0 {
		t.Error("No operations completed")
	}
}

// TestMixedOps_ConcurrentUpdates tests multiple workers updating same documents
func TestMixedOps_ConcurrentUpdates(t *testing.T) {
	const (
		numDocs          = 10
		numWorkers       = 50
		updatesPerWorker = 20
	)

	projectName := "concurrent-updates-test"

	// Create initial documents
	t.Log("Creating initial documents...")
	for i := 0; i < numDocs; i++ {
		doc := storage.Document{
			"_id":     fmt.Sprintf("shared-%d", i),
			"counter": 0,
		}
		if err := createDocument(projectName, "counters", doc); err != nil {
			t.Fatalf("Failed to create doc: %v", err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	var wg sync.WaitGroup
	var successfulUpdates atomic.Int32

	// Multiple workers updating same documents
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

			for u := 0; u < updatesPerWorker; u++ {
				docID := fmt.Sprintf("shared-%d", r.Intn(numDocs))

				update := storage.Document{
					"_id":     docID,
					"worker":  workerID,
					"updated": time.Now().UnixNano(),
				}

				if err := updateDocument(projectName, "counters", docID, update); err == nil {
					successfulUpdates.Add(1)
				}

				// Small delay to reduce contention
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	expectedUpdates := int32(numWorkers * updatesPerWorker)
	t.Logf("📊 Concurrent Updates:")
	t.Logf("  Documents: %d", numDocs)
	t.Logf("  Workers: %d", numWorkers)
	t.Logf("  Successful Updates: %d/%d", successfulUpdates.Load(), expectedUpdates)

	if successfulUpdates.Load() == 0 {
		t.Error("No updates succeeded")
	}

	// Verify all documents still exist and have been updated
	for i := 0; i < numDocs; i++ {
		docID := fmt.Sprintf("shared-%d", i)
		doc, err := getDocument(projectName, "counters", docID)
		if err != nil {
			t.Errorf("Document %s not found after updates", docID)
		} else if doc["worker"] == nil {
			t.Errorf("Document %s was never updated", docID)
		}
	}
}

// TestMixedOps_MultiProjectChaos tests chaotic mix across multiple projects
func TestMixedOps_MultiProjectChaos(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	const (
		numProjects = 10
		numWorkers  = 100
		duration    = 5 * time.Second
	)

	var wg sync.WaitGroup
	var creates, reads, updates, deletes atomic.Int32
	var errors atomic.Int32

	stopChan := make(chan struct{})
	startTime := time.Now()

	// Stop after duration
	go func() {
		time.Sleep(duration)
		close(stopChan)
	}()

	// Workers performing random operations
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

			for {
				select {
				case <-stopChan:
					return
				default:
					projectName := fmt.Sprintf("chaos-project-%d", r.Intn(numProjects))
					collection := fmt.Sprintf("col-%d", r.Intn(5))
					docID := fmt.Sprintf("doc-%d", r.Intn(50))

					operation := r.Intn(4)
					switch operation {
					case 0: // Create
						doc := storage.Document{
							"_id":    docID,
							"worker": workerID,
							"ts":     time.Now().Unix(),
						}
						if err := createDocument(projectName, collection, doc); err == nil {
							creates.Add(1)
						} else {
							errors.Add(1)
						}

					case 1: // Read
						if _, err := getDocument(projectName, collection, docID); err == nil {
							reads.Add(1)
						}

					case 2: // Update
						update := storage.Document{
							"_id":     docID,
							"updated": true,
							"ts":      time.Now().Unix(),
						}
						if err := updateDocument(projectName, collection, docID, update); err == nil {
							updates.Add(1)
						}

					case 3: // Delete
						if err := deleteDocument(projectName, collection, docID); err == nil {
							deletes.Add(1)
						}
					}

					// Small delay
					time.Sleep(5 * time.Millisecond)
				}
			}
		}(i)
	}

	wg.Wait()

	elapsedTime := time.Since(startTime)
	total := creates.Load() + reads.Load() + updates.Load() + deletes.Load()
	throughput := float64(total) / elapsedTime.Seconds()

	t.Logf("📊 Multi-Project Chaos Test:")
	t.Logf("  Duration: %v", elapsedTime)
	t.Logf("  Projects: %d", numProjects)
	t.Logf("  Workers: %d", numWorkers)
	t.Logf("  Creates: %d", creates.Load())
	t.Logf("  Reads: %d", reads.Load())
	t.Logf("  Updates: %d", updates.Load())
	t.Logf("  Deletes: %d", deletes.Load())
	t.Logf("  Total Operations: %d", total)
	t.Logf("  Throughput: %.2f ops/sec", throughput)
	t.Logf("  Errors: %d", errors.Load())

	if total == 0 {
		t.Error("No operations completed")
	}

	// Log distribution
	t.Logf("  Distribution:")
	t.Logf("    Creates: %.1f%%", float64(creates.Load())/float64(total)*100)
	t.Logf("    Reads: %.1f%%", float64(reads.Load())/float64(total)*100)
	t.Logf("    Updates: %.1f%%", float64(updates.Load())/float64(total)*100)
	t.Logf("    Deletes: %.1f%%", float64(deletes.Load())/float64(total)*100)
}

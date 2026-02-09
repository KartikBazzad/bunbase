package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

// serverURL is set by TestMain in setup_test.go (dynamic port).

// TestMatrix_ConcurrentProjects tests multiple projects with concurrent operations
func TestMatrix_ConcurrentProjects(t *testing.T) {
	const (
		numProjects    = 10
		docsPerProject = 500 // 1,000 total documents
	)

	var wg sync.WaitGroup
	var createCount atomic.Int32
	var errorCount atomic.Int32

	// Concurrent creates across multiple projects
	for projID := 0; projID < numProjects; projID++ {
		wg.Add(1)
		go func(projectID int) {
			defer wg.Done()

			projectName := fmt.Sprintf("matrix-project-%d", projectID)

			// Create documents
			for docID := 0; docID < docsPerProject; docID++ {
				doc := storage.Document{
					"_id":       fmt.Sprintf("doc-%d", docID),
					"projectID": projectID,
					"value":     docID,
				}

				err := createDocument(projectName, "items", doc)
				if err != nil {
					errorCount.Add(1)
					t.Errorf("Project %s: Create failed: %v", projectName, err)
					continue
				}

				createCount.Add(1)
			}
		}(projID)
	}

	wg.Wait()

	total := int32(numProjects * docsPerProject)
	t.Logf("Created: %d/%d documents across %d projects", createCount.Load(), total, numProjects)
	t.Logf("Errors: %d", errorCount.Load())

	if errorCount.Load() > 0 {
		t.Errorf("Had %d errors during creation", errorCount.Load())
	}

	if createCount.Load() < total {
		t.Errorf("Expected %d creates, got %d", total, createCount.Load())
	}

	// Spot-check: verify we can read a few documents from each project
	t.Log("Spot-checking reads...")
	var readErrors atomic.Int32
	for projID := 0; projID < numProjects; projID++ {
		wg.Add(1)
		go func(projectID int) {
			defer wg.Done()

			projectName := fmt.Sprintf("matrix-project-%d", projectID)

			// Read first and last document
			for _, docID := range []int{0, docsPerProject - 1} {
				doc, err := getDocument(projectName, "items", fmt.Sprintf("doc-%d", docID))
				if err != nil {
					readErrors.Add(1)
					t.Errorf("Project %s: spot-check read failed: %v", projectName, err)
					continue
				}

				// Verify isolation
				if int(doc["projectID"].(float64)) != projectID {
					t.Errorf("Isolation violated! Expected projectID=%d, got %v", projectID, doc["projectID"])
				}
			}
		}(projID)
	}

	wg.Wait()

	if readErrors.Load() == 0 {
		t.Log(" Isolation and persistence verified!")
	}
}

// TestMatrix_ConcurrentCRUD tests all CRUD operations concurrently on same project
func TestMatrix_ConcurrentCRUD(t *testing.T) {
	const (
		numWriters  = 100 // Create documents
		numReaders  = 100 // Read documents
		numUpdaters = 100 // Update documents
		numDeleters = 50  // Delete documents (~400 total ops)
	)

	projectName := "crud-test-project"
	var wg sync.WaitGroup
	var operations atomic.Int32

	// Writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()

			doc := storage.Document{
				"_id":   fmt.Sprintf("writer-%d", writerID),
				"type":  "writer",
				"value": writerID,
			}

			if err := createDocument(projectName, "crud", doc); err != nil {
				t.Errorf("Writer %d failed: %v", writerID, err)
				return
			}
			operations.Add(1)
		}(i)
	}

	// Small delay to let writers create some docs
	time.Sleep(100 * time.Millisecond)

	// Readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()

			writerID := readerID % numWriters
			docID := fmt.Sprintf("writer-%d", writerID)

			_, err := getDocument(projectName, "crud", docID)
			if err != nil {
				// May not exist yet, that's ok
				return
			}
			operations.Add(1)
		}(i)
	}

	// Updaters
	for i := 0; i < numUpdaters; i++ {
		wg.Add(1)
		go func(updaterID int) {
			defer wg.Done()

			writerID := updaterID % numWriters
			docID := fmt.Sprintf("writer-%d", writerID)

			update := storage.Document{
				"_id":     docID,
				"updated": true,
				"value":   updaterID * 100,
			}

			if err := updateDocument(projectName, "crud", docID, update); err != nil {
				// May not exist yet
				return
			}
			operations.Add(1)
		}(i)
	}

	// Deleters (delete last few)
	for i := 0; i < numDeleters; i++ {
		wg.Add(1)
		go func(deleterID int) {
			defer wg.Done()

			writerID := numWriters - 1 - deleterID
			docID := fmt.Sprintf("writer-%d", writerID)

			time.Sleep(100 * time.Millisecond) // Wait for creation

			if err := deleteDocument(projectName, "crud", docID); err != nil {
				return
			}
			operations.Add(1)
		}(i)
	}

	wg.Wait()

	t.Logf("Total operations completed: %d", operations.Load())

	if operations.Load() == 0 {
		t.Error("No operations completed successfully")
	}
}

// TestMatrix_IsolationGuarantee verifies strict isolation between projects
func TestMatrix_IsolationGuarantee(t *testing.T) {
	const numProjects = 20 // Sufficient to verify isolation

	// Create same document ID in multiple projects with different data
	for i := 0; i < numProjects; i++ {
		projectName := fmt.Sprintf("isolation-test-%d", i)

		doc := storage.Document{
			"_id":       "shared-id",
			"project":   projectName,
			"projectID": i,
			"secret":    fmt.Sprintf("secret-%d", i),
		}

		err := createDocument(projectName, "isolated", doc)
		if err != nil {
			t.Fatalf("Failed to create in project %s: %v", projectName, err)
		}
	}

	// Verify each project only sees its own data
	for i := 0; i < numProjects; i++ {
		projectName := fmt.Sprintf("isolation-test-%d", i)

		retrieved, err := getDocument(projectName, "isolated", "shared-id")
		if err != nil {
			t.Fatalf("Failed to get from project %s: %v", projectName, err)
		}

		// Check isolation
		if retrieved["project"] != projectName {
			t.Errorf("Isolation violated! Expected project=%s, got %v", projectName, retrieved["project"])
		}

		if int(retrieved["projectID"].(float64)) != i {
			t.Errorf("Isolation violated! Expected projectID=%d, got %v", i, retrieved["projectID"])
		}

		expectedSecret := fmt.Sprintf("secret-%d", i)
		if retrieved["secret"] != expectedSecret {
			t.Errorf("Isolation violated! Expected secret=%s, got %v", expectedSecret, retrieved["secret"])
		}
	}

	t.Log("✅ Isolation guarantee verified across all projects")
}

// TestMatrix_HighConcurrency tests server with very high concurrent load
func TestMatrix_HighConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high concurrency test in short mode")
	}

	const (
		numWorkers   = 100
		opsPerWorker = 100 // 10,000 total operations
	)

	var wg sync.WaitGroup
	var successCount atomic.Int32
	var errorCount atomic.Int32

	startTime := time.Now()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			projectName := fmt.Sprintf("load-test-%d", workerID%10) // 10 projects

			for opID := 0; opID < opsPerWorker; opID++ {
				doc := storage.Document{
					"_id":    fmt.Sprintf("worker-%d-op-%d", workerID, opID),
					"worker": workerID,
					"op":     opID,
				}

				err := createDocument(projectName, "load", doc)
				if err != nil {
					errorCount.Add(1)
					continue
				}

				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	duration := time.Since(startTime)
	totalOps := int32(numWorkers * opsPerWorker)
	throughput := float64(successCount.Load()) / duration.Seconds()

	t.Logf("Duration: %v", duration)
	t.Logf("Throughput: %.2f ops/sec", throughput)
	t.Logf("Success: %d/%d", successCount.Load(), totalOps)
	t.Logf("Errors: %d", errorCount.Load())

	if errorCount.Load() > int32(totalOps/10) { // Allow <10% errors
		t.Errorf("Too many errors: %d (%.1f%%)", errorCount.Load(), float64(errorCount.Load())/float64(totalOps)*100)
	}
}

// Helper functions

func createDocument(projectID, collection string, doc storage.Document) error {
	url := fmt.Sprintf("%s/v1/projects/%s/databases/(default)/documents/%s", serverURL, projectID, collection)

	body, _ := json.Marshal(doc)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, body)
	}

	return nil
}

func getDocument(projectID, collection, docID string) (storage.Document, error) {
	url := fmt.Sprintf("%s/v1/projects/%s/databases/(default)/documents/%s/%s", serverURL, projectID, collection, docID)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, body)
	}

	var doc storage.Document
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}

	return doc, nil
}

func updateDocument(projectID, collection, docID string, update storage.Document) error {
	url := fmt.Sprintf("%s/v1/projects/%s/databases/(default)/documents/%s/%s", serverURL, projectID, collection, docID)

	body, _ := json.Marshal(update)
	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, body)
	}

	return nil
}

func deleteDocument(projectID, collection, docID string) error {
	url := fmt.Sprintf("%s/v1/projects/%s/databases/(default)/documents/%s/%s", serverURL, projectID, collection, docID)

	req, _ := http.NewRequest("DELETE", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, body)
	}

	return nil
}

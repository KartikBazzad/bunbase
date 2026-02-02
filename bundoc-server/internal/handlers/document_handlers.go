package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/kartikbazzad/bunbase/bundoc-server/internal/manager"
	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

// DocumentHandlers handles HTTP requests for document operations
type DocumentHandlers struct {
	manager *manager.InstanceManager
}

// NewDocumentHandlers creates document handlers
func NewDocumentHandlers(mgr *manager.InstanceManager) *DocumentHandlers {
	return &DocumentHandlers{manager: mgr}
}

// HandleCreateDocument creates a new document
// POST /v1/projects/{projectId}/databases/(default)/documents/{collection}
func (h *DocumentHandlers) HandleCreateDocument(w http.ResponseWriter, r *http.Request) {
	projectID, collection := h.parseProjectAndCollection(r.URL.Path)
	if projectID == "" || collection == "" {
		h.writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	// Parse document
	var doc storage.Document
	if err := json.Unmarshal(body, &doc); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Acquire database instance
	db, release, err := h.manager.Acquire(projectID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer release()

	// Get or create collection
	coll, err := db.GetCollection(collection)
	if err != nil {
		coll, err = db.CreateCollection(collection)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// Start transaction
	txn, err := db.BeginTransaction(mvcc.ReadCommitted)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Insert document
	err = coll.Insert(txn, doc)
	if err != nil {
		db.RollbackTransaction(txn)
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Commit transaction
	err = db.CommitTransaction(txn)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return created document
	h.writeJSON(w, http.StatusCreated, doc)
}

// HandleGetDocument retrieves a document by ID
// GET /v1/projects/{projectId}/databases/(default)/documents/{collection}/{docId}
func (h *DocumentHandlers) HandleGetDocument(w http.ResponseWriter, r *http.Request) {
	projectID, collection, docID := h.parseProjectCollectionAndDoc(r.URL.Path)
	if projectID == "" || collection == "" || docID == "" {
		h.writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	// Acquire database instance
	db, release, err := h.manager.Acquire(projectID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer release()

	// Get collection
	coll, err := db.GetCollection(collection)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "collection not found")
		return
	}

	// Start transaction
	txn, err := db.BeginTransaction(mvcc.ReadCommitted)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.CommitTransaction(txn)

	// Find document
	doc, err := coll.FindByID(txn, docID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "document not found")
		return
	}

	h.writeJSON(w, http.StatusOK, doc)
}

// HandleUpdateDocument updates a document
// PATCH /v1/projects/{projectId}/databases/(default)/documents/{collection}/{docId}
func (h *DocumentHandlers) HandleUpdateDocument(w http.ResponseWriter, r *http.Request) {
	projectID, collection, docID := h.parseProjectCollectionAndDoc(r.URL.Path)
	if projectID == "" || collection == "" || docID == "" {
		h.writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	// Parse update data
	var updates storage.Document
	if err := json.Unmarshal(body, &updates); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Ensure _id matches
	updates.SetID(storage.DocumentID(docID))

	// Acquire database instance
	db, release, err := h.manager.Acquire(projectID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer release()

	// Get collection
	coll, err := db.GetCollection(collection)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "collection not found")
		return
	}

	// Start transaction
	txn, err := db.BeginTransaction(mvcc.ReadCommitted)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Update document
	err = coll.Update(txn, docID, updates)
	if err != nil {
		db.RollbackTransaction(txn)
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Commit transaction
	err = db.CommitTransaction(txn)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, updates)
}

// HandleDeleteDocument deletes a document
// DELETE /v1/projects/{projectId}/databases/(default)/documents/{collection}/{docId}
func (h *DocumentHandlers) HandleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	projectID, collection, docID := h.parseProjectCollectionAndDoc(r.URL.Path)
	if projectID == "" || collection == "" || docID == "" {
		h.writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	// Acquire database instance
	db, release, err := h.manager.Acquire(projectID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer release()

	// Get collection
	coll, err := db.GetCollection(collection)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "collection not found")
		return
	}

	// Start transaction
	txn, err := db.BeginTransaction(mvcc.ReadCommitted)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Delete document
	err = coll.Delete(txn, docID)
	if err != nil {
		db.RollbackTransaction(txn)
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Commit transaction
	err = db.CommitTransaction(txn)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions

func (h *DocumentHandlers) parseProjectAndCollection(path string) (string, string) {
	// /v1/projects/{projectId}/databases/(default)/documents/{collection}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 {
		return "", ""
	}
	return parts[2], parts[6]
}

func (h *DocumentHandlers) parseProjectCollectionAndDoc(path string) (string, string, string) {
	// /v1/projects/{projectId}/databases/(default)/documents/{collection}/{docId}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 8 {
		return "", "", ""
	}
	return parts[2], parts[6], parts[7]
}

func (h *DocumentHandlers) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *DocumentHandlers) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{
		"error": message,
	})
}

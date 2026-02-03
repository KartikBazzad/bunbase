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

// HandleGetDocument retrieves a document by ID or lists documents
// GET /v1/projects/{projectId}/databases/(default)/documents/{collection}/{docId}
// GET /v1/projects/{projectId}/databases/(default)/documents/{collection}
func (h *DocumentHandlers) HandleGetDocument(w http.ResponseWriter, r *http.Request) {
	projectID, collection, docID := h.parseProjectCollectionAndDoc(r.URL.Path)

	// If missing docID, it might be a LIST request if path matches valid list pattern
	if docID == "" {
		// Try parsing as list request
		pID, col := h.parseProjectAndCollection(r.URL.Path)
		if pID != "" && col != "" {
			h.HandleListDocuments(w, r, pID, col)
			return
		}
		// Otherwise valid error
		h.writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	if projectID == "" || collection == "" {
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

// HandleListDocuments lists documents in a collection
func (h *DocumentHandlers) HandleListDocuments(w http.ResponseWriter, r *http.Request, projectID, collection string) {
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

	// List documents
	// TODO: Pagination
	docs, err := coll.List(txn, 0, 100)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"documents": docs,
	})
}

// HandleListCollections lists all collections in a database
// GET /v1/projects/{projectId}/databases/(default)/collections
func (h *DocumentHandlers) HandleListCollections(w http.ResponseWriter, r *http.Request) {
	projectID := h.parseProjectFromCollectionsPath(r.URL.Path)
	if projectID == "" {
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

	// List collections
	collections := db.ListCollections()

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"collections": collections,
	})
}

// HandleCreateCollection creates a new collection
// POST /v1/projects/{projectId}/databases/(default)/collections
func (h *DocumentHandlers) HandleCreateCollection(w http.ResponseWriter, r *http.Request) {
	projectID := h.parseProjectFromCollectionsPath(r.URL.Path)
	if projectID == "" {
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

	// Parse body for collection name
	// { "name": "my_collection" }
	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Name == "" {
		h.writeError(w, http.StatusBadRequest, "collection name is required")
		return
	}

	// Acquire database instance
	db, release, err := h.manager.Acquire(projectID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer release()

	// Create collection
	_, err = db.CreateCollection(req.Name)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error()) // Often duplicate error
		return
	}

	h.writeJSON(w, http.StatusCreated, map[string]string{
		"name": req.Name,
	})
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

func (h *DocumentHandlers) parseProjectFromCollectionsPath(path string) string {
	// /v1/projects/{projectId}/databases/(default)/collections
	parts := strings.Split(strings.Trim(path, "/"), "/")
	// 0:v1, 1:projects, 2:projectId, 3:databases, 4:default, 5:collections
	if len(parts) < 6 || parts[5] != "collections" {
		return ""
	}
	return parts[2]
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

package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kartikbazzad/bunbase/buncast/pkg/client"
	"github.com/kartikbazzad/bunbase/bundoc"
	"github.com/kartikbazzad/bunbase/bundoc-server/internal/manager"
	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/rules"
	"github.com/kartikbazzad/bunbase/bundoc/storage"
)

// DocumentHandlers handles HTTP requests for document operations
type DocumentHandlers struct {
	manager       *manager.InstanceManager
	buncastClient *client.Client
}

// NewDocumentHandlers creates document handlers
func NewDocumentHandlers(mgr *manager.InstanceManager, buncastClient *client.Client) *DocumentHandlers {
	return &DocumentHandlers{
		manager:       mgr,
		buncastClient: buncastClient,
	}
}

// DocumentChangeEvent is published to Buncast for each document mutation.
type DocumentChangeEvent struct {
	ProjectID  string                 `json:"projectId"`
	Collection string                 `json:"collection"`
	DocID      string                 `json:"docId,omitempty"`
	Op         string                 `json:"op"` // create | update | delete
	Doc        map[string]interface{} `json:"doc,omitempty"`
	Timestamp  time.Time              `json:"ts"`
}

func (h *DocumentHandlers) publishChange(projectID, collection, docID, op string, doc map[string]interface{}) {
	if h.buncastClient == nil {
		return
	}

	topic := fmt.Sprintf("db.%s.collection.%s", projectID, collection)
	ev := DocumentChangeEvent{
		ProjectID:  projectID,
		Collection: collection,
		DocID:      docID,
		Op:         op,
		Doc:        doc,
		Timestamp:  time.Now().UTC(),
	}

	payload, err := json.Marshal(ev)
	if err != nil {
		// Best-effort: log and continue
		log.Printf("[Buncast] Marshal error for topic %s: %v", topic, err)
		return
	}

	go func() {
		log.Printf("[Buncast] Publishing %s event to topic %s (docId=%s)", op, topic, docID)
		if err := h.buncastClient.Publish(topic, payload); err != nil {
			log.Printf("[Buncast] Publish error for topic %s: %v", topic, err)
		} else {
			log.Printf("[Buncast] Successfully published %s event to topic %s", op, topic)
		}
	}()
}

// Helper to extract AuthContext
func (h *DocumentHandlers) getAuthContext(r *http.Request) *rules.AuthContext {
	// 1. Check Admin Key (Server SDK / Console)
	clientKey := r.Header.Get("X-Bunbase-Client-Key")
	if clientKey != "" {
		return &rules.AuthContext{IsAdmin: true}
	}

	// 2. Check User Token
	authHeader := r.Header.Get("X-Bundoc-Auth")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, ":", 2)
		uid := parts[0]
		var claims map[string]interface{}
		if len(parts) > 1 {
			_ = json.Unmarshal([]byte(parts[1]), &claims)
		}
		return &rules.AuthContext{UID: uid, Claims: claims}
	}

	// 3. Unauthenticated
	return nil
}

// HandleGetCollection returns collection metadata including Schema
// GET /v1/projects/{projectId}/databases/(default)/collections/{collection}
func (h *DocumentHandlers) HandleGetCollection(w http.ResponseWriter, r *http.Request) {
	projectID, collection := h.parseProjectAndCollection(r.URL.Path)
	if projectID == "" || collection == "" {
		h.writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	db, release, err := h.manager.Acquire(projectID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer release()

	coll, err := db.GetCollection(collection)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "collection not found")
		return
	}

	schema, _ := coll.GetSchema()
	rules := coll.GetRules()
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"name":   collection,
		"schema": schema,
		"rules":  rules,
	})
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

	// Get Auth
	auth := h.getAuthContext(r)

	// Insert document
	err = coll.Insert(auth, txn, doc)
	if err != nil {
		db.RollbackTransaction(txn)
		h.writeError(w, h.statusFromBundocError(err), err.Error())
		return
	}

	// Commit transaction
	err = db.CommitTransaction(txn)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Publish change event (best-effort)
	var docMap map[string]interface{}
	if raw, err := json.Marshal(doc); err == nil {
		_ = json.Unmarshal(raw, &docMap)
	}
	docID := ""
	if docMap != nil {
		if v, ok := docMap["_id"].(string); ok {
			docID = v
		}
	}
	h.publishChange(projectID, collection, docID, "create", docMap)

	// Return created document
	h.writeJSON(w, http.StatusCreated, doc)
}

// HandleGetDocument retrieves a document by ID or lists documents
// GET /v1/projects/{projectId}/databases/(default)/documents/{collection}/{docId}
// GET /v1/projects/{projectId}/databases/(default)/documents/{collection}
func (h *DocumentHandlers) HandleGetDocument(w http.ResponseWriter, r *http.Request) {
	projectID, pathSuffix := h.parseProjectAndPathSuffix(r.URL.Path)
	if projectID == "" || pathSuffix == "" {
		h.writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	db, release, err := h.manager.Acquire(projectID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer release()

	parts := strings.Split(pathSuffix, "/")
	var docID string
	var foundCollection *bundoc.Collection

	for i := len(parts); i > 0; i-- {
		potentialName := strings.Join(parts[:i], "/")
		coll, err := db.GetCollection(potentialName)
		if err == nil {
			foundCollection = coll
			if i < len(parts) {
				docID = strings.Join(parts[i:], "/")
			}
			break
		}
	}

	// Get Auth
	auth := h.getAuthContext(r)

	if foundCollection == nil {
		fullSuffix := strings.Join(parts, "/")
		if strings.Contains(fullSuffix, "*") {
			queryMap := make(map[string]interface{})
			queryParams := r.URL.Query()
			for k, v := range queryParams {
				if len(v) > 0 {
					queryMap[k] = v[0]
				}
			}

			txn, err := db.BeginTransaction(mvcc.ReadCommitted)
			if err != nil {
				h.writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			defer db.CommitTransaction(txn)

			docs, err := db.FindInGroup(auth, txn, fullSuffix, queryMap)
			if err != nil {
				h.writeError(w, http.StatusInternalServerError, err.Error())
				return
			}

			h.writeJSON(w, http.StatusOK, map[string]interface{}{
				"documents": docs,
			})
			return
		}

		h.writeError(w, http.StatusNotFound, "collection not found")
		return
	}

	// List
	if docID == "" {
		txn, err := db.BeginTransaction(mvcc.ReadCommitted)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer db.CommitTransaction(txn)

		skip := 0
		limit := 100
		if s := r.URL.Query().Get("skip"); s != "" {
			if n, err := strconv.Atoi(s); err == nil {
				skip = n
			}
		}
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil {
				limit = n
			}
		}

		fields := parseFieldsParam(r.URL.Query().Get("fields"))
		var listOpts []bundoc.QueryOptions
		if len(fields) > 0 {
			listOpts = []bundoc.QueryOptions{{Fields: fields}}
		}
		docs, err := foundCollection.List(auth, txn, skip, limit, listOpts...)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]interface{}{
			"documents": docs,
		})
		return
	}

	// Get One
	txn, err := db.BeginTransaction(mvcc.ReadCommitted)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.CommitTransaction(txn)

	doc, err := foundCollection.FindByID(auth, txn, docID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "document not found or permission denied")
		return
	}

	h.writeJSON(w, http.StatusOK, doc)
}

// HandleUpdateDocument updates a document
// PATCH /v1/projects/{projectId}/databases/(default)/documents/{collection}/{docId}
func (h *DocumentHandlers) HandleUpdateDocument(w http.ResponseWriter, r *http.Request) {
	projectID, pathSuffix := h.parseProjectAndPathSuffix(r.URL.Path)
	if projectID == "" || pathSuffix == "" {
		h.writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	db, release, err := h.manager.Acquire(projectID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer release()

	parts := strings.Split(pathSuffix, "/")
	var docID string
	var foundCollection *bundoc.Collection

	for i := len(parts); i > 0; i-- {
		potentialName := strings.Join(parts[:i], "/")
		coll, err := db.GetCollection(potentialName)
		if err == nil {
			foundCollection = coll
			if i < len(parts) {
				docID = strings.Join(parts[i:], "/")
			}
			break
		}
	}

	if foundCollection == nil {
		h.writeError(w, http.StatusNotFound, "collection not found")
		return
	}
	if docID == "" {
		h.writeError(w, http.StatusBadRequest, "document ID required")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	var updates map[string]interface{}
	if err := json.Unmarshal(body, &updates); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	txn, err := db.BeginTransaction(mvcc.ReadCommitted)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	auth := h.getAuthContext(r)

	err = foundCollection.Patch(auth, txn, docID, updates)
	if err != nil {
		db.RollbackTransaction(txn)
		h.writeError(w, h.statusFromBundocError(err), err.Error())
		return
	}

	err = db.CommitTransaction(txn)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Publish change event (best-effort). We don't have the full document here,
	// so we emit only the updates and identifiers.
	h.publishChange(projectID, foundCollection.Name(), docID, "update", updates)

	h.writeJSON(w, http.StatusOK, updates)
}

// HandleDeleteDocument deletes a document
// DELETE /v1/projects/{projectId}/databases/(default)/documents/{collection}/{docId}
func (h *DocumentHandlers) HandleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	projectID, pathSuffix := h.parseProjectAndPathSuffix(r.URL.Path)
	if projectID == "" || pathSuffix == "" {
		h.writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	db, release, err := h.manager.Acquire(projectID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer release()

	parts := strings.Split(pathSuffix, "/")
	var docID string
	var foundCollection *bundoc.Collection

	for i := len(parts); i > 0; i-- {
		potentialName := strings.Join(parts[:i], "/")
		coll, err := db.GetCollection(potentialName)
		if err == nil {
			foundCollection = coll
			if i < len(parts) {
				docID = strings.Join(parts[i:], "/")
			}
			break
		}
	}

	if foundCollection == nil {
		h.writeError(w, http.StatusNotFound, "collection not found")
		return
	}
	if docID == "" {
		h.writeError(w, http.StatusBadRequest, "document ID required")
		return
	}

	txn, err := db.BeginTransaction(mvcc.ReadCommitted)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	auth := h.getAuthContext(r)

	err = foundCollection.Delete(auth, txn, docID)
	if err != nil {
		db.RollbackTransaction(txn)
		h.writeError(w, h.statusFromBundocError(err), err.Error())
		return
	}

	err = db.CommitTransaction(txn)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Publish change event (best-effort). No document body on delete.
	h.publishChange(projectID, foundCollection.Name(), docID, "delete", nil)

	w.WriteHeader(http.StatusNoContent)
}

// HandleQueryDocuments executes a complex query
// POST /v1/projects/{projectId}/databases/(default)/documents/query
func (h *DocumentHandlers) HandleQueryDocuments(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 6 {
		h.writeError(w, http.StatusBadRequest, "invalid path")
		return
	}
	projectID := parts[3]

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	var req struct {
		Collection string                 `json:"collection"`
		Query      map[string]interface{} `json:"query"`
		Skip       int                    `json:"skip"`
		Limit      int                    `json:"limit"`
		SortField  string                 `json:"sortField"`
		SortDesc   bool                   `json:"sortDesc"`
		Fields     []string               `json:"fields"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Collection == "" {
		h.writeError(w, http.StatusBadRequest, "collection is required")
		return
	}

	db, release, err := h.manager.Acquire(projectID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer release()

	auth := h.getAuthContext(r)

	coll, err := db.GetCollection(req.Collection)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "collection not found")
		return
	}

	txn, err := db.BeginTransaction(mvcc.ReadCommitted)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.CommitTransaction(txn)

	opts := bundoc.QueryOptions{
		Skip:      req.Skip,
		Limit:     req.Limit,
		SortField: req.SortField,
		SortDesc:  req.SortDesc,
		Fields:    req.Fields,
	}

	docs, err := coll.FindQuery(auth, txn, req.Query, opts)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("query error: %v", err))
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"documents": docs,
	})
}

// HandleListDocuments lists documents in a collection
func (h *DocumentHandlers) HandleListDocuments(w http.ResponseWriter, r *http.Request, projectID, collection string) {
	db, release, err := h.manager.Acquire(projectID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer release()

	auth := h.getAuthContext(r)

	coll, err := db.GetCollection(collection)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "collection not found")
		return
	}

	txn, err := db.BeginTransaction(mvcc.ReadCommitted)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.CommitTransaction(txn)

	skip := 0
	limit := 100

	if s := r.URL.Query().Get("skip"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			skip = n
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			limit = n
		}
	}

	fields := parseFieldsParam(r.URL.Query().Get("fields"))
	var listOpts []bundoc.QueryOptions
	if len(fields) > 0 {
		listOpts = []bundoc.QueryOptions{{Fields: fields}}
	}
	docs, err := coll.List(auth, txn, skip, limit, listOpts...)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"documents": docs,
	})
}

// parseFieldsParam parses a comma-separated fields query param (e.g. "_id,name,email") into a trimmed slice; empty strings omitted.
func parseFieldsParam(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// HandleListCollections lists all collections in a database
// GET /v1/projects/{projectId}/databases/(default)/collections
func (h *DocumentHandlers) HandleListCollections(w http.ResponseWriter, r *http.Request) {
	projectID := h.parseProjectFromCollectionsPath(r.URL.Path)
	if projectID == "" {
		h.writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	db, release, err := h.manager.Acquire(projectID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer release()

	prefix := r.URL.Query().Get("prefix")
	var collections []string
	if prefix != "" {
		collections = db.ListCollectionsWithPrefix(prefix)
	} else {
		collections = db.ListCollections()
	}

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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	var req struct {
		Name                  string      `json:"name"`
		Schema                interface{} `json:"schema"`
		UpdateIfExists        bool        `json:"update_if_exists"`
		PreventSchemaOverride bool        `json:"prevent_schema_override"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Name == "" {
		h.writeError(w, http.StatusBadRequest, "collection name is required")
		return
	}

	db, release, err := h.manager.Acquire(projectID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer release()

	coll, err := db.CreateCollection(req.Name)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") && req.UpdateIfExists {
			coll, err = db.GetCollection(req.Name)
			if err != nil {
				h.writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if req.Schema != nil {
				schemaBytes, err := json.Marshal(req.Schema)
				if err != nil {
					h.writeError(w, http.StatusBadRequest, "invalid schema format")
					return
				}
				if err := coll.SetSchema(string(schemaBytes)); err != nil {
					h.writeError(w, h.statusFromBundocError(err), err.Error())
					return
				}
			}
			h.writeJSON(w, http.StatusOK, map[string]interface{}{
				"name":   req.Name,
				"schema": req.Schema,
			})
			return
		}
		h.writeError(w, http.StatusConflict, "collection already exists")
		return
	}

	if req.PreventSchemaOverride {
		if err := db.SetCollectionPreventSchemaOverride(req.Name, true); err != nil {
			h.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if req.Schema != nil {
		schemaBytes, err := json.Marshal(req.Schema)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid schema format")
			return
		}

		if err := coll.SetSchema(string(schemaBytes)); err != nil {
			h.writeError(w, h.statusFromBundocError(err), err.Error())
			return
		}
	}

	h.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"name":   req.Name,
		"schema": req.Schema,
	})
}

// HandleUpdateCollection updates collection metadata (Schema)
// PATCH /v1/projects/{projectId}/databases/(default)/collections/{collection}
func (h *DocumentHandlers) HandleUpdateCollection(w http.ResponseWriter, r *http.Request) {
	projectID, collection := h.ParseProjectAndCollectionFromCollectionPath(r.URL.Path)
	if projectID == "" || collection == "" {
		h.writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	var req struct {
		Schema                interface{} `json:"schema"`
		PreventSchemaOverride *bool       `json:"prevent_schema_override"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	db, release, err := h.manager.Acquire(projectID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer release()

	if req.PreventSchemaOverride != nil {
		if err := db.SetCollectionPreventSchemaOverride(collection, *req.PreventSchemaOverride); err != nil {
			h.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	coll, err := db.GetCollection(collection)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "collection not found")
		return
	}

	if req.Schema != nil {
		schemaBytes, err := json.Marshal(req.Schema)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid schema format")
			return
		}
		if err := coll.SetSchema(string(schemaBytes)); err != nil {
			h.writeError(w, h.statusFromBundocError(err), err.Error())
			return
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"status": "collection updated",
	})
}

// HandleIndexOperations manages indexes (List, Create, Delete)
// GET /v1/projects/{projectId}/databases/(default)/indexes?collection=name
func (h *DocumentHandlers) HandleIndexOperations(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 6 {
		h.writeError(w, http.StatusBadRequest, "invalid path")
		return
	}
	projectID := parts[3]

	db, release, err := h.manager.Acquire(projectID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer release()

	switch r.Method {
	case "GET":
		collectionName := r.URL.Query().Get("collection")
		if collectionName == "" {
			h.writeError(w, http.StatusBadRequest, "collection query param required")
			return
		}

		coll, err := db.GetCollection(collectionName)
		if err != nil {
			h.writeError(w, http.StatusNotFound, "collection not found")
			return
		}

		indexes := coll.ListIndexes()
		h.writeJSON(w, http.StatusOK, map[string]interface{}{
			"indexes": indexes,
		})

	case "POST":
		body, err := io.ReadAll(r.Body)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "failed to read body")
			return
		}
		defer r.Body.Close()

		var req struct {
			Collection string `json:"collection"`
			Field      string `json:"field"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		if req.Collection == "" || req.Field == "" {
			h.writeError(w, http.StatusBadRequest, "collection and field are required")
			return
		}

		if strings.Contains(req.Collection, "*") {
			if err := db.EnsureGroupIndex(req.Collection, req.Field); err != nil {
				h.writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			h.writeJSON(w, http.StatusCreated, map[string]string{
				"status":  "group index created",
				"pattern": req.Collection,
				"field":   req.Field,
			})
			return
		}

		coll, err := db.GetCollection(req.Collection)
		if err != nil {
			h.writeError(w, http.StatusNotFound, "collection not found")
			return
		}

		if err := coll.EnsureIndex(req.Field); err != nil {
			h.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		h.writeJSON(w, http.StatusCreated, map[string]string{
			"status": "index created",
			"field":  req.Field,
		})

	case "DELETE":
		collectionName := r.URL.Query().Get("collection")
		field := r.URL.Query().Get("field")
		if collectionName == "" || field == "" {
			h.writeError(w, http.StatusBadRequest, "collection and field query params required")
			return
		}

		coll, err := db.GetCollection(collectionName)
		if err != nil {
			h.writeError(w, http.StatusNotFound, "collection not found")
			return
		}

		if err := coll.DropIndex(field); err != nil {
			h.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandleDeleteIndex deletes a secondary index
// DELETE /v1/projects/{projectId}/databases/(default)/collections/{collection}/indexes/{field}
func (h *DocumentHandlers) HandleDeleteIndex(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	parts := strings.Split(path, "/indexes/")
	if len(parts) != 2 {
		h.writeError(w, http.StatusBadRequest, "invalid index path")
		return
	}

	collectionPath := parts[0]
	field := parts[1]

	projectID, collection := h.ParseProjectAndCollectionFromCollectionPath(collectionPath)

	if projectID == "" || collection == "" {
		h.writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	if field == "" {
		h.writeError(w, http.StatusBadRequest, "field name is required")
		return
	}

	db, release, err := h.manager.Acquire(projectID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer release()

	coll, err := db.GetCollection(collection)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "collection not found")
		return
	}

	if err := coll.DropIndex(field); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions

func (h *DocumentHandlers) parseProjectAndCollection(path string) (string, string) {
	// Robust parsing: find "projects" and collection name.
	// Paths:
	// /v1/projects/{projectId}/databases/(default)/collections/{collection}/documents
	// /v1/projects/{projectId}/databases/(default)/documents/{collection}  (tenant-auth, etc.)
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// Find project ID (after "projects")
	projectID := ""
	for i, p := range parts {
		if p == "projects" && i+1 < len(parts) {
			projectID = parts[i+1]
			break
		}
	}

	// Find collection: after "collections" or after "documents"
	collection := ""
	for i, p := range parts {
		if p == "collections" && i+1 < len(parts) {
			collection = parts[i+1]
			break
		}
	}
	if collection == "" {
		for i, p := range parts {
			if p == "documents" && i+1 < len(parts) {
				collection = parts[i+1]
				break
			}
		}
	}

	return projectID, collection
}

func (h *DocumentHandlers) parseProjectCollectionAndDoc(path string) (string, string, string) {
	// /v1/projects/{projectId}/databases/(default)/documents/{collection}/{docId}
	// Or .../database/collections/{collection}/documents/{docId} <-- Not standard?
	// Actually HandleGetDocument uses this.
	// Check usage: HandleGetDocument maps .../documents/{collection}/{docId} ? No.
	// Current standard for Docs: .../documents/{collection}/{docId} or .../collections/{collection}/documents/{docId}

	parts := strings.Split(strings.Trim(path, "/"), "/")

	projectID := ""
	for i, p := range parts {
		if p == "projects" && i+1 < len(parts) {
			projectID = parts[i+1]
			break
		}
	}

	// Strategy: Use "documents" or "collections" anchor?
	// Handler routing:
	// .../documents/{collection}/{docId} (Old style?)
	// Let's assume strict structure relative to "documents" keyword?

	// If path contains "documents", we assume parts after it.
	// .../documents/{collection}/{docId}
	collection := ""
	docID := ""

	for i, p := range parts {
		if p == "documents" && i+2 < len(parts) {
			collection = parts[i+1]
			docID = parts[i+2]
			break
		}
	}

	// If ID not found? Maybe client uses different path?
	return projectID, collection, docID
}

func (h *DocumentHandlers) parseProjectFromCollectionsPath(path string) string {
	// /v1/projects/{projectId}/databases/(default)/collections
	// /v1/projects/{projectId}/database/collections
	parts := strings.Split(strings.Trim(path, "/"), "/")

	for i, p := range parts {
		if p == "projects" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func (h *DocumentHandlers) ParseProjectAndCollectionFromCollectionPath(path string) (string, string) {
	// /v1/projects/{projectId}/databases/(default)/collections/{collection...}
	// /v1/projects/{projectId}/database/collections/{collection...}
	parts := strings.Split(strings.Trim(path, "/"), "/")

	projectID := ""
	collection := ""

	for i, p := range parts {
		if p == "projects" && i+1 < len(parts) {
			projectID = parts[i+1]
		}
		if p == "collections" && i+1 < len(parts) {
			collection = strings.Join(parts[i+1:], "/")
			break
		}
	}
	return projectID, collection
}

// HandleDeleteCollection drops a collection
// DELETE /v1/projects/{projectId}/databases/(default)/collections/{collection}
func (h *DocumentHandlers) HandleDeleteCollection(w http.ResponseWriter, r *http.Request) {
	projectID, collection := h.ParseProjectAndCollectionFromCollectionPath(r.URL.Path)
	if projectID == "" || collection == "" {
		h.writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	db, release, err := h.manager.Acquire(projectID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer release()

	if err := db.DropCollection(collection); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

func (h *DocumentHandlers) statusFromBundocError(err error) int {
	switch {
	case errors.Is(err, bundoc.ErrInvalidReferenceSchema):
		return http.StatusBadRequest
	case errors.Is(err, bundoc.ErrReferenceTargetNotFound):
		return http.StatusConflict
	case errors.Is(err, bundoc.ErrReferenceRestrictViolation):
		return http.StatusConflict
	case errors.Is(err, bundoc.ErrSchemaOverrideBlocked):
		return http.StatusConflict
	}

	errStr := err.Error()
	if strings.Contains(errStr, "document invalid against schema") || strings.Contains(errStr, "schema validation error") {
		return http.StatusBadRequest
	}

	return http.StatusInternalServerError
}

func (h *DocumentHandlers) parseProjectAndPathSuffix(path string) (string, string) {
	// /v1/projects/{projectId}/databases/(default)/documents/{...}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 6 || parts[5] != "documents" {
		return "", ""
	}
	return parts[2], strings.Join(parts[6:], "/")
}

// HandleUpdateRules updates collection security rules
// PATCH /v1/projects/{projectId}/databases/(default)/collections/{collection}/rules
func (h *DocumentHandlers) HandleUpdateRules(w http.ResponseWriter, r *http.Request) {
	// Parse path
	// /v1/projects/{projectId}/databases/(default)/collections/{collection}/rules
	path := strings.TrimSuffix(r.URL.Path, "/rules") // strip suffix first
	projectID, collection := h.ParseProjectAndCollectionFromCollectionPath(path)

	if projectID == "" || collection == "" {
		h.writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	// Body format: { "rules": { "read": "auth != nil", "write": "false" } }
	var req struct {
		Rules map[string]string `json:"rules"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	db, release, err := h.manager.Acquire(projectID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer release()

	coll, err := db.GetCollection(collection)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "collection not found")
		return
	}

	if err := coll.SetRules(req.Rules); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("failed to set rules: %v", err))
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "rules updated",
		"rules":  req.Rules,
	})
}

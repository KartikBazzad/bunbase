package http

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/kartikbazzad/bunbase/bunder/pkg/store"
)

// KVHandler provides HTTP endpoints for a Bunder Store instance.
type KVHandler struct {
	store store.Store
}

// NewKVHandler creates a new KVHandler.
func NewKVHandler(s store.Store) *KVHandler {
	return &KVHandler{store: s}
}

// ServeHTTP serves GET/PUT/DELETE /kv/:key, GET /keys, GET /health.
func (h *KVHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	switch {
	case path == "health":
		h.serveHealth(w, r)
		return
	case path == "keys" && r.Method == http.MethodGet:
		h.serveKeys(w, r)
		return
	case len(parts) >= 2 && parts[0] == "kv":
		// /kv/:key
		h.serveKV(w, r, parts[1])
		return
	case len(parts) == 1 && parts[0] == "kv" && r.Method == http.MethodGet:
		// /kv (list keys)
		h.serveKeys(w, r)
		return
	}
	http.NotFound(w, r)
}

func (h *KVHandler) serveHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *KVHandler) serveKV(w http.ResponseWriter, r *http.Request, key string) {
	keyB := []byte(key)
	switch r.Method {
	case http.MethodGet:
		v := h.store.Get(keyB)
		if v == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(v)
	case http.MethodPut, http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := h.store.Set(keyB, body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	case http.MethodDelete:
		ok, err := h.store.Delete(keyB)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *KVHandler) serveKeys(w http.ResponseWriter, r *http.Request) {
	pattern := r.URL.Query().Get("pattern")
	if pattern == "" {
		pattern = "*"
	}
	keys := h.store.Keys([]byte(pattern))
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = string(k)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/kartikbazzad/bunbase/bunder/internal/metrics"
)

// HTTPHandler serves REST and SSE: /health, /metrics, /kv/:key (GET/PUT/DELETE), /keys, /subscribe.
type HTTPHandler struct {
	handler *Handler
}

// NewHTTPHandler creates an HTTP handler that delegates to the RESP handler's store.
func NewHTTPHandler(h *Handler) *HTTPHandler {
	return &HTTPHandler{handler: h}
}

// ServeHTTP serves GET/POST /kv/:key, GET /keys, GET /health, GET /metrics, GET /subscribe (SSE).
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	switch {
	case path == "health":
		h.serveHealth(w, r)
		return
	case path == "metrics":
		h.serveMetrics(w, r)
		return
	case path == "subscribe":
		h.serveSubscribe(w, r)
		return
	case len(parts) >= 1 && parts[0] == "kv":
		if len(parts) == 2 {
			h.serveKV(w, r, parts[1])
			return
		}
		if len(parts) == 1 && r.Method == http.MethodGet {
			h.serveKeys(w, r)
			return
		}
	}
	http.NotFound(w, r)
}

func (h *HTTPHandler) serveHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *HTTPHandler) serveMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.Write([]byte(metrics.Default().PrometheusFormat()))
}

func (h *HTTPHandler) serveSubscribe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	w.(http.Flusher).Flush()
	// Placeholder: no Buncast subscription in HTTP; client can poll or use TCP.
	w.Write([]byte("data: {\"msg\":\"connect to TCP for RESP\"}\n\n"))
	w.(http.Flusher).Flush()
}

func (h *HTTPHandler) serveKV(w http.ResponseWriter, r *http.Request, key string) {
	keyB := []byte(key)
	switch r.Method {
	case http.MethodGet:
		v := h.handler.kv.Get(keyB)
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
		if err := h.handler.kv.Set(keyB, body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	case http.MethodDelete:
		ok, err := h.handler.kv.Delete(keyB)
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

func (h *HTTPHandler) serveKeys(w http.ResponseWriter, r *http.Request) {
	pattern := r.URL.Query().Get("pattern")
	if pattern == "" {
		pattern = "*"
	}
	keys := h.handler.kv.Keys([]byte(pattern))
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = string(k)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

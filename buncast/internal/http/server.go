package httpsrv

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/kartikbazzad/bunbase/buncast/internal/broker"
	"github.com/kartikbazzad/bunbase/buncast/internal/config"
	"github.com/kartikbazzad/bunbase/buncast/internal/logger"
)

// Server is the Buncast HTTP server (health, admin, SSE subscribe).
type Server struct {
	cfg    *config.Config
	logger *logger.Logger
	broker *broker.Broker
	server *http.Server
}

// NewServer creates a new HTTP server.
func NewServer(cfg *config.Config, log *logger.Logger, b *broker.Broker) *Server {
	s := &Server{
		cfg:    cfg,
		logger: log,
		broker: b,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/topics", s.handleTopics)
	mux.HandleFunc("/subscribe", s.handleSubscribe)
	s.server = &http.Server{
		Addr:         cfg.HTTP.ListenAddr,
		Handler:      mux,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: 0, // SSE keeps connection open; we don't want to timeout writes
	}
	return s
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.logger.Info("HTTP server listening on %s", s.cfg.HTTP.ListenAddr)
	return s.server.ListenAndServe()
}

// Stop stops the HTTP server.
func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleTopics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	topics := s.broker.ListTopics()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(topics)
}

// sseSubscriber implements broker.Subscriber by writing SSE events to the response writer.
type sseSubscriber struct {
	w       http.ResponseWriter
	flusher http.Flusher
	mu      sync.Mutex
	logger  *logger.Logger
}

func (sub *sseSubscriber) Send(msg *broker.Message) {
	sub.mu.Lock()
	defer sub.mu.Unlock()
	// SSE: "data: " + payload (one line per line, prefix each with "data: ") + "\n\n"
	data := string(msg.Payload)
	// Escape newlines for SSE: each line as "data: ...\n"
	_, _ = sub.w.Write([]byte("data: " + data + "\n\n"))
	if sub.flusher != nil {
		sub.flusher.Flush()
	}
}

func (s *Server) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	topic := r.URL.Query().Get("topic")
	if topic == "" {
		http.Error(w, "missing query parameter: topic", http.StatusBadRequest)
		return
	}

	s.broker.CreateTopic(topic)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	flusher, _ := w.(http.Flusher)
	sub := &sseSubscriber{w: w, flusher: flusher, logger: s.logger}
	s.broker.Subscribe(topic, sub)

	// Keep request alive and detect client disconnect via request context
	<-r.Context().Done()
	s.broker.Unsubscribe(topic, sub)
}

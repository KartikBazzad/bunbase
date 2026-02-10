package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/kartikbazzad/bunbase/buncast/pkg/client"
)

// ChangeType represents the type of document change.
type ChangeType string

const (
	ChangeAdded    ChangeType = "added"
	ChangeModified ChangeType = "modified"
	ChangeRemoved  ChangeType = "removed"
)

// DocumentChangeEvent mirrors bundoc-server's change payload (subset).
type DocumentChangeEvent struct {
	ProjectID  string                 `json:"projectId"`
	Collection string                 `json:"collection"`
	DocID      string                 `json:"docId"`
	Op         string                 `json:"op"`
	Doc        map[string]interface{} `json:"doc,omitempty"`
}

// SubscriptionEvent is what we fan out to browser SSE clients.
type SubscriptionEvent struct {
	Type     ChangeType             `json:"type"`
	Document map[string]interface{} `json:"document,omitempty"`
	DocID    string                 `json:"docId,omitempty"`
}

// KVChangeEvent is the payload for KV set/delete (from bunder-manager via Buncast topic kv.{projectID}).
type KVChangeEvent struct {
	Op    string  `json:"op"`    // "set" or "delete"
	Key   string  `json:"key"`
	Value *string `json:"value,omitempty"` // base64 for set; nil for delete
}

// kvEventWithProjectID is used internally to dispatch KV events.
type kvEventWithProjectID struct {
	projectID string
	ev        KVChangeEvent
}

// QueryPredicate is a simple function that decides if a document matches a query.
type QueryPredicate func(doc map[string]interface{}) bool

// Subscription represents a single logical subscription (collection-only or query-based).
type Subscription struct {
	ProjectID  string
	Collection string
	Predicate  QueryPredicate
	Out        chan SubscriptionEvent
	cancel     context.CancelFunc
}

// SubscriptionManager manages active subscriptions and Buncast fan-out.
type SubscriptionManager struct {
	mu              sync.RWMutex
	subs            map[string][]*Subscription // key: projectID|collection
	activeTopics    map[string]bool           // track which Buncast topics we're subscribed to
	kvSubs          map[string][]chan<- KVChangeEvent // projectID -> channels
	kvActiveTopics  map[string]bool            // track kv.{projectID} subscriptions
	buncast         *client.Client
	socketPath      string
	subscribeCtx   context.Context
	cancel          context.CancelFunc
	eventChan       chan DocumentChangeEvent
	kvEventChan     chan kvEventWithProjectID
}

// NewSubscriptionManager creates a manager. If buncastClient is nil, the manager is a no-op.
func NewSubscriptionManager(buncastClient *client.Client) *SubscriptionManager {
	ctx, cancel := context.WithCancel(context.Background())
	var socketPath string
	if buncastClient != nil {
		// Extract socket path from client (we'll need to pass it separately)
		// For now, we'll store it when creating the manager
		socketPath = "" // Will be set via SetSocketPath if needed
	}
	mgr := &SubscriptionManager{
		subs:           make(map[string][]*Subscription),
		activeTopics:   make(map[string]bool),
		kvSubs:         make(map[string][]chan<- KVChangeEvent),
		kvActiveTopics: make(map[string]bool),
		buncast:        buncastClient,
		socketPath:     socketPath,
		subscribeCtx:   ctx,
		cancel:         cancel,
		eventChan:      make(chan DocumentChangeEvent, 100),
		kvEventChan:    make(chan kvEventWithProjectID, 100),
	}
	if buncastClient != nil {
		go mgr.run()
		go mgr.runKV()
	}
	return mgr
}

// SetSocketPath sets the Buncast socket path for creating subscription clients.
func (m *SubscriptionManager) SetSocketPath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.socketPath = path
}

// key builds the internal map key.
func (m *SubscriptionManager) key(projectID, collection string) string {
	return projectID + "|" + collection
}

// Stop stops all background processing.
func (m *SubscriptionManager) Stop() {
	m.cancel()
}

// run processes events from the event channel and dispatches to subscriptions.
func (m *SubscriptionManager) run() {
	for {
		select {
		case <-m.subscribeCtx.Done():
			return
		case ev := <-m.eventChan:
			m.dispatch(ev)
		}
	}
}

// ensureTopicSubscription subscribes to a Buncast topic if not already subscribed.
func (m *SubscriptionManager) ensureTopicSubscription(topic string) {
	if m.buncast == nil || m.socketPath == "" {
		return
	}
	m.mu.Lock()
	if m.activeTopics[topic] {
		m.mu.Unlock()
		return
	}
	m.activeTopics[topic] = true
	socketPath := m.socketPath
	m.mu.Unlock()

	// Create a dedicated client for this subscription (Subscribe blocks and holds connection)
	subClient := client.New(socketPath)
	
	// Subscribe in a goroutine to avoid blocking
	go func() {
		defer func() {
			_ = subClient.Close()
			// Mark topic as inactive on exit so we can retry if needed
			m.mu.Lock()
			delete(m.activeTopics, topic)
			m.mu.Unlock()
		}()
		
		log.Printf("[SubscriptionManager] Subscribing to topic: %s", topic)
		if err := subClient.Connect(); err != nil {
			log.Printf("[SubscriptionManager] Failed to connect to Buncast for topic %s: %v", topic, err)
			return
		}
		log.Printf("[SubscriptionManager] Connected to Buncast, subscribing to topic: %s", topic)
		
		// Subscribe blocks, so run it in the goroutine
		// Subscribe sends the subscribe command and then reads message frames from the connection
		log.Printf("[SubscriptionManager] Calling Subscribe for topic %s...", topic)
		subscribeErr := subClient.Subscribe(topic, func(msg *client.Message) error {
			log.Printf("[SubscriptionManager] Received message on topic %s: %d bytes", topic, len(msg.Payload))
			var ev DocumentChangeEvent
			if err := json.Unmarshal(msg.Payload, &ev); err != nil {
				log.Printf("[SubscriptionManager] Failed to unmarshal event: %v", err)
				return nil
			}
			log.Printf("[SubscriptionManager] Dispatching event: op=%s docId=%s", ev.Op, ev.DocID)
			select {
			case m.eventChan <- ev:
				log.Printf("[SubscriptionManager] Event dispatched successfully")
			case <-m.subscribeCtx.Done():
				log.Printf("[SubscriptionManager] Context cancelled, stopping subscription")
				return fmt.Errorf("subscription manager stopped")
			default:
				// Drop if channel full (backpressure)
				log.Printf("[SubscriptionManager] Event channel full, dropping event")
			}
			return nil
		})
		
		if subscribeErr != nil {
			log.Printf("[SubscriptionManager] Subscribe error for topic %s: %v", topic, subscribeErr)
		} else {
			log.Printf("[SubscriptionManager] Subscribe ended normally for topic %s (connection closed)", topic)
		}
	}()
}

// Subscribe registers a subscription and returns a channel for events and a cancel func.
func (m *SubscriptionManager) Subscribe(ctx context.Context, projectID, collection string, pred QueryPredicate) (<-chan SubscriptionEvent, context.CancelFunc) {
	out := make(chan SubscriptionEvent, 16)

	subCtx, cancel := context.WithCancel(ctx)
	sub := &Subscription{
		ProjectID:  projectID,
		Collection: collection,
		Predicate:  pred,
		Out:        out,
		cancel:     cancel,
	}

	key := m.key(projectID, collection)
	m.mu.Lock()
	m.subs[key] = append(m.subs[key], sub)
	m.mu.Unlock()

	// Ensure we're subscribed to the Buncast topic for this collection
	topic := fmt.Sprintf("db.%s.collection.%s", projectID, collection)
	m.ensureTopicSubscription(topic)

	// Cleanup when context is done.
	go func() {
		<-subCtx.Done()
		m.removeSubscription(key, sub)
		close(out)
	}()

	return out, cancel
}

func (m *SubscriptionManager) removeSubscription(key string, target *Subscription) {
	m.mu.Lock()
	defer m.mu.Unlock()
	list := m.subs[key]
	if len(list) == 0 {
		return
	}
	out := list[:0]
	for _, s := range list {
		if s != target {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		delete(m.subs, key)
	} else {
		m.subs[key] = out
	}
}

// dispatch fans out a single DocumentChangeEvent to matching subscriptions.
func (m *SubscriptionManager) dispatch(ev DocumentChangeEvent) {
	key := m.key(ev.ProjectID, ev.Collection)

	m.mu.RLock()
	subs := append([]*Subscription(nil), m.subs[key]...)
	m.mu.RUnlock()

	if len(subs) == 0 {
		return
	}

	var changeType ChangeType
	switch ev.Op {
	case "create":
		changeType = ChangeAdded
	case "update":
		changeType = ChangeModified
	case "delete":
		changeType = ChangeRemoved
	default:
		return
	}

	for _, sub := range subs {
		if sub.Predicate != nil && !sub.Predicate(ev.Doc) {
			continue
		}
		select {
		case sub.Out <- SubscriptionEvent{
			Type:     changeType,
			Document: ev.Doc,
			DocID:    ev.DocID,
		}:
		default:
			// Drop if subscriber is slow; this is acceptable for demo / best-effort.
		}
	}
}

// runKV processes KV events and dispatches to KV subscribers.
func (m *SubscriptionManager) runKV() {
	for {
		select {
		case <-m.subscribeCtx.Done():
			return
		case item := <-m.kvEventChan:
			m.dispatchKV(item.projectID, item.ev)
		}
	}
}

// ensureKVTopicSubscription subscribes to Buncast topic kv.{projectID} if not already subscribed.
func (m *SubscriptionManager) ensureKVTopicSubscription(projectID string) {
	if m.buncast == nil || m.socketPath == "" {
		return
	}
	topic := "kv." + projectID
	m.mu.Lock()
	if m.kvActiveTopics[topic] {
		m.mu.Unlock()
		return
	}
	m.kvActiveTopics[topic] = true
	socketPath := m.socketPath
	m.mu.Unlock()

	subClient := client.New(socketPath)
	go func() {
		defer func() {
			_ = subClient.Close()
			m.mu.Lock()
			delete(m.kvActiveTopics, topic)
			m.mu.Unlock()
		}()
		if err := subClient.Connect(); err != nil {
			log.Printf("[SubscriptionManager] KV topic %s: connect failed: %v", topic, err)
			return
		}
		subscribeErr := subClient.Subscribe(topic, func(msg *client.Message) error {
			var ev KVChangeEvent
			if err := json.Unmarshal(msg.Payload, &ev); err != nil {
				return nil
			}
			select {
			case m.kvEventChan <- kvEventWithProjectID{projectID: projectID, ev: ev}:
			case <-m.subscribeCtx.Done():
				return fmt.Errorf("subscription manager stopped")
			default:
			}
			return nil
		})
		if subscribeErr != nil {
			log.Printf("[SubscriptionManager] KV topic %s: subscribe error: %v", topic, subscribeErr)
		}
	}()
}

// SubscribeKV subscribes to KV change events for a project. Returns event channel and cancel.
func (m *SubscriptionManager) SubscribeKV(ctx context.Context, projectID string) (<-chan KVChangeEvent, context.CancelFunc) {
	out := make(chan KVChangeEvent, 16)
	subCtx, cancel := context.WithCancel(ctx)

	m.mu.Lock()
	m.kvSubs[projectID] = append(m.kvSubs[projectID], out)
	m.mu.Unlock()

	m.ensureKVTopicSubscription(projectID)

	go func() {
		<-subCtx.Done()
		m.removeKVSubscription(projectID, out)
		close(out)
	}()

	return out, cancel
}

func (m *SubscriptionManager) removeKVSubscription(projectID string, ch chan<- KVChangeEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	list := m.kvSubs[projectID]
	if len(list) == 0 {
		return
	}
	out := list[:0]
	for _, c := range list {
		if c != ch {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		delete(m.kvSubs, projectID)
	} else {
		m.kvSubs[projectID] = out
	}
}

func (m *SubscriptionManager) dispatchKV(projectID string, ev KVChangeEvent) {
	m.mu.RLock()
	subs := append([]chan<- KVChangeEvent(nil), m.kvSubs[projectID]...)
	m.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

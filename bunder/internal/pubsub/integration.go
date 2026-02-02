// Package pubsub integrates Bunder with Buncast for publishing DML events (SET, DEL, etc.).
package pubsub

import (
	"encoding/json"
	"sync"
	"time"

	buncast "github.com/kartikbazzad/bunbase/buncast/pkg/client"
)

const topicOperations = "bunder.operations"

// Operation represents a single DML operation to publish (e.g. SET key value, DEL key).
type Operation struct {
	Type      string
	Key       string
	Value     []byte
	Timestamp int64
}

// PubSubManager publishes Bunder DML events to Buncast.
type PubSubManager struct {
	mu      sync.Mutex
	client  *buncast.Client
	enabled bool
}

// NewPubSubManager creates a pub/sub manager. If socketPath is empty or Connect fails, Publish is a no-op.
func NewPubSubManager(socketPath string, enabled bool) *PubSubManager {
	return &PubSubManager{
		client:  buncast.New(socketPath),
		enabled: enabled,
	}
}

// PublishOperation publishes a DML operation to Buncast (topic bunder.operations).
func (p *PubSubManager) PublishOperation(op Operation) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	c := p.client
	p.mu.Unlock()
	if c == nil {
		return
	}
	if err := c.Connect(); err != nil {
		return
	}
	if op.Timestamp == 0 {
		op.Timestamp = time.Now().Unix()
	}
	payload, err := json.Marshal(op)
	if err != nil {
		return
	}
	_ = c.CreateTopic(topicOperations)
	_ = c.Publish(topicOperations, payload)
}

// Close closes the Buncast connection.
func (p *PubSubManager) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.client != nil {
		err := p.client.Close()
		p.client = nil
		return err
	}
	return nil
}

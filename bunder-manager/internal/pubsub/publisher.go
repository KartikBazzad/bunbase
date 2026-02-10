package pubsub

import (
	"encoding/base64"
	"encoding/json"
	"sync"

	buncast "github.com/kartikbazzad/bunbase/buncast/pkg/client"
)

// KVEvent is the payload published to Buncast topic kv.{projectID}.
type KVEvent struct {
	Op    string `json:"op"`    // "set" or "delete"
	Key   string `json:"key"`
	Value string `json:"value,omitempty"` // base64-encoded for set; omit for delete
}

// Publisher publishes KV events to Buncast.
type Publisher struct {
	mu     sync.Mutex
	client *buncast.Client
}

// NewPublisher creates a publisher. If socketPath is empty, PublishKV is a no-op.
func NewPublisher(socketPath string) *Publisher {
	if socketPath == "" {
		return &Publisher{}
	}
	return &Publisher{
		client: buncast.New(socketPath),
	}
}

// PublishKV publishes a KV change to topic kv.{projectID}. No-op if publisher was created with empty socket.
func (p *Publisher) PublishKV(projectID, op, key string, value []byte) {
	p.mu.Lock()
	c := p.client
	p.mu.Unlock()
	if c == nil {
		return
	}
	if err := c.Connect(); err != nil {
		return
	}
	topic := "kv." + projectID
	ev := KVEvent{Op: op, Key: key}
	if len(value) > 0 {
		ev.Value = base64.StdEncoding.EncodeToString(value)
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		return
	}
	_ = c.CreateTopic(topic)
	_ = c.Publish(topic, payload)
}

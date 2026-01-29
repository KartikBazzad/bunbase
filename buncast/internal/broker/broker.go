package broker

import (
	"sync"
)

// Message is a published message with optional headers and opaque payload.
type Message struct {
	Topic   string
	Payload []byte
	Headers map[string]string // optional
}

// Subscriber receives messages for one or more topics.
// Send is called for each message; the broker does not block on slow subscribers.
type Subscriber interface {
	Send(msg *Message)
}

// SubscriberFunc adapts a function to Subscriber.
type SubscriberFunc func(msg *Message)

func (f SubscriberFunc) Send(msg *Message) { f(msg) }

// Broker is an in-memory topic broker: fan-out on publish, per-topic ordering.
type Broker struct {
	mu      sync.RWMutex
	topics  map[string]map[Subscriber]struct{} // topic -> set of subscribers
	bufSize int                                // per-subscriber channel buffer (0 = unbuffered send)
}

// New creates a new in-memory broker.
// bufSize is the buffer size for each subscriber's delivery channel; 0 means unbuffered (blocking send).
func New(bufSize int) *Broker {
	return &Broker{
		topics:  make(map[string]map[Subscriber]struct{}),
		bufSize: bufSize,
	}
}

// CreateTopic ensures a topic exists (idempotent).
func (b *Broker) CreateTopic(topic string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.topics[topic] == nil {
		b.topics[topic] = make(map[Subscriber]struct{})
	}
}

// DeleteTopic removes a topic and all its subscribers.
func (b *Broker) DeleteTopic(topic string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.topics, topic)
}

// TopicExists returns whether the topic exists.
func (b *Broker) TopicExists(topic string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, ok := b.topics[topic]
	return ok
}

// ListTopics returns all topic names.
func (b *Broker) ListTopics() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]string, 0, len(b.topics))
	for t := range b.topics {
		out = append(out, t)
	}
	return out
}

// Subscribe adds a subscriber to the given topic. If the topic does not exist, it is created.
func (b *Broker) Subscribe(topic string, sub Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.topics[topic] == nil {
		b.topics[topic] = make(map[Subscriber]struct{})
	}
	b.topics[topic][sub] = struct{}{}
}

// Unsubscribe removes a subscriber from a topic.
func (b *Broker) Unsubscribe(topic string, sub Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if subs := b.topics[topic]; subs != nil {
		delete(subs, sub)
		if len(subs) == 0 {
			delete(b.topics, topic)
		}
	}
}

// Publish sends a message to all subscribers of the topic. Fan-out is non-blocking:
// each subscriber receives the message in a goroutine so slow subscribers do not block others.
func (b *Broker) Publish(msg *Message) {
	if msg == nil {
		return
	}
	b.mu.RLock()
	subs := b.topics[msg.Topic]
	if subs == nil || len(subs) == 0 {
		b.mu.RUnlock()
		return
	}
	// Copy subscriber set so we don't hold the lock while sending
	subList := make([]Subscriber, 0, len(subs))
	for sub := range subs {
		subList = append(subList, sub)
	}
	b.mu.RUnlock()

	for _, sub := range subList {
		sub := sub
		go sub.Send(msg)
	}
}

// SubscriberCount returns the number of subscribers for a topic (0 if topic does not exist).
func (b *Broker) SubscriberCount(topic string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.topics[topic])
}

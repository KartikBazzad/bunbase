# Buncast Architecture

## Overview

Buncast is an in-memory topic broker. Publishers send messages to topics; subscribers receive messages for the topics they subscribe to. Delivery is at-least-once, best-effort (no persistence in v1).

## Components

- **Broker** (`internal/broker`): In-memory map of topic → set of subscribers. On Publish, the broker fans out the message to each subscriber (non-blocking goroutine per subscriber).
- **IPC server** (`internal/ipc`): Unix socket listener; length-prefixed request/response for CreateTopic, DeleteTopic, ListTopics, Publish; long-lived stream for Subscribe (server pushes message frames until client disconnects).
- **HTTP server** (`internal/http`): Health, list topics, and SSE subscribe (GET /subscribe?topic=...). Each SSE connection is registered as a subscriber; disconnect unsubscribes.

## Data flow

1. **Publish (IPC)**: Client sends Publish request (topic + payload) → handler creates topic if needed, broker.Publish(msg) → all subscribers receive the message.
2. **Subscribe (IPC)**: Client sends Subscribe request (topic) → handler registers a connSubscriber that writes message frames to the connection → server streams frames until client closes → server unregisters.
3. **Subscribe (HTTP/SSE)**: Client GET /subscribe?topic=... → server registers an sseSubscriber that writes SSE events → broker delivers messages as SSE data lines → client disconnect unsubscribes.

## Concurrency

- Broker: RWMutex for topic map; Publish copies subscriber set under read lock then sends without holding the lock.
- IPC: One goroutine per connection; Subscribe holds the connection exclusively for the stream.
- HTTP: One handler per request; SSE connection is single-writer (subscriber Send uses mutex).

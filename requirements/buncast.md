# Buncast (Pub/Sub) Requirements

## Purpose

Buncast is the Publish-Subscribe service for Bunbase. It provides an in-memory event bus so that:

- **Platform API** can publish events (e.g. function deployed, project updated).
- **Functions** service can subscribe (e.g. reload config, scale).
- **Platform Web** or CLI can subscribe over HTTP (SSE) for real-time UI.

## Core Requirements

### Topics

- Create and delete topics by name (e.g. `functions.deployments`, `project.{id}.events`).
- Topic names are opaque strings; no schema enforcement in v1.

### Publish

- Publish a message (opaque payload + optional headers) to a topic.
- Fire-and-forget or sync ack depending on transport (IPC ack in v1).

### Subscribe

- Subscribe to one or more topics; receive messages in order per topic (per-subscriber ordering).
- Server-to-server: Unix socket IPC (long-lived stream after Subscribe command).
- Client-facing: HTTP + Server-Sent Events (SSE) for dashboard/CLI.

### Delivery

- At-least-once delivery; v1 in-memory only (no persistent replay).

### Scoping

- Topics may be scoped by project/tenant in the future; v1 has no auth on topics.

## Non-Functional

- **Latency**: Sub-millisecond in-process; single-digit ms over Unix socket.
- **Throughput**: Thousands of messages/sec on one node (v1 single-node).
- **Operability**: Structured logging; optional Prometheus metrics (topics, subs, publish/deliver counts).

## Out of Scope (v1)

- Persistent replay / durable queues.
- Multi-node / clustered broker.
- Exactly-once delivery.
- Schema enforcement (payload remains opaque bytes/JSON).
- Authentication/authorization on topics.

## User Flows

1. **Platform deploys a function** → Platform API publishes to `functions.deployments` with project_id, name, version, service_id. Functions service (optional) subscribes to reload or scale.
2. **Dashboard real-time updates** → Platform Web opens SSE to `/subscribe?topic=...` and displays events (e.g. deploy completed).

# Buncast Design (RFC)

## Overview

Buncast is an in-memory Pub/Sub broker for Bunbase. It exposes:

- **IPC** (Unix socket): request/response for CreateTopic, DeleteTopic, ListTopics, Publish; long-lived stream for Subscribe.
- **HTTP**: health, list topics, SSE subscribe.

## Protocol: IPC

- **Framing**: Length-prefixed (4 bytes little-endian) then body.
- **Request body**: RequestID (8) + Command (1) + PayloadLen (4) + Payload.
- **Response body**: RequestID (8) + Status (1) + PayloadLen (4) + Payload.
- **Commands**: CreateTopic=1, DeleteTopic=2, ListTopics=3, Publish=4, Subscribe=5.
- **Status**: OK=0, Error=1.

### Payloads

- **CreateTopic / DeleteTopic / Subscribe**: TopicLen (2) + Topic (UTF-8).
- **Publish**: TopicLen (2) + Topic + PayloadLen (4) + Payload.
- **ListTopics response**: JSON array of topic strings.

### Subscribe flow

1. Client sends Subscribe request (topic in payload).
2. Server responds with OK.
3. Server streams message frames on the same connection: each frame = length (4) + TopicLen (2) + Topic + PayloadLen (4) + Payload.
4. When client closes the connection, server unregisters the subscriber.

## Protocol: HTTP

- **GET /health**: Returns `{"status":"ok"}`.
- **GET /topics**: Returns JSON array of topic names.
- **GET /subscribe?topic=NAME**: Opens Server-Sent Events stream; each event is the raw message payload. Client disconnect unsubscribes.

## Topic model

- Topics are created implicitly on first Publish or Subscribe, or explicitly via CreateTopic.
- DeleteTopic removes the topic and all subscribers.
- No project/tenant scoping in v1.

## Integration

- **Platform API**: Optional `--buncast-socket`; when set, after DeployFunction succeeds, publish JSON to topic `functions.deployments` with project_id, name, version, service_id.
- **Functions**: Can subscribe to `functions.deployments` over IPC to reload or scale (optional follow-up).
- **DocDB**: Optional future: publish change streams to a topic.

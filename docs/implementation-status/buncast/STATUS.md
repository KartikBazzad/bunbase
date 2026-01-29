# Buncast Implementation Status

## Overview

Buncast is the Pub/Sub service for Bunbase. This document tracks implementation progress.

## Checklist

- [x] Scaffold: module, go.work, cmd/server, config, logger
- [x] Broker: in-memory topics, Subscribe/Publish, fan-out
- [x] IPC server: Unix socket, CreateTopic, DeleteTopic, ListTopics, Publish, Subscribe (stream)
- [x] Go client: pkg/client (Publish, Subscribe, CreateTopic, DeleteTopic, ListTopics)
- [x] HTTP server: health, GET /topics, GET /subscribe (SSE)
- [x] Integration: Platform API optional Buncast client, publish on deploy (topic `functions.deployments`)
- [x] Docs: requirements, planning, architecture, README, configuration, API, implementation-status

## Optional follow-ups

- [ ] Functions service: subscribe to `functions.deployments` for reload/scale
- [ ] Prometheus metrics (topics count, subscriber count, publish/deliver counts)
- [ ] Config file support (YAML/JSON)

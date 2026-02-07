# BunBase Documentation

Welcome to the BunBase documentation. This directory contains cross-cutting documentation for the entire platform.

![BunBase logo](assets/bunbase-logo.svg)

## Product Documentation

- **[Product Catalog](products/README.md)** - Feature and ownership map for every service/product
- **[Architecture](architecture.md)** - High-level system architecture and component overview
- **[Inter-service RPC](inter-service-rpc.md)** - RPC topology and environment configuration
- **[API Paths](api-paths.md)** - Canonical API path conventions across layers

## Engineering Workflow

- **[Development Workflow](development-workflow.md)** - How to work with the BunBase monorepo
- **[Sharing Code Between Services](sharing-code-between-services.md)** - Guide for sharing code across Go services

## User Documentation

Complete guides for using BunBase:

- **[Getting Started](users/getting-started.md)** - Quick start: create account, deploy first function
- **[Writing Functions](users/writing-functions.md)** - How to write effective JavaScript/TypeScript functions
- **[CLI Guide](users/cli-guide.md)** - Complete command-line interface reference
- **[Platform API Reference](users/api-reference.md)** - REST API documentation
- **[Projects Guide](users/projects.md)** - Managing projects and organizing functions
- **[Troubleshooting](users/troubleshooting.md)** - Common issues and solutions

See [User Documentation Index](users/README.md) for the complete user guide.

## Historical Implementation Status

Historical implementation status and progress reports are in [`implementation-status/`](implementation-status/):

- **Platform**: `implementation-status/platform.md`
- **Bundoc**: `implementation-status/bundoc/`
- **Functions**: `implementation-status/functions/`
- **Buncast**: `implementation-status/buncast/` - Pub/Sub service status

## Service-Specific Documentation

Each service maintains its own documentation:

- **Platform API**: `../platform/README.md`
- **Functions**: `../functions/README.md`, `../functions/docs/`
- **Bundoc (Engine)**: `../bundoc/README.md`, `../bundoc/docs/`
- **Bundoc Server**: `../bundoc-server/README.md`
- **Buncast**: `../buncast/README.md`, `../buncast/docs/`
- **Bunder**: `../bunder/README.md`, `../bunder/docs/`
- **Bunder Manager**: `../bunder-manager/README.md`
- **BunKMS**: `../bun-kms/README.md`, `../bun-kms/docs/`
- **Platform Web**: `../platform-web/README.md`

## Planning & Requirements

- **Requirements**: `../requirements/` - Product requirements and user flows
- **Planning**: `../planning/` - Engineering designs, RFCs, and implementation plans

## Quick Links

- [Service Requirements](../requirements/services.md) - Unified per-service requirements
- [Service Implementation](../planning/service-implementation.md) - Current implementation inventory and gaps
- [Service Roadmap](../planning/service-roadmap.md) - Detailed roadmap by service

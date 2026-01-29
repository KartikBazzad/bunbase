# BunBase Documentation

Welcome to the BunBase documentation. This directory contains cross-cutting documentation for the entire platform.

## Core Documentation

- **[Architecture](architecture.md)** - High-level system architecture and component overview
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

## Implementation Status

Historical implementation status and progress reports are in [`implementation-status/`](implementation-status/):

- **Platform**: `implementation-status/platform.md`
- **DocDB**: `implementation-status/docdb/`
- **Functions**: `implementation-status/functions/`
- **Buncast**: `implementation-status/buncast/` - Pub/Sub service status

## Service-Specific Documentation

Each service maintains its own documentation:

- **DocDB**: `../docdb/docs/` - Architecture, usage, configuration, troubleshooting
- **Functions**: `../functions/docs/` - API reference, deployment, function development
- **Platform API**: `../platform/README.md` - API endpoints and setup
- **Platform Web**: `../platform-web/README.md` - Frontend setup and design system
- **Buncast**: `../buncast/README.md`, `../buncast/docs/` - Pub/Sub service, IPC and HTTP API

## Planning & Requirements

- **Requirements**: `../requirements/` - Product requirements and user flows
- **Planning**: `../planning/` - Engineering designs, RFCs, and implementation plans

## Quick Links

- [Monorepo Structure](../planning/monorepo-structure.md) - Repository organization
- [Shared Libraries Plan](../planning/shared-libraries.md) - Code sharing strategy

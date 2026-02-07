# Plan Registry

Registry of all active and completed implementation plans for the BunBase monorepo.

**Last Updated:** January 28, 2026

---

## Active Plans

## Active Plans

### Phase 1: Foundation (Infrastructure)
**Status**: 🚀 Ready for Implementation
**Docs**: [Roadmap](roadmap_phases.md), [Implementation Plan](implementation_plan.md)
**Scope**: Docker, BunAuth, Postgres, MinIO.

### Phase 2: Core Refactor
**Status**: 📋 Planned
**Docs**: [Functions Isolation](functions_runtime_isolation.md), [KMS Integration](kms_bundoc_integration.md), [Project Auth](project_auth_requirements.md)
**Scope**: Platform DB Migration, Functions Preload, Bundoc Encryption, Tenant Auth.

### Phase 3: Client Access
**Status**: 📋 Planned
**Docs**: [SDK Requirements](sdk_requirements.md), [Web Console](web_console_requirements.md), [CLI](cli_requirements.md), [Docs Site](docs_site_requirements.md)
**Scope**: JS SDK, Admin SDK, CLI, Web Console, Documentation.

### Phase 4: Observability
**Status**: 📋 Planned
**Docs**: [Monitoring](monitoring_requirements.md)
**Scope**: Prometheus, Grafana.

---

## Past Plans (Archived)
### v0.2 Improvements and Documentation
**Status**: ✅ Completed
**Date**: Jan 2026

---

## Completed Plans

### v0.1 Implementation

**Status:** ✅ Completed  
**Completed:** January 2026

**Overview:** Core database functionality, WAL rotation, data file checksums, checkpoint-based recovery, graceful shutdown, document corruption detection.

**Related Documents:**

- [Bundoc Status](../docs/implementation-status/bundoc/STATUS.md)
- [Bundoc Roadmap](../docs/implementation-status/bundoc/ROADMAP.md)

---

## Plan Metadata

### Status Values

- **Pending** - Plan created but not started
- **In Progress** - Active implementation
- **Completed** - All items implemented and verified
- **Cancelled** - Plan abandoned or superseded

### Plan Categories

- **Feature Implementation** - New feature development
- **Improvements** - Enhancements to existing features
- **Documentation** - Documentation updates and improvements
- **Testing** - Test coverage improvements
- **Refactoring** - Code quality improvements

---

## Adding a New Plan

When creating a new plan:

1. Create plan document (use naming conventions)
2. Add entry to this registry with:
   - Plan name
   - Status
   - Created date
   - Overview
   - Link to plan file
   - Related documents
3. Update status as work progresses
4. Move to "Completed Plans" when done

---

## Plan Tracking

Plans are tracked in:

- This registry (PLANS.md)
- Individual plan files (in `planning/` or `~/.cursor/plans/`)
- Related implementation status documents
- GitHub issues/PRs (when applicable)

---

## Notes

- Plans created via Cursor AI are stored in `~/.cursor/plans/`
- Plans should be linked from this registry for visibility
- Completed plans provide historical context for decisions
- Active plans help coordinate current work

# Planning Documentation

This directory contains planning documents, RFCs, and implementation plans for the BunBase monorepo.

## Purpose

Planning documents serve to:

- Document design decisions and rationale
- Track implementation plans and progress
- Coordinate work across services
- Provide historical context for decisions

## Plan Registry

See [PLANS.md](PLANS.md) for a registry of all active and completed plans.

## Plan Lifecycle

Plans follow this lifecycle:

1. **Created** - Plan document created with initial scope
2. **In Progress** - Implementation actively underway
3. **Completed** - All plan items implemented and verified
4. **Archived** - Plan moved to historical record

## Plan Structure

Plans should include:

- **Overview** - High-level description of the plan
- **Current State** - Analysis of existing implementation
- **Improvement Areas** - Specific areas to address
- **Implementation Details** - Files to modify, approach to take
- **Success Criteria** - How to measure completion
- **Estimated Effort** - Time/resource estimates

## Creating a Plan

When creating a new plan:

1. Create plan document in this directory or appropriate subdirectory
2. Add entry to [PLANS.md](PLANS.md)
3. Link plan to related implementation status documents
4. Update plan status as work progresses

## Plan Naming Conventions

- Use descriptive names: `v0.2-improvements.md`, `tcp-support-rfc.md`
- Include version numbers when applicable
- Use kebab-case for file names
- Include date in filename if versioned: `plan-2026-01-28.md`

## Related Documentation

- **Requirements:** `../requirements/` - Product requirements and user flows
- **Architecture:** `../docs/architecture.md` - System architecture
- **Implementation Status:** `../docs/implementation-status/` - Implementation tracking
- **Service Implementation Baseline:** `service-implementation.md` - Current per-service implementation map
- **Service Roadmap:** `service-roadmap.md` - Detailed roadmap by service

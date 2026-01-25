# BunBase Development Tickets

This directory contains all development tickets for BunBase services, SDKs, and CLI tool.

## Structure

```
tickets/
├── services/          # Core service tickets
│   ├── auth-service/
│   ├── database-service/
│   ├── storage-service/
│   ├── functions-service/
│   ├── realtime-service/
│   └── api-gateway-service/
├── sdks/              # SDK tickets
│   ├── js-sdk/
│   ├── python-sdk/
│   └── go-sdk/
└── cli/               # CLI tool tickets
```

## Ticket Summary

### Core Services (30 tickets)
- **Authentication Service**: 5 tickets (AUTH-001 to AUTH-005)
- **Database Service**: 6 tickets (DB-001 to DB-006)
- **Storage Service**: 5 tickets (STG-001 to STG-005)
- **Functions Service**: 5 tickets (FN-001 to FN-005)
- **Real-time Service**: 4 tickets (RT-001 to RT-004)
- **API Gateway Service**: 5 tickets (GW-001 to GW-005)

### Client SDKs (16 tickets)
- **JavaScript/TypeScript SDK**: 6 tickets (SDK-JS-001 to SDK-JS-006)
- **Python SDK**: 5 tickets (SDK-PY-001 to SDK-PY-005)
- **Go SDK**: 5 tickets (SDK-GO-001 to SDK-GO-005)

### Developer Tools (6 tickets)
- **CLI Tool**: 6 tickets (CLI-001 to CLI-006)

**Total: 52 tickets**

## Creating GitHub Issues

To create GitHub issues from these tickets:

1. **Authenticate with GitHub CLI** (if not already):
   ```bash
   gh auth login
   ```

2. **Run the script**:
   ```bash
   ./create-github-issues.sh
   ```

The script will:
- Read each ticket file
- Extract the title and content
- Create a GitHub issue with appropriate labels
- Add component and priority labels automatically

## Ticket Format

Each ticket follows this structure:
- **Title**: Clear, descriptive title
- **Component**: Service/SDK name
- **Type**: Feature/Epic
- **Priority**: High/Medium/Low
- **Description**: Overview
- **Requirements**: Detailed requirements
- **Technical Requirements**: API endpoints, schemas, performance
- **Tasks**: Breakdown of subtasks
- **Acceptance Criteria**: Success criteria
- **Dependencies**: Related tickets
- **Estimated Effort**: Story points

## Labels

Issues will be automatically labeled with:
- `type:feature` - All tickets are features
- `component:*` - Based on service/SDK (e.g., `component:auth`, `component:sdk-js`)
- `priority:*` - Based on ticket priority (e.g., `priority:high`, `priority:medium`)

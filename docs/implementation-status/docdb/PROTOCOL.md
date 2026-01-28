# DocDB Shell Protocol Mapping

This document maps shell commands to IPC calls.

## Command → IPC Mapping

| Shell Command | IPC Command | Operation | DB Required? | Description |
|---------------|-------------|-----------|--------------|-------------|
| `.open <name>` | `CmdOpenDB` | `OpCreate` (payload=name) | No | Open or create database |
| `.close` | `CmdCloseDB` | - | Yes | Close current database |
| `.create <id> <payload>` | `CmdExecute` | `OpCreate` | Yes | Create document |
| `.read <id>` | `CmdExecute` | `OpRead` | Yes | Read document |
| `.update <id> <payload>` | `CmdExecute` | `OpUpdate` | Yes | Update document |
| `.delete <id>` | `CmdExecute` | `OpDelete` | Yes | Delete document |
| `.stats` | `CmdStats` | - | No | Print pool statistics |
| `.mem` | `CmdStats` | - | No | Print memory usage |
| `.wal` | `CmdStats` | - | No | Print WAL info |

## Shell State

The shell maintains only:
- `current_db_id` (uint64, 0 = none)
- `transaction_active` (bool, not implemented in v0)

All other state lives on the server.

## IPC Request Flow

1. Parse command and arguments
2. Validate preconditions (database open, valid doc_id, etc.)
3. Decode payload (if applicable)
4. Build IPC request frame
5. Send request via Unix socket
6. Receive response frame
7. Parse response
8. Format output
9. Print result

## Error Handling

- IPC errors abort the command immediately
- Connection loss exits the shell
- Invalid commands do not affect shell state
- Server errors are surfaced verbatim

## Safety Invariants

- No command reordering
- No implicit retries
- No automatic batching
- No round writes
- No mutation without explicit user action

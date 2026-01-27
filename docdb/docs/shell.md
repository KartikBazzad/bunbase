# DocDB Shell (docdbsh)

The DocDB Shell is a debugging and administrative CLI for DocDB. It provides a thin, explicit interface to the IPC protocol with deterministic behavior and transparent error reporting.

## Table of Contents

1. [Overview](#overview)
2. [Installation](#installation)
3. [Quick Start](#quick-start)
4. [Commands](#commands)
5. [Payload Formats](#payload-formats)
6. [Output Format](#output-format)
7. [Design Principles](#design-principles)
8. [Examples](#examples)
9. [Error Handling](#error-handling)
10. [Advanced Usage](#advanced-usage)
11. [Troubleshooting](#troubleshooting)

---

## Overview

### Purpose

The DocDB Shell (`docdbsh`) exists to:

- Exercise the IPC protocol directly
- Inspect database behavior interactively
- Validate correctness, not performance
- Provide a human-facing witness to system state

### Key Characteristics

- **Thin Client**: Every shell command maps directly to one IPC request
- **Explicitness Over Convenience**: No guessing, no defaults that mutate state
- **No Hidden State**: All state (except current DB) is server-owned
- **Failure Transparency**: Errors are printed verbatim and immediately
- **Deterministic Behavior**: Same input → same IPC calls → same output

### Non-Goals (Strictly Enforced)

The shell does NOT include:

- ❌ SQL query language
- ❌ Auto-complete
- ❌ History persistence
- ❌ Scripting language
- ❌ Pipes / redirection
- ❌ Query planner
- ❌ Index inspection
- ❌ Performance benchmarking
- ❌ Transaction support (`.begin`, `.commit`, `.rollback`) - Not in IPC protocol
- ❌ `.list-dbs` - Not in IPC protocol

---

## Installation

### Build from Source

```bash
# Build the shell
go build -o docdbsh ./cmd/docdbsh

# Move to PATH (optional)
sudo mv docdbsh /usr/local/bin/
```

### Requirements

- Go 1.25.6 or higher
- DocDB server running with IPC socket
- No external dependencies (Go stdlib only)

---

## Quick Start

### Start DocDB Server

```bash
./docdb --socket /tmp/docdb.sock
```

### Start Shell

```bash
./docdbsh --socket /tmp/docdb.sock
```

### Example Session

```
$ ./docdbsh
DocDB Shell v0
Connecting to /tmp/docdb.sock...
Connected. Type '.help' for commands.

> .open testdb
OK
db_id=1

> .create 1 raw:"Hello, World!"
OK

> .read 1
OK
len=13
hex=48656c6c6f2c20576f726c6421

> .stats
OK
total_dbs=1
active_dbs=1
total_txns=1
wal_size=64
memory_used=13

> .exit
```

---

## Commands

### Meta Commands

#### `.help`
Show command list and usage information.

```
> .help
```

#### `.exit`
Exit the shell.

```
> .exit
```

You can also use `Ctrl+D` (EOF) to exit.

#### `.clear`
Clear local shell state (database + transaction).

```
> .clear
OK
```

Use this to reset the shell without exiting.

---

### Database Lifecycle

#### `.open <db_name>`
Open or create a database. Sets the current database ID.

```
> .open testdb
OK
db_id=1
```

- Creates database if it doesn't exist
- Returns `db_id` (used internally)
- All subsequent operations use this database
- Error if database cannot be opened

#### `.close`
Close the current database. Clears the current database ID.

```
> .close
OK
```

- Error if no database is open
- Subsequent operations require opening a database first

---

### CRUD Operations

All CRUD commands require an open database.

#### `.create <doc_id> <payload>`
Create a new document with the specified ID and payload.

```
> .create 1 raw:"Hello, World!"
OK
```

- `doc_id`: Unsigned integer (uint64)
- `payload`: See [Payload Formats](#payload-formats)
- Error if document already exists
- Error if memory limit exceeded

#### `.read <doc_id>`
Read a document by ID.

```
> .read 1
OK
len=13
hex=48656c6c6f2c20576f726c6421
```

- Returns length and hex representation
- If JSON payload, also returns pretty-printed JSON
- Error if document not found

#### `.update <doc_id> <payload>`
Update an existing document.

```
> .update 1 raw:"Updated message"
OK
```

- `doc_id`: Unsigned integer
- `payload`: See [Payload Formats](#payload-formats)
- Error if document doesn't exist
- Memory delta tracked (old size freed, new size allocated)

#### `.delete <doc_id>`
Delete a document.

```
> .delete 1
OK
```

- `doc_id`: Unsigned integer
- Error if document doesn't exist
- Memory freed on successful delete

---

### Introspection

#### `.stats`
Print pool statistics.

```
> .stats
OK
total_dbs=1
active_dbs=1
total_txns=5
wal_size=320
memory_used=42
```

#### `.mem`
Print memory usage details.

```
> .mem
OK
memory_used=42
memory_capacity=0
usage_percent=Infinity
```

#### `.wal`
Print WAL information.

```
> .wal
OK
wal_size=320
```

---

## Payload Formats

The shell supports **exactly three** payload formats. No automatic detection.

### Raw String

Format: `raw:"<string>"` or `raw:<string>`

```
> .create 1 raw:"Hello, World!"
> .create 2 raw:Unquoted string
```

- UTF-8 encoded
- Quotes are optional
- Minimal escaping

### Hex Bytes

Format: `hex:<hex_string>`

```
> .create 3 hex:48656c6c6f
```

- Even-length hex string
- Decoded directly to bytes
- Error if invalid hex

### JSON

Format: `json:<json_object>`

```
> .create 4 json:{"name":"Alice","age":30}
```

- Parsed client-side
- Marshaled to bytes
- Server treats as opaque payload
- Pretty-printed when read

### Invalid Payloads

- Missing prefix: `ERROR\npayload must have prefix: raw:, hex:, or json:`
- Invalid JSON: `ERROR\ninvalid json: <error>`
- Invalid hex: `ERROR\ninvalid hex: <error>`

---

## Output Format

### Success

```
OK
```

### Success with Details

```
OK
db_id=1
```

### Read Result

```
OK
len=13
hex=48656c6c6f2c20576f726c6421
```

If JSON:

```
OK
len=24
hex=7b226e616d65223a22416c696365222c22616765223a33307d
json={"age":30,"name":"Alice"}
```

### Error

```
ERROR
<error message>
```

Errors are **never** suppressed or reworded. They are printed verbatim from the server.

---

## Design Principles

### 1. Thin Client

Every shell command maps directly to one IPC request:

```
.create 1 raw:"hello"
→ IPC Request: CmdExecute with OpCreate
→ IPC Response: StatusOK
→ Shell Output: OK
```

### 2. Explicitness Over Convenience

No guessing. No defaults that mutate state:

- Payload format must be explicit (`raw:`, `hex:`, `json:`)
- Database must be explicitly opened
- No automatic retry on failure
- No implicit batching

### 3. No Hidden State

Only shell state maintained:

- `current_db_id` (optional, uint64)
- `transaction_active` (bool, not implemented in v0)

All other state lives on the server.

### 4. Failure Transparency

Errors are printed verbatim:

```
> .read 999
ERROR
document not found
```

No rewording, no interpretation, no suppression.

### 5. Deterministic Behavior

Same input → same IPC calls → same output:

```
> .create 1 raw:"hello"
OK

> .create 1 raw:"hello"
ERROR
document already exists
```

Predictable, reproducible behavior.

---

## Examples

### Basic CRUD

```bash
# Open database
> .open testdb
OK
db_id=1

# Create document with raw string
> .create 1 raw:"Hello, World!"
OK

# Read document
> .read 1
OK
len=13
hex=48656c6c6f2c20576f726c6421

# Update document
> .update 1 raw:"Updated!"
OK

# Delete document
> .delete 1
OK
```

### JSON Payloads

```bash
# Create JSON document
> .create 10 json:{"id":1,"name":"Alice","age":30}
OK

# Read JSON document
> .read 10
OK
len=36
hex=7b226964223a312c226e616d65223a22416c696365222c22616765223a33307d
json={"age":30,"id":1,"name":"Alice"}
```

### Hex Payloads

```bash
# Create binary data
> .create 20 hex:0102030405060708
OK

# Read binary data
> .read 20
OK
len=8
hex=0102030405060708
```

### Multiple Documents

```bash
> .open users
OK
db_id=2

> .create 1 json:{"name":"Alice","age":30}
OK

> .create 2 json:{"name":"Bob","age":25}
OK

> .create 3 json:{"name":"Charlie","age":35}
OK

> .read 2
OK
len=30
hex=7b226e616d65223a22426f62222c22616765223a32357d
json={"age":25,"name":"Bob"}
```

### Statistics

```bash
> .open testdb
OK
db_id=3

> .create 1 raw:"test"
OK

> .stats
OK
total_dbs=3
active_dbs=3
total_txns=6
wal_size=384
memory_used=4

> .mem
OK
memory_used=4
memory_capacity=0
usage_percent=Infinity

> .wal
OK
wal_size=384
```

### Error Handling

```bash
> .read 999
ERROR
document not found

> .create 1 raw:"test"
ERROR
document already exists

> .update 999 raw:"test"
ERROR
document not found

> .delete 999
ERROR
document not found
```

---

## Error Handling

### Connection Errors

```
$ ./docdbsh --socket /tmp/nonexistent.sock
DocDB Shell v0
Connecting to /tmp/nonexistent.sock...
Failed to connect: failed to connect to server
```

Connection loss exits the shell immediately.

### Command Errors

```
> .read 999
ERROR
document not found
```

Command errors do **not** affect shell state.

### Invalid Commands

```
> invalid-command
ERROR
unknown command: invalid-command
```

Invalid commands do **not** affect shell state.

### Database Not Open

```
> .create 1 raw:"test"
ERROR
no database open
```

Open a database with `.open <db_name>` first.

---

## Advanced Usage

### Shell State Management

```bash
# Check current database (internal state)
# Note: db_id is internal, not a command

# Clear state
> .clear
OK

# Close database
> .close
OK
```

### Signal Handling

- `Ctrl+C` (SIGINT): Interrupts current operation, does not exit shell
- `Ctrl+D` (EOF): Exits shell

### Multiple Terminals

You can run multiple shell instances simultaneously:

```bash
# Terminal 1
$ ./docdbsh
> .open testdb

# Terminal 2
$ ./docdbsh
> .open testdb
> .read 1
```

Both interact with the same database.

---

## Troubleshooting

### Cannot Connect to Server

**Symptom**:
```
$ ./docdbsh --socket /tmp/docdb.sock
Failed to connect: failed to connect to server
```

**Solutions**:
1. Start DocDB server: `./docdb --socket /tmp/docdb.sock`
2. Check socket path: Ensure it matches the server's socket path
3. Check permissions: Ensure you have read/write access to socket

### No Database Open

**Symptom**:
```
> .create 1 raw:"test"
ERROR
no database open
```

**Solution**:
```
> .open testdb
OK
```

### Document Already Exists

**Symptom**:
```
> .create 1 raw:"test"
ERROR
document already exists
```

**Solution**:
Use a different document ID, or update the existing document with `.update`.

### Invalid Payload Format

**Symptom**:
```
> .create 1 test
ERROR
payload must have prefix: raw:, hex:, or json:
```

**Solution**:
Use a valid payload format:
```
> .create 1 raw:"test"
```

### Invalid JSON

**Symptom**:
```
> .create 1 json:{invalid}
ERROR
invalid json: invalid character 'i' looking for beginning of value
```

**Solution**:
Use valid JSON:
```
> .create 1 json:{"key":"value"}
```

### Invalid Hex

**Symptom**:
```
> .create 1 hex:xyz
ERROR
invalid hex: encoding/hex: invalid byte: U+0078 'x'
```

**Solution**:
Use valid hex:
```
> .create 1 hex:48656c6c6f
```

---

## Additional Resources

- [Shell README](../cmd/docdbsh/README.md) - Quick reference
- [Protocol Mapping](../cmd/docdbsh/PROTOCOL.md) - Command to IPC mapping
- [Example Session](../cmd/docdbsh/SESSION_TRANSCRIPT.md) - Interactive example
- [Implementation Summary](../cmd/docdbsh/IMPLEMENTATION_SUMMARY.md) - Technical details
- [Main README](../README.md) - Project overview
- [Usage Guide](usage.md) - General DocDB usage

---

## License

Same as DocDB project.

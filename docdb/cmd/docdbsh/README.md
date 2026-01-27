# DocDB Shell (docdbsh)

A debugging and administrative CLI for DocDB.

## Usage

```bash
./docdbsh --socket /tmp/docdb.sock
```

## Commands

### Meta Commands
- `.help` - Show this help message
- `.exit` - Exit the shell
- `.clear` - Clear shell state (database + transaction)

### Database Lifecycle
- `.open <db_name>` - Open or create a database
- `.close` - Close the current database

### CRUD Operations
- `.create <doc_id> <payload>` - Create a document
- `.read <doc_id>` - Read a document
- `.update <doc_id> <payload>` - Update a document
- `.delete <doc_id>` - Delete a document

### Payload Formats

#### Raw String
```
.create 1 raw:"Hello world"
```

#### Hex Bytes
```
.create 2 hex:48656c6c6f
```

#### JSON
```
.create 3 json:{"name":"Alice","age":30}
```

### Introspection
- `.stats` - Print pool statistics
- `.mem` - Print memory usage
- `.wal` - Print WAL info

## Example Session

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

> .create 2 json:{"name":"Alice","age":30}
OK

> .read 2
OK
len=24
hex=7b226e616d65223a22416c696365222c22616765223a33307d
json={"age":30,"name":"Alice"}

> .stats
OK
total_dbs=1
active_dbs=1
total_txns=2
wal_size=128
memory_used=37

> .exit
```

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

## Requirements

- Go 1.25.6 or higher
- DocDB server running with IPC socket

## Design Principles

1. **Thin Client** - Every shell command maps directly to one IPC request
2. **Explicitness Over Convenience** - No guessing, no defaults that mutate state
3. **No Hidden State** - All state (except current DB) is server-owned
4. **Failure Transparency** - Errors are printed verbatim and immediately
5. **Deterministic Behavior** - Same input → same IPC calls → same output

## Scope (v0)

The shell does NOT include:
- Query language
- Auto-complete
- History persistence
- Scripting language
- Pipes / redirection
- Performance benchmarking
- Transaction support (.begin, .commit, .rollback)

## License

Same as DocDB

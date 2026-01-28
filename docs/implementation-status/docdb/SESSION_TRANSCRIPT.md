# Example Shell Session Transcript

```
$ ./docdbsh --socket /tmp/docdb.sock
DocDB Shell v0
Connecting to /tmp/docdb.sock...
Connected. Type '.help' for commands.

> .help
DocDB Shell Commands:

Meta Commands:
  .help     Show this help message
  .exit     Exit the shell
  .clear    Clear shell state (db + tx)

Database Lifecycle:
  .open <db_name>    Open or create database
  .close             Close current database

CRUD Operations:
  .create <doc_id> <payload>      Create document
  .read <doc_id>                   Read document
  .update <doc_id> <payload>      Update document
  .delete <doc_id>                 Delete document

Payload Formats:
  raw:"Hello world"   Raw string
  hex:48656c6c6f      Hex bytes
  json:{"key":"val"}  JSON object

Introspection:
  .stats    Print pool statistics
  .mem      Print memory usage
  .wal      Print WAL info

> .open testdb
OK
db_id=1

> .stats
OK
total_dbs=1
active_dbs=1
total_txns=0
wal_size=0
memory_used=0

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

> .create 3 hex:48656c6c6f
OK

> .read 3
OK
len=5
hex=48656c6c6f

> .update 1 raw:"Updated message"
OK

> .read 1
OK
len=15
hex=55706461746564206d657373616765

> .delete 2
OK

> .read 2
ERROR
document not found

> .stats
OK
total_dbs=1
active_dbs=1
total_txns=5
wal_size=320
memory_used=42

> .mem
OK
memory_used=42
memory_capacity=0
usage_percent=Infinity

> .wal
OK
wal_size=320

> .close
OK

> .read 1
ERROR
no database open

> .clear
OK

> .exit
$
```

## Notes

- The shell maintains state (current database) only during the session
- All data is stored on the server side
- Errors are transparent and come directly from the server
- The shell is a thin client - no caching, no retries, no batching

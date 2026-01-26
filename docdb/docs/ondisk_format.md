# On-Disk Format

This document describes the binary formats used by DocDB for data storage.

## WAL (Write-Ahead Log) Format

### Overview

The WAL is a global append-only log that records all write operations. It enables crash recovery and durability.

### Record Structure

Each WAL record has the following binary format:

```
[8 bytes: record_len]    Total length of this record in bytes
[8 bytes: tx_id]         Transaction ID
[8 bytes: db_id]         Logical database ID
[1 byte:  op_type]       Operation type (1=create, 2=read, 3=update, 4=delete)
[8 bytes: doc_id]        Document ID
[4 bytes: payload_len]   Length of payload
[N bytes: payload]       Document payload (if applicable)
[4 bytes: crc32]        CRC32 checksum of the entire record
```

### Constants

- `record_len_size`: 8 bytes
- `tx_id_size`: 8 bytes
- `db_id_size`: 8 bytes
- `op_type_size`: 1 byte
- `doc_id_size`: 8 bytes
- `payload_len_size`: 4 bytes
- `crc_size`: 4 bytes
- `header_size`: 33 bytes (sum of all sizes except payload and CRC)
- `record_overhead`: 37 bytes (header + CRC)

### Operation Types

| Value | Operation | Description |
|-------|-----------|-------------|
| 1     | Create    | Insert new document |
| 2     | Read      | Fetch document (for logging) |
| 3     | Update    | Replace full document |
| 4     | Delete    | Mark document as deleted |

### CRC32

- Algorithm: IEEE 802.3
- Computed over all bytes in the record **except** the CRC field itself
- Failure to match CRC results in record being ignored during recovery

### Example

Record for creating document ID=1 with payload "hello":

```
0x1D 0x00 0x00 0x00 0x00 0x00 0x00 0x00    # record_len = 29
0x01 0x00 0x00 0x00 0x00 0x00 0x00 0x00    # tx_id = 1
0x01 0x00 0x00 0x00 0x00 0x00 0x00 0x00    # db_id = 1
0x01                                      # op_type = create
0x01 0x00 0x00 0x00 0x00 0x00 0x00 0x00    # doc_id = 1
0x05 0x00 0x00 0x00                        # payload_len = 5
0x68 0x65 0x6C 0x6C 0x6F                   # "hello"
0xXX 0xXX 0xXX 0xXX                        # crc32
```

## Data File Format

### Overview

Data files store document payloads in an append-only manner. No in-place mutations occur.

### Record Structure

Each record in the data file:

```
[4 bytes: payload_len]   Length of payload
[N bytes: payload]       Document payload
```

### Constants

- `payload_len_size`: 4 bytes
- `max_payload_size`: 16 MB (16,777,216 bytes)

### Offsets

- Documents are addressed by byte offset from the start of the file
- Offsets are stored in the in-memory index
- Offpoints to the `payload_len` field of the record

### Example

Record for storing "hello":

```
0x05 0x00 0x00 0x00    # payload_len = 5
0x68 0x65 0x6C 0x6C 0x6F   # "hello"
```

## Catalog Format

### Overview

The catalog stores metadata about logical databases (names, IDs, status).

### Entry Structure

Each catalog entry:

```
[8 bytes: db_id]         Database ID
[2 bytes: name_len]      Length of database name
[1 byte:  status]       Database status (1=active, 2=deleted)
[N bytes: name]          Database name (UTF-8 string)
```

### Constants

- `db_id_size`: 8 bytes
- `name_len_size`: 2 bytes
- `status_size`: 1 byte
- `entry_header_size`: 11 bytes

### Status Values

| Value | Status   | Description |
|-------|----------|-------------|
| 1     | Active   | Database is active and usable |
| 2     | Deleted  | Database is marked for deletion |

### Example

Entry for database ID=1 named "mydb" (active):

```
0x01 0x00 0x00 0x00 0x00 0x00 0x00 0x00    # db_id = 1
0x04 0x00                                # name_len = 4
0x01                                     # status = active
0x6D 0x79 0x64 0x62                       # "mydb"
```

## IPC Frame Format

### Request Frame

```
[8 bytes: request_id]   Unique request identifier
[8 bytes: db_id]       Logical database ID
[1 byte:  command]     Command type (1=open_db, 2=close_db, 3=execute, 4=stats)
[4 bytes: op_count]    Number of operations
[op_count operations...]
```

### Operation Format

Each operation within a request:

```
[1 byte:  op_type]    Operation type
[8 bytes: doc_id]     Document ID
[4 bytes: payload_len] Length of payload (0 if none)
[N bytes: payload]     Payload data
```

### Response Frame

```
[8 bytes: request_id]   Request ID being responded to
[1 byte:  status]      Response status (0=ok, 1=error, 2=not_found, 3=memory_limit)
[4 bytes: data_len]    Length of response data
[N bytes: data]        Response data (command-specific)
```

### Status Values

| Value | Status         | Description |
|-------|---------------|-------------|
| 0     | OK            | Operation succeeded |
| 1     | Error         | General error occurred |
| 2     | NotFound      | Document not found |
| 3     | Conflict      | Document already exists |
| 4     | MemoryLimit   | Memory limit exceeded |

### Command Types

| Value | Command  | Description |
|-------|----------|-------------|
| 1     | OpenDB   | Create/open a logical database |
| 2     | CloseDB  | Close a logical database |
| 3     | Execute  | Execute operations on a database |
| 4     | Stats    | Get pool statistics |

## Endianness

All multi-byte integers are stored in **little-endian** byte order.

## File Naming Conventions

- `db_<name>.data` - Data file for logical database
- `db_<name>.wal` - WAL file for logical database
- `.catalog` - Catalog file (meta-database)
- `.compact` - Temporary file during compaction

## Recovery Procedure

1. Load catalog from `.catalog` file
2. Sequentially read WAL records
3. Validate CRC32 for each record
4. Rebuild in-memory index from valid records
5. Truncate WAL at first corrupted record
6. Begin serving requests

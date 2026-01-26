# @docdb/client

TypeScript/Bun client for DocDB - a file-based ACID document database.

## Features

- 🚀 **Fast**: Built with Bun for maximum performance
- 📝 **Type-safe**: Full TypeScript support with strict types
- 🔌 **Unix Socket**: Native Unix domain socket support
- 🔄 **Batch Operations**: Execute multiple operations efficiently
- 📦 **Dual Build**: ESM + CommonJS support
- 🛠️ **Utilities**: JSON serialization helpers included

## Installation

```bash
bun install @docdb/client
```

## Quick Start

### Basic API (Binary Payloads)

```typescript
import { DocDBClient } from '@docdb/client';

const client = new DocDBClient({
  socketPath: '/tmp/docdb.sock'
});

// Open a database
const dbID = await client.openDB('mydb');

// Create a document
await client.create(dbID, 1n, new TextEncoder().encode('Hello, DocDB!'));

// Read a document
const data = await client.read(dbID, 1n);
console.log(new TextDecoder().decode(data)); // Hello, DocDB!

// Update a document
await client.update(dbID, 1n, new TextEncoder().encode('Updated!'));

// Delete a document
await client.delete(dbID, 1n);

// Get stats
const stats = await client.stats();
console.log(stats);
```

### JSON API (Type-Safe)

```typescript
import { DocDBJSONClient } from '@docdb/client';

interface User {
  id: number;
  name: string;
  email: string;
}

const client = new DocDBJSONClient();

// Create database
const dbID = await client.openDB('usersdb');

// Create JSON document
const user: User = {
  id: 1,
  name: 'John Doe',
  email: 'john@example.com'
};
await client.createJSON(dbID, 1n, user);

// Read JSON document (type-safe)
const fetched = await client.readJSON<User>(dbID, 1n);
console.log(fetched); // { id: 1, name: 'John Doe', email: 'john@example.com' }

// Update JSON document
fetched.name = 'Jane Doe';
await client.updateJSON(dbID, 1n, fetched);

// Delete document
await client.delete(dbID, 1n);
```

### Batch Operations

```typescript
import { DocDBClient, Operation } from '@docdb/client';

const client = new DocDBClient();
const dbID = await client.openDB('batchdb');

// Create multiple documents in one request
const ops: Operation[] = [
  {
    opType: OperationType.Create,
    docID: 1n,
    payload: new TextEncoder().encode('Doc 1')
  },
  {
    opType: OperationType.Create,
    docID: 2n,
    payload: new TextEncoder().encode('Doc 2')
  },
  {
    opType: OperationType.Create,
    docID: 3n,
    payload: new TextEncoder().encode('Doc 3')
  }
];

const results = await client.batchExecute(dbID, ops);
console.log(`Created ${results.length} documents`);
```

## API Reference

### DocDBClient

The main client class for DocDB operations with binary payloads.

#### Constructor

```typescript
new DocDBClient(options?: ClientOptions)
```

**Options:**
- `socketPath` (string, default: `'/tmp/docdb.sock'`): Path to DocDB Unix socket
- `autoConnect` (boolean, default: `true`): Auto-connect on first operation
- `timeout` (number, default: `30000`): Request timeout in milliseconds

#### Methods

**openDB(name: string): Promise<bigint>**
- Create/open a logical database
- Returns: Database ID

**closeDB(dbID: bigint): Promise<void>**
- Close a logical database

**create(dbID: bigint, docID: bigint, payload: Uint8Array): Promise<void>**
- Create a new document
- Throws: `DocDBError` on failure

**read(dbID: bigint, docID: bigint): Promise<Uint8Array>**
- Read a document by ID
- Throws: `DocDBError` if not found

**update(dbID: bigint, docID: bigint, payload: Uint8Array): Promise<void>**
- Update an existing document
- Throws: `DocDBError` on failure

**delete(dbID: bigint, docID: bigint): Promise<void>**
- Delete a document
- Throws: `DocDBError` on failure

**batchExecute(dbID: bigint, ops: Operation[]): Promise<Uint8Array[]>**
- Execute multiple operations atomically
- Returns: Array of response data for each operation

**stats(): Promise<DocDBStats>**
- Get pool statistics

**connect(): Promise<void>**
- Explicitly connect to DocDB server

**disconnect(): Promise<void>**
- Disconnect from DocDB server

### DocDBJSONClient

Type-safe JSON API extending DocDBClient.

#### Methods

**createJSON<T>(dbID: bigint, docID: bigint, data: T): Promise<void>**
- Create a document with JSON payload
- Automatically serializes data to JSON

**readJSON<T>(dbID: bigint, docID: bigint): Promise<T | null>**
- Read a document and deserialize from JSON
- Returns `null` if document not found

**updateJSON<T>(dbID: bigint, docID: bigint, data: T): Promise<void>**
- Update a document with JSON payload
- Automatically serializes data to JSON

All other DocDBClient methods are inherited (openDB, closeDB, delete, batchExecute, stats, connect, disconnect).

## Error Handling

```typescript
import { DocDBError, Status } from '@docdb/client';

try {
  await client.create(dbID, 1n, payload);
} catch (e: DocDBError) {
  if (e.code === Status.NotFound) {
    console.error('Document not found');
  } else if (e.code === Status.Conflict) {
    console.error('Document already exists');
  } else {
    console.error('Error:', e.message);
  }
}
```

## Status Codes

| Code | Name               | Description                              |
|------|--------------------|------------------------------------------|
| 0    | OK                 | Operation successful                      |
| 1    | Error              | General error                           |
| 2    | NotFound          | Document/database not found             |
| 3    | Conflict           | Document already exists                 |
| 4    | MemoryLimit        | Memory limit exceeded                   |

## Building from Source

```bash
# Install dependencies
bun install

# Build library
bun run build

# Run examples
bun run example
bun run example:json

# Run tests
bun test
```

## Protocol Details

This client implements the DocDB binary protocol over Unix domain sockets:

- **Encoding**: Little-endian
- **Frame Format**: Length-prefixed binary frames
- **Max Frame Size**: 16 MB

See [Protocol Documentation](https://github.com/kartikbazzad/docdb) for full protocol specification.

## License

MIT

## Contributing

Contributions welcome! Please open an issue or PR.

## Related Projects

- [DocDB Server](https://github.com/kartikbazzad/docdb) - Go-based DocDB implementation

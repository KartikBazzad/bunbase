# Bundoc API Reference

**Version:** 1.0  
**Last Updated:** February 1, 2026

---

## Table of Contents

1. [Getting Started](#getting-started)
2. [Database](#database)
3. [Collections](#collections)
4. [Transactions](#transactions)
5. [Documents](#documents)
6. [Error Handling](#error-handling)
7. [Examples](#examples)

---

## Getting Started

### Installation

```bash
go get github.com/kartikbazzad/bunbase/bundoc
```

### Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/kartikbazzad/bunbase/bundoc"
    "github.com/kartikbazzad/bunbase/bundoc/mvcc"
    "github.com/kartikbazzad/bunbase/bundoc/storage"
)

func main() {
    // Open database
    opts := bundoc.DefaultOptions("./mydb")
    db, err := bundoc.Open(opts)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Create collection
    users, err := db.CreateCollection("users")
    if err != nil {
        log.Fatal(err)
    }

    // Insert document
    txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
    doc := storage.Document{
        "_id":  "user-1",
        "name": "Alice",
        "age":  30,
    }
    users.Insert(txn, doc)
   db.txnMgr.Commit(txn)

    fmt.Println("Document inserted!")
}
```

---

## Database

### Options

```go
type Options struct {
    Path           string // Database directory (required)
    BufferPoolSize int    // Number of pages to cache (default: 256)
    WALSegmentSize int64  // WAL segment size in bytes (default: 64MB)
}
```

#### DefaultOptions

```go
func DefaultOptions(path string) *Options
```

Creates default options with recommended settings.

**Example:**

```go
opts := bundoc.DefaultOptions("./data")
opts.BufferPoolSize = 512 // Customize if needed
```

---

### Open

```go
func Open(opts *Options) (*Database, error)
```

Opens or creates a database at the specified path.

**Parameters:**

- `opts`: Configuration options

**Returns:**

- `*Database`: Database instance
- `error`: Error if opening fails

**Example:**

```go
db, err := bundoc.Open(bundoc.DefaultOptions("./mydb"))
if err != nil {
    log.Fatal(err)
}
defer db.Close()
```

**Errors:**

- `ErrInvalidOptions`: Invalid configuration
- `ErrDatabaseCorrupted`: Database files corrupted
- OS errors (permission denied, etc.)

---

### Close

```go
func (db *Database) Close() error
```

Closes the database and releases resources.

**Important:**

- All active transactions should be committed or rolled back before closing
- Any operations after Close() will return errors

**Example:**

```go
err := db.Close()
if err != nil {
    log.Printf("Error closing database: %v", err)
}
```

---

###Collections

#### CreateCollection

```go
func (db *Database) CreateCollection(name string) (*Collection, error)
```

Creates a new collection.

**Parameters:**

- `name`: Collection name (must be unique)

**Returns:**

- `*Collection`: Collection instance
- `error`: Error if creation fails

**Example:**

```go
users, err := db.CreateCollection("users")
if err != nil {
    log.Fatal(err)
}
```

**Errors:**

- `ErrCollectionExists`: Collection already exists
- `ErrInvalidName`: Invalid collection name

---

#### GetCollection

```go
func (db *Database) GetCollection(name string) (*Collection, error)
```

Retrieves an existing collection.

**Example:**

```go
users, err := db.GetCollection("users")
if err != nil {
    log.Fatal(err)
}
```

**Errors:**

- `ErrCollectionNotFound`: Collection doesn't exist

---

#### DropCollection

```go
func (db *Database) DropCollection(name string) error
```

Deletes a collection and all its documents.

**Warning:** This operation is irreversible!

**Example:**

```go
err := db.DropCollection("temp_data")
if err != nil {
    log.Fatal(err)
}
```

---

#### ListCollections

```go
func (db *Database) ListCollections() []string
```

Returns names of all collections.

**Example:**

```go
collections := db.ListCollections()
for _, name := range collections {
    fmt.Println(name)
}
```

---

## Transactions

### Isolation Levels

```go
const (
    ReadUncommitted  // Dirty reads allowed
    ReadCommitted    // Read only committed data (recommended)
    RepeatableRead   // Consistent snapshot
    Serializable     // Full serializability
)
```

**Recommendations:**

- **Read Committed**: Good default for most workloads
- **Repeatable Read**: When you need consistent reads across queries
- **Serializable**: Maximum isolation (same as RepeatableRead currently)

---

### BeginTransaction

```go
func (db *Database) BeginTransaction(level IsolationLevel) (*Transaction, error)
```

Starts a new transaction.

**Example:**

```go
txn, err := db.BeginTransaction(mvcc.ReadCommitted)
if err != nil {
    log.Fatal(err)
}
```

---

### Commit

```go
func (db *Database) txnMgr.Commit(txn *Transaction) error
```

Commits a transaction, making all changes durable.

**Example:**

```go
err := db.txnMgr.Commit(txn)
if err != nil {
    log.Fatal(err)
}
```

**What happens:**

1. Write all changes to WAL
2. Wait for `fsync()` (via group commit)
3. Update MVCC version chains
4. Release locks

---

### Rollback

```go
func (db *Database) txnMgr.Rollback(txn *Transaction) error
```

Rolls back a transaction, discarding all changes.

**Example:**

```go
err := db.txnMgr.Rollback(txn)
if err != nil {
    log.Fatal(err)
}
```

---

## Collections

### Name

```go
func (c *Collection) Name() string
```

Returns the collection name.

---

### Insert

```go
func (c *Collection) Insert(txn *Transaction, doc Document) error
```

Inserts a document.

**Parameters:**

- `txn`: Active transaction
- `doc`: Document to insert

**Auto-ID Generation:**
If `_id` is not provided or empty, a unique ID is generated automatically.

**Example:**

```go
doc := storage.Document{
    "_id":  "user-42",  // Optional
    "name": "Bob",
    "age":  25,
}
err := users.Insert(txn, doc)
```

**Errors:**

- `ErrDuplicateKey`: Document with same `_id` already exists

---

### FindByID

```go
func (c *Collection) FindByID(txn *Transaction, id string) (Document, error)
```

Finds a document by ID.

**Example:**

```go
doc, err := users.FindByID(txn, "user-42")
if err != nil {
    log.Printf("Not found: %v", err)
    return
}
fmt.Printf("Name: %s\n", doc["name"])
```

**Errors:**

- `ErrDocumentNotFound`: Document doesn't exist

**MVCC Behavior:**
Returns the version visible to the transaction's snapshot timestamp.

---

### Update

```go
func (c *Collection) Update(txn *Transaction, id string, doc Document) error
```

Updates a document (replaces it entirely).

**Important:** This is a full replacement, not a partial update.

**Example:**

```go
updated := storage.Document{
    "_id":  "user-42",
    "name": "Bob Smith",  // Updated
    "age":  26,           // Updated
}
err := users.Update(txn, "user-42", updated)
```

**Errors:**

- `ErrDocumentNotFound`: Document doesn't exist

**MVCC Behavior:**
Creates a new version while keeping old versions for concurrent readers.

---

### Delete

```go
func (c *Collection) Delete(txn *Transaction, id string) error
```

Deletes a document (tombstone-based).

**Example:**

```go
err := users.Delete(txn, "user-42")
if err != nil {
    log.Printf("Failed to delete: %v", err)
}
```

**MVCC Behavior:**
Marks the document as deleted with a tombstone. Actual bytes are removed during garbage collection.

---

### Cross-Collection References

Collections can declare **reference fields** that point to documents in another collection. References are enforced at write time (strict FK semantics) and support configurable delete behavior.

**Schema extension:** In the collection schema (set via `SetSchema`), add `x-bundoc-ref` to a property:

```json
{
  "type": "object",
  "properties": {
    "author_id": {
      "type": "string",
      "x-bundoc-ref": {
        "collection": "users",
        "field": "_id",
        "on_delete": "set_null"
      }
    }
  }
}
```

- **`collection`** (required): Target collection name.
- **`field`** (optional): Target field; v1 only supports `_id`.
- **`on_delete`** (optional): Action when the referenced document is deleted. Defaults to `set_null`.
  - **`restrict`**: Delete of the target document fails if any dependent document exists (returns conflict).
  - **`set_null`**: Dependent document’s reference field is set to `null` (schema must allow null for that field).
  - **`cascade`**: Dependent documents are deleted recursively (cycles are guarded).

**Write behavior:** On Insert, Update, and Patch, every reference field is validated: the referenced document must exist in the target collection. If it does not, the operation fails with `ErrReferenceTargetNotFound` (HTTP 409 when used via bundoc-server).

**Read behavior:** In v1 there is no automatic expansion or embedding of referenced documents; reference fields store only the target ID.

**Scope:** References are within the same database (project) only.

---

## Documents

### Document Type

```go
type Document map[string]interface{}
```

JSON-compatible key-value map.

**Supported Value Types:**

- `string`
- `int`, `int64`, `float64`
- `bool`
- `[]interface{}` (arrays)
- `map[string]interface{}` (nested objects)
- `nil`

**Reserved Keys:**

- `_id`: Document identifier (string)

---

### GetID / SetID

```go
func (d Document) GetID() (DocumentID, bool)
func (d Document) SetID(id DocumentID)
```

**Example:**

```go
doc := storage.Document{"name": "Alice"}

// Set ID
doc.SetID("user-1")

// Get ID
id, hasID := doc.GetID()
if hasID {
    fmt.Println("ID:", id)
}
```

---

### Serialize / Deserialize

```go
func (d Document) Serialize() ([]byte, error)
func DeserializeDocument(data []byte) (Document, error)
```

Converts documents to/from JSON bytes.

**Example:**

```go
// Serialize
bytes, err := doc.Serialize()

// Deserialize
doc, err := storage.DeserializeDocument(bytes)
```

---

## Error Handling

### Common Errors

```go
var (
    ErrDatabaseClosed     = errors.New("database is closed")
    ErrCollectionExists   = errors.New("collection already exists")
    ErrCollectionNotFound = errors.New("collection not found")
    ErrDocumentNotFound   = errors.New("document not found")
    ErrDuplicateKey       = errors.New("duplicate key")
    ErrInvalidDocument    = errors.New("invalid document")
    ErrTransactionAborted = errors.New("transaction aborted")

    // Reference (FK) errors — use errors.Is(err, bundoc.Err...) for checks
    ErrInvalidReferenceSchema   = errors.New("invalid reference schema")   // 400
    ErrReferenceTargetNotFound  = errors.New("reference target not found")  // 409
    ErrReferenceRestrictViolation = errors.New("reference restrict violation") // 409
)
```

**Reference errors (HTTP mapping when using bundoc-server):** Schema/validation and invalid reference schema return 400. Missing reference target on write and restrict violation on delete return 409.

### Error Handling Pattern

```go
doc, err := users.FindByID(txn, "user-42")
if err != nil {
    if errors.Is(err, storage.ErrDocumentNotFound) {
        // Handle not found
        fmt.Println("User doesn't exist")
    } else {
        // Other errors
        log.Fatal(err)
    }
}
```

---

## Examples

### Example 1: Basic CRUD

```go
func basicCRUD() {
    db, _ := bundoc.Open(bundoc.DefaultOptions("./data"))
    defer db.Close()

    users, _ := db.CreateCollection("users")

    // INSERT
    txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
    doc := storage.Document{
        "_id":  "alice",
        "name": "Alice",
        "age":  30,
    }
    users.Insert(txn, doc)
    db.txnMgr.Commit(txn)

    // FIND
    txn, _ = db.BeginTransaction(mvcc.ReadCommitted)
    found, _ := users.FindByID(txn, "alice")
    fmt.Println(found)
    db.txnMgr.Commit(txn)

    // UPDATE
    txn, _ = db.BeginTransaction(mvcc.ReadCommitted)
    updated := storage.Document{
        "_id":  "alice",
        "name": "Alice Smith",
        "age":  31,
    }
    users.Update(txn, "alice", updated)
    db.txnMgr.Commit(txn)

    // DELETE
    txn, _ = db.BeginTransaction(mvcc.ReadCommitted)
    users.Delete(txn, "alice")
    db.txnMgr.Commit(txn)
}
```

---

### Example 2: Concurrent Writes

```go
func concurrentWrites() {
    db, _ := bundoc.Open(bundoc.DefaultOptions("./data"))
    defer db.Close()

    users, _ := db.CreateCollection("users")

    var wg sync.WaitGroup

    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()

            txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
            doc := storage.Document{
                "_id":    fmt.Sprintf("user-%d", id),
                "worker": id,
            }
            users.Insert(txn, doc)
            db.txnMgr.Commit(txn)
        }(i)
    }

    wg.Wait()
    fmt.Println("All inserts complete!")
}
```

---

### Example 3: Transactional Integrity

```go
func transferMoney(db *bundoc.Database, from, to string, amount float64) error {
    accounts, _ := db.GetCollection("accounts")

    txn, err := db.BeginTransaction(mvcc.RepeatableRead)
    if err != nil {
        return err
    }

    // Read sender
    sender, err := accounts.FindByID(txn, from)
    if err != nil {
        db.txnMgr.Rollback(txn)
        return err
    }

    // Check balance
    balance := sender["balance"].(float64)
    if balance < amount {
        db.txnMgr.Rollback(txn)
        return errors.New("insufficient funds")
    }

    // Read receiver
    receiver, err := accounts.FindByID(txn, to)
    if err != nil {
        db.txnMgr.Rollback(txn)
        return err
    }

    // Update balances
    sender["balance"] = balance - amount
    receiver["balance"] = receiver["balance"].(float64) + amount

    accounts.Update(txn, from, sender)
    accounts.Update(txn, to, receiver)

    // Commit atomically
    return db.txnMgr.Commit(txn)
}
```

---

### Example 4: Read-Your-Own-Writes

```go
func readYourOwnWrites() {
    db, _ := bundoc.Open(bundoc.DefaultOptions("./data"))
    defer db.Close()

    users, _ := db.CreateCollection("users")

    txn, _ := db.BeginTransaction(mvcc.ReadCommitted)

    // Insert
    doc := storage.Document{"_id": "test", "value": 1}
    users.Insert(txn, doc)

    // Read immediately (before commit)
    found, _ := users.FindByID(txn, "test")
    fmt.Println(found) // {"_id": "test", "value": 1}

    // Update
    found["value"] = 2
    users.Update(txn, "test", found)

    // Read updated value
    updated, _ := users.FindByID(txn, "test")
    fmt.Println(updated) // {"_id": "test", "value": 2}

    db.txnMgr.Commit(txn)
}
```

---

## Best Practices

### 1. Always Close Database

```go
db, err := bundoc.Open(opts)
if err != nil {
    return err
}
defer db.Close() // ✅ Always defer Close()
```

---

### 2. Commit or Rollback Transactions

```go
txn, _ := db.BeginTransaction(mvcc.ReadCommitted)

// Do work...

if err != nil {
    db.txnMgr.Rollback(txn) // ✅ Rollback on error
    return err
}

db.txnMgr.Commit(txn) // ✅ Commit on success
```

---

### 3. Use Read Committed by Default

```go
// ✅ Good default
txn, _ := db.BeginTransaction(mvcc.ReadCommitted)

// Only use RepeatableRead when you need consistent snapshots
txn, _ := db.BeginTransaction(mvcc.RepeatableRead)
```

---

### 4. Handle Errors

```go
// ❌ Bad
users.Insert(txn, doc)

// ✅ Good
if err := users.Insert(txn, doc); err != nil {
    log.Printf("Insert failed: %v", err)
    db.txnMgr.Rollback(txn)
    return err
}
```

---

### 5. Don't Reuse Transactions

```go
// ❌ Bad
txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
users.Insert(txn, doc1)
db.txnMgr.Commit(txn)
users.Insert(txn, doc2) // ❌ Transaction already committed!

// ✅ Good
txn1, _ := db.BeginTransaction(mvcc.ReadCommitted)
users.Insert(txn1, doc1)
db.txnMgr.Commit(txn1)

txn2, _ := db.BeginTransaction(mvcc.ReadCommitted)
users.Insert(txn2, doc2)
db.txnMgr.Commit(txn2)
```

---

## Performance Tips

1. **Batch writes** in single transaction when possible
2. **Use larger buffer pool** for read-heavy workloads
3. **Concurrent writes** benefit from group commits automatically
4. **Read Committed** is faster than Repeatable Read

---

**For architecture details**: See [ARCHITECTURE.md](./ARCHITECTURE.md)  
**For performance tuning**: See [PERFORMANCE.md](./PERFORMANCE.md)  
** For configuration**: See [CONFIGURATION.md](./CONFIGURATION.md)

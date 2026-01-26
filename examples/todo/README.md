# BunStore Todo Test Suite

A comprehensive interactive testing ground for BunStore APIs, JS SDK, and real-time features.

## Overview

This example serves as a testing suite for BunStore functionality, including:
- CRUD operations (Create, Read, Update, Delete)
- Real-time subscriptions
- Batch operations
- Transactions
- Query filtering and sorting
- Performance testing

## Features

### 1. Create Todo
Create new todo items with title and completion status.

### 2. List Todos
Query todos with various filters:
- Filter by completion status
- Sort by createdAt or title
- Limit results
- Pagination support

### 3. Update Todo
Update existing todo items by ID.

### 4. Delete Todo
Delete todo items by ID.

### 5. Real-time Subscriptions
Subscribe to collection changes and receive real-time updates when documents are created, updated, or deleted.

### 6. Batch Operations
Perform multiple operations atomically using write batches.

### 7. Transactions
Execute multiple operations in a transaction with rollback support.

### 8. Query Examples
Demonstrate various query patterns:
- Filtering by field values
- Sorting and ordering
- Pagination with offset/limit

### 9. Performance Testing
Test API performance with concurrent operations and measure throughput.

## Setup

1. Install dependencies:
```bash
bun install
```

2. Set environment variables (optional):
```bash
export BUNBASE_API_KEY="your-api-key"
export BUNBASE_URL="http://localhost:3000/api"
```

3. Run the test suite:
```bash
bun run index.ts
```

## Usage

The interactive CLI will present a menu with numbered options. Select an option to test different features:

```
=== BunStore Todo Test Suite ===
1. Create todo
2. List todos (with filters)
3. Update todo
4. Delete todo
5. Subscribe to real-time updates
6. Batch operations
7. Transaction example
8. Query examples
9. Performance test
0. Exit
================================
```

## Testing Scenarios

### Basic CRUD
1. Create a few todos
2. List all todos
3. Update a todo's completion status
4. Delete a todo

### Real-time Testing
1. Start a real-time subscription (option 5)
2. In another terminal or session, create/update/delete todos
3. Observe real-time updates in the subscription

### Batch Operations
1. Use option 6 to create multiple todos in a batch
2. Verify all todos are created atomically

### Performance Testing
1. Use option 9 to run performance tests
2. Adjust the number of operations to test different loads
3. Observe throughput and latency metrics

## API Reference

This example uses the BunStore API:

- `bunstore.collection(path)` - Get a collection reference
- `collection.doc(id?)` - Get a document reference
- `docRef.set(data)` - Create or update a document
- `docRef.get()` - Get a document
- `docRef.update(data)` - Update a document
- `docRef.delete()` - Delete a document
- `collection.where(field, op, value)` - Filter documents
- `collection.orderBy(field, direction)` - Sort documents
- `collection.limit(n)` - Limit results
- `collection.offset(n)` - Pagination
- `collection.onSnapshot(callback)` - Real-time subscription
- `bunstore.batch()` - Create a write batch
- `bunstore.runTransaction(callback)` - Run a transaction

## Event System

BunStore uses an Event Emitter system to send real-time events when documents are:
- Created (`document:created`)
- Updated (`document:updated`)
- Deleted (`document:deleted`)

These events are broadcast via WebSocket to subscribed clients.

## Load Testing

For comprehensive load testing, see the `load-test.ts` file and related load testing infrastructure.

## Troubleshooting

### Connection Issues
- Ensure the BunBase server is running
- Check that the API key is valid
- Verify the baseURL is correct

### Real-time Not Working
- Check WebSocket connection status
- Ensure API key authentication is working
- Verify server-side event emitter is running

### Performance Issues
- Check server logs for errors
- Monitor database performance
- Adjust batch sizes if needed

## Contributing

This test suite is designed to be extended with additional test scenarios. Add new test cases to demonstrate BunStore features and edge cases.

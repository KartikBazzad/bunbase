# BunBase JavaScript/TypeScript SDK

Official JavaScript/TypeScript SDK for BunBase.

## Installation

```bash
npm install @bunbase/js-sdk
# or
bun add @bunbase/js-sdk
```

## Usage

```typescript
import { createClient } from "@bunbase/js-sdk";

const client = createClient({
  apiKey: "your-api-key",
  baseURL: "https://api.bunbase.com",
  projectId: "your-project-id",
});

// Authentication
const { user, session } = await client.auth.signUp(
  "user@example.com",
  "password123",
  "John Doe"
);

// Database operations
const document = await client.database.create(
  "db-id",
  "collection-id",
  { name: "John", age: 30 }
);

// Storage operations
const file = await client.storage.upload(
  "bucket-id",
  fileBlob,
  { path: "uploads/file.jpg" }
);

// Realtime
client.realtime.connect({
  userId: user.id,
  onMessage: (message) => {
    console.log("Received:", message);
  },
});
```

## Modules

- **Auth**: User authentication and session management
- **Database**: Document CRUD operations, queries, batch operations
- **Storage**: File upload/download, bucket management
- **Realtime**: WebSocket connections, channels, pub/sub

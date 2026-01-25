# JavaScript/TypeScript SDK Requirements

## Overview

The JavaScript/TypeScript SDK provides a type-safe, developer-friendly interface for integrating BunBase services into web, mobile, and Node.js applications.

## Supported Platforms

- **Browser** (ES6+, Chrome, Firefox, Safari, Edge)
- **Node.js** (v18+)
- **Bun** (latest)
- **Deno** (latest)
- **React Native** (iOS/Android)
- **React** (with React hooks)
- **Vue** (with composables)
- **Svelte** (with stores)
- **Angular** (with services)

## Installation

```bash
# npm
npm install @bunbase/sdk

# yarn
yarn add @bunbase/sdk

# pnpm
pnpm add @bunbase/sdk

# bun
bun add @bunbase/sdk
```

## Core Features

### 1. Initialization & Configuration

```typescript
import { createClient } from "@bunbase/sdk";

const bunbase = createClient({
  apiKey: "bunbase_pk_live_xxx",
  project: "my-project",
  region: "us-east-1", // optional
  options: {
    auth: {
      persistSession: true,
      autoRefresh: true,
      storage: localStorage, // or custom storage
    },
    realtime: {
      autoConnect: true,
      reconnect: true,
      heartbeat: 30000, // ms
    },
    fetch: {
      timeout: 30000, // ms
      retries: 3,
      retryDelay: 1000,
    },
  },
});
```

### 2. Authentication Module

```typescript
// Register
const { user, session, error } = await bunbase.auth.signUp({
  email: "user@example.com",
  password: "securepassword",
  metadata: {
    name: "John Doe",
  },
});

// Login
const { user, session, error } = await bunbase.auth.signIn({
  email: "user@example.com",
  password: "securepassword",
});

// OAuth
const { url, error } = await bunbase.auth.signInWithOAuth({
  provider: "google",
  redirectTo: "https://myapp.com/callback",
});

// Magic Link
const { error } = await bunbase.auth.signInWithMagicLink({
  email: "user@example.com",
  redirectTo: "https://myapp.com/verify",
});

// Get current user
const { user, error } = await bunbase.auth.getUser();

// Update user
const { user, error } = await bunbase.auth.updateUser({
  data: {
    name: "Jane Doe",
    avatar: "https://...",
  },
});

// Sign out
const { error } = await bunbase.auth.signOut();

// Listen to auth state changes
bunbase.auth.onAuthStateChange((event, session) => {
  console.log("Auth state:", event, session);
});
```

### 3. Database Module

```typescript
// Get documents
const { data, error } = await bunbase
  .from("users")
  .select("*")
  .eq("status", "active")
  .order("createdAt", { ascending: false })
  .limit(10);

// Get single document
const { data, error } = await bunbase
  .from("users")
  .select("*")
  .eq("id", "user-123")
  .single();

// Insert document
const { data, error } = await bunbase.from("users").insert({
  name: "John Doe",
  email: "john@example.com",
  age: 25,
});

// Update document
const { data, error } = await bunbase
  .from("users")
  .update({ age: 26 })
  .eq("id", "user-123");

// Delete document
const { data, error } = await bunbase
  .from("users")
  .delete()
  .eq("id", "user-123");

// Advanced queries
const { data, error } = await bunbase
  .from("orders")
  .select("*, customer:customers(*), items:order_items(*)")
  .gte("total", 100)
  .lte("total", 1000)
  .in("status", ["pending", "processing"])
  .or("priority.eq.high,amount.gte.500");

// Full-text search
const { data, error } = await bunbase
  .from("articles")
  .select("*")
  .textSearch("title", "machine learning");

// Real-time subscriptions
const subscription = bunbase
  .from("messages")
  .on("INSERT", (payload) => {
    console.log("New message:", payload.new);
  })
  .on("UPDATE", (payload) => {
    console.log("Updated message:", payload.new);
  })
  .on("DELETE", (payload) => {
    console.log("Deleted message:", payload.old);
  })
  .subscribe();

// Unsubscribe
subscription.unsubscribe();
```

### 4. Storage Module

```typescript
// Upload file
const { data, error } = await bunbase.storage
  .from("avatars")
  .upload("public/user-123.jpg", file, {
    cacheControl: "3600",
    upsert: true,
    metadata: {
      userId: "user-123",
    },
  });

// Download file
const { data, error } = await bunbase.storage
  .from("avatars")
  .download("public/user-123.jpg");

// List files
const { data, error } = await bunbase.storage.from("avatars").list("public/", {
  limit: 100,
  offset: 0,
  sortBy: { column: "created_at", order: "desc" },
});

// Delete file
const { data, error } = await bunbase.storage
  .from("avatars")
  .remove(["public/user-123.jpg"]);

// Get public URL
const { data } = bunbase.storage
  .from("avatars")
  .getPublicUrl("public/user-123.jpg", {
    transform: {
      width: 200,
      height: 200,
      format: "webp",
    },
  });

// Create signed URL
const { data, error } = await bunbase.storage
  .from("private-files")
  .createSignedUrl("documents/contract.pdf", 3600); // 1 hour

// Upload with progress
const { data, error } = await bunbase.storage
  .from("videos")
  .upload("large-video.mp4", file, {
    onProgress: (progress) => {
      console.log(`Upload progress: ${progress}%`);
    },
  });
```

### 5. Functions Module

```typescript
// Invoke function
const { data, error } = await bunbase.functions.invoke("send-email", {
  body: {
    to: "user@example.com",
    subject: "Welcome!",
    template: "welcome",
  },
});

// Invoke with headers
const { data, error } = await bunbase.functions.invoke("protected-function", {
  body: { action: "delete" },
  headers: {
    "X-Custom-Header": "value",
  },
});

// Streaming response
const stream = await bunbase.functions.stream("generate-report", {
  body: { reportType: "monthly" },
});

for await (const chunk of stream) {
  console.log("Received chunk:", chunk);
}
```

### 6. Real-time Module

```typescript
// Connect to real-time
const realtime = bunbase.realtime;

// Subscribe to channel
const channel = realtime.channel("chat:room-123");

await channel.subscribe((status) => {
  if (status === "SUBSCRIBED") {
    console.log("Connected to channel");
  }
});

// Send messages
await channel.send({
  type: "broadcast",
  event: "new-message",
  payload: {
    text: "Hello!",
    userId: "user-123",
  },
});

// Listen to messages
channel.on("new-message", (payload) => {
  console.log("New message:", payload);
});

// Presence
const presenceChannel = realtime.channel("presence:room-123", {
  presence: true,
});

await presenceChannel.subscribe();

// Track presence
await presenceChannel.track({
  user: "John",
  status: "online",
});

// Listen to presence
presenceChannel.on("presence", { event: "join" }, (payload) => {
  console.log("User joined:", payload);
});

presenceChannel.on("presence", { event: "leave" }, (payload) => {
  console.log("User left:", payload);
});

// Get presence state
const presenceState = await presenceChannel.presenceState();
console.log("Online users:", Object.keys(presenceState));
```

## TypeScript Support

### Auto-generated Types

```typescript
// Define your database schema
interface Database {
  users: {
    id: string;
    email: string;
    name: string;
    age: number;
    createdAt: Date;
  };
  posts: {
    id: string;
    title: string;
    content: string;
    authorId: string;
    published: boolean;
  };
}

// Type-safe client
const bunbase = createClient<Database>({
  apiKey: "xxx",
});

// Fully typed queries
const { data, error } = await bunbase
  .from("users") // auto-complete available
  .select("id, name, email") // auto-complete for columns
  .eq("age", 25); // type-checked

// data is typed as:
// { id: string; name: string; email: string; }[] | null
```

### Type Generation CLI

```bash
# Generate types from database schema
bunbase types generate --project my-project --output ./types/database.ts

# Watch mode
bunbase types generate --watch
```

## Framework Integrations

### React Hooks

```typescript
import { useAuth, useQuery, useMutation, useSubscription } from '@bunbase/react';

function MyComponent() {
  // Auth hook
  const { user, loading, signIn, signOut } = useAuth();

  // Query hook
  const { data, loading, error, refetch } = useQuery(
    ['users'],
    () => bunbase.from('users').select('*')
  );

  // Mutation hook
  const { mutate, loading: saving } = useMutation(
    (userData) => bunbase.from('users').insert(userData),
    {
      onSuccess: () => refetch()
    }
  );

  // Subscription hook
  useSubscription(
    bunbase.from('messages').on('INSERT'),
    (payload) => {
      console.log('New message:', payload.new);
    }
  );

  return <div>{/* ... */}</div>;
}
```

### Vue Composables

```typescript
import { useAuth, useQuery, useMutation } from "@bunbase/vue";

export default {
  setup() {
    const { user, signIn, signOut } = useAuth();
    const { data: users, loading } = useQuery(["users"], () =>
      bunbase.from("users").select("*"),
    );

    return { user, users, loading };
  },
};
```

### Svelte Stores

```typescript
import { auth, query } from "@bunbase/svelte";

const user = auth();
const users = query(["users"], () => bunbase.from("users").select("*"));

// In component
$: console.log($user);
$: console.log($users);
```

## Error Handling

```typescript
try {
  const { data, error } = await bunbase.from("users").select("*");

  if (error) {
    // Handle BunBase error
    console.error("Error code:", error.code);
    console.error("Error message:", error.message);
    console.error("Error details:", error.details);
  }
} catch (err) {
  // Handle network or unexpected errors
  console.error("Unexpected error:", err);
}
```

## Offline Support

```typescript
const bunbase = createClient({
  apiKey: "xxx",
  options: {
    offline: {
      enabled: true,
      storage: "indexeddb",
      syncInterval: 30000, // ms
      conflictResolution: "server-wins", // or 'client-wins', 'manual'
    },
  },
});

// Queue mutations while offline
const { data, error } = await bunbase
  .from("notes")
  .insert({ title: "Offline note" });
// Automatically synced when back online
```

## Performance Features

### Request Batching

```typescript
// Enable automatic batching
const bunbase = createClient({
  apiKey: "xxx",
  options: {
    batching: {
      enabled: true,
      maxBatchSize: 10,
      batchWindow: 10, // ms
    },
  },
});
```

### Caching

```typescript
import { createClient, CachePolicy } from "@bunbase/sdk";

const bunbase = createClient({
  apiKey: "xxx",
  options: {
    cache: {
      enabled: true,
      policy: CachePolicy.CacheFirst, // or NetworkFirst, CacheOnly, NetworkOnly
      ttl: 300, // seconds
    },
  },
});

// Per-query cache control
const { data } = await bunbase
  .from("users")
  .select("*")
  .cache({ ttl: 600, policy: CachePolicy.CacheFirst });
```

## Testing Utilities

```typescript
import { createMockClient } from "@bunbase/sdk/testing";

const mockBunbase = createMockClient();

mockBunbase.from("users").select.mockResolvedValue({
  data: [{ id: "1", name: "Test User" }],
  error: null,
});

// Use in tests
const { data } = await mockBunbase.from("users").select("*");
expect(data).toHaveLength(1);
```

## Bundle Size

- Core SDK: ~15KB (gzipped)
- With Auth: ~25KB (gzipped)
- With Realtime: ~35KB (gzipped)
- Full SDK: ~45KB (gzipped)
- Tree-shakeable modules

## Browser Support

- Chrome/Edge: Last 2 versions
- Firefox: Last 2 versions
- Safari: Last 2 versions
- iOS Safari: 12+
- Android Chrome: Last 2 versions

## Documentation Requirements

- Getting started guide
- API reference (auto-generated)
- Migration guides
- Framework integration guides
- Best practices
- TypeScript guide
- Examples repository
- Video tutorials

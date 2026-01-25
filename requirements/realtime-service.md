# Real-time Service Requirements

## Overview

The Real-time Service enables live, bidirectional communication between clients and servers for building collaborative applications, live updates, chat systems, and real-time dashboards.

## Core Features

### 1. Real-time Communication Protocols

- **WebSocket Support**
  - Full-duplex communication
  - Binary and text messages
  - Connection lifecycle management
  - Automatic reconnection
  - Heartbeat/ping-pong

- **Server-Sent Events (SSE)**
  - Unidirectional server-to-client
  - HTTP-based streaming
  - Automatic reconnection
  - Event types and IDs

- **Long Polling**
  - Fallback for restrictive networks
  - Automatic upgrade to WebSocket
  - Timeout handling

### 2. Channels & Rooms

- **Public Channels**
  - Anyone can subscribe
  - Broadcast messages
  - No authentication required

- **Private Channels**
  - Authentication required
  - Authorization checks
  - Encrypted messages

- **Presence Channels**
  - Track online users
  - Join/leave events
  - User metadata
  - Typing indicators
  - User count

- **Room Management**
  - Dynamic room creation
  - Room metadata
  - Room permissions
  - Room capacity limits
  - Room lifecycle events

### 3. Pub/Sub Messaging

- Publish messages to channels
- Subscribe to channels
- Unsubscribe from channels
- Message filtering
- Message acknowledgment
- Message ordering guarantees
- Delivery guarantees (at-least-once, exactly-once)

### 4. Presence Features

- Online/offline status
- User metadata (name, avatar, etc.)
- Last seen timestamp
- Typing indicators
- Custom presence states
- Presence timeout configuration

### 5. Database Live Queries

- Subscribe to database queries
- Real-time query results
- Automatic updates on data changes
- Filtering and sorting
- Pagination support
- Query result caching

### 6. Broadcast Features

- Broadcast to all clients
- Broadcast to specific users
- Broadcast to rooms/channels
- Broadcast with filters
- Broadcast with exclusions
- Message prioritization

## Technical Requirements

### API Endpoints

```
# WebSocket Connection
WS     /realtime                     - WebSocket connection

# REST API for Server-side
POST   /realtime/channels/:id/broadcast - Broadcast message
GET    /realtime/channels/:id/presence  - Get presence info
POST   /realtime/channels/:id/kick      - Kick user from channel
GET    /realtime/connections            - List active connections
DELETE /realtime/connections/:id        - Disconnect client
```

### WebSocket Message Protocol

```typescript
// Client -> Server Messages
{
  "type": "subscribe",
  "channel": "chat:room-123",
  "auth": "token..."
}

{
  "type": "unsubscribe",
  "channel": "chat:room-123"
}

{
  "type": "message",
  "channel": "chat:room-123",
  "event": "new-message",
  "data": {
    "text": "Hello!",
    "userId": "user-456"
  }
}

{
  "type": "presence:update",
  "channel": "chat:room-123",
  "data": {
    "status": "typing"
  }
}

// Server -> Client Messages
{
  "type": "subscribed",
  "channel": "chat:room-123",
  "presence": {
    "users": [...],
    "count": 42
  }
}

{
  "type": "message",
  "channel": "chat:room-123",
  "event": "new-message",
  "data": {...},
  "timestamp": "2026-01-25T09:00:00Z",
  "sender": "user-456"
}

{
  "type": "presence:join",
  "channel": "chat:room-123",
  "user": {
    "id": "user-789",
    "name": "John",
    "metadata": {...}
  }
}

{
  "type": "error",
  "code": "RT_003",
  "message": "Permission denied"
}
```

### Database Live Queries

```typescript
// Subscribe to live query
const subscription = db
  .collection("messages")
  .where("roomId", "==", "room-123")
  .orderBy("createdAt", "desc")
  .limit(50)
  .onSnapshot((snapshot) => {
    snapshot.changes.forEach((change) => {
      if (change.type === "added") {
        console.log("New message:", change.doc.data());
      }
      if (change.type === "modified") {
        console.log("Modified message:", change.doc.data());
      }
      if (change.type === "removed") {
        console.log("Removed message:", change.doc.data());
      }
    });
  });

// Unsubscribe
subscription.unsubscribe();
```

### Channel Naming Convention

```
public:*                  - Public channels
private:user-*            - Private user channels
private:room-*            - Private room channels
presence:*                - Presence channels
db:collection:*           - Database live queries
```

### Performance Requirements

- Connection establishment: < 100ms
- Message latency: < 50ms (p95)
- Concurrent connections per server: 50,000+
- Messages per second per channel: 10,000+
- Channel subscriptions per connection: 100
- Horizontal scaling support
- Geographic distribution

### Scalability Features

- Multi-server deployment
- Sticky sessions
- Message routing between servers
- Distributed pub/sub
- Connection balancing
- Auto-scaling based on connections

## Security Features

### Authentication

- Token-based authentication
- API key authentication
- Custom authentication handlers
- Session validation
- Token refresh

### Authorization

- Channel-level permissions
- User-based access control
- IP whitelisting
- Rate limiting per connection
- Message size limits

### Channel Access Control

```typescript
{
  "channel": "private:room-123",
  "rules": {
    "subscribe": "auth.uid != null && data.members.includes(auth.uid)",
    "publish": "auth.uid != null && data.members.includes(auth.uid)",
    "presence": true
  }
}
```

### Message Security

- Message encryption in transit (TLS)
- Optional end-to-end encryption
- Message validation
- XSS prevention
- Content filtering
- Spam detection

## Client SDK Features

### Connection Management

```typescript
import { Realtime } from "@bunbase/sdk";

const realtime = new Realtime({
  apiKey: "your-api-key",
  autoConnect: true,
  reconnect: true,
  reconnectDelay: 1000, // ms
  maxReconnectAttempts: 10,
});

// Connection events
realtime.on("connected", () => console.log("Connected"));
realtime.on("disconnected", () => console.log("Disconnected"));
realtime.on("error", (error) => console.error(error));

// Manual connection control
realtime.connect();
realtime.disconnect();
```

### Channel Subscription

```typescript
const channel = realtime.channel("chat:room-123", {
  presence: true,
});

await channel.subscribe();

// Listen to messages
channel.on("new-message", (message) => {
  console.log("New message:", message);
});

// Send messages
await channel.send("new-message", {
  text: "Hello!",
  userId: currentUser.id,
});

// Unsubscribe
await channel.unsubscribe();
```

### Presence

```typescript
const presenceChannel = realtime.channel("presence:room-123");

await presenceChannel.subscribe();

// Update presence
await presenceChannel.updatePresence({
  status: "online",
  name: "John",
  avatar: "https://...",
});

// Listen to presence events
presenceChannel.on("presence:join", (user) => {
  console.log("User joined:", user);
});

presenceChannel.on("presence:leave", (user) => {
  console.log("User left:", user);
});

presenceChannel.on("presence:update", (user) => {
  console.log("User updated:", user);
});

// Get current presence
const presence = await presenceChannel.getPresence();
console.log("Online users:", presence.users);
console.log("User count:", presence.count);
```

### Database Live Queries

```typescript
const subscription = realtime
  .database()
  .collection("messages")
  .where("roomId", "==", "room-123")
  .orderBy("createdAt", "desc")
  .onSnapshot((snapshot) => {
    // Handle changes
  });
```

## Server-side Integration

### Broadcasting from Functions

```typescript
import { realtime } from "@bunbase/sdk";

export async function onNewOrder(event) {
  const order = event.data;

  // Broadcast to admin dashboard
  await realtime.channel("admin:orders").broadcast("new-order", {
    orderId: order.id,
    amount: order.amount,
    timestamp: new Date(),
  });

  // Notify specific user
  await realtime
    .channel(`private:user-${order.userId}`)
    .broadcast("order-update", {
      status: "confirmed",
      orderId: order.id,
    });
}
```

### Presence Management

```typescript
// Get channel presence
const presence = await realtime.channel("chat:room-123").getPresence();

// Kick user from channel
await realtime.channel("chat:room-123").kick("user-id");

// Get connection info
const connections = await realtime.getConnections({
  userId: "user-123",
});

// Disconnect user
await realtime.disconnectUser("user-123");
```

## Use Case Examples

### Live Chat

- Real-time messaging
- Typing indicators
- Online presence
- Read receipts
- Message history
- File sharing

### Collaborative Editing

- Real-time document updates
- Cursor positions
- Selection tracking
- Conflict resolution
- Version history

### Live Dashboard

- Real-time metrics
- Live charts
- Alert notifications
- System status
- Activity feeds

### Multiplayer Games

- Player positions
- Game state sync
- Matchmaking
- Leaderboards
- Chat

### Live Notifications

- Push notifications
- Activity feeds
- Real-time alerts
- System announcements

## Monitoring & Observability

### Metrics

- Active connections (total, per channel)
- Messages per second
- Connection duration
- Message latency (p50, p95, p99)
- Reconnection rate
- Error rate
- Bandwidth usage
- Channel subscription count

### Logging

- Connection lifecycle events
- Message delivery
- Authentication failures
- Authorization failures
- Channel events
- Presence updates

### Alerting

- High connection count (>80% capacity)
- High error rate (>5%)
- High message latency (>1s)
- Server health issues
- Abnormal disconnect rate

## Rate Limiting

### Connection Limits

- Max connections per user: 10
- Max connections per IP: 100
- Connection rate: 100/minute per IP

### Message Limits

- Messages per second per connection: 100
- Message size: 128KB
- Presence updates per second: 10
- Channel subscriptions per connection: 100

## Error Handling

### Error Codes

- `RT_001`: Connection failed
- `RT_002`: Authentication failed
- `RT_003`: Permission denied
- `RT_004`: Channel not found
- `RT_005`: Message too large
- `RT_006`: Rate limit exceeded
- `RT_007`: Invalid message format
- `RT_008`: Channel capacity exceeded
- `RT_009`: Presence update failed

### Retry Strategies

- Exponential backoff
- Jitter for reconnection
- Max retry attempts
- Fallback to long polling

## Advanced Features

### Message Persistence

- Store messages in database
- Message history retrieval
- Replay messages on reconnection
- Message expiry

### Message Filtering

- Server-side filtering
- Client-side filtering
- Content-based routing
- User preferences

### Analytics

- User engagement metrics
- Channel activity
- Popular channels
- Peak usage times
- Geographic distribution

## Documentation Requirements

- Getting started guide
- Channel concepts
- Presence guide
- Live queries guide
- Security best practices
- Scaling guide
- SDK reference
- Example applications

# Go SDK Requirements

## Overview

The Go SDK provides a performant, idiomatic Go interface for integrating BunBase services into Go applications, microservices, and cloud-native systems.

## Supported Go Versions

- Go 1.20+
- Go 1.21+
- Go 1.22+

## Installation

```bash
go get github.com/bunbase/bunbase-go
```

## Core Features

### 1. Initialization

```go
package main

import (
    "context"
    "github.com/bunbase/bunbase-go"
)

func main() {
    client, err := bunbase.NewClient(bunbase.Config{
        APIKey:  "bunbase_pk_live_xxx",
        Project: "my-project",
        Region:  "us-east-1",
        Options: bunbase.Options{
            Timeout:    30 * time.Second,
            Retries:    3,
            MaxRetries: 5,
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
}
```

### 2. Authentication Module

```go
// Register user
user, session, err := client.Auth.SignUp(ctx, bunbase.SignUpRequest{
    Email:    "user@example.com",
    Password: "securepassword",
    Metadata: map[string]interface{}{
        "name": "John Doe",
    },
})

// Login
user, session, err := client.Auth.SignIn(ctx, bunbase.SignInRequest{
    Email:    "user@example.com",
    Password: "securepassword",
})

// OAuth
authURL, err := client.Auth.SignInWithOAuth(ctx, bunbase.OAuthRequest{
    Provider:   "google",
    RedirectTo: "https://myapp.com/callback",
})

// Get current user
user, err := client.Auth.GetUser(ctx)

// Update user
user, err := client.Auth.UpdateUser(ctx, bunbase.UpdateUserRequest{
    Data: map[string]interface{}{
        "name": "Jane Doe",
    },
})

// Sign out
err := client.Auth.SignOut(ctx)

// Auth state change listener
client.Auth.OnAuthStateChange(func(event string, session *bunbase.Session) {
    log.Printf("Auth event: %s", event)
})
```

### 3. Database Module

```go
// Define types
type User struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    Age       int       `json:"age"`
    CreatedAt time.Time `json:"created_at"`
}

// Query documents
var users []User
err := client.From("users").
    Select("*").
    Eq("status", "active").
    Order("created_at", bunbase.OrderDesc).
    Limit(10).
    Execute(ctx, &users)

// Get single document
var user User
err := client.From("users").
    Select("*").
    Eq("id", "user-123").
    Single().
    Execute(ctx, &user)

// Insert document
var newUser User
err := client.From("users").
    Insert(User{
        Name:  "John Doe",
        Email: "john@example.com",
        Age:   25,
    }).
    Execute(ctx, &newUser)

// Batch insert
var newUsers []User
err := client.From("users").
    Insert([]User{
        {Name: "User 1", Email: "user1@example.com"},
        {Name: "User 2", Email: "user2@example.com"},
    }).
    Execute(ctx, &newUsers)

// Update document
var updatedUser User
err := client.From("users").
    Update(map[string]interface{}{"age": 26}).
    Eq("id", "user-123").
    Execute(ctx, &updatedUser)

// Delete document
err := client.From("users").
    Delete().
    Eq("id", "user-123").
    Execute(ctx, nil)

// Advanced filtering
type Order struct {
    ID         string  `json:"id"`
    Total      float64 `json:"total"`
    Priority   string  `json:"priority"`
    CustomerID string  `json:"customer_id"`
}

var orders []Order
err := client.From("orders").
    Select("*, customer:customers(*), items:order_items(*)").
    Gte("total", 100).
    Lte("total", 1000).
    Or(
        bunbase.Eq("priority", "high"),
        bunbase.Gte("amount", 500),
    ).
    Execute(ctx, &orders)

// Full-text search
var articles []Article
err := client.From("articles").
    Select("*").
    TextSearch("title", "machine learning").
    Execute(ctx, &articles)

// Real-time subscriptions
subscription := client.From("messages").
    OnInsert(func(payload bunbase.Payload) {
        log.Printf("New message: %v", payload.New)
    }).
    OnUpdate(func(payload bunbase.Payload) {
        log.Printf("Updated message: %v", payload.New)
    }).
    Subscribe(ctx)

defer subscription.Unsubscribe()
```

### 4. Storage Module

```go
// Upload file
file, err := os.Open("avatar.jpg")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

uploadResp, err := client.Storage.
    From("avatars").
    Upload(ctx, "public/user-123.jpg", file, bunbase.UploadOptions{
        CacheControl: "3600",
        Upsert:       true,
        Metadata: map[string]string{
            "userId": "user-123",
        },
    })

// Download file
data, err := client.Storage.
    From("avatars").
    Download(ctx, "public/user-123.jpg")

// Save to file
err = os.WriteFile("downloaded.jpg", data, 0644)

// List files
files, err := client.Storage.
    From("avatars").
    List(ctx, "public/", bunbase.ListOptions{
        Limit: 100,
        SortBy: bunbase.SortBy{
            Column: "created_at",
            Order:  bunbase.OrderDesc,
        },
    })

// Delete file
err := client.Storage.
    From("avatars").
    Remove(ctx, []string{"public/user-123.jpg"})

// Get public URL
url := client.Storage.
    From("avatars").
    GetPublicURL("public/user-123.jpg", bunbase.TransformOptions{
        Width:  200,
        Height: 200,
        Format: "webp",
    })

// Create signed URL
signedURL, err := client.Storage.
    From("private-files").
    CreateSignedURL(ctx, "documents/contract.pdf", 3600)

// Upload with progress
file, _ := os.Open("large-file.mp4")
defer file.Close()

err := client.Storage.
    From("videos").
    UploadWithProgress(ctx, "large-video.mp4", file, bunbase.UploadOptions{
        OnProgress: func(progress int) {
            log.Printf("Upload progress: %d%%", progress)
        },
    })
```

### 5. Functions Module

```go
// Invoke function
type EmailRequest struct {
    To       string `json:"to"`
    Subject  string `json:"subject"`
    Template string `json:"template"`
}

var response map[string]interface{}
err := client.Functions.Invoke(ctx, "send-email", EmailRequest{
    To:       "user@example.com",
    Subject:  "Welcome!",
    Template: "welcome",
}, &response)

// Invoke with custom headers
err := client.Functions.InvokeWithOptions(ctx, "protected-function",
    map[string]interface{}{"action": "delete"},
    bunbase.InvokeOptions{
        Headers: map[string]string{
            "X-Custom-Header": "value",
        },
    },
    &response,
)
```

### 6. Real-time Module

```go
// Subscribe to channel
channel := client.Realtime.Channel("chat:room-123")

err := channel.Subscribe(ctx, func(status bunbase.SubscriptionStatus) {
    log.Printf("Subscription status: %s", status)
})

// Send messages
err := channel.Send(ctx, bunbase.Message{
    Type:  "broadcast",
    Event: "new-message",
    Payload: map[string]interface{}{
        "text":   "Hello!",
        "userId": "user-123",
    },
})

// Listen to messages
channel.On("new-message", func(payload map[string]interface{}) {
    log.Printf("New message: %v", payload)
})

// Presence
presenceChannel := client.Realtime.Channel("presence:room-123", bunbase.ChannelOptions{
    Presence: true,
})

presenceChannel.Subscribe(ctx, nil)

// Track presence
err := presenceChannel.Track(ctx, map[string]interface{}{
    "user":   "John",
    "status": "online",
})

// Listen to presence
presenceChannel.OnPresence("join", func(payload map[string]interface{}) {
    log.Printf("User joined: %v", payload)
})

presenceChannel.OnPresence("leave", func(payload map[string]interface{}) {
    log.Printf("User left: %v", payload)
})
```

## Context Support

```go
import "context"

// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

var users []User
err := client.From("users").Select("*").Execute(ctx, &users)

// With cancellation
ctx, cancel := context.WithCancel(context.Background())
go func() {
    // Cancel after some condition
    cancel()
}()

err := client.Functions.Invoke(ctx, "long-running-task", nil, nil)
```

## Concurrency Support

```go
// Concurrent requests with WaitGroup
var wg sync.WaitGroup
results := make(chan []User, 3)

for i := 0; i < 3; i++ {
    wg.Add(1)
    go func(page int) {
        defer wg.Done()
        var users []User
        err := client.From("users").
            Select("*").
            Limit(100).
            Offset(page * 100).
            Execute(ctx, &users)
        if err == nil {
            results <- users
        }
    }(i)
}

wg.Wait()
close(results)

// Collect results
var allUsers []User
for users := range results {
    allUsers = append(allUsers, users...)
}
```

## Error Handling

```go
import "github.com/bunbase/bunbase-go/errors"

var users []User
err := client.From("users").Select("*").Execute(ctx, &users)

if err != nil {
    switch e := err.(type) {
    case *errors.AuthError:
        log.Printf("Auth error: %s", e.Message)
    case *errors.DatabaseError:
        log.Printf("Database error: %s (code: %s)", e.Message, e.Code)
    case *errors.RateLimitError:
        log.Printf("Rate limited. Retry after: %ds", e.RetryAfter)
    case *errors.NetworkError:
        log.Printf("Network error: %s", e.Message)
    default:
        log.Printf("Unknown error: %v", err)
    }
}
```

## Connection Pooling

```go
client, err := bunbase.NewClient(bunbase.Config{
    APIKey: "xxx",
    Options: bunbase.Options{
        PoolSize:        10,
        PoolMaxSize:     20,
        PoolTimeout:     30 * time.Second,
        PoolIdleTimeout: 5 * time.Minute,
    },
})
```

## Retry & Backoff

```go
import "github.com/bunbase/bunbase-go/retry"

client, err := bunbase.NewClient(bunbase.Config{
    APIKey: "xxx",
    Options: bunbase.Options{
        Retries: 3,
        Backoff: retry.ExponentialBackoff{
            Base:     1 * time.Second,
            MaxDelay: 60 * time.Second,
            Jitter:   true,
        },
    },
})
```

## Middleware Support

```go
// Custom middleware
func loggingMiddleware(next bunbase.Handler) bunbase.Handler {
    return func(req *bunbase.Request) (*bunbase.Response, error) {
        start := time.Now()
        log.Printf("Request: %s %s", req.Method, req.URL)

        resp, err := next(req)

        duration := time.Since(start)
        log.Printf("Response: %d (took %v)", resp.StatusCode, duration)

        return resp, err
    }
}

// Use middleware
client.Use(loggingMiddleware)
```

## Testing Utilities

```go
import "github.com/bunbase/bunbase-go/testing"

func TestUserQuery(t *testing.T) {
    mockClient := testing.NewMockClient()

    // Mock response
    mockClient.OnFrom("users").
        OnSelect("*").
        Return([]User{{ID: "1", Name: "Test User"}}, nil)

    // Test your code
    var users []User
    err := mockClient.From("users").Select("*").Execute(ctx, &users)

    assert.NoError(t, err)
    assert.Len(t, users, 1)
}
```

## Performance Features

- Connection pooling
- Request batching
- Response caching
- Context cancellation
- Struct marshaling/unmarshaling
- Zero-copy operations where possible
- Memory pooling

## Documentation Requirements

- Getting started guide
- API reference (godoc)
- Examples repository
- Migration guide
- Best practices
- Performance tuning guide
- Testing guide

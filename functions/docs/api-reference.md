# API Reference

Complete API documentation for BunBase Functions.

## HTTP Gateway API

The HTTP gateway provides REST endpoints for function invocation and management.

### Base URL

```
http://localhost:8080
```

### Endpoints

#### Health Check

```http
GET /health
```

Returns server health status.

**Response:**
```
OK
```

**Status Codes:**
- `200 OK`: Server is healthy

---

#### Invoke Function

```http
POST /functions/{function-name}
GET /functions/{function-name}
PUT /functions/{function-name}
DELETE /functions/{function-name}
```

Invoke a deployed function.

**Path Parameters:**
- `function-name`: Name or ID of the function to invoke

**Query Parameters:**
- Any query parameters are passed to the function handler

**Request Body:**
- Optional: JSON, text, or binary body
- Content-Type header determines body format

**Headers:**
- All headers (except `Host`, `Content-Length`) are passed to the function
- `X-Invocation-Id`: Automatically added (can be overridden)

**Response:**
- Status code from function handler
- Headers from function handler
- Body from function handler

**Example:**

```bash
curl -X POST "http://localhost:8080/functions/hello-world?name=Alice" \
  -H "Content-Type: application/json" \
  -d '{"greeting": "Hi"}'
```

**Status Codes:**
- `200 OK`: Function executed successfully
- `400 Bad Request`: Invalid request or function error
- `404 Not Found`: Function not found or not deployed
- `500 Internal Server Error`: Server error or function timeout
- `504 Gateway Timeout`: Function execution timeout

---

## IPC Protocol (Unix Socket)

The IPC protocol uses Unix domain sockets for inter-service communication.

### Connection

```go
import "github.com/kartikbazzad/bunbase/functions/pkg/client"

cli := client.New("/tmp/functions.sock")
err := cli.Connect()
defer cli.Close()
```

### Commands

#### Invoke Function

```go
result, err := cli.Invoke(&client.InvokeRequest{
    FunctionID: "my-function",
    Method:     "POST",
    Path:       "/",
    Headers:    map[string]string{"Content-Type": "application/json"},
    Query:      map[string]string{"name": "Alice"},
    Body:       []byte(`{"data": "value"}`),
})
```

**Request:**
- `FunctionID`: Function identifier
- `Method`: HTTP method
- `Path`: Request path
- `Headers`: Request headers map
- `Query`: Query parameters map
- `Body`: Request body bytes

**Response:**
- `Success`: Boolean indicating success
- `Status`: HTTP status code
- `Headers`: Response headers map
- `Body`: Response body bytes
- `Error`: Error message if failed

---

## Go Client API

### Client

```go
type Client struct {
    // ...
}

func New(socketPath string) *Client
func (c *Client) Connect() error
func (c *Client) Close() error
```

### Methods

#### Invoke

```go
func (c *Client) Invoke(req *InvokeRequest) (*InvokeResult, error)
```

Invoke a function via IPC.

**InvokeRequest:**
```go
type InvokeRequest struct {
    FunctionID string
    Method     string
    Path       string
    Headers    map[string]string
    Query      map[string]string
    Body       []byte
}
```

**InvokeResult:**
```go
type InvokeResult struct {
    Success bool
    Status  int
    Headers map[string]string
    Body    []byte
    Error   string
}
```

---

## Function Handler API

### Handler Signature

```typescript
export default async function handler(req: Request): Promise<Response>
```

### Request Object

Standard Web API `Request` object:

```typescript
interface Request {
  method: string;
  url: string;
  headers: Headers;
  body: ReadableStream | null;
  bodyUsed: boolean;
  
  json(): Promise<any>;
  text(): Promise<string>;
  arrayBuffer(): Promise<ArrayBuffer>;
  formData(): Promise<FormData>;
}
```

### Response Object

Standard Web API `Response` constructor:

```typescript
Response.json(data, init?: ResponseInit): Response
Response.text(text, init?: ResponseInit): Response
Response.redirect(url: string, status?: number): Response

interface ResponseInit {
  status?: number;
  statusText?: string;
  headers?: HeadersInit;
}
```

### Environment Variables

```typescript
process.env.VARIABLE_NAME
```

Access environment variables set at deployment time.

### Invocation ID

```typescript
const invocationId = req.headers.get("X-Invocation-Id");
```

Unique identifier for each invocation.

---

## Configuration API

### Server Configuration

Command-line flags:

```bash
./functions \
  --data-dir ./data \
  --socket /tmp/functions.sock \
  --log-level debug \
  --http-port 8080
```

Environment variables:

```bash
FUNCTIONS_DATA_DIR=./data
FUNCTIONS_SOCKET=/tmp/functions.sock
FUNCTIONS_LOG_LEVEL=debug
FUNCTIONS_HTTP_PORT=8080
```

### Function Configuration

Set via metadata store:

- `max_workers`: Maximum workers per function
- `warm_workers`: Number of warm workers to maintain
- `timeout_ms`: Execution timeout in milliseconds
- `memory_mb`: Memory limit in MB

See [Configuration Guide](configuration.md) for details.

---

## Error Codes

### HTTP Status Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 400 | Bad Request (invalid input or function error) |
| 404 | Not Found (function not found or not deployed) |
| 500 | Internal Server Error |
| 504 | Gateway Timeout (function timeout) |

### IPC Error Codes

| Code | Meaning |
|------|---------|
| `FUNCTION_NOT_FOUND` | Function doesn't exist |
| `FUNCTION_NOT_DEPLOYED` | Function exists but not deployed |
| `WORKER_UNAVAILABLE` | No workers available |
| `TIMEOUT` | Execution timeout |
| `BUNDLE_LOAD_ERROR` | Failed to load function bundle |
| `HANDLER_ERROR` | Function handler threw error |

---

## Rate Limits

Current version (v1) has no rate limiting. Future versions will support:
- Per-function rate limits
- Per-client rate limits
- Burst limits

---

## Security Considerations

### v1 Limitations

- Single-tenant only
- No strong sandboxing
- Network access open
- File system access limited

### Best Practices

- Validate all inputs
- Sanitize outputs
- Use environment variables for secrets
- Set appropriate timeouts
- Monitor resource usage

---

## See Also

- [Architecture Guide](architecture.md) - System internals
- [Protocol Specification](protocol.md) - IPC protocol details
- [Configuration Guide](configuration.md) - All configuration options

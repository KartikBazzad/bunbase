# Testing Functions

This guide shows how to create and test functions using the API.

## Prerequisites

1. Server must be running: `cd services/server && bun run src/index.ts`
2. You need an API key or be logged in via the dashboard
3. You need a project ID

## Step 1: Get Your API Key

If you don't have an API key, create one via the dashboard or use session-based auth.

## Step 2: Create a Function

```bash
curl -X POST http://localhost:3000/api/functions \
  -H "Content-Type: application/json" \
  -H "X-API-Key: YOUR_API_KEY" \
  -d '{
    "name": "hello-world",
    "runtime": "bun",
    "handler": "handler",
    "code": "export async function handler(req: Request): Promise<Response> { const url = new URL(req.url); const name = url.searchParams.get(\"name\") || \"World\"; return Response.json({ message: `Hello, ${name}!`, timestamp: new Date().toISOString() }); }"
  }'
```

Response:
```json
{
  "id": "function-id-here",
  "name": "hello-world",
  "runtime": "bun",
  "handler": "handler",
  "status": "draft",
  ...
}
```

Save the `id` from the response.

## Step 3: Deploy the Function

```bash
curl -X POST http://localhost:3000/api/functions/FUNCTION_ID/deploy \
  -H "X-API-Key: YOUR_API_KEY"
```

Response:
```json
{
  "message": "Function deployed successfully",
  "version": "1.0.0",
  "deploymentId": "..."
}
```

## Step 4: Invoke the Function

### Option A: Via Invoke Endpoint

```bash
curl -X POST http://localhost:3000/api/functions/FUNCTION_ID/invoke \
  -H "Content-Type: application/json" \
  -H "X-API-Key: YOUR_API_KEY" \
  -d '{
    "method": "GET",
    "url": "http://localhost:3000/api/functions/hello-world?name=Alice",
    "headers": {}
  }'
```

### Option B: Via Direct HTTP Endpoint (by name)

```bash
curl http://localhost:3000/api/functions/hello-world?name=Alice \
  -H "X-API-Key: YOUR_API_KEY"
```

### Option C: Via Direct HTTP Endpoint (by ID)

```bash
curl http://localhost:3000/api/functions/FUNCTION_ID/invoke/?name=Alice \
  -H "X-API-Key: YOUR_API_KEY"
```

## Step 5: View Logs

```bash
curl http://localhost:3000/api/functions/FUNCTION_ID/logs \
  -H "X-API-Key: YOUR_API_KEY"
```

## Step 6: View Metrics

```bash
curl http://localhost:3000/api/functions/FUNCTION_ID/metrics \
  -H "X-API-Key: YOUR_API_KEY"
```

## Quick Test Script

Use the provided `quick-test.ts` script:

```bash
cd examples/functions
API_KEY=your_api_key bun run quick-test.ts
```

## Example Functions

All example functions are in `examples/functions/`:

- `hello-world.ts` - Simple greeting
- `echo.ts` - Echoes request data
- `json-processor.ts` - Processes JSON
- `calculator.ts` - Calculator API
- `env-demo.ts` - Environment variables demo

## Using the Example Functions

To use the example functions, read the file and pass it as the `code` field:

```bash
# Read function code
CODE=$(cat examples/functions/hello-world.ts)

# Create function
curl -X POST http://localhost:3000/api/functions \
  -H "Content-Type: application/json" \
  -H "X-API-Key: YOUR_API_KEY" \
  -d "{
    \"name\": \"hello-world\",
    \"runtime\": \"bun\",
    \"handler\": \"handler\",
    \"code\": $(echo "$CODE" | jq -Rs .)
  }"
```

## Troubleshooting

### Function not found
- Make sure the function is deployed (status should be "deployed")
- Check that you're using the correct function ID or name

### Execution failed
- Check function logs: `GET /functions/{id}/logs`
- Verify the function code is valid
- Check that the handler function is exported correctly

### Cold start delays
- First invocation may be slower (cold start)
- Subsequent invocations use warm cache (faster)

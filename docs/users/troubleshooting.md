# Troubleshooting Guide

Common issues and solutions when using BunBase.

## Authentication Issues

### "Not authenticated" Error

**Problem**: API requests return 401 Unauthorized.

**Solutions:**

1. **CLI**: Run `bunbase login --email ... --password ...` to authenticate
2. **API**: Ensure session cookies are included in requests
3. **Dashboard**: Log out and log back in

### Session Expired

**Problem**: Previously working requests now fail.

**Solution**: Re-authenticate:

```bash
bunbase login --email you@example.com --password 'your-password'
```

## Function Deployment Issues

### "Function deployment failed"

**Problem**: Function won't deploy.

**Check:**

1. **Function file exists**: Verify the file path is correct
2. **Default export**: Ensure function exports default handler:
   ```typescript
   export default async function handler(req: Request): Promise<Response> {
     // ...
   }
   ```
3. **Runtime supported**: Use `bun` (and `quickjs-ng` only when your environment/runtime wiring supports it)
4. **Bundle format**: If bundling manually, ensure correct format

**Solution**: Review function logs in dashboard for detailed errors.

### "Function not found" After Deployment

**Problem**: Function deployed but can't be invoked.

**Check:**

1. Function is visible in dashboard function list or via `GET /api/projects/:id/functions`
2. Function status is `deployed` (check dashboard)
3. Bundle file exists at expected path
4. Function name matches exactly (case-sensitive)

**Solution**: Redeploy the function or check function logs.

### Bundle Encoding Issues

**Problem**: Base64 encoding errors when deploying.

**Solution**: Ensure bundle is properly encoded:

```bash
# Correct way
BUNDLE=$(base64 -i bundle.js)
# or
BUNDLE=$(cat bundle.js | base64)
```

## Function Execution Issues

### Function Timeout

**Problem**: Function execution times out after 30 seconds.

**Solutions:**

1. Optimize function code
2. Break work into smaller chunks
3. Use async/await properly
4. Check for blocking operations

**Check logs** for execution time:

```bash
# View function logs in dashboard
# or check functions service logs
```

### Function Returns 500 Error

**Problem**: Function executes but returns server error.

**Check:**

1. Function logs for exceptions
2. Console output in function code
3. Error handling in function

**Debug:**

```typescript
export default async function handler(req: Request): Promise<Response> {
  try {
    // Your code
  } catch (error) {
    console.error("Error:", error);
    return Response.json({ error: error.message }, { status: 500 });
  }
}
```

### Function Not Receiving Request Body

**Problem**: `req.json()` returns empty or undefined.

**Check:**

1. Content-Type header: `application/json`
2. Request body is valid JSON
3. Body is not empty

**Example:**

```typescript
const contentType = req.headers.get("Content-Type");
if (contentType !== "application/json") {
  return Response.json({ error: "Invalid content type" }, { status: 400 });
}

const body = await req.json();
```

## Project Issues

### "Project not found"

**Problem**: Can't access project.

**Check:**

1. Project ID is correct: `bunbase projects list`
2. You're the project owner
3. Project wasn't deleted

**Solution**: Verify project exists and you have access.

### "No active project"

**Problem**: CLI says no active project set.

**Solution**: Set active project:

```bash
bunbase projects use <project-id>
```

## CLI Issues

### Command Not Found

**Problem**: `bunbase` command not found.

**Solution**: Add to PATH or use full path:

```bash
# Add to PATH
export PATH=$PATH:/path/to/bunbase

# Or use full path
/path/to/bunbase projects list
```

### Permission Denied

**Problem**: Can't execute CLI binary.

**Solution**: Make executable:

```bash
chmod +x bunbase
```

## Environment Variables

### Environment Variable Not Available

**Problem**: `process.env.MY_VAR` is undefined.

**Check:**

1. Variable is set in dashboard/API
2. Variable name matches exactly (case-sensitive)
3. Function was redeployed after setting variable

**Solution**: Set environment variable and redeploy function.

## Network Issues

### Can't Connect to Platform API

**Problem**: Connection refused or timeout.

**Check:**

1. Platform API is running: `curl http://localhost:3001/health`
2. Correct port (default: 3001)
3. Firewall/network settings

**Solution**: Start platform API:

```bash
cd platform
./platform-server --port 3001
```

### Can't Connect to Functions Service

**Problem**: Function invocations fail.

**Check:**

1. Functions service is running
2. Socket path is correct (default: `/tmp/functions.sock`)
3. Permissions on socket file

**Solution**: Start functions service:

```bash
cd functions
./functions --socket /tmp/functions.sock
```

## Performance Issues

### Slow Function Execution

**Problem**: Functions take too long to execute.

**Check:**

1. Function code efficiency
2. External API calls (timeouts, retries)
3. Database queries (if using Bundoc APIs)
4. Cold start vs warm execution

**Solutions:**

- Optimize code
- Cache results when possible
- Use warm workers (keep functions active)

### High Memory Usage

**Problem**: Functions consume too much memory.

**Check:**

1. Memory usage in function logs
2. Large data structures
3. Memory leaks

**Solutions:**

- Optimize data structures
- Release unused references
- Process data in chunks

## Getting More Help

### Check Logs

1. **Function logs**: Dashboard → Project → Function → Logs
2. **Platform API logs**: Check terminal where `platform-server` is running
3. **Functions service logs**: Check terminal where `functions` is running

### Debug Mode

Enable debug logging:

```bash
# Functions service
./functions --log-level debug

# Platform API (if supported)
./platform-server --log-level debug
```

### Common Error Messages

| Error                | Cause                 | Solution                            |
| -------------------- | --------------------- | ----------------------------------- |
| "Not authenticated"  | No valid session      | Run `bunbase login --email ... --password ...` |
| "Project not found"  | Invalid project ID    | Verify with `bunbase projects list` |
| "Function not found" | Function not deployed | Deploy function first               |
| "Timeout"            | Function took >30s    | Optimize function code              |
| "Invalid bundle"     | Bundle format error   | Check bundle encoding               |

## See Also

- [Getting Started](getting-started.md)
- [Writing Functions](writing-functions.md)
- [CLI Guide](cli-guide.md)
- [Platform API Reference](api-reference.md)

# Troubleshooting Guide

Common issues and solutions for BunBase Functions.

## Server Won't Start

### Port Already in Use

**Error:**
```
bind: address already in use
```

**Solution:**
```bash
# Find process using port 8080
lsof -i :8080

# Kill the process or use a different port
./functions --http-port 8081
```

### Socket File Exists

**Error:**
```
bind: address already in use (Unix socket)
```

**Solution:**
```bash
# Remove existing socket
rm /tmp/functions.sock

# Or use a different socket path
./functions --socket /tmp/functions-alt.sock
```

### Permission Denied

**Error:**
```
permission denied: creating directory
```

**Solution:**
```bash
# Ensure data directory is writable
chmod 755 ./data
mkdir -p ./data/bundles ./data/logs

# Or use a different data directory
./functions --data-dir ~/.functions-data
```

---

## Function Not Found

### Function Not Registered

**Error:**
```
Function not found
```

**Check:**
```bash
sqlite3 data/metadata.db "SELECT * FROM functions;"
```

**Solution:**
Register the function (see [Getting Started](getting-started.md)).

### Function Not Deployed

**Error:**
```
Function not deployed
```

**Check:**
```bash
sqlite3 data/metadata.db "SELECT id, name, status FROM functions;"
```

**Solution:**
Ensure function status is `deployed`:
```sql
UPDATE functions SET status = 'deployed' WHERE id = 'your-function-id';
```

### Bundle Path Incorrect

**Error:**
```
Failed to load bundle: Cannot find module
```

**Check:**
```bash
# Verify bundle exists
ls -la data/bundles/your-function/v1/bundle.js

# Check path in database
sqlite3 data/metadata.db "SELECT bundle_path FROM function_versions WHERE function_id = 'your-function-id';"
```

**Solution:**
Update bundle path to absolute path or ensure relative path is correct.

---

## Worker Won't Start

### Bun Not Found

**Error:**
```
exec: "bun": executable file not found in $PATH
```

**Solution:**
```bash
# Install Bun
curl -fsSL https://bun.sh/install | bash

# Verify installation
bun --version

# Add to PATH if needed
export PATH="$HOME/.bun/bin:$PATH"
```

### Bundle Export Error

**Error:**
```
No handler function found. Expected default export or named 'handler' export.
```

**Check your bundle:**
```bash
# Inspect bundle
cat data/bundles/your-function/v1/bundle.js | grep -E "export|default"
```

**Solution:**
Ensure your function exports a default handler:
```typescript
export default async function handler(req: Request): Promise<Response> {
  // ...
}
```

### Bundle Syntax Error

**Error:**
```
Failed to load bundle: SyntaxError
```

**Solution:**
```bash
# Test bundle directly
bun data/bundles/your-function/v1/bundle.js

# Rebuild bundle
bun build your-function.ts --outdir ./data/bundles/your-function/v1 --target bun --outfile bundle.js
```

---

## Invocation Timeout

### Function Takes Too Long

**Error:**
```
invocation deadline exceeded
```

**Check:**
- Function execution time
- External API calls
- Database queries
- Long-running loops

**Solution:**
1. Optimize function code
2. Increase timeout:
   ```go
   // In gateway configuration
   DefaultTimeout: 60 * time.Second
   ```
3. Use async operations with timeouts:
   ```typescript
   const controller = new AbortController();
   setTimeout(() => controller.abort(), 25000);
   
   const response = await fetch(url, {
     signal: controller.signal
   });
   ```

### Worker Process Hung

**Symptoms:**
- Invocation hangs indefinitely
- Worker process still running
- No response received

**Solution:**
```bash
# Check worker processes
ps aux | grep bun

# Kill hung workers (server will respawn)
pkill -f "bun.*worker.ts"

# Restart server
```

---

## Memory Issues

### Worker Killed (OOM)

**Error:**
```
Worker process exited unexpectedly
```

**Check:**
```bash
# Check system memory
free -h

# Check worker memory usage
ps aux | grep bun
```

**Solution:**
1. Reduce function memory usage
2. Increase worker memory limit:
   ```go
   // In worker configuration
   MemoryLimitMB: 512
   ```
3. Reduce concurrent workers:
   ```go
   MaxWorkersPerFunction: 5
   ```

---

## IPC Communication Errors

### Socket Connection Failed

**Error:**
```
connect: no such file or directory
```

**Solution:**
```bash
# Verify socket exists
ls -la /tmp/functions.sock

# Check server is running
curl http://localhost:8080/health

# Restart server if needed
```

### Message Parsing Error

**Error:**
```
failed to unmarshal message: invalid character
```

**Cause:**
- Console.log output breaking NDJSON protocol
- Malformed messages

**Solution:**
- Ensure all console output goes to stderr (handled automatically)
- Check worker logs for errors
- Verify worker.ts is using correct message format

---

## Performance Issues

### Slow Cold Starts

**Symptoms:**
- First invocation takes > 500ms
- Subsequent invocations are fast

**Solution:**
- This is expected for cold starts
- Use warm workers to avoid cold starts:
  ```go
  WarmWorkersPerFunction: 2
  ```
- Pre-warm functions on deployment

### High Memory Usage

**Symptoms:**
- System memory exhausted
- Workers killed frequently

**Solution:**
1. Profile function memory usage
2. Reduce bundle size
3. Limit concurrent workers
4. Set memory limits per function

### Slow Warm Invocations

**Symptoms:**
- Warm invocations take > 50ms
- Function code is simple

**Check:**
- External API calls
- Database queries
- Network latency
- Function code efficiency

**Solution:**
- Optimize function code
- Cache external responses
- Use connection pooling
- Reduce external dependencies

---

## Logging Issues

### No Logs Appearing

**Check:**
```bash
# Verify log level
./functions --log-level debug

# Check log files
ls -la data/logs/

# Check database logs
sqlite3 data/logs.db "SELECT * FROM logs LIMIT 10;"
```

### Too Many Logs

**Solution:**
```bash
# Reduce log level
./functions --log-level warn

# Or in code
logger.SetLevel(logger.LevelWarn)
```

---

## Database Issues

### Database Locked

**Error:**
```
database is locked
```

**Solution:**
```bash
# Check for other processes accessing database
lsof data/metadata.db

# Close other connections
# Restart server
```

### Corrupted Database

**Error:**
```
database disk image is malformed
```

**Solution:**
```bash
# Backup first
cp data/metadata.db data/metadata.db.backup

# Try to recover
sqlite3 data/metadata.db ".recover" | sqlite3 data/metadata.db.recovered

# Or recreate database (loses data)
rm data/metadata.db
# Server will recreate on startup
```

---

## Getting Help

### Enable Debug Logging

```bash
./functions --log-level debug
```

### Check Worker Logs

```bash
# Server logs
tail -f server.log

# Worker stderr (captured by server)
# Check server output for "Worker X stderr:" messages
```

### Inspect Database

```bash
# Functions
sqlite3 data/metadata.db "SELECT * FROM functions;"

# Versions
sqlite3 data/metadata.db "SELECT * FROM function_versions;"

# Deployments
sqlite3 data/metadata.db "SELECT * FROM function_deployments;"

# Logs
sqlite3 data/logs.db "SELECT * FROM logs ORDER BY created_at DESC LIMIT 10;"
```

### Check Processes

```bash
# Server process
ps aux | grep functions

# Worker processes
ps aux | grep "bun.*worker.ts"

# Socket
ls -la /tmp/functions.sock
```

---

## Common Patterns

### Function Works Locally But Not in Server

1. Check bundle path is absolute or correct relative path
2. Verify all dependencies are bundled
3. Check environment variables are set
4. Verify function exports default handler

### Intermittent Failures

1. Check for race conditions in function code
2. Verify worker pool limits aren't exceeded
3. Check for memory leaks
4. Review timeout settings

### First Invocation Fails, Second Succeeds

1. This is often a cold start issue
2. Check worker startup logs
3. Verify bundle loads correctly
4. Use warm workers to avoid cold starts

---

## Still Having Issues?

1. Check [Architecture Guide](architecture.md) for system understanding
2. Review [Protocol Specification](protocol.md) for IPC details
3. Enable debug logging and review logs
4. Test function directly with Bun
5. Check server and worker process status

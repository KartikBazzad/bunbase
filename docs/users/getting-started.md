# Getting Started with BunBase

Welcome to BunBase! This guide will help you get started with the platform, from creating your account to deploying your first function.

## What is BunBase?

BunBase is a developer platform that lets you:

- **Deploy JavaScript/TypeScript functions** with minimal configuration
- **Manage projects** and organize your functions
- **Store and query data** using DocDB, our embedded document database
- **Build serverless applications** with fast, warm execution

## Quick Start

### 1. Create an Account

Visit the BunBase dashboard and sign up:

```bash
# If running locally, navigate to:
http://localhost:5173
```

Click **"Sign Up"** and provide:

- Email address
- Password (minimum 8 characters)

### 2. Create Your First Project

After logging in, you'll see the dashboard. Click **"Create Project"** and give it a name:

- Project names are converted to URL-friendly slugs (e.g., "My App" → "my-app")
- Each project can contain multiple functions
- Projects help organize your functions and manage access

### 3. Deploy Your First Function

#### Option A: Using the CLI (Recommended)

1. **Install the BunBase CLI** (if not already installed):

   ```bash
   # Build from source
   cd platform
   go build -o bunbase ./cmd/cli
   ```

2. **Login via CLI**:

   ```bash
   ./bunbase auth login
   # Enter your email and password
   ```

3. **Select your project**:

   ```bash
   ./bunbase projects list
   ./bunbase projects use <project-id>
   ```

4. **Create a function file**:

   ```typescript
   // hello.ts
   export default async function handler(req: Request): Promise<Response> {
     const name = new URL(req.url).searchParams.get("name") || "World";
     return Response.json({ message: `Hello, ${name}!` });
   }
   ```

5. **Deploy the function**:
   ```bash
   ./bunbase deploy hello.ts --name hello --runtime bun --handler default
   ```

#### Option B: Using the Web Dashboard

1. Navigate to your project
2. Click **"Deploy Function"**
3. Upload your function file or paste the code
4. Provide function details:
   - Name: `hello`
   - Runtime: `bun`
   - Handler: `default`
5. Click **"Deploy"**

### 4. Invoke Your Function

Once deployed, you can invoke your function via HTTP:

```bash
curl "http://localhost:8080/functions/hello?name=Alice"
```

Response:

```json
{
  "message": "Hello, Alice!"
}
```

## Next Steps

- **[Writing Functions](writing-functions.md)** - Learn how to write effective functions
- **[Using the CLI](cli-guide.md)** - Complete CLI reference
- **[Platform API](api-reference.md)** - REST API documentation
- **[Managing Projects](projects.md)** - Project management guide
- **[DocDB Guide](../docdb/docs/usage.md)** - Using the document database

## Common Questions

### What runtimes are supported?

Currently, BunBase supports:

- **Bun** - Fast JavaScript/TypeScript runtime (recommended)

### How do I handle environment variables?

Environment variables can be set per function in the dashboard or via CLI. They're available in your function via `process.env`.

### Can I use npm packages?

Yes! Bun supports npm packages. Use `bun install` in your function directory before bundling.

### How do I debug functions?

- Check function logs in the dashboard
- Use `console.log()` in your function code
- Review execution metrics in the project dashboard

### What are the limits?

Current limits (v1):

- Function timeout: 30 seconds
- Memory per function: 200MB
- Max concurrent invocations: 100 per function

## Getting Help

- **Documentation**: Browse the [user guides](README.md)
- **Issues**: Check the [troubleshooting guide](troubleshooting.md)
- **Support**: Contact support or open an issue on GitHub

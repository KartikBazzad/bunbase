/**
 * Function Worker Script
 * Runs in a worker thread to execute function code
 */

// Using Bun native APIs

interface WorkerMessage {
  functionId: string;
  version: string;
  codePath: string;
  request: {
    method: string;
    url: string;
    headers: Record<string, string>;
    body?: any;
  };
  env: Record<string, string>;
  timeout: number;
}

interface WorkerResponse {
  success: boolean;
  response?: {
    status: number;
    headers: Record<string, string>;
    body: any;
  };
  error?: string;
  logs: Array<{ level: string; message: string; timestamp: Date }>;
  executionTime: number;
  memoryUsed?: number;
}

/**
 * Cleanup execution context (timers, globals, module cache)
 */
function cleanupExecutionContext(): void {
  // Clear all active timers
  // Note: In Bun workers, we can't easily enumerate all timers
  // But we can clear common ones and reset state

  // Reset any global state that might persist
  // (Add specific cleanup as needed based on function behavior)
}

// Listen for messages from main thread
self.addEventListener("message", async (event: MessageEvent<WorkerMessage>) => {
  const task = event.data;
  const startTime = Date.now();
  const logs: Array<{ level: string; message: string; timestamp: Date }> = [];

  // Cleanup before execution
  cleanupExecutionContext();

  // Save original environment
  const originalEnv = { ...process.env };

  try {
    // Set environment variables (isolated per execution)
    for (const [key, value] of Object.entries(task.env)) {
      process.env[key] = value;
    }

    // Read function code using Bun.file
    const codeFile = Bun.file(task.codePath);
    if (!(await codeFile.exists())) {
      throw new Error(`Function code not found: ${task.codePath}`);
    }

    const code = await codeFile.text();

    // Create a Request object
    const request = new Request(task.request.url, {
      method: task.request.method,
      headers: task.request.headers,
      body: task.request.body
        ? JSON.stringify(task.request.body)
        : undefined,
    });

    // Execute function code
    // For Bun, we can directly import TypeScript files
    // Convert file path to file:// URL for cross-platform compatibility
    // Use cache-busting query parameter to force fresh import
    const { pathToFileURL } = await import("url");
    const modulePath = pathToFileURL(task.codePath).href + "?t=" + Date.now();
    
    // Import with fresh module cache (cache-busting via query param)
    const handlerModule = await import(modulePath);
    
    // Get handler function
    let handler: (req: Request) => Promise<Response>;
    if (handlerModule.handler) {
      handler = handlerModule.handler;
    } else if (handlerModule.default) {
      handler = handlerModule.default;
    } else {
      throw new Error("Function must export a 'handler' function");
    }

    // Execute handler
    const response = await handler(request);

    // Restore original environment
    process.env = originalEnv;

    // Extract response data
    const responseBody = await response.text();
    let parsedBody: any;
    try {
      parsedBody = JSON.parse(responseBody);
    } catch {
      parsedBody = responseBody;
    }

    const result: WorkerResponse = {
      success: true,
      response: {
        status: response.status,
        headers: Object.fromEntries(response.headers.entries()),
        body: parsedBody,
      },
      logs,
      executionTime: Date.now() - startTime,
    };

    // Cleanup after execution
    cleanupExecutionContext();

    // Send result back to main thread
    self.postMessage(result);
  } catch (error: any) {
    // Cleanup on error
    cleanupExecutionContext();

    // Restore environment if it was set
    if (typeof originalEnv !== "undefined") {
      process.env = originalEnv;
    }

    const result: WorkerResponse = {
      success: false,
      error: error.message || "Function execution failed",
      logs: [
        ...logs,
        {
          level: "error",
          message: error.message || "Unknown error",
          timestamp: new Date(),
        },
      ],
      executionTime: Date.now() - startTime,
    };

    self.postMessage(result);
  }
});

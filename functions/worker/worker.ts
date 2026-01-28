/**
 * BunBase Functions Worker
 *
 * This script runs as a Bun process and handles function invocations.
 * It communicates with the Go control plane via stdin/stdout using NDJSON.
 */

import type { BodyInit } from "bun";

interface Message {
  id: string;
  type: "ready" | "invoke" | "response" | "log" | "error";
  payload: any;
}

interface InvokePayload {
  method: string;
  path: string;
  headers: Record<string, string>;
  query: Record<string, string>;
  body: string; // base64-encoded
  deadline_ms: number;
}

interface ResponsePayload {
  status: number;
  headers: Record<string, string>;
  body: string; // base64-encoded
}

interface LogPayload {
  level: "info" | "warn" | "error" | "debug";
  message: string;
  metadata?: Record<string, any>;
}

interface ErrorPayload {
  message: string;
  stack?: string;
  code?: string;
}

// Get bundle path from environment
const BUNDLE_PATH = process.env.BUNDLE_PATH;
const WORKER_ID = process.env.WORKER_ID || `worker-${Date.now()}`;

if (!BUNDLE_PATH) {
  console.error("BUNDLE_PATH environment variable is required");
  process.exit(1);
}

// Intercept console methods FIRST (before any logging)
// This ensures all console output goes through the log message system
const originalLog = console.log;
const originalError = console.error;
const originalWarn = console.warn;
const originalDebug = console.debug;

let currentInvocationId: string | null = null;

function interceptLog(
  level: "info" | "warn" | "error" | "debug",
  ...args: any[]
) {
  // During bundle loading (no invocation ID), send logs to stderr to avoid breaking NDJSON protocol
  // After bundle loading, logs go to stdout via original methods AND log messages if there's an invocation
  if (!currentInvocationId) {
    // No active invocation - send to stderr to avoid interfering with NDJSON on stdout
    // Use the appropriate original console method based on log level
    const logMessage = args
      .map((arg) =>
        typeof arg === "object" ? JSON.stringify(arg) : String(arg),
      )
      .join(" ");

    switch (level) {
      case "info":
        originalError(`[INFO] ${logMessage}`);
        break;
      case "debug":
        originalError(`[DEBUG] ${logMessage}`);
        break;
      case "warn":
        originalError(`[WARN] ${logMessage}`);
        break;
      case "error":
        originalError(`[ERROR] ${logMessage}`);
        break;
    }
    return;
  }

  // Active invocation - DON'T use original methods (they write to stdout and break NDJSON)
  // Instead, just send as NDJSON log message
  const message = args
    .map((arg) => (typeof arg === "object" ? JSON.stringify(arg) : String(arg)))
    .join(" ");

  sendLog(currentInvocationId, {
    level,
    message,
  });
}

// Override console methods
console.log = (...args: any[]) => interceptLog("info", ...args);
console.error = (...args: any[]) => interceptLog("error", ...args);
console.warn = (...args: any[]) => interceptLog("warn", ...args);
console.debug = (...args: any[]) => interceptLog("debug", ...args);

// Load function bundle
let handler: ((req: Request) => Promise<Response>) | null = null;

try {
  const bundle = await import(BUNDLE_PATH);

  // Try default export first
  if (bundle.default && typeof bundle.default === "function") {
    handler = bundle.default;
  } else if (typeof bundle.handler === "function") {
    handler = bundle.handler;
  } else {
    throw new Error(
      "No handler function found. Expected default export or named 'handler' export.",
    );
  }
} catch (error: any) {
  console.error(`Failed to load bundle: ${error.message}`);
  sendError("bundle-load", {
    message: `Failed to load bundle: ${error.message}`,
    stack: error.stack,
    code: "BUNDLE_LOAD_ERROR",
  });
  process.exit(1);
}

// Send READY message immediately
const readyMsg =
  JSON.stringify({
    id: WORKER_ID,
    type: "ready",
    payload: {},
  }) + "\n";

process.stdout.write(readyMsg);
// Try to flush (Bun might buffer, but flush may not be available)
try {
  const stdout = process.stdout as any;
  if (stdout && typeof stdout.flush === "function") {
    stdout.flush();
  }
} catch (e) {
  console.error(`Failed to flush ready message: ${e}`);
  // Flush not available or failed - that's OK, message should still be sent
}

// Console interception is already set up earlier (before bundle loading)

// Helper functions
function sendMessage(msg: Message) {
  const json = JSON.stringify(msg);
  const line = json + "\n";
  process.stdout.write(line);
  // Try to flush (may not be available in Bun)
  try {
    const stdout = process.stdout as any;
    if (stdout && typeof stdout.flush === "function") {
      stdout.flush();
    }
  } catch (e) {
    // Flush not available - that's OK, message should still be sent
  }
}

function sendLog(invocationId: string, payload: LogPayload) {
  sendMessage({
    id: invocationId,
    type: "log",
    payload,
  });
}

function sendError(invocationId: string, payload: ErrorPayload) {
  sendMessage({
    id: invocationId,
    type: "error",
    payload,
  });
}

function sendResponse(invocationId: string, payload: ResponsePayload) {
  sendMessage({
    id: invocationId,
    type: "response",
    payload,
  });
}

// Helper to create Request object from invoke payload
function createRequest(payload: InvokePayload): Request {
  const url = new URL(payload.path, "http://localhost");

  // Add query parameters
  for (const [key, value] of Object.entries(payload.query)) {
    url.searchParams.set(key, value);
  }

  // Decode body
  let body: any = null;
  if (payload.body) {
    try {
      const decoded = Buffer.from(payload.body, "base64");
      body = decoded;
    } catch (error) {
      // If decoding fails, use as-is
      body = payload.body;
    }
  }

  // Create Request object
  return new Request(url.toString(), {
    method: payload.method,
    headers: payload.headers,
    body: body,
  });
}

// Helper to convert Response to ResponsePayload
async function responseToPayload(response: Response): Promise<ResponsePayload> {
  const headers: Record<string, string> = {};
  response.headers.forEach((value, key) => {
    headers[key] = value;
  });

  // Read body and encode as base64
  let body = "";
  try {
    const bodyBuffer = await response.arrayBuffer();
    if (bodyBuffer.byteLength > 0) {
      body = Buffer.from(bodyBuffer).toString("base64");
    }
  } catch (error) {
    console.error(`Failed to read response body: ${error}`);
  }

  return {
    status: response.status,
    headers,
    body,
  };
}

// Main message loop
async function processMessage(msg: Message) {
  if (msg.type === "invoke") {
    const payload = msg.payload as InvokePayload;
    currentInvocationId = msg.id;

    // Check deadline
    if (payload.deadline_ms > 0) {
      const deadline = Date.now() + payload.deadline_ms;
      if (Date.now() >= deadline) {
        sendError(msg.id, {
          message: "Invocation deadline exceeded",
          code: "DEADLINE_EXCEEDED",
        });
        currentInvocationId = null;
        return;
      }
    }

    try {
      // Create Request object
      const request = createRequest(payload);

      // Execute handler
      if (!handler) {
        throw new Error("Handler not loaded");
      }

      const response = await handler(request);

      // Convert Response to payload
      const responsePayload = await responseToPayload(response);

      // Send response
      sendResponse(msg.id, responsePayload);
    } catch (error: any) {
      sendError(msg.id, {
        message: error.message || "Handler execution failed",
        stack: error.stack,
        code: "HANDLER_ERROR",
      });
    } finally {
      currentInvocationId = null;
    }
  }
}

// Read messages from stdin
const reader = Bun.stdin.stream().getReader();
const decoder = new TextDecoder();
let buffer = "";

while (true) {
  const { done, value } = await reader.read();

  if (done) {
    break;
  }

  buffer += decoder.decode(value, { stream: true });

  // Process complete lines (NDJSON)
  const lines = buffer.split("\n");
  buffer = lines.pop() || ""; // Keep incomplete line in buffer

  for (const line of lines) {
    if (line.trim() === "") {
      continue;
    }

    try {
      const msg = JSON.parse(line) as Message;
      await processMessage(msg);
    } catch (error: any) {
      // Ignore malformed JSON (log to stderr but don't crash)
      console.error(`Failed to parse message: ${error.message}`);
    }
  }
}

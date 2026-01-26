/**
 * Function Process Script
 * Runs in isolated process to execute function code
 */

// Using Bun native APIs

const codePath = process.env.FUNCTION_CODE_PATH!;
const requestJson = process.env.FUNCTION_REQUEST!;
const timeout = parseInt(process.env.FUNCTION_TIMEOUT || "30000");

interface ProcessResult {
  success: boolean;
  response?: {
    status: number;
    headers: Record<string, string>;
    body: any;
  };
  error?: string;
  logs: Array<{ level: string; message: string; timestamp: Date }>;
}

async function main() {
  const startTime = Date.now();
  const logs: Array<{ level: string; message: string; timestamp: Date }> = [];

  try {
    // Read function code using Bun.file
    const codeFile = Bun.file(codePath);
    if (!(await codeFile.exists())) {
      throw new Error(`Function code not found: ${codePath}`);
    }

    // Parse request
    const requestData = JSON.parse(requestJson);
    const request = new Request(requestData.url, {
      method: requestData.method,
      headers: requestData.headers,
      body: requestData.body ? JSON.stringify(requestData.body) : undefined,
    });

    // Import function module
    const { pathToFileURL } = await import("url");
    const modulePath = pathToFileURL(codePath).href + "?t=" + Date.now();
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

    // Execute handler with timeout
    const executionPromise = handler(request);
    const timeoutPromise = new Promise<Response>((_, reject) => {
      setTimeout(() => reject(new Error("Function timeout")), timeout);
    });

    const response = await Promise.race([executionPromise, timeoutPromise]);

    // Extract response data
    const responseBody = await response.text();
    let parsedBody: any;
    try {
      parsedBody = JSON.parse(responseBody);
    } catch {
      parsedBody = responseBody;
    }

    const result: ProcessResult = {
      success: true,
      response: {
        status: response.status,
        headers: Object.fromEntries(response.headers.entries()),
        body: parsedBody,
      },
      logs,
    };

    // Output result as JSON
    console.log(JSON.stringify(result));
  } catch (error: any) {
    const result: ProcessResult = {
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
    };

    console.log(JSON.stringify(result));
    process.exit(1);
  }
}

main();

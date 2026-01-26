import { serve } from "bun";
import index from "./index.html";
import app from "./server";
import { logger } from "./lib/logger";
import { initializeLogBuffer } from "./lib/function-log-buffer";
import { flushLogsToStorage } from "./lib/function-log-storage";
import { db, projects } from "./db";
import { eq } from "drizzle-orm";
import { getFunctionByName } from "./lib/function-helpers";
import { executeFunction } from "./lib/function-executor";
import { join } from "path";

// Initialize function log buffer
initializeLogBuffer(flushLogsToStorage);

/**
 * Handle subdomain-based function routing
 * Supports patterns like: project-name.localhost:3000/functions/function-name
 */
async function handleSubdomainRouting(req: Request): Promise<Response | null> {
  const host = req.headers.get("host");
  if (!host) {
    return null;
  }

  // Parse subdomain (e.g., "project-name" from "project-name.localhost:3000")
  const hostParts = host.split(".");
  if (hostParts.length < 2) {
    return null; // No subdomain
  }

  const subdomain = hostParts[0];
  if (!subdomain || subdomain === "localhost" || subdomain === "127.0.0.1") {
    return null; // No valid subdomain
  }

  const url = new URL(req.url);

  // Only handle function routes on subdomains
  // Pattern: /functions/{function-name} or just /{function-name}
  const pathMatch = url.pathname.match(/^\/(?:functions\/)?([^\/]+)$/);
  if (!pathMatch || !pathMatch[1]) {
    return null; // Not a function route
  }

  const functionName = pathMatch[1];

  try {
    // Look up project by name (using subdomain as project identifier)
    // For now, we'll use the subdomain as the project name
    // In production, you might want to add a dedicated subdomain field
    const [project] = await db
      .select()
      .from(projects)
      .where(eq(projects.name, subdomain))
      .limit(1);

    if (!project) {
      return new Response(
        JSON.stringify({
          error: {
            message: `Project not found for subdomain: ${subdomain}`,
            code: "PROJECT_NOT_FOUND",
          },
        }),
        {
          status: 404,
          headers: { "Content-Type": "application/json" },
        },
      );
    }

    // Get function by name
    const func = await getFunctionByName(project.id, functionName);

    if (func.status !== "deployed") {
      return new Response(
        JSON.stringify({
          error: {
            message: "Function is not deployed",
            code: "FUNCTION_NOT_DEPLOYED",
          },
        }),
        {
          status: 400,
          headers: { "Content-Type": "application/json" },
        },
      );
    }

    // Parse request
    const method = req.method;
    const headers: Record<string, string> = {};
    req.headers.forEach((value, key) => {
      // Skip certain headers
      if (
        !key.toLowerCase().startsWith("x-api-key") &&
        key.toLowerCase() !== "authorization" &&
        key.toLowerCase() !== "host"
      ) {
        headers[key] = value;
      }
    });

    // Get request body if present
    let requestBody: any = undefined;
    if (req.body && method !== "GET" && method !== "HEAD") {
      try {
        const contentType = req.headers.get("content-type") || "";
        if (contentType.includes("application/json")) {
          requestBody = await req.json();
        } else if (contentType.includes("text/")) {
          requestBody = await req.text();
        } else if (contentType.includes("application/x-www-form-urlencoded")) {
          const formData = await req.formData();
          const formObj: Record<string, any> = {};
          formData.forEach((value, key) => {
            formObj[key] = value;
          });
          requestBody = formObj;
        } else {
          const buffer = await req.arrayBuffer();
          if (buffer.byteLength > 0) {
            requestBody = Array.from(new Uint8Array(buffer));
          }
        }
      } catch (error) {
        console.warn("Failed to parse request body:", error);
      }
    }

    // Execute function
    const result = await executeFunction(func.id, project.id, {
      method,
      url: url.pathname + url.search,
      headers,
      body: requestBody,
    });

    if (!result.success) {
      return new Response(
        JSON.stringify({
          error: {
            message: result.error || "Function execution failed",
            code: "EXECUTION_ERROR",
          },
        }),
        {
          status: 500,
          headers: { "Content-Type": "application/json" },
        },
      );
    }

    // Return the function's response
    if (result.response) {
      const responseHeaders = new Headers();
      Object.entries(result.response.headers || {}).forEach(([key, value]) => {
        responseHeaders.set(key, value);
      });

      // Convert response body to appropriate format
      let responseBody: BodyInit;
      if (typeof result.response.body === "string") {
        responseBody = result.response.body;
      } else if (result.response.body instanceof ArrayBuffer) {
        responseBody = result.response.body;
      } else {
        responseBody = JSON.stringify(result.response.body);
        responseHeaders.set("Content-Type", "application/json");
      }

      return new Response(responseBody, {
        status: result.response.status || 200,
        headers: responseHeaders,
      });
    }

    return new Response(
      JSON.stringify({
        error: {
          message: "Function did not return a response",
          code: "NO_RESPONSE",
        },
      }),
      {
        status: 500,
        headers: { "Content-Type": "application/json" },
      },
    );
  } catch (error: any) {
    logger.error("Subdomain routing error", error);
    return new Response(
      JSON.stringify({
        error: {
          message: error.message || "Internal server error",
          code: "INTERNAL_ERROR",
        },
      }),
      {
        status: 500,
        headers: { "Content-Type": "application/json" },
      },
    );
  }
}

const frontendServer = serve({
  routes: {
    "/*": index,
  },
  port: 3001,
  development: process.env.NODE_ENV !== "production" && {
    // Enable browser hot reloading in development

    // Echo console logs from the browser to the server
    console: true,
  },
});

const server = serve({
  fetch: async (req) => {
    const url = new URL(req.url);
    if (url.hostname.split(".").length > 1) {
      const subdomainResponse = await handleSubdomainRouting(req);
      if (subdomainResponse) {
        return subdomainResponse;
      }
    }
    return app.fetch(req);
  },

  development: process.env.NODE_ENV !== "production" && {
    // Enable browser hot reloading in development

    // Echo console logs from the browser to the server
    console: true,
  },
});

logger.info(`🚀 Server running at ${server.url}`);
logger.info(`🚀 Frontend server running at ${frontendServer.url}`);

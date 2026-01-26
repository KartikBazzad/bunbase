import { Elysia } from "elysia";
import { logger } from "../lib/logger";
import {
  generateCorrelationId,
  logRequest,
  logError,
} from "../lib/logger-utils";
import { logProjectApiCall } from "../lib/project-logger-utils";

/**
 * Logger middleware for ElysiaJS
 * - Generates correlation IDs for requests
 * - Logs incoming requests and responses
 * - Tracks request duration
 * - Integrates with error handler
 */
export const loggerMiddleware = new Elysia({ name: "logger" })
  .derive(({ request, headers }) => {
    // Get or generate correlation ID
    const correlationId =
      headers["x-correlation-id"] ||
      headers["x-request-id"] ||
      generateCorrelationId();

    // Set correlation ID in logger context
    logger.setCorrelationId(correlationId);

    return {
      correlationId,
    };
  })
  .onRequest(({ request, correlationId, set }) => {
    // Log incoming request
    const startTime = Date.now();
    const url = new URL(request.url);
    const method = request.method;
    const path = url.pathname;

    // Store start time and correlation ID in context for response logging
    (request as any).__startTime = startTime;
    (request as any).__correlationId = correlationId;

    // Simple console log for incoming API requests
    const timestamp = new Date().toISOString();
    const queryString = url.search ? `?${url.searchParams.toString()}` : "";
    const userAgent = request.headers.get("user-agent") || "Unknown";
    const ip = request.headers.get("x-forwarded-for") || 
               request.headers.get("x-real-ip") || 
               "Unknown";

    console.log(
      `[${timestamp}] ${method} ${path}${queryString} | IP: ${ip} | ID: ${correlationId}`
    );

    logger.debug("Incoming request", {
      method,
      path,
      query: url.search,
      headers: Object.fromEntries(request.headers.entries()),
      correlationId,
    });
  })
  .onAfterHandle(({ request, response, set, correlationId, apiKey }) => {
    // Log response
    const startTime = (request as any).__startTime;
    const duration = startTime ? Date.now() - startTime : 0;
    const statusCode = set.status || 200;

    const url = new URL(request.url);
    
    // Simple console log for API response
    const timestamp = new Date().toISOString();
    const statusEmoji = statusCode >= 500 ? "❌" : statusCode >= 400 ? "⚠️" : "✅";
    const statusColor = statusCode >= 500 ? "\x1b[31m" : statusCode >= 400 ? "\x1b[33m" : "\x1b[32m";
    const resetColor = "\x1b[0m";
    
    console.log(
      `${statusEmoji} [${timestamp}] ${request.method} ${url.pathname} → ${statusColor}${statusCode}${resetColor} (${duration}ms) | ID: ${correlationId}`
    );

    logRequest(
      request.method,
      url.pathname,
      statusCode,
      duration,
      correlationId,
      {
        query: url.search,
      },
    );

    // Also log to project logger if projectId is available
    // Check apiKey context first (from apiKeyResolver), then path params, then query params
    let projectId: string | undefined;
    if (apiKey?.projectId) {
      projectId = apiKey.projectId;
    } else {
      // Try to extract from path (e.g., /projects/:id/...)
      const pathMatch = url.pathname.match(/\/projects\/([^\/]+)/);
      if (pathMatch) {
        projectId = pathMatch[1];
      } else {
        // Try query params
        projectId = url.searchParams.get("projectId") || undefined;
      }
    }

    if (projectId) {
      logProjectApiCall(projectId, request.method, url.pathname, statusCode, {
        query: url.search,
        duration,
        correlationId,
      });
    }

    // Add correlation ID to response headers
    if (response instanceof Response) {
      response.headers.set("X-Correlation-ID", correlationId);
    }
  })
  .onError(({ error, request, set, correlationId, apiKey }) => {
    // Log errors
    const startTime = (request as any).__startTime;
    const duration = startTime ? Date.now() - startTime : 0;
    const statusCode = set.status || 500;

    const url = new URL(request.url);
    
    // Simple console log for API errors
    const timestamp = new Date().toISOString();
    const errorMessage = error instanceof Error ? error.message : String(error);
    console.error(
      `❌ [${timestamp}] ${request.method} ${url.pathname} → ERROR ${statusCode} (${duration}ms) | ${errorMessage} | ID: ${correlationId}`
    );
    
    logError(error, {
      method: request.method,
      path: url.pathname,
      statusCode,
      duration,
      correlationId,
    });

    // Also log to project logger if projectId is available
    let projectId: string | undefined;
    if (apiKey?.projectId) {
      projectId = apiKey.projectId;
    } else {
      const pathMatch = url.pathname.match(/\/projects\/([^\/]+)/);
      if (pathMatch) {
        projectId = pathMatch[1];
      } else {
        projectId = url.searchParams.get("projectId") || undefined;
      }
    }

    if (projectId) {
      logProjectApiCall(projectId, request.method, url.pathname, statusCode, {
        query: url.search,
        duration,
        correlationId,
        error: error instanceof Error ? error.message : String(error),
      });
    }
  });

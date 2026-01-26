import { logger, Logger, LogLevel } from "./logger";

/**
 * Generate a correlation ID
 */
export function generateCorrelationId(): string {
  return `${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
}

/**
 * Create a logger instance for a specific service
 */
export function createServiceLogger(service: string): Logger {
  return logger.child(service);
}

/**
 * Log HTTP request details
 */
export function logRequest(
  method: string,
  path: string,
  statusCode: number,
  duration: number,
  correlationId?: string,
  context?: Record<string, unknown>,
): void {
  const logLevel: LogLevel =
    statusCode >= 500 ? "error" : statusCode >= 400 ? "warn" : "info";

  logger.info(
    `${method} ${path} ${statusCode} ${duration}ms`,
    {
      method,
      path,
      statusCode,
      duration,
      ...context,
    },
    {
      type: "http_request",
    },
  );
}

/**
 * Log HTTP error
 */
export function logError(
  error: Error | unknown,
  context?: Record<string, unknown>,
): void {
  logger.error("Request error", error, context);
}

/**
 * Log database operation
 */
export function logDatabaseOperation(
  operation: string,
  table?: string,
  duration?: number,
  context?: Record<string, unknown>,
): void {
  logger.debug(
    `Database ${operation}${table ? ` on ${table}` : ""}${duration ? ` (${duration}ms)` : ""}`,
    {
      operation,
      table,
      duration,
      ...context,
    },
    {
      type: "database_operation",
    },
  );
}

/**
 * Log authentication event
 */
export function logAuthEvent(
  event: string,
  userId?: string,
  context?: Record<string, unknown>,
): void {
  logger.info(
    `Auth event: ${event}${userId ? ` (user: ${userId})` : ""}`,
    {
      event,
      userId,
      ...context,
    },
    {
      type: "auth_event",
    },
  );
}

/**
 * Log API key operation
 */
export function logApiKeyOperation(
  operation: string,
  apiKeyId?: string,
  context?: Record<string, unknown>,
): void {
  logger.debug(
    `API key ${operation}${apiKeyId ? ` (key: ${apiKeyId})` : ""}`,
    {
      operation,
      apiKeyId,
      ...context,
    },
    {
      type: "api_key_operation",
    },
  );
}

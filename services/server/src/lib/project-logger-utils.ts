import {
  ProjectLogger,
  type LogLevel,
  type ProjectLogRecord,
} from "./project-logger";

export interface ProjectLogOptions {
  limit?: number;
  offset?: number;
  level?: LogLevel;
  startDate?: Date;
  endDate?: Date;
  type?: string;
  search?: string;
}

/**
 * Get a project logger instance
 */
export function getProjectLogger(projectId: string): ProjectLogger {
  return ProjectLogger.getInstance(projectId);
}

/**
 * Log a generic project operation
 */
export function logProjectOperation(
  projectId: string,
  operation: string,
  context?: Record<string, unknown>,
): void {
  const logger = getProjectLogger(projectId);
  logger.info(
    `Project operation: ${operation}`,
    context,
    undefined,
    "project_operation",
  );
}

/**
 * Log a database operation within a project
 */
export function logProjectDatabaseOperation(
  projectId: string,
  operation: string,
  table?: string,
  context?: Record<string, unknown>,
): void {
  const logger = getProjectLogger(projectId);
  logger.info(
    `Database ${operation}${table ? ` on ${table}` : ""}`,
    {
      operation,
      table,
      ...context,
    },
    undefined,
    "database_operation",
  );
}

/**
 * Log an API call for a project
 */
export function logProjectApiCall(
  projectId: string,
  method: string,
  path: string,
  statusCode: number,
  context?: Record<string, unknown>,
): void {
  const logger = getProjectLogger(projectId);
  const logLevel: LogLevel =
    statusCode >= 500 ? "error" : statusCode >= 400 ? "warn" : "info";

  logger[logLevel](
    `${method} ${path} ${statusCode}`,
    {
      method,
      path,
      statusCode,
      ...context,
    },
    undefined,
    "api_call",
  );
}

/**
 * Log a storage operation for a project
 */
export function logProjectStorageOperation(
  projectId: string,
  operation: string,
  fileId?: string,
  context?: Record<string, unknown>,
): void {
  const logger = getProjectLogger(projectId);
  logger.info(
    `Storage ${operation}${fileId ? ` (file: ${fileId})` : ""}`,
    {
      operation,
      fileId,
      ...context,
    },
    undefined,
    "storage_operation",
  );
}

/**
 * Log an authentication event for a project
 */
export function logProjectAuthEvent(
  projectId: string,
  event: string,
  context?: Record<string, unknown>,
): void {
  const logger = getProjectLogger(projectId);
  logger.info(
    `Auth event: ${event}`,
    {
      event,
      ...context,
    },
    undefined,
    "auth_event",
  );
}

/**
 * Retrieve project logs from the database
 */
export async function getProjectLogs(
  projectId: string,
  options: ProjectLogOptions = {},
): Promise<ProjectLogRecord[]> {
  const logger = getProjectLogger(projectId);
  return logger.getLogs(options);
}

/**
 * Convert project log records to ActivityItem format for dashboard display
 */
export function convertLogsToActivityItems(logs: ProjectLogRecord[]): Array<{
  id: string;
  title: string;
  description?: string;
  timestamp: Date;
  type?: "success" | "warning" | "error" | "info";
}> {
  return logs.map((log) => {
    // Determine activity type based on log level
    let activityType: "success" | "warning" | "error" | "info" = "info";
    if (log.level === "error") {
      activityType = "error";
    } else if (log.level === "warn") {
      activityType = "warning";
    } else if (log.level === "info" && log.type === "project_operation") {
      activityType = "success";
    }

    // Generate title from message
    let title = log.message;
    if (log.type === "database_operation" && log.context?.operation) {
      title = `${log.context.operation} ${log.context.table || "document"}`;
    } else if (log.type === "storage_operation" && log.context?.operation) {
      title = `${log.context.operation} file`;
    } else if (log.type === "auth_event" && log.context?.event) {
      title = `Auth: ${log.context.event}`;
    } else if (
      log.type === "api_call" &&
      log.context?.method &&
      log.context?.path
    ) {
      title = `${log.context.method} ${log.context.path}`;
    }

    // Generate description from context
    let description: string | undefined;
    if (log.context) {
      const contextParts: string[] = [];
      if (log.context.documentId) {
        contextParts.push(`Document: ${log.context.documentId}`);
      }
      if (log.context.collectionId) {
        contextParts.push(`Collection: ${log.context.collectionId}`);
      }
      if (log.context.fileId) {
        contextParts.push(`File: ${log.context.fileId}`);
      }
      if (log.context.statusCode) {
        contextParts.push(`Status: ${log.context.statusCode}`);
      }
      if (contextParts.length > 0) {
        description = contextParts.join(" • ");
      }
    }

    return {
      id: `log-${log.id}`,
      title,
      description,
      timestamp: log.timestamp,
      type: activityType,
    };
  });
}

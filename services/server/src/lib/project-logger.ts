import { Database } from "bun:sqlite";
import { join } from "path";

export type LogLevel = "debug" | "info" | "warn" | "error";

export interface ProjectLogEntry {
  level: LogLevel;
  message: string;
  context?: Record<string, unknown>;
  correlationId?: string;
  service?: string;
  type?: string;
  metadata?: Record<string, unknown>;
  timestamp?: Date;
  projectId: string;
}

export interface ProjectLoggerConfig {
  level?: LogLevel;
  format?: "json" | "plain" | "auto";
  persist?: boolean;
  service?: string;
  dbPath?: string;
}

const LOG_LEVELS: Record<LogLevel, number> = {
  debug: 0,
  info: 1,
  warn: 2,
  error: 3,
};

/**
 * Project-scoped logger that stores logs in separate SQLite files per project
 */
export class ProjectLogger {
  private static instances: Map<string, ProjectLogger> = new Map();
  private db: Database | null = null;
  private config: Required<ProjectLoggerConfig>;
  private correlationId: string | undefined;
  private initialized = false;
  private projectId: string;

  private constructor(projectId: string, config: ProjectLoggerConfig = {}) {
    this.projectId = projectId;
    this.config = {
      level: (process.env.LOG_LEVEL as LogLevel) || "info",
      format: (process.env.LOG_FORMAT as "json" | "plain" | "auto") || "auto",
      persist: process.env.LOG_PERSIST !== "false",
      service: process.env.LOG_SERVICE || "server",
      dbPath: process.env.LOG_DB_PATH || ".logs",
      ...config,
    };
  }

  /**
   * Get or create a logger instance for a specific project
   */
  static getInstance(
    projectId: string,
    config?: ProjectLoggerConfig,
  ): ProjectLogger {
    if (!ProjectLogger.instances.has(projectId)) {
      ProjectLogger.instances.set(
        projectId,
        new ProjectLogger(projectId, config),
      );
    }
    return ProjectLogger.instances.get(projectId)!;
  }

  /**
   * Initialize the logger database connection and schema
   */
  private async initialize(): Promise<void> {
    if (this.initialized) return;

    if (this.config.persist) {
      try {
        // Resolve database path relative to server root
        // import.meta.dir is services/server/src/lib, so go up to services/server
        const serverRoot = join(import.meta.dir, "../../");
        const projectLogsDir = join(
          serverRoot,
          this.config.dbPath,
          "projects",
          this.projectId,
        );

        // Ensure directory exists using Bun
        const dirFile = Bun.file(projectLogsDir);
        if (!(await dirFile.exists())) {
          // Create directory by writing a keep file
          await Bun.write(join(projectLogsDir, ".keep"), "");
        }

        const dbPath = join(projectLogsDir, "logs.db");

        if (process.env.NODE_ENV === "development") {
          console.log(
            `[ProjectLogger] Initializing database for project ${this.projectId}`,
          );
          console.log(`[ProjectLogger] Database path: ${dbPath}`);
        }

        // Initialize Bun.sqlite database
        this.db = new Database(dbPath);

        // Create logs table if not exists
        this.db.exec(`
          CREATE TABLE IF NOT EXISTS logs (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            level TEXT NOT NULL,
            message TEXT NOT NULL,
            context TEXT,
            correlationId TEXT,
            service TEXT,
            type TEXT,
            metadata TEXT,
            timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
            projectId TEXT NOT NULL
          );

          CREATE INDEX IF NOT EXISTS idx_logs_projectId ON logs(projectId);
          CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp);
          CREATE INDEX IF NOT EXISTS idx_logs_level ON logs(level);
          CREATE INDEX IF NOT EXISTS idx_logs_type ON logs(type);
        `);

        this.initialized = true;

        if (process.env.NODE_ENV === "development") {
          console.log(
            `Project logger initialized for project ${this.projectId} at ${dbPath}`,
          );
        }
      } catch (error) {
        // If database initialization fails, log to console and continue
        console.error(
          `Failed to initialize project logger database for project ${this.projectId}:`,
          error,
        );
        if (error instanceof Error) {
          console.error(`Error details: ${error.message}`);
          console.error(`Stack: ${error.stack}`);
        }
        // Don't disable persist - allow retry on next log
        this.initialized = true;
      }
    } else {
      this.initialized = true;
    }
  }

  /**
   * Set correlation ID for the current context
   */
  setCorrelationId(id: string): void {
    this.correlationId = id;
  }

  /**
   * Get current correlation ID
   */
  getCorrelationId(): string | undefined {
    return this.correlationId;
  }

  /**
   * Check if log level should be logged
   */
  private shouldLog(level: LogLevel): boolean {
    return LOG_LEVELS[level] >= LOG_LEVELS[this.config.level];
  }

  /**
   * Get format mode (auto-detect based on NODE_ENV if auto)
   */
  private getFormat(): "json" | "plain" {
    if (this.config.format === "auto") {
      return process.env.NODE_ENV === "production" ? "json" : "plain";
    }
    return this.config.format;
  }

  /**
   * Format log entry for console output
   */
  private formatLog(entry: ProjectLogEntry): string {
    const format = this.getFormat();

    if (format === "json") {
      return JSON.stringify({
        timestamp: entry.timestamp || new Date().toISOString(),
        level: entry.level,
        message: entry.message,
        projectId: entry.projectId,
        ...(entry.context && { context: entry.context }),
        ...(entry.correlationId && { correlationId: entry.correlationId }),
        ...(entry.service && { service: entry.service }),
        ...(entry.type && { type: entry.type }),
        ...(entry.metadata && { metadata: entry.metadata }),
      });
    }

    // Plain format
    const timestamp = (entry.timestamp || new Date()).toISOString();
    const level = entry.level.toUpperCase().padEnd(5);
    const service = entry.service ? `[${entry.service}]` : "";
    const correlation = entry.correlationId ? `[${entry.correlationId}]` : "";
    const project = `[project:${entry.projectId}]`;
    const context = entry.context ? ` ${JSON.stringify(entry.context)}` : "";

    return `${timestamp} ${level} ${service}${correlation}${project} ${entry.message}${context}`;
  }

  /**
   * Write log to database (async, non-blocking)
   */
  private async writeToDatabase(entry: ProjectLogEntry): Promise<void> {
    if (!this.config.persist) return;

    try {
      await this.initialize();

      if (!this.db) {
        // Database initialization failed, skip writing
        if (process.env.NODE_ENV === "development") {
          console.warn(
            `Project logger database not available for project ${this.projectId}`,
          );
        }
        return;
      }

      const stmt = this.db.prepare(`
        INSERT INTO logs (level, message, context, correlationId, service, type, metadata, timestamp, projectId)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
      `);

      const result = stmt.run(
        entry.level,
        entry.message,
        entry.context ? JSON.stringify(entry.context) : null,
        entry.correlationId || null,
        entry.service || this.config.service,
        entry.type || null,
        entry.metadata ? JSON.stringify(entry.metadata) : null,
        entry.timestamp
          ? entry.timestamp.toISOString()
          : new Date().toISOString(),
        entry.projectId,
      );

      if (process.env.NODE_ENV === "development" && result.changes === 0) {
        console.warn(
          `[ProjectLogger] No rows inserted for project ${this.projectId}`,
        );
      }
    } catch (error) {
      // Log errors in development to help debug
      if (process.env.NODE_ENV === "development") {
        console.error(
          `Failed to write log to project database for project ${this.projectId}:`,
          error,
        );
        if (error instanceof Error) {
          console.error(`Error details: ${error.message}`);
          console.error(`Stack: ${error.stack}`);
        }
      }
    }
  }

  /**
   * Internal log method
   */
  private async log(
    level: LogLevel,
    message: string,
    context?: Record<string, unknown>,
    metadata?: Record<string, unknown>,
    type?: string,
  ): Promise<void> {
    if (!this.shouldLog(level)) return;

    const entry: ProjectLogEntry = {
      level,
      message,
      context,
      correlationId: this.correlationId,
      service: this.config.service,
      type,
      metadata,
      timestamp: new Date(),
      projectId: this.projectId,
    };

    // Output to console
    const formatted = this.formatLog(entry);
    const consoleMethod =
      level === "error"
        ? console.error
        : level === "warn"
          ? console.warn
          : level === "debug"
            ? console.debug
            : console.log;
    consoleMethod(formatted);

    // Write to database (async, non-blocking)
    if (this.config.persist) {
      this.writeToDatabase(entry).catch((error) => {
        // Error already handled in writeToDatabase, but log in development for debugging
        if (process.env.NODE_ENV === "development") {
          console.error(
            `[ProjectLogger] Failed to write log for project ${this.projectId}:`,
            error,
          );
        }
      });
    }
  }

  /**
   * Log debug message
   */
  debug(
    message: string,
    context?: Record<string, unknown>,
    metadata?: Record<string, unknown>,
    type?: string,
  ): void {
    this.log("debug", message, context, metadata, type).catch(() => {
      // Error already handled
    });
  }

  /**
   * Log info message
   */
  info(
    message: string,
    context?: Record<string, unknown>,
    metadata?: Record<string, unknown>,
    type?: string,
  ): void {
    this.log("info", message, context, metadata, type).catch(() => {
      // Error already handled
    });
  }

  /**
   * Log warning message
   */
  warn(
    message: string,
    context?: Record<string, unknown>,
    metadata?: Record<string, unknown>,
    type?: string,
  ): void {
    this.log("warn", message, context, metadata, type).catch(() => {
      // Error already handled
    });
  }

  /**
   * Log error message
   */
  error(
    message: string,
    error?: Error | unknown,
    context?: Record<string, unknown>,
    metadata?: Record<string, unknown>,
    type?: string,
  ): void {
    const errorContext: Record<string, unknown> = {
      ...context,
    };

    if (error instanceof Error) {
      errorContext.error = {
        name: error.name,
        message: error.message,
        stack: error.stack,
      };
    } else if (error) {
      errorContext.error = error;
    }

    this.log("error", message, errorContext, metadata, type).catch(() => {
      // Error already handled
    });
  }

  /**
   * Close database connection
   */
  close(): void {
    if (this.db) {
      this.db.close();
      this.db = null;
      this.initialized = false;
    }
  }

  /**
   * Retrieve logs from the database
   */
  async getLogs(
    options: {
      limit?: number;
      offset?: number;
      level?: LogLevel;
      startDate?: Date;
      endDate?: Date;
      type?: string;
      search?: string;
    } = {},
  ): Promise<ProjectLogRecord[]> {
    if (!this.config.persist || !this.db) {
      return [];
    }

    try {
      await this.initialize();

      if (!this.db) {
        return [];
      }

      let query = "SELECT * FROM logs WHERE projectId = ?";
      const params: any[] = [this.projectId];

      // Add filters
      if (options.level) {
        query += " AND level = ?";
        params.push(options.level);
      }

      if (options.type) {
        query += " AND type = ?";
        params.push(options.type);
      }

      if (options.startDate) {
        query += " AND timestamp >= ?";
        params.push(options.startDate.toISOString());
      }

      if (options.endDate) {
        query += " AND timestamp <= ?";
        params.push(options.endDate.toISOString());
      }

      if (options.search) {
        query += " AND message LIKE ?";
        params.push(`%${options.search}%`);
      }

      // Order by timestamp descending (newest first)
      query += " ORDER BY timestamp DESC";

      // Add limit
      if (options.limit) {
        query += " LIMIT ?";
        params.push(options.limit);
      } else {
        // Default limit of 100
        query += " LIMIT ?";
        params.push(100);
      }

      // Add offset for pagination
      if (options.offset) {
        query += " OFFSET ?";
        params.push(options.offset);
      }

      const stmt = this.db.prepare(query);
      const rows = stmt.all(...params) as any[];

      return rows.map((row) => ({
        id: row.id,
        level: row.level as LogLevel,
        message: row.message,
        context: row.context ? JSON.parse(row.context) : undefined,
        correlationId: row.correlationId || undefined,
        service: row.service || undefined,
        type: row.type || undefined,
        metadata: row.metadata ? JSON.parse(row.metadata) : undefined,
        timestamp: new Date(row.timestamp),
        projectId: row.projectId,
      }));
    } catch (error) {
      console.error(
        `Failed to retrieve logs for project ${this.projectId}:`,
        error,
      );
      return [];
    }
  }

  /**
   * Close all project logger instances
   */
  static closeAll(): void {
    for (const logger of ProjectLogger.instances.values()) {
      logger.close();
    }
    ProjectLogger.instances.clear();
  }
}

export interface ProjectLogRecord {
  id: number;
  level: LogLevel;
  message: string;
  context?: Record<string, unknown>;
  correlationId?: string;
  service?: string;
  type?: string;
  metadata?: Record<string, unknown>;
  timestamp: Date;
  projectId: string;
}

import { Database } from "bun:sqlite";
import { join } from "path";
import { mkdir } from "fs/promises";
import { existsSync } from "fs";

export type LogLevel = "debug" | "info" | "warn" | "error";

export interface LogEntry {
  level: LogLevel;
  message: string;
  context?: Record<string, unknown>;
  correlationId?: string;
  service?: string;
  metadata?: Record<string, unknown>;
  timestamp?: Date;
}

export interface LoggerConfig {
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

export class Logger {
  private static instance: Logger;
  private db: Database | null = null;
  private config: Required<LoggerConfig>;
  private correlationId: string | undefined;
  private initialized = false;

  private constructor(config: LoggerConfig = {}) {
    this.config = {
      level: (process.env.LOG_LEVEL as LogLevel) || "info",
      format: (process.env.LOG_FORMAT as "json" | "plain" | "auto") || "auto",
      persist: process.env.LOG_PERSIST !== "false",
      service: process.env.LOG_SERVICE || "server",
      dbPath: process.env.LOG_DB_PATH || ".logs",
      ...config,
    };
  }

  static getInstance(config?: LoggerConfig): Logger {
    if (!Logger.instance) {
      Logger.instance = new Logger(config);
    }
    return Logger.instance;
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
        const dbPath = join(serverRoot, this.config.dbPath);

        // Ensure parent directory exists (in case dbPath includes subdirectories)
        const dbDir = join(dbPath, "..");
        if (!existsSync(dbDir)) {
          await mkdir(dbDir, { recursive: true });
        }

        // Initialize Bun.sqlite database
        this.db = new Database(dbPath);

        // Create logs table if not exists
        this.db.run(`
          CREATE TABLE IF NOT EXISTS logs (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            level TEXT NOT NULL,
            message TEXT NOT NULL,
            context TEXT,
            correlationId TEXT,
            service TEXT,
            timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
            metadata TEXT
          );

          CREATE INDEX IF NOT EXISTS idx_logs_correlationId ON logs(correlationId);
          CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp);
          CREATE INDEX IF NOT EXISTS idx_logs_level ON logs(level);
        `);

        this.initialized = true;
      } catch (error) {
        // If database initialization fails, log to console and continue
        console.error("Failed to initialize logger database:", error);
        this.config.persist = false;
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
   * Create a child logger with service-specific context
   */
  child(service: string): Logger {
    const child = new Logger({
      ...this.config,
      service,
    });
    child.correlationId = this.correlationId;
    child.db = this.db;
    child.initialized = this.initialized;
    return child;
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
  private formatLog(entry: LogEntry): string {
    const format = this.getFormat();

    if (format === "json") {
      return JSON.stringify({
        timestamp: entry.timestamp || new Date().toISOString(),
        level: entry.level,
        message: entry.message,
        ...(entry.context && { context: entry.context }),
        ...(entry.correlationId && { correlationId: entry.correlationId }),
        ...(entry.service && { service: entry.service }),
        ...(entry.metadata && { metadata: entry.metadata }),
      });
    }

    // Plain format
    const timestamp = (entry.timestamp || new Date()).toISOString();
    const level = entry.level.toUpperCase().padEnd(5);
    const service = entry.service ? `[${entry.service}]` : "";
    const correlation = entry.correlationId ? `[${entry.correlationId}]` : "";
    const context = entry.context ? ` ${JSON.stringify(entry.context)}` : "";

    return `${timestamp} ${level} ${service}${correlation} ${entry.message}${context}`;
  }

  /**
   * Write log to database (async, non-blocking)
   */
  private async writeToDatabase(entry: LogEntry): Promise<void> {
    if (!this.config.persist || !this.db) return;

    try {
      await this.initialize();

      if (this.db) {
        const stmt = this.db.prepare(`
          INSERT INTO logs (level, message, context, correlationId, service, timestamp, metadata)
          VALUES (?, ?, ?, ?, ?, ?, ?)
        `);

        stmt.run(
          entry.level,
          entry.message,
          entry.context ? JSON.stringify(entry.context) : null,
          entry.correlationId || null,
          entry.service || this.config.service,
          entry.timestamp?.toISOString() || new Date().toISOString(),
          entry.metadata ? JSON.stringify(entry.metadata) : null,
        );
      }
    } catch (error) {
      // Silently fail - don't break the application if logging fails
      // Only log to console in development
      if (process.env.NODE_ENV === "development") {
        console.error("Failed to write log to database:", error);
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
  ): Promise<void> {
    if (!this.shouldLog(level)) return;

    const entry: LogEntry = {
      level,
      message,
      context,
      correlationId: this.correlationId,
      service: this.config.service,
      metadata,
      timestamp: new Date(),
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
      this.writeToDatabase(entry).catch(() => {
        // Error already handled in writeToDatabase
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
  ): void {
    this.log("debug", message, context, metadata).catch(() => {
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
  ): void {
    this.log("info", message, context, metadata).catch(() => {
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
  ): void {
    this.log("warn", message, context, metadata).catch(() => {
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

    this.log("error", message, errorContext, metadata).catch(() => {
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
}

// Export singleton instance
export const logger = Logger.getInstance();

/**
 * Async Log Buffer
 * Buffers function logs in memory and flushes to storage asynchronously
 */

export interface LogEntry {
  id: string;
  functionId: string;
  executionId: string;
  level: string;
  message: string;
  metadata?: Record<string, any>;
  timestamp: Date;
}

type FlushHandler = (logs: LogEntry[]) => Promise<void>;

/**
 * Async log buffer with batch flushing
 */
export class FunctionLogBuffer {
  private queue: LogEntry[] = [];
  private maxQueueSize: number;
  private flushInterval: number;
  private flushTimer: NodeJS.Timeout | null = null;
  private flushHandler: FlushHandler;
  private flushing: boolean = false;
  private stopped: boolean = false;

  constructor(
    flushHandler: FlushHandler,
    maxQueueSize: number = 10000,
    flushInterval: number = 1000,
  ) {
    this.flushHandler = flushHandler;
    this.maxQueueSize = maxQueueSize;
    this.flushInterval = flushInterval;
  }

  /**
   * Append a log entry to the buffer
   */
  append(log: LogEntry): void {
    if (this.stopped) {
      return;
    }

    this.queue.push(log);

    // Flush if queue is 80% full
    if (this.queue.length >= this.maxQueueSize * 0.8) {
      this.flush().catch((error) => {
        console.error("Error flushing log buffer:", error);
      });
    }
  }

  /**
   * Flush all buffered logs
   */
  async flush(): Promise<void> {
    if (this.flushing || this.queue.length === 0) {
      return;
    }

    this.flushing = true;
    const logsToFlush = this.queue.splice(0);
    this.flushing = false;

    if (logsToFlush.length > 0) {
      try {
        await this.flushHandler(logsToFlush);
      } catch (error) {
        // On error, put logs back (but limit to prevent memory explosion)
        if (this.queue.length < this.maxQueueSize) {
          this.queue.unshift(...logsToFlush);
        }
        // Log error but don't throw (accept log loss in serverless)
        console.error("Failed to flush logs:", error);
      }
    }
  }

  /**
   * Start periodic flushing
   */
  start(): void {
    if (this.flushTimer) {
      return;
    }

    this.stopped = false;
    this.flushTimer = setInterval(() => {
      this.flush().catch((error) => {
        console.error("Error in periodic log flush:", error);
      });
    }, this.flushInterval);
  }

  /**
   * Stop periodic flushing and flush remaining logs
   */
  async stop(): Promise<void> {
    this.stopped = true;

    if (this.flushTimer) {
      clearInterval(this.flushTimer);
      this.flushTimer = null;
    }

    // Flush remaining logs
    await this.flush();
  }

  /**
   * Get current queue size
   */
  getQueueSize(): number {
    return this.queue.length;
  }

  /**
   * Clear all buffered logs (use with caution)
   */
  clear(): void {
    this.queue = [];
  }
}

// Global log buffer instance
let globalLogBuffer: FunctionLogBuffer | null = null;

/**
 * Get the global log buffer
 */
export function getLogBuffer(): FunctionLogBuffer {
  return globalLogBuffer!;
}

/**
 * Initialize the global log buffer
 */
export function initializeLogBuffer(flushHandler: FlushHandler): FunctionLogBuffer {
  if (globalLogBuffer) {
    return globalLogBuffer;
  }

  globalLogBuffer = new FunctionLogBuffer(
    flushHandler,
    parseInt(process.env.FUNCTION_LOG_BUFFER_SIZE || "10000"),
    parseInt(process.env.FUNCTION_LOG_FLUSH_INTERVAL || "1000"),
  );

  globalLogBuffer.start();

  // Flush on process exit
  process.on("SIGTERM", async () => {
    await globalLogBuffer?.stop();
  });

  process.on("SIGINT", async () => {
    await globalLogBuffer?.stop();
  });

  return globalLogBuffer;
}

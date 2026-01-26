/**
 * Concurrency Controller
 * Manages per-function and global concurrency limits
 */

/**
 * Simple semaphore implementation
 */
class Semaphore {
  private count: number;
  private maxCount: number;
  private waitQueue: Array<() => void> = [];

  constructor(maxCount: number) {
    this.maxCount = maxCount;
    this.count = maxCount;
  }

  async acquire(): Promise<boolean> {
    if (this.count > 0) {
      this.count--;
      return true;
    }

    // Wait for release
    return new Promise((resolve) => {
      this.waitQueue.push(() => {
        this.count--;
        resolve(true);
      });
    });
  }

  release(): void {
    if (this.waitQueue.length > 0) {
      const next = this.waitQueue.shift()!;
      next();
    } else {
      this.count++;
      if (this.count > this.maxCount) {
        this.count = this.maxCount;
      }
    }
  }

  getCurrentCount(): number {
    return this.maxCount - this.count;
  }

  getAvailable(): number {
    return this.count;
  }
}

/**
 * Concurrency controller for function execution
 */
export class ConcurrencyController {
  private globalSemaphore: Semaphore;
  private functionSemaphores: Map<string, Semaphore> = new Map();
  private defaultMaxConcurrency: number;

  constructor(
    globalMaxConcurrency: number = 100,
    defaultMaxConcurrency: number = 10,
  ) {
    this.globalSemaphore = new Semaphore(globalMaxConcurrency);
    this.defaultMaxConcurrency = defaultMaxConcurrency;
  }

  /**
   * Acquire a permit for function execution
   * Returns true if acquired, false if limit exceeded
   */
  async acquire(functionId: string, maxConcurrency?: number): Promise<boolean> {
    // Check global limit first
    const globalAcquired = await this.globalSemaphore.acquire();
    if (!globalAcquired) {
      return false;
    }

    // Get or create function semaphore
    let functionSemaphore = this.functionSemaphores.get(functionId);
    if (!functionSemaphore) {
      const limit = maxConcurrency || this.defaultMaxConcurrency;
      functionSemaphore = new Semaphore(limit);
      this.functionSemaphores.set(functionId, functionSemaphore);
    }

    // Acquire function-level permit
    const functionAcquired = await functionSemaphore.acquire();
    if (!functionAcquired) {
      // Release global permit if function limit exceeded
      this.globalSemaphore.release();
      return false;
    }

    return true;
  }

  /**
   * Release permits after function execution
   */
  release(functionId: string): void {
    const functionSemaphore = this.functionSemaphores.get(functionId);
    if (functionSemaphore) {
      functionSemaphore.release();
    }
    this.globalSemaphore.release();
  }

  /**
   * Get current concurrency for a function
   */
  getCurrentConcurrency(functionId: string): number {
    const functionSemaphore = this.functionSemaphores.get(functionId);
    if (!functionSemaphore) {
      return 0;
    }
    return functionSemaphore.getCurrentCount();
  }

  /**
   * Get global concurrency
   */
  getGlobalConcurrency(): number {
    return this.globalSemaphore.getCurrentCount();
  }

  /**
   * Update function concurrency limit
   */
  setFunctionLimit(functionId: string, maxConcurrency: number): void {
    const existing = this.functionSemaphores.get(functionId);
    if (existing) {
      // Update existing semaphore (simple approach: create new one)
      this.functionSemaphores.delete(functionId);
    }
    this.functionSemaphores.set(functionId, new Semaphore(maxConcurrency));
  }

  /**
   * Remove function semaphore (cleanup)
   */
  removeFunction(functionId: string): void {
    this.functionSemaphores.delete(functionId);
  }
}

// Global concurrency controller instance
let globalConcurrencyController: ConcurrencyController | null = null;

/**
 * Get the global concurrency controller
 */
export function getConcurrencyController(): ConcurrencyController {
  if (!globalConcurrencyController) {
    globalConcurrencyController = new ConcurrencyController(
      parseInt(process.env.FUNCTION_MAX_GLOBAL_CONCURRENCY || "100"),
      parseInt(process.env.FUNCTION_DEFAULT_MAX_CONCURRENCY || "10"),
    );
  }
  return globalConcurrencyController;
}

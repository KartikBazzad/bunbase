/**
 * Function Worker Thread Management
 * Manages Bun worker threads for function execution with warm cache
 */

import { join } from "path";
import type { Worker } from "bun";

export interface WorkerTask {
  functionId: string;
  version: string;
  codePath: string;
  request: {
    method: string;
    url: string;
    headers: Record<string, string>;
    body?: any;
  };
  env: Record<string, string>;
  timeout: number;
  memory?: number;
}

export interface WorkerResult {
  success: boolean;
  response?: {
    status: number;
    headers: Record<string, string>;
    body: any;
  };
  error?: string;
  logs: Array<{ level: string; message: string; timestamp: Date }>;
  executionTime: number;
  memoryUsed?: number;
}

/**
 * Cached worker with metadata
 */
interface CachedWorker {
  worker: Bun.Worker;
  functionId: string;
  version: string;
  lastUsed: number;
  inUse: boolean;
}

/**
 * Warm worker cache with LRU eviction
 */
class WarmWorkerCache {
  private cache: Map<string, CachedWorker> = new Map();
  private maxCacheSize: number;
  private idleTimeout: number;
  private evictionInterval: NodeJS.Timeout | null = null;

  constructor(maxCacheSize: number = 50, idleTimeout: number = 60000) {
    this.maxCacheSize = maxCacheSize;
    this.idleTimeout = idleTimeout;
    this.startEvictionTimer();
  }

  /**
   * Get cache key for function+version
   */
  private getCacheKey(functionId: string, version: string): string {
    return `${functionId}:${version}`;
  }

  /**
   * Get a warm worker for function+version
   */
  getWorker(functionId: string, version: string): Bun.Worker | null {
    const key = this.getCacheKey(functionId, version);
    const cached = this.cache.get(key);

    if (cached && !cached.inUse) {
      // Check if worker is still alive and not idle
      const now = Date.now();
      if (now - cached.lastUsed < this.idleTimeout) {
        cached.lastUsed = now;
        cached.inUse = true;
        return cached.worker;
      } else {
        // Worker is idle, remove it
        this.cache.delete(key);
        try {
          cached.worker.terminate();
        } catch {
          // Ignore termination errors
        }
      }
    }

    return null;
  }

  /**
   * Cache a worker for function+version
   */
  cacheWorker(
    functionId: string,
    version: string,
    worker: Bun.Worker,
  ): void {
    const key = this.getCacheKey(functionId, version);
    const now = Date.now();

    // Evict if cache is full (LRU)
    if (this.cache.size >= this.maxCacheSize && !this.cache.has(key)) {
      this.evictLRU();
    }

    // Remove existing entry if present
    const existing = this.cache.get(key);
    if (existing && existing.worker !== worker) {
      try {
        existing.worker.terminate();
      } catch {
        // Ignore termination errors
      }
    }

    this.cache.set(key, {
      worker,
      functionId,
      version,
      lastUsed: now,
      inUse: false,
    });
  }

  /**
   * Mark worker as available (not in use)
   */
  releaseWorker(functionId: string, version: string): void {
    const key = this.getCacheKey(functionId, version);
    const cached = this.cache.get(key);
    if (cached) {
      cached.inUse = false;
      cached.lastUsed = Date.now();
    }
  }

  /**
   * Evict least recently used worker
   */
  private evictLRU(): void {
    let oldestKey: string | null = null;
    let oldestTime = Infinity;

    for (const [key, cached] of this.cache.entries()) {
      if (!cached.inUse && cached.lastUsed < oldestTime) {
        oldestTime = cached.lastUsed;
        oldestKey = key;
      }
    }

    if (oldestKey) {
      const cached = this.cache.get(oldestKey)!;
      this.cache.delete(oldestKey);
      try {
        cached.worker.terminate();
      } catch {
        // Ignore termination errors
      }
    }
  }

  /**
   * Evict idle workers
   */
  evictIdle(): void {
    const now = Date.now();
    const toEvict: string[] = [];

    for (const [key, cached] of this.cache.entries()) {
      if (!cached.inUse && now - cached.lastUsed >= this.idleTimeout) {
        toEvict.push(key);
      }
    }

    for (const key of toEvict) {
      const cached = this.cache.get(key)!;
      this.cache.delete(key);
      try {
        cached.worker.terminate();
      } catch {
        // Ignore termination errors
      }
    }
  }

  /**
   * Start eviction timer
   */
  private startEvictionTimer(): void {
    this.evictionInterval = setInterval(() => {
      this.evictIdle();
    }, 10000); // Check every 10 seconds
  }

  /**
   * Clear all cached workers
   */
  clear(): void {
    if (this.evictionInterval) {
      clearInterval(this.evictionInterval);
      this.evictionInterval = null;
    }

    for (const cached of this.cache.values()) {
      try {
        cached.worker.terminate();
      } catch {
        // Ignore termination errors
      }
    }
    this.cache.clear();
  }
}

/**
 * Worker pool manager with warm cache
 */
export class WorkerPool {
  private workers: Bun.Worker[] = [];
  private availableWorkers: Bun.Worker[] = [];
  private maxWorkers: number;
  private workerScript: string;
  private warmCache: WarmWorkerCache;

  constructor(maxWorkers: number = 10) {
    this.maxWorkers = maxWorkers;
    // Use file path for worker script (Bun can handle TypeScript directly)
    this.workerScript = join(import.meta.dir, "function-worker-script.ts");
    this.warmCache = new WarmWorkerCache(
      parseInt(process.env.FUNCTION_WARM_CACHE_SIZE || "50"),
      parseInt(process.env.FUNCTION_WORKER_IDLE_TIMEOUT || "60000"),
    );
  }

  /**
   * Get a warm worker for function+version, or create a new one
   */
  private async getWorker(
    functionId: string,
    version: string,
  ): Promise<Bun.Worker> {
    // Try warm cache first
    const warmWorker = this.warmCache.getWorker(functionId, version);
    if (warmWorker) {
      return warmWorker as Bun.Worker;
    }

    // Try available workers pool
    if (this.availableWorkers.length > 0) {
      return this.availableWorkers.pop()!;
    }

    // Create new worker if under limit
    if (this.workers.length < this.maxWorkers) {
      const worker = new Bun.Worker(this.workerScript, {
        type: "module",
      });
      this.workers.push(worker as any);
      return worker;
    }

    // Wait for a worker to become available
    return new Promise((resolve) => {
      const checkAvailable = () => {
        // Check warm cache again
        const warm = this.warmCache.getWorker(functionId, version);
        if (warm) {
          resolve(warm);
          return;
        }

        if (this.availableWorkers.length > 0) {
          resolve(this.availableWorkers.pop()!);
        } else {
          setTimeout(checkAvailable, 10);
        }
      };
      checkAvailable();
    });
  }

  /**
   * Return a worker to the pool or warm cache
   */
  private returnWorker(
    worker: Bun.Worker,
    functionId: string,
    version: string,
  ): void {
    // Try to cache in warm cache
    this.warmCache.cacheWorker(functionId, version, worker);
    // Also add to available pool as fallback
    if (!this.availableWorkers.includes(worker)) {
      this.availableWorkers.push(worker);
    }
  }

  /**
   * Execute a function in a worker (with warm cache affinity)
   */
  async execute(task: WorkerTask): Promise<WorkerResult> {
    const worker = await this.getWorker(task.functionId, task.version);
    const startTime = Date.now();
    const isColdStart = !this.warmCache.getWorker(
      task.functionId,
      task.version,
    );

    try {
      const result = await new Promise<WorkerResult>((resolve, reject) => {
        const timeout = setTimeout(() => {
          worker.terminate();
          reject(new Error("Function execution timeout"));
        }, task.timeout * 1000);

        worker.onmessage = (event: MessageEvent) => {
          clearTimeout(timeout);
          this.returnWorker(worker, task.functionId, task.version);
          resolve(event.data as WorkerResult);
        };

        worker.onerror = (error) => {
          clearTimeout(timeout);
          this.returnWorker(worker, task.functionId, task.version);
          reject(error);
        };

        worker.postMessage(task);
      });

      result.executionTime = Date.now() - startTime;
      // Mark as cold start if execution took > 500ms and was first call
      if (isColdStart && result.executionTime > 500) {
        // This will be tracked in metrics
      }
      return result;
    } catch (error: any) {
      this.returnWorker(worker, task.functionId, task.version);
      return {
        success: false,
        error: error.message || "Worker execution failed",
        logs: [],
        executionTime: Date.now() - startTime,
      };
    }
  }

  /**
   * Shutdown all workers
   */
  async shutdown(): Promise<void> {
    this.warmCache.clear();
    for (const worker of this.workers) {
      try {
        worker.terminate();
      } catch {
        // Ignore termination errors
      }
    }
    this.workers = [];
    this.availableWorkers = [];
  }
}

// Global worker pool instance
let globalWorkerPool: WorkerPool | null = null;

/**
 * Get the global worker pool
 */
export function getWorkerPool(): WorkerPool {
  if (!globalWorkerPool) {
    globalWorkerPool = new WorkerPool(
      parseInt(process.env.FUNCTION_WORKER_POOL_SIZE || "10"),
    );
  }
  return globalWorkerPool;
}

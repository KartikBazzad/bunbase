/**
 * Core BunBase Client
 */

import { AuthModule } from "./modules/auth";
import { DatabaseModule } from "./modules/database";
import { StorageModule } from "./modules/storage";
import { RealtimeModule } from "./modules/realtime";

export interface BunBaseConfig {
  apiKey: string;
  baseURL?: string;
  projectId?: string;
  timeout?: number;
  retries?: number;
  retryDelay?: number;
}

export class BunBaseClient {
  public readonly auth: AuthModule;
  public readonly database: DatabaseModule;
  public readonly storage: StorageModule;
  public readonly realtime: RealtimeModule;

  private config: Required<BunBaseConfig>;

  constructor(config: BunBaseConfig) {
    // Validate config
    if (!config.apiKey) {
      throw new Error("API key is required");
    }

    // Set defaults
    this.config = {
      apiKey: config.apiKey,
      baseURL: config.baseURL || "http://localhost:3000/api",
      projectId: config.projectId || "",
      timeout: config.timeout || 30000,
      retries: config.retries || 3,
      retryDelay: config.retryDelay || 1000,
    };

    // Initialize modules (pass client instance for request method)
    this.auth = new AuthModule(this.config, this);
    this.database = new DatabaseModule(this.config, this);
    this.storage = new StorageModule(this.config, this);
    this.realtime = new RealtimeModule(this.config, this);
  }

  /**
   * Make an HTTP request
   */
  async request<T>(
    method: string,
    path: string,
    options?: {
      body?: any;
      headers?: Record<string, string>;
      query?: Record<string, string | number | boolean>;
    },
  ): Promise<T> {
    const url = new URL(path, this.config.baseURL);
    
    // Add query parameters
    if (options?.query) {
      for (const [key, value] of Object.entries(options.query)) {
        url.searchParams.append(key, String(value));
      }
    }

    // Add projectId to query if configured
    if (this.config.projectId) {
      url.searchParams.append("projectId", this.config.projectId);
    }

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      "Authorization": `Bearer ${this.config.apiKey}`,
      ...options?.headers,
    };

    let retries = this.config.retries;
    let lastError: Error | null = null;

    while (retries >= 0) {
      try {
        const controller = new AbortController();
        const timeoutId = setTimeout(
          () => controller.abort(),
          this.config.timeout,
        );

        const response = await fetch(url.toString(), {
          method,
          headers,
          body: options?.body ? JSON.stringify(options.body) : undefined,
          signal: controller.signal,
        });

        clearTimeout(timeoutId);

        if (!response.ok) {
          const error = await response.json().catch(() => ({
            error: { message: response.statusText, code: response.status },
          }));
          throw new Error(error.error?.message || `HTTP ${response.status}`);
        }

        const data = await response.json();
        return data as T;
      } catch (error) {
        lastError = error instanceof Error ? error : new Error(String(error));
        
        if (retries > 0 && this.isRetryableError(lastError)) {
          // Exponential backoff with jitter
          const delay = this.config.retryDelay * Math.pow(2, this.config.retries - retries) + Math.random() * 1000;
          await new Promise((resolve) => setTimeout(resolve, delay));
          retries--;
        } else {
          throw lastError;
        }
      }
    }

    throw lastError || new Error("Request failed");
  }

  private isRetryableError(error: Error): boolean {
    // Retry on network errors or 5xx errors
    return (
      error.name === "AbortError" ||
      error.message.includes("network") ||
      error.message.includes("timeout")
    );
  }
}

/**
 * Create a new BunBase client instance
 */
export function createClient(config: BunBaseConfig): BunBaseClient {
  return new BunBaseClient(config);
}

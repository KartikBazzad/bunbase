/**
 * Core BunBase Client
 */

import { AuthModule } from "./modules/auth";
import { DatabaseModule } from "./modules/database";
import { StorageModule } from "./modules/storage";
import { RealtimeModule } from "./modules/realtime";
import { BunStore } from "./bunstore";

export interface BunBaseConfig {
  apiKey: string;
  baseURL?: string;
  projectId?: string;
  databaseId?: string; // Optional - defaults to the database resolved from API key
  timeout?: number;
  retries?: number;
  retryDelay?: number;
}

export class BunBaseClient {
  public readonly auth: AuthModule;
  public readonly database: DatabaseModule;
  public readonly storage: StorageModule;
  public readonly realtime: RealtimeModule;
  private _bunstore?: BunStore;

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
      databaseId: config.databaseId || "",
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
      useCookies?: boolean; // For Better Auth cookie-based authentication
    },
  ): Promise<T> {
    // Construct URL properly - if path starts with /, it replaces baseURL path
    // So we need to ensure baseURL ends with / or handle path differently
    const baseURL = this.config.baseURL.endsWith("/")
      ? this.config.baseURL.slice(0, -1)
      : this.config.baseURL;
    // Remove leading / from path to append it properly
    const normalizedPath = path.startsWith("/") ? path.slice(1) : path;
    const url = new URL(`${baseURL}/${normalizedPath}`);

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

    // For Better Auth endpoints, use cookies instead of Bearer token
    const isAuthEndpoint = path.startsWith("/auth/");
    const useCookies = options?.useCookies ?? isAuthEndpoint;

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      ...options?.headers,
    };

    // Only add Authorization header if not using cookies (Better Auth uses cookies)
    if (!useCookies) {
      headers.Authorization = `Bearer ${this.config.apiKey}`;
    }

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
          credentials: useCookies ? "include" : "same-origin", // Include cookies for Better Auth
        });

        clearTimeout(timeoutId);

        if (!response.ok) {
          let errorData;
          const contentType = response.headers.get("content-type");
          if (contentType?.includes("application/json")) {
            try {
              errorData = await response.json();
            } catch {
              errorData = {
                error: { message: response.statusText, code: response.status },
              };
            }
          } else {
            const text = await response.text().catch(() => response.statusText);
            errorData = {
              error: {
                message: text || response.statusText,
                code: response.status,
              },
            };
          }

          // Create a more informative error
          const errorMessage =
            errorData.error?.message || `HTTP ${response.status}`;
          const errorCode = errorData.error?.code || `HTTP_${response.status}`;
          const error = new Error(errorMessage);
          (error as any).code = errorCode;
          (error as any).status = response.status;
          (error as any).details = errorData.error;
          throw error;
        }

        const contentType = response.headers.get("content-type");
        if (!contentType?.includes("application/json")) {
          const text = await response.text();
          throw new Error(
            `Expected JSON response but got ${contentType}. Response: ${text.substring(0, 200)}`,
          );
        }

        const data = await response.json();
        return data as T;
      } catch (error) {
        lastError = error instanceof Error ? error : new Error(String(error));

        if (retries > 0 && this.isRetryableError(lastError)) {
          // Exponential backoff with jitter
          const delay =
            this.config.retryDelay *
              Math.pow(2, this.config.retries - retries) +
            Math.random() * 1000;
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

  /**
   * Get BunStore instance
   */
  bunstore(): BunStore {
    if (!this._bunstore) {
      this._bunstore = new BunStore(this, this.config);
    }
    return this._bunstore;
  }
}

/**
 * Create a new BunBase client instance
 */
export function createClient(config: BunBaseConfig): BunBaseClient {
  return new BunBaseClient(config);
}

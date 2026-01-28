/**
 * Core BunBase Server Client
 */

import { FunctionsModule } from "./modules/functions";
import { AdminModule } from "./modules/admin";

export interface ServerSDKConfig {
  apiKey?: string; // For API key auth
  baseURL?: string; // API base URL
  projectId?: string; // Project ID
  useCookies?: boolean; // For session-based auth
  cookieHeader?: string; // Cookie header string for session auth
  timeout?: number;
  retries?: number;
  retryDelay?: number;
}

export class ServerClient {
  public readonly functions: FunctionsModule;
  public readonly admin: AdminModule;

  private config: Required<Omit<ServerSDKConfig, "apiKey" | "useCookies">> & {
    apiKey?: string;
    useCookies: boolean;
  };

  constructor(config: ServerSDKConfig) {
    // Validate config - either apiKey or useCookies must be provided
    if (!config.apiKey && !config.useCookies) {
      throw new Error(
        "Either API key or useCookies must be provided for authentication",
      );
    }

    // Set defaults
    this.config = {
      apiKey: config.apiKey,
      baseURL: config.baseURL || "http://localhost:3000/api",
      projectId: config.projectId || "",
      timeout: config.timeout || 30000,
      retries: config.retries || 3,
      retryDelay: config.retryDelay || 1000,
      useCookies: config.useCookies || false,
    };

    // Initialize modules (pass client instance for request method)
    this.functions = new FunctionsModule(this.config, this);
    this.admin = new AdminModule(this.config, this);
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
      useCookies?: boolean; // Override default cookie usage
    },
  ): Promise<T> {
    // Construct URL properly
    const baseURL = this.config.baseURL.endsWith("/")
      ? this.config.baseURL.slice(0, -1)
      : this.config.baseURL;
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

    // Determine if we should use cookies
    const useCookies = options?.useCookies ?? this.config.useCookies;

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      ...options?.headers,
    };

    // Add cookies if using session-based auth
    if (useCookies && this.config.cookieHeader) {
      headers.Cookie = this.config.cookieHeader;
    }

    // Only add Authorization header if not using cookies
    if (!useCookies && this.config.apiKey) {
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
          credentials: useCookies ? "include" : "same-origin",
        });

        clearTimeout(timeoutId);

        if (!response.ok) {
          let errorData: any;
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
            errorData?.error?.message || `HTTP ${response.status}`;
          const errorCode = errorData?.error?.code || `HTTP_${response.status}`;
          const error = new Error(errorMessage);
          (error as any).code = errorCode;
          (error as any).status = response.status;
          (error as any).details = errorData?.error;
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
}

/**
 * Create a new BunBase Server client instance
 */
export function createServerClient(
  config: ServerSDKConfig,
): ServerClient {
  return new ServerClient(config);
}

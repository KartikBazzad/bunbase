/**
 * Functions Module
 */

import type { ServerSDKConfig, ServerClient } from "../client";
import type {
  HTTPFunctionOptions,
  CallableFunctionOptions,
  FunctionResponse,
  FunctionLog,
  FunctionMetrics,
} from "../types";

export interface FunctionsModuleOptions {
  // Additional function-specific options
}

export class FunctionsModule {
  private client: { request: ServerClient["request"] };

  constructor(
    private config: ServerSDKConfig & { useCookies: boolean },
    client: ServerClient,
    private options?: FunctionsModuleOptions,
  ) {
    this.client = client;
  }

  /**
   * Create an HTTP function
   */
  async createHTTPFunction(
    options: HTTPFunctionOptions,
  ): Promise<FunctionResponse> {
    // Store type and path/methods in metadata for now
    // Server will need to be updated to support these fields
    const response = await this.client.request<FunctionResponse>(
      "POST",
      "/functions",
      {
        body: {
          name: options.name,
          runtime: options.runtime,
          handler: options.handler,
          code: options.code,
          memory: options.memory,
          timeout: options.timeout,
          // Note: type, path, methods will need server-side support
          // For now, store in code or metadata
        },
      },
    );
    // Add type and path info to response
    return {
      ...response,
      type: "http" as const,
      path: options.path,
      methods: options.methods,
    };
  }

  /**
   * Create a callable function
   */
  async createCallableFunction(
    options: CallableFunctionOptions,
  ): Promise<FunctionResponse> {
    const response = await this.client.request<FunctionResponse>(
      "POST",
      "/functions",
      {
        body: {
          name: options.name,
          runtime: options.runtime,
          handler: options.handler,
          code: options.code,
          memory: options.memory,
          timeout: options.timeout,
        },
      },
    );
    // Add type to response
    return {
      ...response,
      type: "callable" as const,
    };
  }

  /**
   * List all functions
   */
  async list(): Promise<FunctionResponse[]> {
    return this.client.request<FunctionResponse[]>("GET", "/functions");
  }

  /**
   * Get function details
   */
  async get(id: string): Promise<FunctionResponse> {
    return this.client.request<FunctionResponse>("GET", `/functions/${id}`);
  }

  /**
   * Update function
   */
  async update(
    id: string,
    options: Partial<HTTPFunctionOptions | CallableFunctionOptions>,
  ): Promise<FunctionResponse> {
    return this.client.request<FunctionResponse>("PUT", `/functions/${id}`, {
      body: options,
    });
  }

  /**
   * Delete function
   */
  async delete(id: string): Promise<void> {
    await this.client.request("DELETE", `/functions/${id}`);
  }

  /**
   * Deploy function
   */
  async deploy(id: string): Promise<{ message: string; version: string }> {
    return this.client.request<{ message: string; version: string }>(
      "POST",
      `/functions/${id}/deploy`,
    );
  }

  /**
   * Invoke function
   */
  async invoke(
    id: string,
    data?: {
      body?: any;
      headers?: Record<string, string>;
    },
  ): Promise<{ result: any; executionTime: number }> {
    return this.client.request<{ result: any; executionTime: number }>(
      "POST",
      `/functions/${id}/invoke`,
      {
        body: data || {},
      },
    );
  }

  /**
   * Get function logs
   */
  async getLogs(
    id: string,
    options?: {
      limit?: number;
      offset?: number;
    },
  ): Promise<{ logs: FunctionLog[]; total: number }> {
    return this.client.request<{ logs: FunctionLog[]; total: number }>(
      "GET",
      `/functions/${id}/logs`,
      {
        query: options,
      },
    );
  }

  /**
   * Get function metrics
   */
  async getMetrics(id: string): Promise<FunctionMetrics> {
    return this.client.request<FunctionMetrics>(
      "GET",
      `/functions/${id}/metrics`,
    );
  }

  /**
   * Set environment variable
   */
  async setEnv(id: string, key: string, value: string): Promise<void> {
    await this.client.request("POST", `/functions/${id}/env`, {
      body: { key, value },
    });
  }

  /**
   * Delete environment variable
   */
  async deleteEnv(id: string, key: string): Promise<void> {
    await this.client.request("DELETE", `/functions/${id}/env/${key}`);
  }
}

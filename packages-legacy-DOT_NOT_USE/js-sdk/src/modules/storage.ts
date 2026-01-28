/**
 * Storage Module
 */

import type { BunBaseConfig, BunBaseClient } from "../client";
import type { StorageFile } from "../types";

export interface StorageModuleOptions {
  // Additional storage-specific options
}

export class StorageModule {
  private client: { request: BunBaseClient["request"] };

  constructor(
    private config: BunBaseConfig,
    client: BunBaseClient,
    private options?: StorageModuleOptions,
  ) {
    this.client = client;
  }

  /**
   * Upload a file
   */
  async upload(
    bucketId: string,
    file: File | Blob,
    options?: {
      path?: string;
      metadata?: Record<string, any>;
    },
  ): Promise<StorageFile> {
    const formData = new FormData();
    formData.append("file", file);
    if (options?.path) {
      formData.append("path", options.path);
    }
    if (options?.metadata) {
      formData.append("metadata", JSON.stringify(options.metadata));
    }

    // Use fetch directly for file uploads
    const baseURL = this.config.baseURL || "http://localhost:3000";
    const url = new URL(
      `/storage/buckets/${bucketId}/upload`,
      baseURL,
    );
    if (this.config.projectId) {
      url.searchParams.append("projectId", this.config.projectId);
    }

    const response = await fetch(url.toString(), {
      method: "POST",
      headers: {
        Authorization: `Bearer ${this.config.apiKey}`,
      },
      body: formData,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({
        error: { message: response.statusText },
      }));
      throw new Error(error.error?.message || `HTTP ${response.status}`);
    }

    return response.json();
  }

  /**
   * Download a file
   */
  async download(bucketId: string, path: string): Promise<Blob> {
    const baseURL = this.config.baseURL || "http://localhost:3000";
    const url = new URL(
      `/storage/buckets/${bucketId}/files/${path}`,
      baseURL,
    );
    if (this.config.projectId) {
      url.searchParams.append("projectId", this.config.projectId);
    }

    const response = await fetch(url.toString(), {
      headers: {
        Authorization: `Bearer ${this.config.apiKey}`,
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to download file: ${response.statusText}`);
    }

    return response.blob();
  }

  /**
   * Delete a file
   */
  async delete(bucketId: string, path: string): Promise<void> {
    await this.client.request(
      "DELETE",
      `/storage/buckets/${bucketId}/files/${path}`,
      {
        query: this.config.projectId ? { projectId: this.config.projectId } : undefined,
      },
    );
  }

  /**
   * List files in a bucket
   */
  async list(
    bucketId: string,
    options?: {
      prefix?: string;
      limit?: number;
      offset?: number;
    },
  ): Promise<{
    files: StorageFile[];
    total: number;
    limit: number;
    offset: number;
  }> {
    return this.client.request(
      "GET",
      `/storage/buckets/${bucketId}/files`,
      {
        query: {
          ...(this.config.projectId ? { projectId: this.config.projectId } : {}),
          ...options,
        },
      },
    );
  }
}

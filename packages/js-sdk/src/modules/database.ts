/**
 * Database Module
 */

import type { BunBaseConfig, BunBaseClient } from "../client";
import type { DatabaseDocument, DatabaseQuery } from "../types";

export interface DatabaseModuleOptions {
  // Additional database-specific options
}

export class DatabaseModule {
  private client: { request: BunBaseClient["request"] };

  constructor(
    private config: BunBaseConfig,
    client: BunBaseClient,
    private options?: DatabaseModuleOptions,
  ) {
    this.client = client;
  }

  /**
   * Create a document
   */
  async create(
    databaseId: string,
    collectionId: string,
    data: Record<string, any>,
  ): Promise<DatabaseDocument> {
    const response = await this.client.request<{ data: DatabaseDocument }>(
      "POST",
      `/databases/${databaseId}/collections/${collectionId}/documents`,
      {
        body: { data },
        query: { projectId: this.config.projectId },
      },
    );
    return response.data;
  }

  /**
   * Get a document by ID
   */
  async get(
    databaseId: string,
    collectionId: string,
    documentId: string,
  ): Promise<DatabaseDocument> {
    const response = await this.client.request<{ data: DatabaseDocument }>(
      "GET",
      `/databases/${databaseId}/collections/${collectionId}/documents/${documentId}`,
      {
        query: { projectId: this.config.projectId },
      },
    );
    return response.data;
  }

  /**
   * Query documents
   */
  async query(
    databaseId: string,
    collectionId: string,
    query: DatabaseQuery = {},
  ): Promise<{
    data: DatabaseDocument[];
    total: number;
    limit: number;
    offset: number;
  }> {
    return this.client.request(
      "GET",
      `/databases/${databaseId}/collections/${collectionId}/documents`,
      {
        query: {
          projectId: this.config.projectId,
          ...query,
        },
      },
    );
  }

  /**
   * Update a document (full replacement)
   */
  async update(
    databaseId: string,
    collectionId: string,
    documentId: string,
    data: Record<string, any>,
  ): Promise<DatabaseDocument> {
    const response = await this.client.request<{ data: DatabaseDocument }>(
      "PUT",
      `/databases/${databaseId}/collections/${collectionId}/documents/${documentId}`,
      {
        body: { data },
        query: { projectId: this.config.projectId },
      },
    );
    return response.data;
  }

  /**
   * Patch a document (partial update)
   */
  async patch(
    databaseId: string,
    collectionId: string,
    documentId: string,
    data: Record<string, any>,
  ): Promise<DatabaseDocument> {
    const response = await this.client.request<{ data: DatabaseDocument }>(
      "PATCH",
      `/databases/${databaseId}/collections/${collectionId}/documents/${documentId}`,
      {
        body: { data },
        query: { projectId: this.config.projectId },
      },
    );
    return response.data;
  }

  /**
   * Delete a document
   */
  async delete(
    databaseId: string,
    collectionId: string,
    documentId: string,
  ): Promise<void> {
    await this.client.request(
      "DELETE",
      `/databases/${databaseId}/collections/${collectionId}/documents/${documentId}`,
      {
        query: { projectId: this.config.projectId },
      },
    );
  }

  /**
   * Upsert a document (create if not exists, update if exists)
   */
  async upsert(
    databaseId: string,
    collectionId: string,
    documentId: string,
    data: Record<string, any>,
  ): Promise<DatabaseDocument> {
    const response = await this.client.request<{ data: DatabaseDocument }>(
      "PUT",
      `/databases/${databaseId}/collections/${collectionId}/documents/${documentId}/upsert`,
      {
        body: { data },
        query: { projectId: this.config.projectId },
      },
    );
    return response.data;
  }

  /**
   * Batch operations
   */
  async batch(
    databaseId: string,
    collectionId: string,
    operations: Array<{
      type: "create" | "update" | "upsert" | "delete";
      documentId?: string;
      data?: Record<string, any>;
    }>,
  ): Promise<{
    results: Array<{
      success: boolean;
      documentId?: string;
      error?: string;
      data?: Record<string, any>;
    }>;
    successCount: number;
    errorCount: number;
  }> {
    return this.client.request(
      "POST",
      `/databases/${databaseId}/collections/${collectionId}/documents/batch`,
      {
        body: { operations },
        query: { projectId: this.config.projectId },
      },
    );
  }
}

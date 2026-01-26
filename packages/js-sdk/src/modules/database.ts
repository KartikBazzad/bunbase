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
   * @param collectionId - Collection name or path
   * @param data - Document data
   * @param databaseId - Optional database ID (uses config default if not provided)
   */
  async create(
    collectionId: string,
    data: Record<string, any>,
    databaseId?: string,
  ): Promise<DatabaseDocument> {
    // Database ID is resolved automatically from API key on the server
    // Only use provided databaseId if explicitly passed (for multi-database scenarios)
    // Use the new Firebase-style API: /api/db/collections/:name/documents
    // Collections are auto-created if they don't exist (Firebase behavior)
    const response = await this.client.request<
      DatabaseDocument | { data: DatabaseDocument }
    >("POST", `/db/${collectionId}`, {
      body: { data },
      // Optionally include databaseId in query if provided
      query: databaseId ? { databaseId } : undefined,
    });
    // Response format: { data: { documentId, collectionId, path, data, createdAt, updatedAt } }
    // Handle both wrapped and unwrapped formats for backward compatibility
    if (
      "data" in response &&
      response.data &&
      typeof response.data === "object" &&
      "documentId" in response.data
    ) {
      return response.data as DatabaseDocument;
    }
    return response as DatabaseDocument;
  }

  /**
   * Get a document by ID
   * @param collectionId - Collection name or path
   * @param documentId - Document ID
   * @param databaseId - Optional database ID (uses config default if not provided)
   */
  async get(
    collectionId: string,
    documentId: string,
    databaseId?: string,
  ): Promise<DatabaseDocument> {
    // Database ID is resolved automatically from API key on the server
    // Only use provided databaseId if explicitly passed (for multi-database scenarios)
    // Use the new Firebase-style API: /api/db/:collection/:id
    const response = await this.client.request<{ data: DatabaseDocument }>(
      "GET",
      `/db/${collectionId}/${documentId}`,
      {
        query: databaseId ? { databaseId } : undefined,
      },
    );
    return response.data;
  }

  /**
   * Query documents
   * @param collectionId - Collection name or path
   * @param query - Query options (filter, sort, limit, offset)
   * @param databaseId - Optional database ID (uses config default if not provided)
   */
  async query(
    collectionId: string,
    query: DatabaseQuery = {},
    databaseId?: string,
  ): Promise<{
    data: DatabaseDocument[];
    total: number;
    limit: number;
    offset: number;
  }> {
    // Database ID is resolved automatically from API key on the server
    // Only use provided databaseId if explicitly passed (for multi-database scenarios)
    const queryParams: Record<string, string | number | boolean> = {};

    if (databaseId) {
      queryParams.databaseId = databaseId;
    }
    if (query.filter) {
      queryParams.filter = JSON.stringify(query.filter);
    }
    if (query.sort) {
      queryParams.sort = JSON.stringify(query.sort);
    }
    if (query.limit !== undefined) {
      queryParams.limit = query.limit;
    }
    if (query.offset !== undefined) {
      queryParams.offset = query.offset;
    }

    // Use the new Firebase-style API: /api/db/:collection
    return this.client.request("GET", `/db/${collectionId}`, {
      query: Object.keys(queryParams).length > 0 ? queryParams : undefined,
    });
  }

  /**
   * Update a document (full replacement)
   * @param collectionId - Collection name or path
   * @param documentId - Document ID
   * @param data - Document data
   * @param databaseId - Optional database ID (uses config default if not provided)
   */
  async update(
    collectionId: string,
    documentId: string,
    data: Record<string, any>,
    databaseId?: string,
  ): Promise<DatabaseDocument> {
    // Database ID is resolved automatically from API key on the server
    // Use the new Firebase-style API: /api/db/:collection/:id
    const response = await this.client.request<{ data: DatabaseDocument }>(
      "PUT",
      `/db/${collectionId}/${documentId}`,
      {
        body: { data },
        query: databaseId ? { databaseId } : undefined,
      },
    );
    return response.data;
  }

  /**
   * Patch a document (partial update)
   * @param collectionId - Collection name or path
   * @param documentId - Document ID
   * @param data - Partial document data
   * @param databaseId - Optional database ID (uses config default if not provided)
   */
  async patch(
    collectionId: string,
    documentId: string,
    data: Record<string, any>,
    databaseId?: string,
  ): Promise<DatabaseDocument> {
    // Database ID is resolved automatically from API key on the server
    // Use the new Firebase-style API: /api/db/:collection/:id
    const response = await this.client.request<{ data: DatabaseDocument }>(
      "PATCH",
      `/db/${collectionId}/${documentId}`,
      {
        body: { data },
        query: databaseId ? { databaseId } : undefined,
      },
    );
    return response.data;
  }

  /**
   * Delete a document
   * @param collectionId - Collection name or path
   * @param documentId - Document ID
   * @param databaseId - Optional database ID (uses config default if not provided)
   */
  async delete(
    collectionId: string,
    documentId: string,
    databaseId?: string,
  ): Promise<void> {
    // Database ID is resolved automatically from API key on the server
    // Use the new Firebase-style API: /api/db/:collection/:id
    await this.client.request("DELETE", `/db/${collectionId}/${documentId}`, {
      query: databaseId ? { databaseId } : undefined,
    });
  }

  /**
   * Upsert a document (create if not exists, update if exists)
   * @param collectionId - Collection name or path
   * @param documentId - Document ID
   * @param data - Document data
   * @param databaseId - Optional database ID (uses config default if not provided)
   */
  async upsert(
    collectionId: string,
    documentId: string,
    data: Record<string, any>,
    databaseId?: string,
  ): Promise<DatabaseDocument> {
    // Database ID is resolved automatically from API key on the server
    // Use the new Firebase-style API: /api/db/:collection/:id
    // Upsert can be done with PUT (create or update)
    const response = await this.client.request<{ data: DatabaseDocument }>(
      "PUT",
      `/db/${collectionId}/${documentId}`,
      {
        body: { data },
        query: databaseId ? { databaseId } : undefined,
      },
    );
    return response.data;
  }

  /**
   * Batch operations
   * @param collectionId - Collection name or path
   * @param operations - Array of batch operations
   * @param databaseId - Optional database ID (uses config default if not provided)
   */
  async batch(
    collectionId: string,
    operations: Array<{
      type: "create" | "update" | "upsert" | "delete";
      documentId?: string;
      data?: Record<string, any>;
    }>,
    databaseId?: string,
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
    // Database ID is resolved automatically from API key on the server
    // Use the new Firebase-style API: /api/db/:collection/batch
    return this.client.request("POST", `/db/${collectionId}/batch`, {
      body: { operations },
      query: databaseId ? { databaseId } : undefined,
    });
  }
}

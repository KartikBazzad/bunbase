/**
 * Admin Module
 */

import type { ServerSDKConfig, ServerClient } from "../client";
import type {
  Project,
  Application,
  Database,
  StorageBucket,
  Collection,
  APIKey,
  FunctionResponse,
} from "../types";

export interface AdminModuleOptions {
  // Additional admin-specific options
}

export class AdminModule {
  private client: { request: ServerClient["request"] };

  constructor(
    private config: ServerSDKConfig & { useCookies: boolean },
    client: ServerClient,
    private options?: AdminModuleOptions,
  ) {
    this.client = client;
  }

  // Projects Management
  projects = {
    /**
     * List all projects
     */
    list: async (): Promise<Project[]> => {
      const response = await this.client.request<{ data: Project[] }>(
        "GET",
        "/projects",
        { useCookies: true },
      );
      return response.data;
    },

    /**
     * Create a project
     */
    create: async (data: {
      name: string;
      description?: string;
    }): Promise<Project> => {
      const response = await this.client.request<{ data: Project }>(
        "POST",
        "/projects",
        {
          body: data,
          useCookies: true,
        },
      );
      return response.data;
    },

    /**
     * Get project by ID
     */
    get: async (id: string): Promise<Project> => {
      const response = await this.client.request<{ data: Project }>(
        "GET",
        `/projects/${id}`,
        { useCookies: true },
      );
      return response.data;
    },

    /**
     * Update project
     */
    update: async (
      id: string,
      data: { name?: string; description?: string },
    ): Promise<Project> => {
      const response = await this.client.request<{ data: Project }>(
        "PATCH",
        `/projects/${id}`,
        {
          body: data,
          useCookies: true,
        },
      );
      return response.data;
    },

    /**
     * Delete project
     */
    delete: async (id: string): Promise<void> => {
      await this.client.request("DELETE", `/projects/${id}`, {
        useCookies: true,
      });
    },

    /**
     * Get project logs
     */
    getLogs: async (
      id: string,
      options?: { limit?: number; offset?: number },
    ): Promise<any> => {
      return this.client.request("GET", `/projects/${id}/logs`, {
        query: options,
        useCookies: true,
      });
    },
  };

  // Applications Management
  applications = {
    /**
     * List applications for a project
     */
    list: async (projectId: string): Promise<Application[]> => {
      const response = await this.client.request<{ data: Application[] }>(
        "GET",
        `/applications/project/${projectId}`,
        { useCookies: true },
      );
      return response.data;
    },

    /**
     * Create an application
     */
    create: async (
      projectId: string,
      data: { name: string; description?: string; type?: "web" },
    ): Promise<Application> => {
      const response = await this.client.request<{ data: Application }>(
        "POST",
        `/applications/project/${projectId}`,
        {
          body: data,
          useCookies: true,
        },
      );
      return response.data;
    },

    /**
     * Get application by ID
     */
    get: async (id: string): Promise<Application> => {
      const response = await this.client.request<{ data: Application }>(
        "GET",
        `/applications/${id}`,
        { useCookies: true },
      );
      return response.data;
    },

    /**
     * Update application
     */
    update: async (
      id: string,
      data: { name?: string; description?: string; type?: "web" },
    ): Promise<Application> => {
      const response = await this.client.request<{ data: Application }>(
        "PATCH",
        `/applications/${id}`,
        {
          body: data,
          useCookies: true,
        },
      );
      return response.data;
    },

    /**
     * Delete application
     */
    delete: async (id: string): Promise<void> => {
      await this.client.request("DELETE", `/applications/${id}`, {
        useCookies: true,
      });
    },

    /**
     * Generate API key for application
     */
    generateKey: async (id: string): Promise<APIKey> => {
      const response = await this.client.request<{ data: APIKey }>(
        "POST",
        `/applications/${id}/keys`,
        { useCookies: true },
      );
      return response.data;
    },

    /**
     * Revoke API key for application
     */
    revokeKey: async (id: string): Promise<void> => {
      await this.client.request("DELETE", `/applications/${id}/keys`, {
        useCookies: true,
      });
    },
  };

  // Databases Management
  databases = {
    /**
     * List databases for a project
     */
    list: async (projectId: string): Promise<Database[]> => {
      const response = await this.client.request<{ data: Database[] }>(
        "GET",
        `/databases/project/${projectId}`,
        { useCookies: true },
      );
      return response.data;
    },

    /**
     * Create a database
     */
    create: async (
      projectId: string,
      data: { name: string },
    ): Promise<Database> => {
      const response = await this.client.request<{ data: Database }>(
        "POST",
        `/databases/project/${projectId}`,
        {
          body: data,
          useCookies: true,
        },
      );
      return response.data;
    },

    /**
     * Get database by ID
     */
    get: async (id: string, projectId: string): Promise<Database> => {
      const response = await this.client.request<{ data: Database }>(
        "GET",
        `/databases/${id}`,
        {
          query: { projectId },
          useCookies: true,
        },
      );
      return response.data;
    },

    /**
     * Delete database
     */
    delete: async (id: string, projectId: string): Promise<void> => {
      await this.client.request("DELETE", `/databases/${id}`, {
        query: { projectId },
        useCookies: true,
      });
    },
  };

  // Storage Management
  storage = {
    buckets: {
      /**
       * List storage buckets for a project
       */
      list: async (projectId: string): Promise<StorageBucket[]> => {
        const response = await this.client.request<StorageBucket[]>(
          "GET",
          `/storage/buckets`,
          {
            query: { projectId },
            useCookies: true,
          },
        );
        return Array.isArray(response) ? response : [];
      },

      /**
       * Create a storage bucket
       */
      create: async (
        projectId: string,
        data: { name: string },
      ): Promise<StorageBucket> => {
        return this.client.request<StorageBucket>("POST", `/storage/buckets`, {
          body: data,
          query: { projectId },
          useCookies: true,
        });
      },

      /**
       * Get bucket by ID
       */
      get: async (id: string): Promise<StorageBucket> => {
        return this.client.request<StorageBucket>(
          "GET",
          `/storage/buckets/${id}`,
          { useCookies: true },
        );
      },

      /**
       * Delete bucket
       */
      delete: async (id: string): Promise<void> => {
        await this.client.request("DELETE", `/storage/buckets/${id}`, {
          useCookies: true,
        });
      },
    },

    files: {
      /**
       * List files in a bucket
       */
      list: async (
        bucketId: string,
        options?: { prefix?: string; limit?: number; offset?: number },
      ): Promise<{ files: any[]; total: number }> => {
        return this.client.request<{ files: any[]; total: number }>(
          "GET",
          `/storage/buckets/${bucketId}/files`,
          {
            query: options,
            useCookies: true,
          },
        );
      },

      /**
       * Delete a file
       */
      delete: async (bucketId: string, path: string): Promise<void> => {
        await this.client.request(
          "DELETE",
          `/storage/buckets/${bucketId}/files/${path}`,
          { useCookies: true },
        );
      },
    },
  };

  // Collections Management
  collections = {
    /**
     * List collections in a database
     */
    list: async (databaseId: string, projectId: string): Promise<Collection[]> => {
      const response = await this.client.request<{ data: Collection[] }>(
        "GET",
        `/databases/${databaseId}/collections`,
        {
          query: { projectId },
          useCookies: true,
        },
      );
      return response.data;
    },

    /**
     * Create a collection
     */
    create: async (
      databaseId: string,
      projectId: string,
      data: { name: string },
    ): Promise<Collection> => {
      const response = await this.client.request<{ data: Collection }>(
        "POST",
        `/databases/${databaseId}/collections`,
        {
          body: data,
          query: { projectId },
          useCookies: true,
        },
      );
      return response.data;
    },

    /**
     * Get collection by ID
     */
    get: async (
      databaseId: string,
      collectionId: string,
      projectId: string,
    ): Promise<Collection> => {
      const response = await this.client.request<{ data: Collection }>(
        "GET",
        `/databases/${databaseId}/collections/${collectionId}`,
        {
          query: { projectId },
          useCookies: true,
        },
      );
      return response.data;
    },

    /**
     * Delete collection
     */
    delete: async (
      databaseId: string,
      collectionId: string,
      projectId: string,
    ): Promise<void> => {
      await this.client.request(
        "DELETE",
        `/databases/${databaseId}/collections/${collectionId}`,
        {
          query: { projectId },
          useCookies: true,
        },
      );
    },
  };

  // Functions Management (admin-level)
  functions = {
    /**
     * List all functions in a project
     */
    list: async (projectId: string): Promise<FunctionResponse[]> => {
      return this.client.request<FunctionResponse[]>("GET", `/functions`, {
        query: { projectId },
        useCookies: true,
      });
    },

    /**
     * Get function by ID
     */
    get: async (id: string): Promise<FunctionResponse> => {
      return this.client.request<FunctionResponse>("GET", `/functions/${id}`, {
        useCookies: true,
      });
    },

    /**
     * Update function
     */
    update: async (
      id: string,
      data: Partial<FunctionResponse>,
    ): Promise<FunctionResponse> => {
      return this.client.request<FunctionResponse>(
        "PUT",
        `/functions/${id}`,
        {
          body: data,
          useCookies: true,
        },
      );
    },

    /**
     * Delete function
     */
    delete: async (id: string): Promise<void> => {
      await this.client.request("DELETE", `/functions/${id}`, {
        useCookies: true,
      });
    },
  };
}

import { apiClient } from "../client";
import { toast } from "sonner";

/**
 * API helper functions with error handling
 */

export async function handleApiCall<T>(
  promise: Promise<{ data?: T; error?: { message: string; code?: string } }>,
  options?: {
    showError?: boolean;
    showSuccess?: boolean;
    successMessage?: string;
  },
): Promise<T | null> {
  try {
    const response = await promise;

    if (response.error) {
      if (options?.showError !== false) {
        toast.error(response.error.message || "An error occurred");
      }
      return null;
    }

    if (response.data) {
      if (options?.showSuccess && options?.successMessage) {
        toast.success(options.successMessage);
      }
      return response.data;
    }

    return null;
  } catch (error) {
    if (options?.showError !== false) {
      toast.error(
        error instanceof Error ? error.message : "An unexpected error occurred",
      );
    }
    return null;
  }
}

// Projects API
export const projectsApi = {
  list: () => apiClient.api.projects.get(),
  get: (id: string) => apiClient.api.projects({ id }).get(),
  create: (data: { name: string; description: string }) =>
    apiClient.api.projects.post(data),
  update: (id: string, data: { name?: string; description?: string }) =>
    apiClient.api.projects({ id }).patch(data),
  delete: (id: string) => apiClient.api.projects({ id }).delete(),
};

// Applications API
export const applicationsApi = {
  list: (projectId: string) =>
    apiClient.api.applications["project"]({ projectId }).get(),
  get: (id: string) => apiClient.api.applications({ id }).get(),
  create: (
    projectId: string,
    data: { name: string; description: string; type?: "web" },
  ) => apiClient.api.applications["project"]({ projectId }).post(data),
  update: (
    id: string,
    data: { name?: string; description?: string; type?: "web" },
  ) => apiClient.api.applications({ id }).patch(data),
  delete: (id: string) => apiClient.api.applications({ id }).delete(),
  // API Key management
  generateKey: (id: string) => apiClient.api.applications({ id }).keys.post(),
  getKey: (id: string) => apiClient.api.applications({ id }).keys.get(),
  revokeKey: (id: string) => apiClient.api.applications({ id }).keys.delete(),
};

// Databases API
export const databasesApi = {
  list: (projectId: string) =>
    apiClient.api.databases["project"]({ projectId }).get(),
  get: (id: string) => apiClient.api.databases({ id }).get(),
  create: (projectId: string, data: { name: string }) =>
    apiClient.api.databases["project"]({ projectId }).post(data),
  delete: (id: string) => apiClient.api.databases({ id }).delete(),
};

// Auth Providers API
export const authProvidersApi = {
  get: (projectId: string) =>
    apiClient.api.auth["project"]({ projectId }).get(),
  update: (projectId: string, data: { providers: string[] }) =>
    apiClient.api.auth["project"]({ projectId }).patch(data),
  toggleProvider: (projectId: string, provider: string) =>
    apiClient.api.auth["project"]({ projectId }).providers({ provider }).post(),
};

// Collections API
export const collectionsApi = {
  list: (databaseId: string, parentPath?: string) =>
    apiClient.api.databases({ id: databaseId }).collections.get({
      query: parentPath ? { parentPath } : undefined,
    }),
  get: (databaseId: string, collectionId: string) =>
    apiClient.api
      .databases({ id: databaseId })
      .collections({ collectionId })
      .get(),
  getByPath: (databaseId: string, path: string) =>
    apiClient.api.databases({ id: databaseId }).collections["by-path"].get({
      query: { path },
    }),
  create: (
    databaseId: string,
    data: {
      name: string;
      parentPath?: string;
      parentDocumentId?: string;
    },
  ) => apiClient.api.databases({ id: databaseId }).collections.post(data),
  update: (databaseId: string, collectionId: string, data: { name?: string }) =>
    apiClient.api
      .databases({ id: databaseId })
      .collections({ collectionId })
      .patch(data),
  delete: (databaseId: string, collectionId: string) =>
    apiClient.api
      .databases({ id: databaseId })
      .collections({ collectionId })
      .delete(),
};

// Documents API
export const documentsApi = {
  list: (
    databaseId: string,
    collectionId: string,
    query?: {
      filter?: Record<string, any>;
      sort?: Record<string, "asc" | "desc">;
      limit?: number;
      offset?: number;
    },
  ) =>
    apiClient.api
      .databases({ id: databaseId })
      .collections({ collectionId })
      .documents.get({
        query,
      }),
  listByPath: (
    databaseId: string,
    collectionPath: string,
    query?: {
      filter?: Record<string, any>;
      sort?: Record<string, "asc" | "desc">;
      limit?: number;
      offset?: number;
    },
  ) =>
    apiClient.api.databases({ id: databaseId }).documents.get({
      query: {
        collectionPath,
        ...query,
      },
    }),
  get: (databaseId: string, collectionId: string, documentId: string) =>
    apiClient.api
      .databases({ id: databaseId })
      .collections({ collectionId })
      .documents({ documentId })
      .get(),
  getByPath: (databaseId: string, path: string) =>
    apiClient.api.databases({ id: databaseId }).documents["by-path"].get({
      query: { path },
    }),
  create: (
    databaseId: string,
    collectionId: string,
    data: { data: Record<string, any> },
  ) =>
    apiClient.api
      .databases({ id: databaseId })
      .collections({ collectionId })
      .documents.post(data),
  update: (
    databaseId: string,
    collectionId: string,
    documentId: string,
    data: { data: Record<string, any> },
  ) =>
    apiClient.api
      .databases({ id: databaseId })
      .collections({ collectionId })
      .documents({ documentId })
      .put(data),
  patch: (
    databaseId: string,
    collectionId: string,
    documentId: string,
    data: { data: Record<string, any> },
  ) =>
    apiClient.api
      .databases({ id: databaseId })
      .collections({ collectionId })
      .documents({ documentId })
      .patch(data),
  delete: (databaseId: string, collectionId: string, documentId: string) =>
    apiClient.api
      .databases({ id: databaseId })
      .collections({ collectionId })
      .documents({ documentId })
      .delete(),
  getSubcollections: (
    databaseId: string,
    collectionId: string,
    documentId: string,
  ) =>
    apiClient.api
      .databases({ id: databaseId })
      .collections({ collectionId })
      .documents({ documentId })
      .subcollections.get(),
};

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
  getLogs: (
    id: string,
    options?: {
      limit?: number;
      offset?: number;
      level?: string;
      type?: string;
      startDate?: string;
      endDate?: string;
      search?: string;
      full?: boolean;
    },
  ) =>
    apiClient.api.projects({ id }).logs.get({
      query: options
        ? {
            limit: options.limit?.toString(),
            offset: options.offset?.toString(),
            level: options.level,
            type: options.type,
            startDate: options.startDate,
            endDate: options.endDate,
            search: options.search,
            full: options.full ? "true" : undefined,
          }
        : undefined,
    }),
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

// Databases API - simplified for single database per project
export const databasesApi = {
  list: (projectId: string) =>
    apiClient.api.databases["project"]({ projectId }).get(),
};

// Auth Providers API
export const authProvidersApi = {
  get: (projectId: string) =>
    apiClient.api.projects({ id: projectId }).authProviders.get(),
  update: (projectId: string, data: { providers: string[] }) =>
    apiClient.api.projects({ id: projectId }).authProviders.patch(data),
  toggleProvider: (projectId: string, provider: string) =>
    apiClient.api
      .projects({ id: projectId })
      .authProviders({ provider })
      .post(),
  // OAuth Configuration
  getOAuthConfig: (projectId: string, provider: string) =>
    apiClient.api
      .projects({ id: projectId })
      .authProviders.oauth({ provider })
      .get(),
  saveOAuthConfig: (
    projectId: string,
    provider: string,
    config: {
      clientId: string;
      clientSecret: string;
      redirectUri?: string;
      scopes?: string[];
    },
  ) =>
    apiClient.api
      .projects({ id: projectId })
      .authProviders.oauth({ provider })
      .post(config),
  testOAuthConnection: (projectId: string, provider: string) =>
    apiClient.api
      .projects({ id: projectId })
      .authProviders.oauth({ provider })
      .test.post(),
  deleteOAuthConfig: (projectId: string, provider: string) =>
    apiClient.api
      .projects({ id: projectId })
      .authProviders.oauth({ provider })
      .delete(),
  // Settings
  getSettings: (projectId: string) =>
    apiClient.api.projects({ id: projectId }).authProviders.settings.get(),
  updateSettings: (
    projectId: string,
    settings: {
      requireEmailVerification?: boolean;
      rateLimitMax?: number;
      rateLimitWindow?: number;
      sessionExpirationDays?: number;
      minPasswordLength?: number;
      requireUppercase?: boolean;
      requireLowercase?: boolean;
      requireNumbers?: boolean;
      requireSpecialChars?: boolean;
      mfaEnabled?: boolean;
      mfaRequired?: boolean;
    },
  ) =>
    apiClient.api
      .projects({ id: projectId })
      .authProviders.settings.patch(settings),
  // Statistics
  getStatistics: (projectId: string) =>
    apiClient.api.projects({ id: projectId }).authProviders.statistics.get(),
};

// Users API
export const usersApi = {
  list: (
    projectId: string,
    filters?: {
      limit?: number;
      offset?: number;
      search?: string;
      emailVerified?: boolean;
    },
  ) =>
    apiClient.api.projects({ id: projectId }).users.get({
      query: filters
        ? {
            limit: filters.limit?.toString(),
            offset: filters.offset?.toString(),
            search: filters.search,
            emailVerified: filters.emailVerified?.toString(),
          }
        : undefined,
    }),
  get: (projectId: string, userId: string) =>
    apiClient.api.projects({ id: projectId }).users({ userId }).get(),
  verifyEmail: (projectId: string, userId: string) =>
    apiClient.api
      .projects({ id: projectId })
      .users({ userId })
      .resendVerification.post(),
  delete: (projectId: string, userId: string) =>
    apiClient.api.projects({ id: projectId }).users({ userId }).delete(),
};

// Collections API
export const collectionsApi = {
  list: (projectId: string, parentPath?: string) =>
    apiClient.api.projects({ id: projectId }).collections.get({
      query: parentPath ? { parentPath } : undefined,
    }),
  get: (projectId: string, collectionId: string) =>
    apiClient.api
      .projects({ id: projectId })
      .collections({ collectionId })
      .get(),
  getByPath: (projectId: string, path: string) =>
    apiClient.api.projects({ id: projectId }).collections["by-path"].get({
      query: { path },
    }),
  create: (
    projectId: string,
    data: {
      name: string;
      parentPath?: string;
      parentDocumentId?: string;
    },
  ) => apiClient.api.projects({ id: projectId }).collections.post(data),
  update: (projectId: string, collectionId: string, data: { name?: string }) =>
    apiClient.api
      .projects({ id: projectId })
      .collections({ collectionId })
      .patch(data),
  delete: (projectId: string, collectionId: string) =>
    apiClient.api
      .projects({ id: projectId })
      .collections({ collectionId })
      .delete(),
};

// Documents API
export const documentsApi = {
  list: (
    projectId: string,
    collectionId: string,
    query?: {
      filter?: Record<string, any>;
      sort?: Record<string, "asc" | "desc">;
      limit?: number;
      offset?: number;
    },
  ) =>
    apiClient.api
      .projects({ id: projectId })
      .collections({ collectionId })
      .documents.get({
        query,
      }),
  listByPath: (
    projectId: string,
    collectionPath: string,
    query?: {
      filter?: Record<string, any>;
      sort?: Record<string, "asc" | "desc">;
      limit?: number;
      offset?: number;
    },
  ) =>
    apiClient.api.projects({ id: projectId }).documents.get({
      query: {
        collectionPath,
        ...query,
      },
    }),
  get: (projectId: string, collectionId: string, documentId: string) =>
    apiClient.api
      .projects({ id: projectId })
      .collections({ collectionId })
      .documents({ documentId })
      .get(),
  getByPath: (projectId: string, path: string) =>
    apiClient.api.projects({ id: projectId }).documents["by-path"].get({
      query: { path },
    }),
  create: (
    projectId: string,
    collectionId: string,
    data: { data: Record<string, any> },
  ) =>
    apiClient.api
      .projects({ id: projectId })
      .collections({ collectionId })
      .documents.post(data),
  update: (
    projectId: string,
    collectionId: string,
    documentId: string,
    data: { data: Record<string, any> },
  ) =>
    apiClient.api
      .projects({ id: projectId })
      .collections({ collectionId })
      .documents({ documentId })
      .put(data),
  patch: (
    projectId: string,
    collectionId: string,
    documentId: string,
    data: { data: Record<string, any> },
  ) =>
    apiClient.api
      .projects({ id: projectId })
      .collections({ collectionId })
      .documents({ documentId })
      .patch(data),
  delete: (projectId: string, collectionId: string, documentId: string) =>
    apiClient.api
      .projects({ id: projectId })
      .collections({ collectionId })
      .documents({ documentId })
      .delete(),
  getSubcollections: (
    projectId: string,
    collectionId: string,
    documentId: string,
  ) =>
    apiClient.api
      .projects({ id: projectId })
      .collections({ collectionId })
      .documents({ documentId })
      .subcollections.get(),
};

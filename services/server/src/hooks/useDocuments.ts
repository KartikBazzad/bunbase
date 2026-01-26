import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { documentsApi, handleApiCall } from "../lib/api";

interface QueryOptions {
  filter?: Record<string, any>;
  sort?: Record<string, "asc" | "desc">;
  limit?: number;
  offset?: number;
}

export function useDocuments(
  projectId: string,
  collectionId: string,
  queryOptions?: QueryOptions
) {
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ["documents", projectId, collectionId, queryOptions],
    queryFn: async () => {
      const result = await documentsApi.list(projectId, collectionId, queryOptions);
      if (result.error) {
        throw new Error(result.error.message);
      }
      return result.data || { data: [], total: 0, limit: 50, offset: 0 };
    },
    enabled: !!projectId && !!collectionId,
  });

  const createMutation = useMutation({
    mutationFn: async (data: { data: Record<string, any> }) => {
      return handleApiCall(documentsApi.create(projectId, collectionId, data), {
        showSuccess: true,
        successMessage: "Document created successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["documents", projectId, collectionId] });
    },
  });

  const updateMutation = useMutation({
    mutationFn: async ({
      documentId,
      data,
    }: {
      documentId: string;
      data: { data: Record<string, any> };
    }) => {
      return handleApiCall(
        documentsApi.update(projectId, collectionId, documentId, data),
        {
          showSuccess: true,
          successMessage: "Document updated successfully",
        }
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["documents", projectId, collectionId] });
    },
  });

  const patchMutation = useMutation({
    mutationFn: async ({
      documentId,
      data,
    }: {
      documentId: string;
      data: { data: Record<string, any> };
    }) => {
      return handleApiCall(documentsApi.patch(projectId, collectionId, documentId, data), {
        showSuccess: true,
        successMessage: "Document updated successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["documents", projectId, collectionId] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (documentId: string) => {
      return handleApiCall(documentsApi.delete(projectId, collectionId, documentId), {
        showSuccess: true,
        successMessage: "Document deleted successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["documents", projectId, collectionId] });
    },
  });

  return {
    documents: data?.data || [],
    total: data?.total || 0,
    limit: data?.limit || 50,
    offset: data?.offset || 0,
    isLoading,
    error,
    createDocument: createMutation.mutateAsync,
    updateDocument: updateMutation.mutateAsync,
    patchDocument: patchMutation.mutateAsync,
    deleteDocument: deleteMutation.mutateAsync,
    isCreating: createMutation.isPending,
    isUpdating: updateMutation.isPending,
    isPatching: patchMutation.isPending,
    isDeleting: deleteMutation.isPending,
  };
}

export function useDocumentByPath(projectId: string, path: string) {
  const { data, isLoading, error } = useQuery({
    queryKey: ["document", projectId, path],
    queryFn: async () => {
      const result = await documentsApi.getByPath(projectId, path);
      if (result.error) {
        throw new Error(result.error.message);
      }
      return result.data?.data;
    },
    enabled: !!projectId && !!path,
  });

  return {
    document: data,
    isLoading,
    error,
  };
}

export function useSubcollections(
  projectId: string,
  collectionId: string,
  documentId: string
) {
  const { data, isLoading, error } = useQuery({
    queryKey: ["subcollections", projectId, collectionId, documentId],
    queryFn: async () => {
      const result = await documentsApi.getSubcollections(projectId, collectionId, documentId);
      if (result.error) {
        throw new Error(result.error.message);
      }
      return result.data?.data || [];
    },
    enabled: !!projectId && !!collectionId && !!documentId,
  });

  return {
    subcollections: data || [],
    isLoading,
    error,
  };
}

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { documentsApi, handleApiCall } from "../lib/api";

interface QueryOptions {
  filter?: Record<string, any>;
  sort?: Record<string, "asc" | "desc">;
  limit?: number;
  offset?: number;
}

export function useDocuments(
  databaseId: string,
  collectionId: string,
  queryOptions?: QueryOptions
) {
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ["documents", databaseId, collectionId, queryOptions],
    queryFn: async () => {
      const result = await documentsApi.list(databaseId, collectionId, queryOptions);
      if (result.error) {
        throw new Error(result.error.message);
      }
      return result.data || { data: [], total: 0, limit: 50, offset: 0 };
    },
    enabled: !!databaseId && !!collectionId,
  });

  const createMutation = useMutation({
    mutationFn: async (data: { data: Record<string, any> }) => {
      return handleApiCall(documentsApi.create(databaseId, collectionId, data), {
        showSuccess: true,
        successMessage: "Document created successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["documents", databaseId, collectionId] });
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
        documentsApi.update(databaseId, collectionId, documentId, data),
        {
          showSuccess: true,
          successMessage: "Document updated successfully",
        }
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["documents", databaseId, collectionId] });
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
      return handleApiCall(documentsApi.patch(databaseId, collectionId, documentId, data), {
        showSuccess: true,
        successMessage: "Document updated successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["documents", databaseId, collectionId] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (documentId: string) => {
      return handleApiCall(documentsApi.delete(databaseId, collectionId, documentId), {
        showSuccess: true,
        successMessage: "Document deleted successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["documents", databaseId, collectionId] });
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

export function useDocumentByPath(databaseId: string, path: string) {
  const { data, isLoading, error } = useQuery({
    queryKey: ["document", databaseId, path],
    queryFn: async () => {
      const result = await documentsApi.getByPath(databaseId, path);
      if (result.error) {
        throw new Error(result.error.message);
      }
      return result.data?.data;
    },
    enabled: !!databaseId && !!path,
  });

  return {
    document: data,
    isLoading,
    error,
  };
}

export function useSubcollections(
  databaseId: string,
  collectionId: string,
  documentId: string
) {
  const { data, isLoading, error } = useQuery({
    queryKey: ["subcollections", databaseId, collectionId, documentId],
    queryFn: async () => {
      const result = await documentsApi.getSubcollections(databaseId, collectionId, documentId);
      if (result.error) {
        throw new Error(result.error.message);
      }
      return result.data?.data || [];
    },
    enabled: !!databaseId && !!collectionId && !!documentId,
  });

  return {
    subcollections: data || [],
    isLoading,
    error,
  };
}

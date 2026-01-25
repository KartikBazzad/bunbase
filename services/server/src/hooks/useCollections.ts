import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { collectionsApi, handleApiCall } from "../lib/api";

export function useCollections(databaseId: string, parentPath?: string) {
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ["collections", databaseId, parentPath],
    queryFn: async () => {
      const result = await collectionsApi.list(databaseId, parentPath);
      if (result.error) {
        throw new Error(result.error.message);
      }
      return result.data?.data || [];
    },
    enabled: !!databaseId,
  });

  const createMutation = useMutation({
    mutationFn: async (data: {
      name: string;
      parentPath?: string;
      parentDocumentId?: string;
    }) => {
      return handleApiCall(collectionsApi.create(databaseId, data), {
        showSuccess: true,
        successMessage: "Collection created successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["collections", databaseId] });
    },
  });

  const updateMutation = useMutation({
    mutationFn: async ({
      collectionId,
      data,
    }: {
      collectionId: string;
      data: { name?: string };
    }) => {
      return handleApiCall(collectionsApi.update(databaseId, collectionId, data), {
        showSuccess: true,
        successMessage: "Collection updated successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["collections", databaseId] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (collectionId: string) => {
      return handleApiCall(collectionsApi.delete(databaseId, collectionId), {
        showSuccess: true,
        successMessage: "Collection deleted successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["collections", databaseId] });
    },
  });

  return {
    collections: data || [],
    isLoading,
    error,
    createCollection: createMutation.mutateAsync,
    updateCollection: updateMutation.mutateAsync,
    deleteCollection: deleteMutation.mutateAsync,
    isCreating: createMutation.isPending,
    isUpdating: updateMutation.isPending,
    isDeleting: deleteMutation.isPending,
  };
}

export function useCollectionByPath(databaseId: string, path: string) {
  const { data, isLoading, error } = useQuery({
    queryKey: ["collection", databaseId, path],
    queryFn: async () => {
      const result = await collectionsApi.getByPath(databaseId, path);
      if (result.error) {
        throw new Error(result.error.message);
      }
      return result.data?.data;
    },
    enabled: !!databaseId && !!path,
  });

  return {
    collection: data,
    isLoading,
    error,
  };
}

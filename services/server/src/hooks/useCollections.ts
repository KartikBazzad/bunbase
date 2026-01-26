import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { collectionsApi, handleApiCall } from "../lib/api";

export function useCollections(projectId: string, parentPath?: string) {
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ["collections", projectId, parentPath],
    queryFn: async () => {
      const result = await collectionsApi.list(projectId, parentPath);
      if (result.error) {
        throw new Error(result.error.message);
      }
      return result.data?.data || [];
    },
    enabled: !!projectId,
  });

  const createMutation = useMutation({
    mutationFn: async (data: {
      name: string;
      parentPath?: string;
      parentDocumentId?: string;
    }) => {
      return handleApiCall(collectionsApi.create(projectId, data), {
        showSuccess: true,
        successMessage: "Collection created successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["collections", projectId] });
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
      return handleApiCall(collectionsApi.update(projectId, collectionId, data), {
        showSuccess: true,
        successMessage: "Collection updated successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["collections", projectId] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (collectionId: string) => {
      return handleApiCall(collectionsApi.delete(projectId, collectionId), {
        showSuccess: true,
        successMessage: "Collection deleted successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["collections", projectId] });
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

export function useCollectionByPath(projectId: string, path: string) {
  const { data, isLoading, error } = useQuery({
    queryKey: ["collection", projectId, path],
    queryFn: async () => {
      const result = await collectionsApi.getByPath(projectId, path);
      if (result.error) {
        throw new Error(result.error.message);
      }
      return result.data?.data;
    },
    enabled: !!projectId && !!path,
  });

  return {
    collection: data,
    isLoading,
    error,
  };
}

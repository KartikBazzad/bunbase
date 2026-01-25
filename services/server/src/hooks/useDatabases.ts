import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { databasesApi, handleApiCall } from "../lib/api";

export function useDatabases(projectId: string) {
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ["databases", projectId],
    queryFn: async () => {
      const result = await databasesApi.list(projectId);
      if (result.error) {
        throw new Error(result.error.message);
      }
      return result.data?.data || [];
    },
    enabled: !!projectId,
  });

  const createMutation = useMutation({
    mutationFn: async (data: { name: string }) => {
      return handleApiCall(databasesApi.create(projectId, data), {
        showSuccess: true,
        successMessage: "Database created successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["databases", projectId] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      return handleApiCall(databasesApi.delete(id), {
        showSuccess: true,
        successMessage: "Database deleted successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["databases", projectId] });
    },
  });

  return {
    databases: data || [],
    isLoading,
    error,
    createDatabase: createMutation.mutateAsync,
    deleteDatabase: deleteMutation.mutateAsync,
    isCreating: createMutation.isPending,
    isDeleting: deleteMutation.isPending,
  };
}

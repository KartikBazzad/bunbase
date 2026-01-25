import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { applicationsApi, handleApiCall } from "../lib/api";

export function useApplications(projectId: string) {
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ["applications", projectId],
    queryFn: async () => {
      const result = await applicationsApi.list(projectId);
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
      description: string;
      type?: "web";
    }) => {
      return handleApiCall(applicationsApi.create(projectId, data), {
        showSuccess: true,
        successMessage: "Application created successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["applications", projectId] });
    },
  });

  const updateMutation = useMutation({
    mutationFn: async ({
      id,
      data,
    }: {
      id: string;
      data: { name?: string; description?: string; type?: "web" };
    }) => {
      return handleApiCall(applicationsApi.update(id, data), {
        showSuccess: true,
        successMessage: "Application updated successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["applications", projectId] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      return handleApiCall(applicationsApi.delete(id), {
        showSuccess: true,
        successMessage: "Application deleted successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["applications", projectId] });
    },
  });

  return {
    applications: data || [],
    isLoading,
    error,
    createApplication: createMutation.mutateAsync,
    updateApplication: updateMutation.mutateAsync,
    deleteApplication: deleteMutation.mutateAsync,
    isCreating: createMutation.isPending,
    isUpdating: updateMutation.isPending,
    isDeleting: deleteMutation.isPending,
  };
}

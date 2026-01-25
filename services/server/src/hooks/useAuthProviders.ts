import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { authProvidersApi, handleApiCall } from "../lib/api";

export function useAuthProviders(projectId: string) {
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ["authProviders", projectId],
    queryFn: async () => {
      const result = await authProvidersApi.get(projectId);
      if (result.error) {
        throw new Error(result.error.message);
      }
      return result.data?.data || null;
    },
    enabled: !!projectId,
  });

  const updateMutation = useMutation({
    mutationFn: async (data: { providers: string[] }) => {
      return handleApiCall(authProvidersApi.update(projectId, data), {
        showSuccess: true,
        successMessage: "Auth providers updated successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["authProviders", projectId],
      });
    },
  });

  const toggleProviderMutation = useMutation({
    mutationFn: async (provider: string) => {
      return handleApiCall(authProvidersApi.toggleProvider(projectId, provider), {
        showSuccess: true,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["authProviders", projectId],
      });
    },
  });

  return {
    authConfig: data,
    isLoading,
    error,
    updateProviders: updateMutation.mutateAsync,
    toggleProvider: toggleProviderMutation.mutateAsync,
    isUpdating: updateMutation.isPending,
    isToggling: toggleProviderMutation.isPending,
  };
}

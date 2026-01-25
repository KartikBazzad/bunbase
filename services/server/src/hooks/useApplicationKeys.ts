import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { applicationsApi, handleApiCall } from "../lib/api";

export function useApplicationKeys(applicationId: string) {
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ["applicationKeys", applicationId],
    queryFn: async () => {
      const result = await applicationsApi.getKey(applicationId);
      if (result.error) {
        throw new Error(result.error.message);
      }
      return result.data?.data || null;
    },
    enabled: !!applicationId,
  });

  const generateMutation = useMutation({
    mutationFn: async () => {
      return handleApiCall(applicationsApi.generateKey(applicationId), {
        showSuccess: true,
        successMessage: "API key generated successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["applicationKeys", applicationId],
      });
    },
  });

  const revokeMutation = useMutation({
    mutationFn: async () => {
      return handleApiCall(applicationsApi.revokeKey(applicationId), {
        showSuccess: true,
        successMessage: "API key revoked successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["applicationKeys", applicationId],
      });
    },
  });

  return {
    key: data,
    isLoading,
    error,
    generateKey: generateMutation.mutateAsync,
    revokeKey: revokeMutation.mutateAsync,
    isGenerating: generateMutation.isPending,
    isRevoking: revokeMutation.isPending,
  };
}

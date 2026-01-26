import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { usersApi, handleApiCall } from "../lib/api";

export function useProjectUsers(
  projectId: string,
  filters?: {
    limit?: number;
    offset?: number;
    search?: string;
    emailVerified?: boolean;
  },
) {
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ["projectUsers", projectId, filters],
    queryFn: async () => {
      const result = await usersApi.list(projectId, filters);
      if (result.error) {
        throw new Error(result.error.message);
      }
      return result.data || null;
    },
    enabled: !!projectId,
  });

  const verifyEmailMutation = useMutation({
    mutationFn: async (userId: string) => {
      return handleApiCall(usersApi.verifyEmail(projectId, userId), {
        showSuccess: true,
        successMessage: "Verification email sent",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["projectUsers", projectId],
      });
    },
  });

  const deleteUserMutation = useMutation({
    mutationFn: async (userId: string) => {
      return handleApiCall(usersApi.delete(projectId, userId), {
        showSuccess: true,
        successMessage: "User deleted successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["projectUsers", projectId],
      });
    },
  });

  return {
    users: data?.data || [],
    total: data?.total || 0,
    limit: data?.limit || 20,
    offset: data?.offset || 0,
    isLoading,
    error,
    verifyUserEmail: verifyEmailMutation.mutateAsync,
    deleteUser: deleteUserMutation.mutateAsync,
    isVerifying: verifyEmailMutation.isPending,
    isDeleting: deleteUserMutation.isPending,
  };
}

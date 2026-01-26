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

  // OAuth Configuration
  const getOAuthConfig = async (provider: string) => {
    const result = await authProvidersApi.getOAuthConfig(projectId, provider);
    if (result.error) {
      throw new Error(result.error.message);
    }
    return result.data?.data || null;
  };

  const saveOAuthConfigMutation = useMutation({
    mutationFn: async (data: {
      provider: string;
      clientId: string;
      clientSecret: string;
      redirectUri?: string;
      scopes?: string[];
    }) => {
      return handleApiCall(
        authProvidersApi.saveOAuthConfig(projectId, data.provider, {
          clientId: data.clientId,
          clientSecret: data.clientSecret,
          redirectUri: data.redirectUri,
          scopes: data.scopes,
        }),
        {
          showSuccess: true,
          successMessage: "OAuth configuration saved successfully",
        },
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["authProviders", projectId],
      });
      queryClient.invalidateQueries({
        queryKey: ["oauthConfig", projectId],
      });
    },
  });

  const testOAuthConnectionMutation = useMutation({
    mutationFn: async (provider: string) => {
      return handleApiCall(
        authProvidersApi.testOAuthConnection(projectId, provider),
        {
          showSuccess: false, // We'll handle the message manually
        },
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["oauthConfig", projectId],
      });
    },
  });

  const deleteOAuthConfigMutation = useMutation({
    mutationFn: async (provider: string) => {
      return handleApiCall(
        authProvidersApi.deleteOAuthConfig(projectId, provider),
        {
          showSuccess: true,
          successMessage: "OAuth configuration removed",
        },
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["oauthConfig", projectId],
      });
    },
  });

  // Settings
  const getAuthSettings = async () => {
    const result = await authProvidersApi.getSettings(projectId);
    if (result.error) {
      throw new Error(result.error.message);
    }
    return result.data?.data || null;
  };

  const updateAuthSettingsMutation = useMutation({
    mutationFn: async (settings: {
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
    }) => {
      return handleApiCall(
        authProvidersApi.updateSettings(projectId, settings),
        {
          showSuccess: true,
          successMessage: "Auth settings updated successfully",
        },
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["authSettings", projectId],
      });
    },
  });

  // Statistics
  const getStatistics = async () => {
    const result = await authProvidersApi.getStatistics(projectId);
    if (result.error) {
      throw new Error(result.error.message);
    }
    return result.data?.data || null;
  };

  return {
    authConfig: data,
    isLoading,
    error,
    updateProviders: updateMutation.mutateAsync,
    toggleProvider: toggleProviderMutation.mutateAsync,
    isUpdating: updateMutation.isPending,
    isToggling: toggleProviderMutation.isPending,
    // OAuth
    getOAuthConfig,
    saveOAuthConfig: saveOAuthConfigMutation.mutateAsync,
    testOAuthConnection: testOAuthConnectionMutation.mutateAsync,
    deleteOAuthConfig: deleteOAuthConfigMutation.mutateAsync,
    isSavingOAuth: saveOAuthConfigMutation.isPending,
    isTestingOAuth: testOAuthConnectionMutation.isPending,
    // Settings
    getAuthSettings,
    updateAuthSettings: updateAuthSettingsMutation.mutateAsync,
    isUpdatingSettings: updateAuthSettingsMutation.isPending,
    // Statistics
    getStatistics,
  };
}

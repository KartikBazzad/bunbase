import { useQuery } from "@tanstack/react-query";
import { databasesApi } from "../lib/api";

// Simplified hook for single database per project
export function useDatabases(projectId: string) {
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

  return {
    databases: data || [],
    isLoading,
    error,
  };
}

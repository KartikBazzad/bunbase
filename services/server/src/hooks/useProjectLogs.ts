import { useQuery } from "@tanstack/react-query";
import { projectsApi } from "../lib/api";

export interface ProjectLogOptions {
  limit?: number;
  offset?: number;
  level?: "debug" | "info" | "warn" | "error";
  type?: string;
  startDate?: Date;
  endDate?: Date;
  search?: string;
  full?: boolean;
}

export interface ProjectLogActivity {
  id: string;
  title: string;
  description?: string;
  timestamp: Date;
  type?: "success" | "warning" | "error" | "info";
}

export interface ProjectLogRecord {
  id: string;
  level: "debug" | "info" | "warn" | "error";
  message: string;
  context?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  correlationId?: string;
  service?: string;
  type?: string;
  timestamp: Date;
  projectId: string;
}

export function useProjectLogs(
  projectId: string,
  options: ProjectLogOptions = {},
) {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["project-logs", projectId, options],
    queryFn: async () => {
      const result = await projectsApi.getLogs(projectId, {
        ...options,
        startDate: options.startDate?.toISOString(),
        endDate: options.endDate?.toISOString(),
      });
      if (result.error) {
        throw new Error(result.error.message);
      }

      // If full logs requested, return ProjectLogRecord[]
      if (options.full) {
        return {
          logs: (result.data?.data || []) as ProjectLogRecord[],
          total: (result.data as any)?.total,
          hasMore: (result.data as any)?.hasMore,
        };
      }

      // Otherwise return activity items for backward compatibility
      return {
        logs: (result.data?.data || []) as ProjectLogActivity[],
      };
    },
    enabled: !!projectId,
    refetchInterval: options.full ? false : 30000, // Only auto-refresh for activity view
  });

  return {
    logs: data?.logs || [],
    total: (data as any)?.total,
    hasMore: (data as any)?.hasMore,
    isLoading,
    error,
    refetch,
  };
}

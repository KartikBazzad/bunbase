import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { projectsApi, handleApiCall } from "../lib/api";
import { toast } from "sonner";

export function useProjects() {
  const queryClient = useQueryClient();

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["projects"],
    queryFn: async () => {
      try {
        const result = await projectsApi.list();
        if (result.error) {
          const errorMessage =
            result.error.message || "Failed to fetch projects";
          toast.error(errorMessage);
          throw new Error(errorMessage);
        }
        return result.data?.data || [];
      } catch (err) {
        // Log the full error for debugging
        console.error("Error fetching projects:", err);
        const errorMessage =
          err instanceof Error ? err.message : "Failed to fetch projects";
        toast.error(errorMessage);
        throw err;
      }
    },
    retry: 1,
  });

  const createMutation = useMutation({
    mutationFn: async (data: { name: string; description: string }) => {
      return handleApiCall(projectsApi.create(data), {
        showSuccess: true,
        successMessage: "Project created successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["projects"] });
    },
  });

  const updateMutation = useMutation({
    mutationFn: async ({
      id,
      data,
    }: {
      id: string;
      data: { name?: string; description?: string };
    }) => {
      return handleApiCall(projectsApi.update(id, data), {
        showSuccess: true,
        successMessage: "Project updated successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["projects"] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      return handleApiCall(projectsApi.delete(id), {
        showSuccess: true,
        successMessage: "Project deleted successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["projects"] });
    },
  });

  return {
    projects: data || [],
    isLoading,
    error,
    refetch,
    createProject: createMutation.mutateAsync,
    updateProject: updateMutation.mutateAsync,
    deleteProject: deleteMutation.mutateAsync,
    isCreating: createMutation.isPending,
    isUpdating: updateMutation.isPending,
    isDeleting: deleteMutation.isPending,
  };
}

export function useProject(id: string) {
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ["project", id],
    queryFn: async () => {
      const result = await projectsApi.get(id);
      if (result.error) {
        throw new Error(result.error.message);
      }
      return result.data?.data || null;
    },
    enabled: !!id,
  });

  const updateMutation = useMutation({
    mutationFn: async (data: { name?: string; description?: string }) => {
      return handleApiCall(projectsApi.update(id, data), {
        showSuccess: true,
        successMessage: "Project updated successfully",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["project", id] });
      queryClient.invalidateQueries({ queryKey: ["projects"] });
    },
  });

  return {
    project: data,
    isLoading,
    error,
    updateProject: updateMutation.mutateAsync,
    isUpdating: updateMutation.isPending,
  };
}

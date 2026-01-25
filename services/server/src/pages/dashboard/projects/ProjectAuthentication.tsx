import { useParams } from "react-router-dom";
import { useProject } from "../../../hooks/useProjects";
import { Skeleton } from "../../../components/ui/skeleton";
import { AuthProviderConfig } from "../../../components/auth-providers/AuthProviderConfig";

export function ProjectAuthentication() {
  const { id } = useParams<{ id: string }>();
  const { project, isLoading } = useProject(id || "");

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-96 w-full" />
      </div>
    );
  }

  if (!project) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-center">
          <h2 className="text-2xl font-bold">Project not found</h2>
          <p className="text-muted-foreground mt-2">
            The project you're looking for doesn't exist or you don't have
            access to it.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">{project.name}</h1>
        <p className="text-muted-foreground mt-1">{project.description}</p>
      </div>

      <div className="space-y-4">
        <AuthProviderConfig projectId={project.id} />
      </div>
    </div>
  );
}

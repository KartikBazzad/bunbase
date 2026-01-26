import { useParams } from "react-router-dom";
import { useProject } from "../../../hooks/useProjects";
import { Skeleton } from "../../../components/ui/skeleton";
import { useNavigate } from "react-router-dom";
import { useEffect } from "react";
import { Database } from "lucide-react";

export function ProjectDatabases() {
  const { id } = useParams<{ id: string }>();
  const { project, isLoading } = useProject(id || "");
  const navigate = useNavigate();

  useEffect(() => {
    // Redirect to explore page since we only have one database per project
    if (project && !isLoading) {
      navigate(`/dashboard/projects/${project.id}/explore`, { replace: true });
    }
  }, [project, isLoading, navigate]);

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Skeleton className="h-10 w-10 rounded-lg" />
          <div className="space-y-2">
            <Skeleton className="h-6 w-48" />
            <Skeleton className="h-4 w-64" />
          </div>
        </div>
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
    <div className="flex items-center justify-center py-12">
      <div className="text-center space-y-4">
        <div className="flex justify-center">
          <div className="p-3 rounded-full bg-muted">
            <Database className="h-6 w-6 text-muted-foreground animate-pulse" />
          </div>
        </div>
        <p className="text-sm text-muted-foreground">Redirecting to database explorer...</p>
      </div>
    </div>
  );
}

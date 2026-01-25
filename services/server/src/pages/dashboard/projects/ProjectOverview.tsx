import { useParams } from "react-router-dom";
import { useProject } from "../../../hooks/useProjects";
import { Skeleton } from "../../../components/ui/skeleton";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../../../components/ui/card";
import { format } from "date-fns";

export function ProjectOverview() {
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
        <div className="grid gap-4 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Project Information</CardTitle>
              <CardDescription>Basic project details</CardDescription>
            </CardHeader>
            <CardContent className="space-y-2">
              <div>
                <span className="text-sm font-medium">Name:</span>
                <p className="text-sm text-muted-foreground">{project.name}</p>
              </div>
              <div>
                <span className="text-sm font-medium">Description:</span>
                <p className="text-sm text-muted-foreground">
                  {project.description}
                </p>
              </div>
              <div>
                <span className="text-sm font-medium">Created:</span>
                <p className="text-sm text-muted-foreground">
                  {format(new Date(project.createdAt), "PPpp")}
                </p>
              </div>
              <div>
                <span className="text-sm font-medium">Last Updated:</span>
                <p className="text-sm text-muted-foreground">
                  {format(new Date(project.updatedAt), "PPpp")}
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}

import { useParams } from "react-router-dom";
import { useProject } from "../../../hooks/useProjects";
import { useProjectLogs } from "../../../hooks/useProjectLogs";
import { Skeleton } from "../../../components/ui/skeleton";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../../../components/ui/card";
import { StatsCard } from "../../../components/ui/stats-card";
import {
  ActivityTimeline,
  type ActivityItem,
} from "../../../components/ui/activity-timeline";
import { format, formatDistanceToNow } from "date-fns";
import { FolderKanban, Layers, Database, Activity, Clock } from "lucide-react";

export function ProjectOverview() {
  const { id } = useParams<{ id: string }>();
  const { project, isLoading } = useProject(id || "");
  const { logs, isLoading: logsLoading } = useProjectLogs(id || "", {
    limit: 10,
  });

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-32 w-full" />
        <div className="grid gap-4 md:grid-cols-3">
          <Skeleton className="h-32" />
          <Skeleton className="h-32" />
          <Skeleton className="h-32" />
        </div>
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

  // Convert logs to ActivityItem format
  // If no logs yet, show project creation as fallback
  const recentActivities: ActivityItem[] =
    logs.length > 0
      ? logs.map((log) => ({
          id: log.id,
          title: log.title,
          description: log.description,
          timestamp: new Date(log.timestamp),
          type: log.type,
        }))
      : [
          {
            id: "project-created",
            title: "Project created",
            description: format(new Date(project.createdAt), "MMMM d, yyyy"),
            icon: FolderKanban,
            timestamp: new Date(project.createdAt),
            type: "success",
          },
        ];

  return (
    <div className="space-y-8">
      <div className="space-y-2">
        <h1 className="text-3xl font-bold tracking-tight">{project.name}</h1>
        <p className="text-muted-foreground">{project.description}</p>
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Clock className="h-4 w-4" />
          <span>
            Last updated{" "}
            {formatDistanceToNow(new Date(project.updatedAt), {
              addSuffix: true,
            })}
          </span>
        </div>
      </div>

      <section className="grid gap-4 md:grid-cols-3">
        <StatsCard
          title="Applications"
          value="0"
          icon={Layers}
          description="Deployed apps"
        />
        <StatsCard
          title="Databases"
          value="0"
          icon={Database}
          description="Data stores"
        />
        <StatsCard
          title="API Calls"
          value="2.4K"
          icon={Activity}
          description="Last 24 hours"
          trend={{ value: 15, isPositive: true }}
        />
      </section>

      <div className="grid gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <Card>
            <CardHeader>
              <CardTitle>Project Details</CardTitle>
              <CardDescription>Basic project information</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-1">
                  <p className="text-sm font-medium">Project ID</p>
                  <p className="text-sm text-muted-foreground font-mono">
                    {project.id}
                  </p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium">Created</p>
                  <p className="text-sm text-muted-foreground">
                    {format(new Date(project.createdAt), "MMMM d, yyyy")}
                  </p>
                </div>
              </div>
              <div className="space-y-1">
                <p className="text-sm font-medium">Description</p>
                <p className="text-sm text-muted-foreground">
                  {project.description}
                </p>
              </div>
            </CardContent>
          </Card>
        </div>

        <div>
          <h3 className="text-xl font-semibold mb-4">Recent Activity</h3>
          {logsLoading ? (
            <Card>
              <CardContent className="p-6">
                <Skeleton className="h-4 w-full mb-2" />
                <Skeleton className="h-4 w-3/4 mb-2" />
                <Skeleton className="h-4 w-1/2" />
              </CardContent>
            </Card>
          ) : (
            <ActivityTimeline activities={recentActivities} />
          )}
        </div>
      </div>
    </div>
  );
}

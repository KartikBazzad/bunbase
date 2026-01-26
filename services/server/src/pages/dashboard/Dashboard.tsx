import { useState } from "react";
import { useAuth } from "../../hooks/useAuth";
import { useProjects } from "../../hooks/useProjects";
import { useApplications } from "../../hooks/useApplications";
import { ProjectList } from "../../components/projects/ProjectList";
import { CreateProjectDialog } from "../../components/projects/CreateProjectDialog";
import { StatsCard } from "../../components/ui/stats-card";
import { ActivityTimeline, type ActivityItem } from "../../components/ui/activity-timeline";
import { Button } from "../../components/ui/button";
import { Plus, FolderKanban, Layers, Activity } from "lucide-react";

export function Dashboard() {
  const { user } = useAuth();
  const { projects, isLoading: projectsLoading, createProject, isCreating } = useProjects();
  const { applications } = useApplications("");
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);

  const getGreeting = () => {
    const hour = new Date().getHours();
    if (hour < 12) return "Good morning";
    if (hour < 18) return "Good afternoon";
    return "Good evening";
  };

  const totalProjects = projects.length;
  const totalApplications = applications.length;

  const recentActivities: ActivityItem[] = [
    {
      id: "1",
      title: "New project created",
      description: projects[0]?.name || "Project",
      icon: FolderKanban,
      timestamp: new Date(Date.now() - 1000 * 60 * 5),
      type: "success",
    },
    {
      id: "2",
      title: "Application deployed",
      description: "Production build successful",
      icon: Layers,
      timestamp: new Date(Date.now() - 1000 * 60 * 30),
      type: "success",
    },
    {
      id: "3",
      title: "Database backup completed",
      description: "Automatic backup",
      icon: Activity,
      timestamp: new Date(Date.now() - 1000 * 60 * 60 * 2),
      type: "info",
    },
  ];

  const handleCreate = async (data: { name: string; description: string }) => {
    await createProject(data);
  };

  return (
    <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
      <section className="animate-in fade-in slide-in-from-bottom-2 duration-500 delay-100">
        <h1 className="text-3xl font-bold tracking-tight">
          {getGreeting()}, {user?.name?.split(" ")[0] || "there"}
        </h1>
        <p className="text-muted-foreground mt-2">
          Here's what's happening with your projects today.
        </p>
      </section>

      <section className="grid gap-4 md:grid-cols-2 lg:grid-cols-4 animate-in fade-in slide-in-from-bottom-2 duration-500 delay-200">
        <StatsCard
          title="Total Projects"
          value={totalProjects}
          icon={FolderKanban}
          description={`${totalProjects === 1 ? "project" : "projects"} created`}
        />
        <StatsCard
          title="Applications"
          value={totalApplications}
          icon={Layers}
          description={`${totalApplications === 1 ? "application" : "applications"} deployed`}
        />
        <StatsCard
          title="Active Deployments"
          value={totalApplications}
          description="Currently running"
          trend={{ value: 12, isPositive: true }}
        />
        <StatsCard
          title="API Requests"
          value="12.4K"
          description="Last 24 hours"
          trend={{ value: 8, isPositive: true }}
        />
      </section>

      <div className="grid gap-6 lg:grid-cols-3 animate-in fade-in slide-in-from-bottom-2 duration-500 delay-300">
        <div className="lg:col-span-2">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-2xl font-bold tracking-tight">Projects</h2>
              <p className="text-muted-foreground">
                Manage your projects and applications
              </p>
            </div>
            <Button size="sm" onClick={() => setIsCreateDialogOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              New Project
            </Button>
          </div>
          <ProjectList />
        </div>

        <div className="space-y-4">
          <div>
            <h2 className="text-xl font-semibold mb-4">Recent Activity</h2>
            <ActivityTimeline activities={recentActivities} />
          </div>
        </div>
      </div>

      <CreateProjectDialog
        open={isCreateDialogOpen}
        onOpenChange={setIsCreateDialogOpen}
        onCreate={handleCreate}
        isCreating={isCreating}
      />
    </div>
  );
}

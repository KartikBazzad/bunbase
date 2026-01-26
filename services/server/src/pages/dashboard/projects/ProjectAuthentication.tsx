import { useState } from "react";
import { useParams } from "react-router-dom";
import { useProject } from "../../../hooks/useProjects";
import { Skeleton } from "../../../components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../../../components/ui/tabs";
import { AuthProviderConfig } from "../../../components/auth-providers/AuthProviderConfig";
import { EmailPasswordSettings } from "../../../components/auth-providers/EmailPasswordSettings";
import { AdvancedAuthSettings } from "../../../components/auth-providers/AdvancedAuthSettings";
import { ProviderStatistics } from "../../../components/auth-providers/ProviderStatistics";
import { ProjectUsersList } from "../../../components/auth-providers/ProjectUsersList";

export function ProjectAuthentication() {
  const { id } = useParams<{ id: string }>();
  const { project, isLoading } = useProject(id || "");
  const [activeTab, setActiveTab] = useState("providers");

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
            The project you're looking for doesn't exist or you don't have access to it.
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

      <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
        <TabsList className="grid w-full grid-cols-5">
          <TabsTrigger value="providers">Providers</TabsTrigger>
          <TabsTrigger value="email-password">Email & Password</TabsTrigger>
          <TabsTrigger value="advanced">Advanced</TabsTrigger>
          <TabsTrigger value="statistics">Statistics</TabsTrigger>
          <TabsTrigger value="users">Users</TabsTrigger>
        </TabsList>

        <TabsContent value="providers" className="space-y-4 mt-6">
          <AuthProviderConfig projectId={project.id} />
        </TabsContent>

        <TabsContent value="email-password" className="space-y-4 mt-6">
          <EmailPasswordSettings projectId={project.id} />
        </TabsContent>

        <TabsContent value="advanced" className="space-y-4 mt-6">
          <AdvancedAuthSettings projectId={project.id} />
        </TabsContent>

        <TabsContent value="statistics" className="space-y-4 mt-6">
          <ProviderStatistics projectId={project.id} />
        </TabsContent>

        <TabsContent value="users" className="space-y-4 mt-6">
          <ProjectUsersList projectId={project.id} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

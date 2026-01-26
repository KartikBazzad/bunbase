import { useProjects } from "../../hooks/useProjects";
import { ProjectCard } from "./ProjectCard";
import { CreateProjectDialog } from "./CreateProjectDialog";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { Plus, Loader2, Search, Rocket } from "lucide-react";
import { useState } from "react";
import { Card, CardContent } from "../ui/card";

export function ProjectList() {
  const {
    projects,
    isLoading,
    error,
    deleteProject,
    isDeleting,
    createProject,
    isCreating,
    refetch,
  } = useProjects();
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");

  const handleCreate = async (data: { name: string; description: string }) => {
    await createProject(data);
  };

  const handleDelete = async (id: string) => {
    await deleteProject(id);
  };

  const filteredProjects = projects.filter(
    (project) =>
      project.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      project.description.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-12 px-6">
          <div className="mb-4 flex size-16 items-center justify-center rounded-full bg-destructive/10">
            <Rocket className="h-8 w-8 text-destructive" />
          </div>
          <div className="text-center">
            <h3 className="text-lg font-semibold">Failed to load projects</h3>
            <p className="text-sm text-muted-foreground mt-2">
              {error instanceof Error
                ? error.message
                : "An error occurred while loading projects"}
            </p>
            <Button size="sm" className="mt-4" onClick={() => refetch()}>
              Try Again
            </Button>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <>
      {projects.length > 0 && (
        <div className="mb-4">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="text"
              placeholder="Search projects..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9"
            />
          </div>
        </div>
      )}

      {filteredProjects.length === 0 ? (
        searchQuery ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-12 px-6">
              <div className="mb-4 flex size-16 items-center justify-center rounded-full bg-muted">
                <Search className="h-8 w-8 text-muted-foreground" />
              </div>
              <div className="text-center">
                <h3 className="text-lg font-semibold">No results found</h3>
                <p className="text-sm text-muted-foreground mt-2">
                  Try adjusting your search terms
                </p>
              </div>
            </CardContent>
          </Card>
        ) : (
          <Card className="border-dashed">
            <CardContent className="flex flex-col items-center justify-center py-12 px-6">
              <div className="mb-4 flex size-16 items-center justify-center rounded-full bg-primary/10">
                <Rocket className="h-8 w-8 text-primary" />
              </div>
              <div className="text-center">
                <h3 className="text-xl font-semibold">No projects yet</h3>
                <p className="text-sm text-muted-foreground mt-2">
                  Create your first project to start building with BunBase.
                  Projects help you organize your applications, databases, and
                  resources.
                </p>
              </div>
              <Button
                size="lg"
                className="mt-6"
                onClick={() => setIsCreateDialogOpen(true)}
              >
                <Plus className="mr-2 h-5 w-5" />
                Create Your First Project
              </Button>
            </CardContent>
          </Card>
        )
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-2">
          {filteredProjects.map((project) => (
            <ProjectCard
              key={project.id}
              project={project}
              onDelete={handleDelete}
              isDeleting={isDeleting}
            />
          ))}
        </div>
      )}

      <CreateProjectDialog
        open={isCreateDialogOpen}
        onOpenChange={setIsCreateDialogOpen}
        onCreate={handleCreate}
        isCreating={isCreating}
      />
    </>
  );
}

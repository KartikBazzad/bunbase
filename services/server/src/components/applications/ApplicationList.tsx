import { useApplications } from "../../hooks/useApplications";
import { CreateApplicationDialog } from "./CreateApplicationDialog";
import { ApplicationConfigDialog } from "./ApplicationConfigDialog";
import { Button } from "../ui/button";
import { Plus, Loader2, Trash2, Key, Rocket } from "lucide-react";
import { useState } from "react";
import { Card, CardContent } from "../ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../ui/table";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "../ui/alert-dialog";

interface ApplicationListProps {
  projectId: string;
}

export function ApplicationList({ projectId }: ApplicationListProps) {
  const {
    applications,
    isLoading,
    createApplication,
    deleteApplication,
    isCreating,
    isDeleting,
  } = useApplications(projectId);
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const [configDialogState, setConfigDialogState] = useState<{
    open: boolean;
    applicationId: string;
    applicationName: string;
  } | null>(null);

  const handleCreate = async (data: {
    name: string;
    description: string;
    type?: "web";
  }) => {
    await createApplication(data);
  };

  const handleDelete = async (id: string) => {
    await deleteApplication(id);
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <div className="space-y-4 animate-in fade-in slide-in-from-bottom-2 duration-300">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Applications</h2>
          <p className="text-muted-foreground">
            Manage applications in this project
          </p>
        </div>
        <Button onClick={() => setIsCreateDialogOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          New Application
        </Button>
      </div>

      {applications.length === 0 ? (
        <Card className="border-dashed">
          <CardContent className="flex flex-col items-center justify-center py-12 px-6">
            <div className="mb-4 flex size-16 items-center justify-center rounded-full bg-primary/10">
              <Rocket className="h-8 w-8 text-primary" />
            </div>
            <div className="text-center">
              <h3 className="text-xl font-semibold">No applications yet</h3>
              <p className="text-sm text-muted-foreground mt-2">
                Create your first application to start deploying your project
              </p>
            </div>
            <Button
              className="mt-6"
              onClick={() => setIsCreateDialogOpen(true)}
            >
              <Plus className="mr-2 h-4 w-4" />
              Create Application
            </Button>
          </CardContent>
        </Card>
      ) : (
        <Card className="overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Description</TableHead>
                <TableHead>Type</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {applications.map((app) => (
                <TableRow key={app.id} className="group hover:bg-muted/50 transition-colors">
                  <TableCell className="font-medium">{app.name}</TableCell>
                  <TableCell className="text-muted-foreground">{app.description}</TableCell>
                  <TableCell>
                    <span className="inline-flex items-center rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary">
                      {app.type}
                    </span>
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity"
                        onClick={() =>
                          setConfigDialogState({
                            open: true,
                            applicationId: app.id,
                            applicationName: app.name,
                          })
                        }
                        title="Configuration"
                      >
                        <Key className="h-4 w-4" />
                      </Button>
                      <AlertDialog>
                        <AlertDialogTrigger asChild>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8 text-destructive opacity-0 group-hover:opacity-100 transition-opacity"
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>Delete application</AlertDialogTitle>
                            <AlertDialogDescription>
                              Are you sure you want to delete "{app.name}"? This action cannot be undone.
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel>Cancel</AlertDialogCancel>
                            <AlertDialogAction
                              onClick={() => handleDelete(app.id)}
                              disabled={isDeleting}
                              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                            >
                              {isDeleting ? "Deleting..." : "Delete"}
                            </AlertDialogAction>
                          </AlertDialogFooter>
                        </AlertDialogContent>
                      </AlertDialog>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      )}

      <CreateApplicationDialog
        open={isCreateDialogOpen}
        onOpenChange={setIsCreateDialogOpen}
        onCreate={handleCreate}
        isCreating={isCreating}
      />

      {configDialogState && (
        <ApplicationConfigDialog
          open={configDialogState.open}
          onOpenChange={(open) => {
            if (!open) {
              setConfigDialogState(null);
            } else {
              setConfigDialogState({
                ...configDialogState,
                open: true,
              });
            }
          }}
          applicationId={configDialogState.applicationId}
          applicationName={configDialogState.applicationName}
        />
      )}
    </div>
  );
}

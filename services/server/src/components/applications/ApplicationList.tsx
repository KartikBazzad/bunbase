import { useApplications } from "../../hooks/useApplications";
import { CreateApplicationDialog } from "./CreateApplicationDialog";
import { ApplicationConfigDialog } from "./ApplicationConfigDialog";
import { Button } from "../ui/button";
import { Plus, Loader2, Trash2, Key } from "lucide-react";
import { useState } from "react";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "../ui/empty";
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
    <div className="space-y-4">
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
        <Empty>
          <EmptyMedia variant="icon">
            <Plus className="h-8 w-8" />
          </EmptyMedia>
          <EmptyHeader>
            <EmptyTitle>No applications yet</EmptyTitle>
            <EmptyDescription>
              Create your first application for this project
            </EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            <Button onClick={() => setIsCreateDialogOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              Create Application
            </Button>
          </EmptyContent>
        </Empty>
      ) : (
        <div className="rounded-md border">
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
                <TableRow key={app.id}>
                  <TableCell className="font-medium">{app.name}</TableCell>
                  <TableCell>{app.description}</TableCell>
                  <TableCell>
                    <span className="inline-flex items-center rounded-full bg-primary/10 px-2 py-1 text-xs font-medium text-primary">
                      {app.type}
                    </span>
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-2">
                      <Button
                        variant="ghost"
                        size="icon"
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
                          <Button variant="ghost" size="icon">
                            <Trash2 className="h-4 w-4 text-destructive" />
                          </Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>Are you sure?</AlertDialogTitle>
                            <AlertDialogDescription>
                              This will permanently delete the application "
                              {app.name}". This action cannot be undone.
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
        </div>
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

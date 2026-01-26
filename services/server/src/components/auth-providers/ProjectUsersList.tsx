import { useState } from "react";
import { useProjectUsers } from "../../hooks/useProjectUsers";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../ui/table";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../ui/card";
import { Input } from "../ui/input";
import { Button } from "../ui/button";
import { Badge } from "../ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "../ui/dropdown-menu";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "../ui/alert-dialog";
import { Loader2, Search, MoreVertical, CheckCircle, Mail, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { format } from "date-fns";

interface ProjectUsersListProps {
  projectId: string;
}

export function ProjectUsersList({ projectId }: ProjectUsersListProps) {
  const [search, setSearch] = useState("");
  const [emailVerified, setEmailVerified] = useState<boolean | undefined>(undefined);
  const [page, setPage] = useState(0);
  const [userToDelete, setUserToDelete] = useState<{ id: string; name: string } | null>(null);

  const limit = 20;
  const {
    users,
    total,
    isLoading,
    verifyUserEmail,
    deleteUser,
    isDeleting,
  } = useProjectUsers(projectId, {
    limit,
    offset: page * limit,
    search: search || undefined,
    emailVerified,
  });

  const handleVerifyEmail = async (userId: string, name: string) => {
    try {
      await verifyUserEmail(userId);
      toast.success(`Verification email sent to ${name}`);
    } catch (error) {
      // Error handled by hook
    }
  };

  const handleDelete = async () => {
    if (!userToDelete) return;
    try {
      await deleteUser(userToDelete.id);
      toast.success(`${userToDelete.name} has been deleted`);
      setUserToDelete(null);
    } catch (error) {
      // Error handled by hook
    }
  };

  const totalPages = Math.ceil((total || 0) / limit);

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Users</CardTitle>
          <CardDescription>Manage project users and their authentication status</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-col sm:flex-row gap-4">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search users by name or email..."
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setPage(0);
                }}
                className="pl-9"
              />
            </div>
            <div className="flex gap-2">
              <Button
                variant={emailVerified === undefined ? "default" : "outline"}
                onClick={() => setEmailVerified(undefined)}
                size="sm"
              >
                All
              </Button>
              <Button
                variant={emailVerified === true ? "default" : "outline"}
                onClick={() => setEmailVerified(true)}
                size="sm"
              >
                Verified
              </Button>
              <Button
                variant={emailVerified === false ? "default" : "outline"}
                onClick={() => setEmailVerified(false)}
                size="sm"
              >
                Unverified
              </Button>
            </div>
          </div>

          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : (
            <>
              <div className="rounded-md border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Email</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {users.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                          No users found
                        </TableCell>
                      </TableRow>
                    ) : (
                      users.map((user) => (
                        <TableRow key={user.id}>
                          <TableCell className="font-medium">{user.name}</TableCell>
                          <TableCell>{user.email}</TableCell>
                          <TableCell>
                            <div className="flex gap-2">
                              {user.emailVerified ? (
                                <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">
                                  <CheckCircle className="h-3 w-3 mr-1" />
                                  Verified
                                </Badge>
                              ) : (
                                <Badge variant="outline">Unverified</Badge>
                              )}
                              {user.provider && (
                                <Badge variant="outline" className="text-xs">
                                  {user.provider}
                                </Badge>
                              )}
                            </div>
                          </TableCell>
                          <TableCell className="text-muted-foreground">
                            {format(new Date(user.createdAt), "MMM d, yyyy")}
                          </TableCell>
                          <TableCell className="text-right">
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button variant="ghost" size="icon">
                                  <MoreVertical className="h-4 w-4" />
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                {!user.emailVerified && (
                                  <DropdownMenuItem
                                    onClick={() => handleVerifyEmail(user.id, user.name)}
                                  >
                                    <Mail className="h-4 w-4 mr-2" />
                                    Send Verification Email
                                  </DropdownMenuItem>
                                )}
                                <DropdownMenuItem
                                  onClick={() => setUserToDelete({ id: user.id, name: user.name })}
                                  className="text-destructive"
                                >
                                  <Trash2 className="h-4 w-4 mr-2" />
                                  Delete User
                                </DropdownMenuItem>
                              </DropdownMenuContent>
                            </DropdownMenu>
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </div>

              {totalPages > 1 && (
                <div className="flex items-center justify-between">
                  <p className="text-sm text-muted-foreground">
                    Showing {page * limit + 1} to {Math.min((page + 1) * limit, total)} of {total}{" "}
                    users
                  </p>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setPage((p) => Math.max(0, p - 1))}
                      disabled={page === 0}
                    >
                      Previous
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
                      disabled={page >= totalPages - 1}
                    >
                      Next
                    </Button>
                  </div>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>

      <AlertDialog open={!!userToDelete} onOpenChange={(open) => !open && setUserToDelete(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete User</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete {userToDelete?.name}? This action cannot be undone and
              will permanently remove the user and all associated data.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              disabled={isDeleting}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {isDeleting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Deleting...
                </>
              ) : (
                "Delete"
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

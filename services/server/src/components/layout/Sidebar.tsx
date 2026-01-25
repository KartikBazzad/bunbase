import { Link, useLocation, useParams, useNavigate } from "react-router-dom";
import {
  LayoutDashboard,
  FolderKanban,
  LogOut,
  Database,
  Shield,
} from "lucide-react";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarTrigger,
  useSidebar,
} from "../ui/sidebar";
import { Avatar, AvatarFallback, AvatarImage } from "../ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../ui/dropdown-menu";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../ui/select";
import { useAuth } from "../../hooks/useAuth";
import { useProjects } from "../../hooks/useProjects";
import { Button } from "../ui/button";
import { Skeleton } from "../ui/skeleton";

const menuItems = [
  {
    title: "Dashboard",
    icon: LayoutDashboard,
    href: "/dashboard",
  },
];

const projectMenuItems = [
  {
    title: "Overview",
    icon: LayoutDashboard,
    path: "",
  },
  {
    title: "Applications",
    icon: FolderKanban,
    path: "/applications",
  },
  {
    title: "Databases",
    icon: Database,
    path: "/databases",
  },
  {
    title: "Authentication",
    icon: Shield,
    path: "/authentication",
  },
];

function SidebarContentComponent() {
  const location = useLocation();
  const params = useParams();
  const navigate = useNavigate();
  const { user, signOut } = useAuth();
  const { projects, isLoading: isLoadingProjects } = useProjects();

  const projectId = params.id;
  const isProjectPage = location.pathname.startsWith("/dashboard/projects/");

  // Determine which project section is active based on pathname
  const getActivePath = () => {
    if (!isProjectPage || !projectId) return "";
    const path = location.pathname.replace(
      `/dashboard/projects/${projectId}`,
      "",
    );
    return path || "";
  };

  const activePath = getActivePath();

  const getInitials = (name?: string) => {
    if (!name) return "U";
    return name
      .split(" ")
      .map((n) => n[0])
      .join("")
      .toUpperCase()
      .slice(0, 2);
  };

  const handleProjectChange = (value: string) => {
    if (value === "all") {
      navigate("/dashboard");
    } else {
      navigate(`/dashboard/projects/${value}`);
    }
  };

  const currentProject = projects.find((p) => p.id === projectId);

  return (
    <Sidebar collapsible="icon">
      <SidebarContent>
        {/* Project Selector */}
        <SidebarGroup>
          <SidebarGroupLabel>Project</SidebarGroupLabel>
          <SidebarGroupContent>
            {isLoadingProjects ? (
              <Skeleton className="h-9 w-full" />
            ) : (
              <Select
                value={isProjectPage && projectId ? projectId : "all"}
                onValueChange={handleProjectChange}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Select a project">
                    {isProjectPage && currentProject
                      ? currentProject.name
                      : "All Projects"}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Projects</SelectItem>
                  {projects.map((project) => (
                    <SelectItem key={project.id} value={project.id}>
                      {project.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </SidebarGroupContent>
        </SidebarGroup>

        {/* Navigation Items */}
        <SidebarGroup>
          <SidebarGroupLabel>Navigation</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {isProjectPage
                ? // Project-specific navigation
                  projectMenuItems.map((item) => {
                    const href = `/dashboard/projects/${projectId}${item.path}`;
                    const isActive = activePath === item.path;
                    return (
                      <SidebarMenuItem key={item.path || "overview"}>
                        <SidebarMenuButton asChild isActive={isActive}>
                          <Link to={href}>
                            <item.icon />
                            <span>{item.title}</span>
                          </Link>
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    );
                  })
                : // Dashboard navigation
                  menuItems.map((item) => (
                    <SidebarMenuItem key={item.href}>
                      <SidebarMenuButton
                        asChild
                        isActive={location.pathname === item.href}
                      >
                        <Link to={item.href}>
                          <item.icon />
                          <span>{item.title}</span>
                        </Link>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              className="w-full justify-start gap-2 h-auto p-2"
            >
              <Avatar className="h-8 w-8">
                <AvatarImage src={user?.image || undefined} alt={user?.name} />
                <AvatarFallback>{getInitials(user?.name)}</AvatarFallback>
              </Avatar>
              <div className="flex flex-col items-start text-left flex-1 min-w-0">
                <span className="text-sm font-medium truncate w-full">
                  {user?.name || "User"}
                </span>
                <span className="text-xs text-muted-foreground truncate w-full">
                  {user?.email}
                </span>
              </div>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56">
            <DropdownMenuLabel>My Account</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={() => signOut()}>
              <LogOut className="mr-2 h-4 w-4" />
              <span>Log out</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarFooter>
    </Sidebar>
  );
}

export function AppSidebar() {
  return <SidebarContentComponent />;
}

export { SidebarTrigger, useSidebar };

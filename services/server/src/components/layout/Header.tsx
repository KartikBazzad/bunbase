import { useLocation, Link, useParams } from "react-router-dom";
import { SidebarTrigger } from "./Sidebar";
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from "../ui/breadcrumb";
import { useProject } from "../../hooks/useProjects";
import { Skeleton } from "../ui/skeleton";
import { ChevronRight } from "lucide-react";

export function Header() {
  const location = useLocation();
  const params = useParams();
  const { project, isLoading: projectLoading } = useProject(params.id || "");
  const pathSegments = location.pathname.split("/").filter(Boolean);

  const getProjectSectionTitle = (path: string) => {
    if (path === "") return "Overview";
    if (path === "/applications") return "Applications";
    if (path === "/databases") return "Databases";
    if (path === "/explore") return "Database";
    if (path === "/authentication") return "Authentication";
    if (path === "/logs") return "Logs";
    return path.replace(/^\//, "");
  };

  const isProjectPage = pathSegments[0] === "dashboard" && pathSegments[1] === "projects" && pathSegments[2];

  return (
    <header className="sticky top-0 z-10 flex h-16 items-center gap-4 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 px-4">
      <SidebarTrigger />
      <Breadcrumb className="flex items-center">
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link to="/dashboard" className="hover:text-primary transition-colors">
                Dashboard
              </Link>
            </BreadcrumbLink>
          </BreadcrumbItem>

          {pathSegments.length > 0 && pathSegments[0] === "dashboard" && (
            <>
              <BreadcrumbSeparator>
                <ChevronRight className="h-4 w-4" />
              </BreadcrumbSeparator>
              <BreadcrumbItem>
                <BreadcrumbLink asChild>
                  <Link to="/dashboard">Projects</Link>
                </BreadcrumbLink>
              </BreadcrumbItem>
            </>
          )}

          {isProjectPage && (
            <>
              <BreadcrumbSeparator>
                <ChevronRight className="h-4 w-4" />
              </BreadcrumbSeparator>
              <BreadcrumbItem>
                {projectLoading ? (
                  <Skeleton className="h-4 w-24" />
                ) : project ? (
                  <BreadcrumbLink asChild>
                    <Link to={`/dashboard/projects/${project.id}`}>
                      {project.name}
                    </Link>
                  </BreadcrumbLink>
                ) : (
                  <BreadcrumbPage>Project</BreadcrumbPage>
                )}
              </BreadcrumbItem>
              {pathSegments[3] && (
                <>
                  <BreadcrumbSeparator>
                    <ChevronRight className="h-4 w-4" />
                  </BreadcrumbSeparator>
                  <BreadcrumbItem>
                    <BreadcrumbPage>
                      {getProjectSectionTitle(`/${pathSegments[3]}`)}
                    </BreadcrumbPage>
                  </BreadcrumbItem>
                </>
              )}
            </>
          )}
        </BreadcrumbList>
      </Breadcrumb>

      {isProjectPage && project && (
        <div className="ml-auto flex items-center gap-2">
          <div className="h-2 w-2 rounded-full bg-green-500" />
          <span className="text-xs text-muted-foreground">Active</span>
        </div>
      )}
    </header>
  );
}

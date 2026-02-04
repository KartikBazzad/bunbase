import { ReactNode, useEffect, useState } from "react";
import { Link, Outlet, useLocation, useParams } from "react-router-dom";
import { Menu, PanelLeftClose } from "lucide-react";
import { Sidebar } from "./Sidebar";
import { api } from "../../lib/api";

const DRAWER_ID = "dashboard-drawer";

interface NavbarProject {
  id: string;
  name: string;
  slug: string;
}

export function DashboardLayout({ children }: { children?: ReactNode }) {
  const location = useLocation();
  const { id } = useParams<{ id: string }>();
  const [project, setProject] = useState<NavbarProject | null>(null);
  const [loading, setLoading] = useState(false);
  const [sidebarOpen, setSidebarOpen] = useState(true);

  const isProjectRoute = location.pathname.startsWith("/projects/");

  useEffect(() => {
    if (!isProjectRoute || !id) {
      setProject(null);
      return;
    }
    setLoading(true);
    api
      .getProject(id)
      .then((data) => setProject(data as NavbarProject))
      .catch(() => setProject(null))
      .finally(() => setLoading(false));
  }, [id, isProjectRoute]);

  return (
    <div className="drawer lg:drawer-open">
      <input
        id={DRAWER_ID}
        type="checkbox"
        className="drawer-toggle"
        aria-hidden
        checked={sidebarOpen}
        onChange={(e) => setSidebarOpen(e.target.checked)}
      />
      <div className="drawer-content flex flex-col min-h-screen bg-base-200">
        <nav className="navbar min-h-16 w-full bg-base-100 border-b border-base-300 sticky top-0 z-30">
          <button
            type="button"
            className="btn btn-square btn-ghost drawer-button"
            onClick={() => setSidebarOpen((prev) => !prev)}
            aria-label={sidebarOpen ? "Collapse sidebar" : "Expand sidebar"}
          >
            {sidebarOpen ? (
              <PanelLeftClose className="size-5" />
            ) : (
              <Menu className="size-5" />
            )}
          </button>
          <div className="flex-1 px-4 flex items-center gap-4">
            {isProjectRoute ? (
              <>
                <Link
                  to="/dashboard"
                  className="text-base-content/70 hover:text-base-content flex items-center gap-2"
                >
                  ← Back to Dashboard
                </Link>
                <span className="font-semibold text-base-content">
                  {loading ? "…" : (project?.name ?? "Project")}
                </span>
                {project?.slug && (
                  <span className="bg-base-300 text-xs px-2 py-1 rounded text-base-content/50">
                    {project.slug}
                  </span>
                )}
              </>
            ) : (
              <span className="font-semibold text-base-content">
                BunBase Platform
              </span>
            )}
          </div>
        </nav>
        <main className="flex-1 overflow-auto">
          <div className="p-4 lg:p-6 h-full min-h-0 flex flex-col">
            {children ?? <Outlet />}
          </div>
        </main>
      </div>

      <div className="drawer-side is-drawer-close:overflow-visible z-40">
        <label
          htmlFor={DRAWER_ID}
          aria-label="Close sidebar"
          className="drawer-overlay"
        />
        <div className="flex min-h-full flex-col bg-base-100 border-r border-base-300 is-drawer-close:w-14 is-drawer-open:w-64 transition-[width] duration-200 ease-in-out overflow-x-hidden">
          <Sidebar />
        </div>
      </div>
    </div>
  );
}

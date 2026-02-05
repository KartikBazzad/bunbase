import { Link, useLocation } from "react-router-dom";
import { useAuth } from "../../hooks/useAuth";
import { ThemeSwitcher } from "../ThemeSwitcher";
import { PortalTooltip } from "../ui/PortalTooltip";
import {
  FolderKanban,
  LayoutDashboard,
  Home,
  Database,
  Zap,
  Settings,
  Key,
  LogOut,
} from "lucide-react";

const linkBase =
  "flex items-center gap-3 px-3 py-2 rounded-lg transition-colors is-drawer-close:justify-center is-drawer-close:px-2 w-full";

export function Sidebar() {
  const { user, logout } = useAuth();
  const location = useLocation();

  const menuItems = [
    { label: "All Projects", icon: LayoutDashboard, path: "/dashboard" },
  ];

  const isActive = (path: string) => {
    const pathname = location.pathname;
    return pathname === path || pathname.startsWith(path + "/");
  };

  const projectBase = location.pathname.startsWith("/projects/")
    ? location.pathname.replace(
        /\/(overview|database|functions|settings|authentication).*$/,
        ""
      )
    : "";

  return (
    <div className="flex min-h-screen flex-col min-w-0 overflow-x-hidden">
      {/* Brand */}
      <div className="min-h-16 p-3 border-b border-base-300 flex items-center gap-2 is-drawer-close:justify-center">
        <FolderKanban className="w-6 h-6 text-primary shrink-0" />
        <span className="font-bold is-drawer-close:hidden">BunBase</span>
      </div>

      {/* Main Navigation - flex-1 so footer stays at bottom */}
      <nav className="flex-1 min-h-0 h-full overflow-y-auto p-2 flex flex-col min-w-0">
        <div className="mb-4">
          <p className="px-3 text-xs font-semibold text-base-content/50 mb-2 uppercase tracking-wider is-drawer-close:hidden">
            Dashboard
          </p>
          {menuItems.map((item) => (
            <PortalTooltip key={item.path} label={item.label}>
              <Link
                to={item.path}
                className={`${linkBase} ${
                  isActive(item.path)
                    ? "bg-primary text-primary-content"
                    : "hover:bg-base-200 text-base-content/70"
                }`}
              >
                <item.icon className="w-5 h-5 shrink-0" />
                <span className="text-sm font-medium is-drawer-close:hidden">
                  {item.label}
                </span>
              </Link>
            </PortalTooltip>
          ))}
        </div>

        {projectBase && (
          <div className="mb-4 w-full">
            <p className="px-3 text-xs font-semibold text-base-content/50 mb-2 uppercase tracking-wider is-drawer-close:hidden">
              Project
            </p>
            <div className="space-y-1 w-full">
              <PortalTooltip label="Overview">
                <Link
                  to={`${projectBase}/overview`}
                  className={`${linkBase} ${
                    location.pathname.endsWith("/overview")
                      ? "bg-primary text-primary-content"
                      : "hover:bg-base-200 text-base-content/70"
                  }`}
                >
                  <Home className="w-5 h-5 shrink-0" />
                  <span className="text-sm font-medium is-drawer-close:hidden">
                    Overview
                  </span>
                </Link>
              </PortalTooltip>
              <PortalTooltip label="Database">
                <Link
                  to={`${projectBase}/database`}
                  className={`${linkBase} ${
                    location.pathname.endsWith("/database")
                      ? "bg-primary text-primary-content"
                      : "hover:bg-base-200 text-base-content/70"
                  }`}
                >
                  <Database className="w-5 h-5 shrink-0" />
                  <span className="text-sm font-medium is-drawer-close:hidden">
                    Database
                  </span>
                </Link>
              </PortalTooltip>
              <PortalTooltip label="Functions">
                <Link
                  to={`${projectBase}/functions`}
                  className={`${linkBase} ${
                    location.pathname.endsWith("/functions") ||
                    location.pathname.includes("/functions/logs")
                      ? "bg-primary text-primary-content"
                      : "hover:bg-base-200 text-base-content/70"
                  }`}
                >
                  <Zap className="w-5 h-5 shrink-0" />
                  <span className="text-sm font-medium is-drawer-close:hidden">
                    Functions
                  </span>
                </Link>
              </PortalTooltip>
              <PortalTooltip label="Authentication">
                <Link
                  to={`${projectBase}/authentication`}
                  className={`${linkBase} ${
                    location.pathname.includes("/authentication")
                      ? "bg-primary text-primary-content"
                      : "hover:bg-base-200 text-base-content/70"
                  }`}
                >
                  <Key className="w-5 h-5 shrink-0" />
                  <span className="text-sm font-medium is-drawer-close:hidden">
                    Authentication
                  </span>
                </Link>
              </PortalTooltip>
              <PortalTooltip label="Settings">
                <Link
                  to={`${projectBase}/settings`}
                  className={`${linkBase} ${
                    location.pathname.endsWith("/settings")
                      ? "bg-primary text-primary-content"
                      : "hover:bg-base-200 text-base-content/70"
                  }`}
                >
                  <Settings className="w-5 h-5 shrink-0" />
                  <span className="text-sm font-medium is-drawer-close:hidden">
                    Settings
                  </span>
                </Link>
              </PortalTooltip>
            </div>
          </div>
        )}
      </nav>

      {/* Sidebar footer: Account (profile) in column, stays at bottom */}
      <div className="border-t border-base-300 pt-4 p-2 flex flex-col shrink-0 min-w-0 gap-2">
        {/* <div className="flex items-center gap-2 px-3 is-drawer-close:justify-center">
          <User className="w-5 h-5 shrink-0 text-base-content/70" />
          <span className="text-xs font-semibold text-base-content/50 uppercase tracking-wider is-drawer-close:hidden">
            Account
          </span>
        </div> */}
        <div
          className="truncate text-sm px-3 text-base-content/70 is-drawer-close:hidden"
          title={user?.name || user?.email || ""}
        >
          {user?.name || user?.email}
        </div>
        <div className="flex items-center gap-2 px-3 is-drawer-close:justify-center">
          <span className="text-xs font-semibold text-base-content/50 uppercase tracking-wider is-drawer-close:hidden">
            Appearance
          </span>
          <ThemeSwitcher />
        </div>
        <PortalTooltip label="Logout">
          <button
            type="button"
            onClick={() => logout()}
            className={`${linkBase} w-full hover:bg-error hover:text-error-content text-base-content/70 is-drawer-close:justify-center`}
            aria-label="Logout"
          >
            <LogOut className="w-5 h-5 shrink-0" />
            <span className="text-sm is-drawer-close:hidden">Logout</span>
          </button>
        </PortalTooltip>
      </div>
    </div>
  );
}

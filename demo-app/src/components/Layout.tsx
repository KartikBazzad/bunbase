import { Outlet, NavLink } from "react-router-dom";

export function Layout() {
  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="border-b bg-white px-4 py-3">
        <div className="mx-auto flex max-w-4xl items-center justify-between">
          <span className="font-semibold text-gray-800">BunBase Demo</span>
          <div className="flex gap-4">
            <NavLink
              to="/"
              className={({ isActive }) =>
                isActive ? "text-blue-600" : "text-gray-600 hover:text-gray-900"
              }
            >
              Home
            </NavLink>
            <NavLink
              to="/documents"
              className={({ isActive }) =>
                isActive ? "text-blue-600" : "text-gray-600 hover:text-gray-900"
              }
            >
              Documents
            </NavLink>
            <NavLink
              to="/references"
              className={({ isActive }) =>
                isActive ? "text-blue-600" : "text-gray-600 hover:text-gray-900"
              }
            >
              References
            </NavLink>
            <NavLink
              to="/functions"
              className={({ isActive }) =>
                isActive ? "text-blue-600" : "text-gray-600 hover:text-gray-900"
              }
            >
              Functions
            </NavLink>
            <NavLink
              to="/settings"
              className={({ isActive }) =>
                isActive ? "text-blue-600" : "text-gray-600 hover:text-gray-900"
              }
            >
              Settings
            </NavLink>
          </div>
        </div>
      </nav>
      <main className="mx-auto max-w-4xl px-4 py-6">
        <Outlet />
      </main>
    </div>
  );
}

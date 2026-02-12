import {
  BrowserRouter,
  Routes,
  Route,
  Navigate,
  Outlet,
  useParams,
} from "react-router-dom";
import { AuthProvider, useAuth } from "./hooks/useAuth";
import { useInstanceStatus } from "./hooks/useInstanceStatus";
import { ThemeProvider } from "./contexts/ThemeContext";
import { ProtectedRoute } from "./components/auth/ProtectedRoute";
import { Login } from "./pages/Login";
import { SignUp } from "./pages/SignUp";
import { Setup } from "./pages/Setup";
import { DashboardLayout } from "./components/layout/DashboardLayout";
import { Dashboard } from "./pages/Dashboard";
import { ProjectOverview } from "./pages/ProjectOverview";
import { Database } from "./pages/Database";
import { Functions } from "./pages/Functions";
import { FunctionLogs } from "./pages/FunctionLogs";
import { Settings } from "./pages/Settings";
import { Authentication } from "./pages/Authentication";
import { Storage } from "./pages/Storage";
import { NotFound } from "./pages/NotFound";
import "./App.css";

function RedirectProjectToOverview() {
  const { id } = useParams();
  return (
    <Navigate to={id ? `/projects/${id}/overview` : "/dashboard"} replace />
  );
}

function RootRedirect() {
  const { user, loading: authLoading } = useAuth();
  const { status, loading: statusLoading } = useInstanceStatus();

  if (authLoading || statusLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <span className="loading loading-spinner loading-lg text-primary" />
      </div>
    );
  }
  if (
    !user &&
    status?.deployment_mode === "self_hosted" &&
    !status?.setup_complete
  ) {
    return <Navigate to="/setup" replace />;
  }
  return <Navigate to="/dashboard" replace />;
}

function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/setup" element={<Setup />} />
            <Route path="/login" element={<Login />} />
            <Route path="/signup" element={<SignUp />} />
            <Route
              path="/dashboard"
              element={
                <ProtectedRoute>
                  <DashboardLayout>
                    <Outlet />
                  </DashboardLayout>
                </ProtectedRoute>
              }
            >
              <Route index element={<Dashboard />} />
            </Route>
            <Route
              path="/projects/:id/overview"
              element={
                <ProtectedRoute>
                  <DashboardLayout>
                    <ProjectOverview />
                  </DashboardLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/projects/:id/database"
              element={
                <ProtectedRoute>
                  <DashboardLayout>
                    <Database />
                  </DashboardLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/projects/:id/functions/logs"
              element={
                <ProtectedRoute>
                  <DashboardLayout>
                    <FunctionLogs />
                  </DashboardLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/projects/:id/functions"
              element={
                <ProtectedRoute>
                  <DashboardLayout>
                    <Functions />
                  </DashboardLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/projects/:id/settings"
              element={
                <ProtectedRoute>
                  <DashboardLayout>
                    <Settings />
                  </DashboardLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/projects/:id/authentication"
              element={
                <ProtectedRoute>
                  <DashboardLayout>
                    <Authentication />
                  </DashboardLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/projects/:id/storage"
              element={
                <ProtectedRoute>
                  <DashboardLayout>
                    <Storage />
                  </DashboardLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/projects/:id"
              element={<RedirectProjectToOverview />}
            />
            <Route path="/" element={<RootRedirect />} />
            <Route path="*" element={<NotFound />} />
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </ThemeProvider>
  );
}

export default App;

import {
  BrowserRouter,
  Routes,
  Route,
  Navigate,
  Outlet,
  useParams,
} from "react-router-dom";
import { AuthProvider } from "./hooks/useAuth";
import { ThemeProvider } from "./contexts/ThemeContext";
import { ProtectedRoute } from "./components/auth/ProtectedRoute";
import { Login } from "./pages/Login";
import { SignUp } from "./pages/SignUp";
import { DashboardLayout } from "./components/layout/DashboardLayout";
import { Dashboard } from "./pages/Dashboard";
import { ProjectOverview } from "./pages/ProjectOverview";
import { Database } from "./pages/Database";
import { Functions } from "./pages/Functions";
import { FunctionLogs } from "./pages/FunctionLogs";
import { Settings } from "./pages/Settings";
import { Authentication } from "./pages/Authentication";
import { NotFound } from "./pages/NotFound";
import "./App.css";

function RedirectProjectToOverview() {
  const { id } = useParams();
  return (
    <Navigate to={id ? `/projects/${id}/overview` : "/dashboard"} replace />
  );
}

function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
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
              path="/projects/:id"
              element={<RedirectProjectToOverview />}
            />
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="*" element={<NotFound />} />
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </ThemeProvider>
  );
}

export default App;

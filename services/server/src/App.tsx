import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "./components/ui/sonner";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { SignIn } from "./pages/auth/SignIn";
import { SignUp } from "./pages/auth/SignUp";
import { ForgotPassword } from "./pages/auth/ForgotPassword";
import { ResetPassword } from "./pages/auth/ResetPassword";
import { VerifyEmail } from "./pages/auth/VerifyEmail";
import { ProtectedRoute } from "./components/ProtectedRoute";
import { DashboardLayout } from "./components/layout/DashboardLayout";
import { Dashboard } from "./pages/dashboard/Dashboard";
import { ProjectOverview } from "./pages/dashboard/projects/ProjectOverview";
import { ProjectApplications } from "./pages/dashboard/projects/ProjectApplications";
import { ProjectDatabases } from "./pages/dashboard/projects/ProjectDatabases";
import { ProjectAuthentication } from "./pages/dashboard/projects/ProjectAuthentication";
import { ProjectLogs } from "./pages/dashboard/projects/ProjectLogs";
import { DatabaseExplorer } from "./pages/dashboard/databases/DatabaseExplorer";
import { NotFound } from "./pages/NotFound";
import { useAuth } from "./hooks/useAuth";

// Create a query client
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

function AppRoutes() {
  const { isAuthenticated, isPending } = useAuth();

  if (isPending) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-muted-foreground">Loading...</div>
      </div>
    );
  }

  return (
    <Routes>
      <Route
        path="/auth/sign-in"
        element={
          isAuthenticated ? <Navigate to="/dashboard" replace /> : <SignIn />
        }
      />
      <Route
        path="/auth/sign-up"
        element={
          isAuthenticated ? <Navigate to="/dashboard" replace /> : <SignUp />
        }
      />
      <Route
        path="/auth/forgot-password"
        element={
          isAuthenticated ? (
            <Navigate to="/dashboard" replace />
          ) : (
            <ForgotPassword />
          )
        }
      />
      <Route
        path="/auth/reset-password"
        element={
          isAuthenticated ? (
            <Navigate to="/dashboard" replace />
          ) : (
            <ResetPassword />
          )
        }
      />
      <Route path="/auth/verify-email" element={<VerifyEmail />} />
      <Route
        path="/dashboard"
        element={
          <ProtectedRoute>
            <DashboardLayout>
              <Dashboard />
            </DashboardLayout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/dashboard/projects/:id"
        element={
          <ProtectedRoute>
            <DashboardLayout>
              <ProjectOverview />
            </DashboardLayout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/dashboard/projects/:id/applications"
        element={
          <ProtectedRoute>
            <DashboardLayout>
              <ProjectApplications />
            </DashboardLayout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/dashboard/projects/:id/explore"
        element={
          <ProtectedRoute>
            <DashboardLayout>
              <DatabaseExplorer />
            </DashboardLayout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/dashboard/projects/:id/explore/collections/:collectionId"
        element={
          <ProtectedRoute>
            <DashboardLayout>
              <DatabaseExplorer />
            </DashboardLayout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/dashboard/projects/:id/explore/collections/:collectionId/documents/:documentId"
        element={
          <ProtectedRoute>
            <DashboardLayout>
              <DatabaseExplorer />
            </DashboardLayout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/dashboard/projects/:id/authentication"
        element={
          <ProtectedRoute>
            <DashboardLayout>
              <ProjectAuthentication />
            </DashboardLayout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/dashboard/projects/:id/logs"
        element={
          <ProtectedRoute>
            <DashboardLayout>
              <ProjectLogs />
            </DashboardLayout>
          </ProtectedRoute>
        }
      />
      <Route path="/" element={<Navigate to="/dashboard" replace />} />
      <Route path="*" element={<NotFound />} />
    </Routes>
  );
}

export function App() {
  return (
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <AppRoutes />
          <Toaster />
        </BrowserRouter>
      </QueryClientProvider>
    </ErrorBoundary>
  );
}

export default App;

import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { AuthProvider } from "./contexts/AuthContext";
import { ConfigProvider } from "./contexts/ConfigContext";
import { Login } from "./pages/Login";
import { SignUp } from "./pages/SignUp";
import { Home } from "./pages/Home";
import { Documents } from "./pages/Documents";
import { References } from "./pages/References";
import { Functions } from "./pages/Functions";
import { KV } from "./pages/KV";
import { Storage } from "./pages/Storage";
import { Settings } from "./pages/Settings";
import { Layout } from "./components/Layout";
import { useAuth } from "./contexts/AuthContext";
import { useConfig } from "./contexts/ConfigContext";

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isConfigured } = useConfig();
  const { isLoggedIn, isLoadingSession } = useAuth();

  if (!isConfigured) {
    return <Navigate to="/settings" replace />;
  }
  if (isLoadingSession) {
    return (
      <div className="flex items-center justify-center py-12">
        <span className="text-gray-500">Loading…</span>
      </div>
    );
  }
  if (!isLoggedIn) {
    return <Navigate to="/login" replace />;
  }
  return <>{children}</>;
}

function GuestRoute({ children }: { children: React.ReactNode }) {
  const { isConfigured } = useConfig();
  const { isLoggedIn, isLoadingSession } = useAuth();

  if (!isConfigured) {
    return <Navigate to="/settings" replace />;
  }
  if (isLoadingSession) {
    return (
      <div className="flex items-center justify-center py-12">
        <span className="text-gray-500">Loading…</span>
      </div>
    );
  }
  if (isLoggedIn) {
    return <Navigate to="/" replace />;
  }
  return <>{children}</>;
}

export default function App() {
  return (
    <ConfigProvider>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route
              path="/login"
              element={
                <GuestRoute>
                  <Login />
                </GuestRoute>
              }
            />
            <Route
              path="/signup"
              element={
                <GuestRoute>
                  <SignUp />
                </GuestRoute>
              }
            />
            <Route path="/" element={<Layout />}>
              <Route
                index
                element={
                  <ProtectedRoute>
                    <Home />
                  </ProtectedRoute>
                }
              />
              <Route
                path="documents"
                element={
                  <ProtectedRoute>
                    <Documents />
                  </ProtectedRoute>
                }
              />
              <Route
                path="references"
                element={
                  <ProtectedRoute>
                    <References />
                  </ProtectedRoute>
                }
              />
              <Route
                path="functions"
                element={
                  <ProtectedRoute>
                    <Functions />
                  </ProtectedRoute>
                }
              />
              <Route
                path="kv"
                element={
                  <ProtectedRoute>
                    <KV />
                  </ProtectedRoute>
                }
              />
              <Route
                path="storage"
                element={
                  <ProtectedRoute>
                    <Storage />
                  </ProtectedRoute>
                }
              />
              <Route path="settings" element={<Settings />} />
            </Route>
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </ConfigProvider>
  );
}

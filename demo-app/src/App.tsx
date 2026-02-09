import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { AuthProvider } from "./contexts/AuthContext";
import { ConfigProvider } from "./contexts/ConfigContext";
import { Login } from "./pages/Login";
import { SignUp } from "./pages/SignUp";
import { Home } from "./pages/Home";
import { Documents } from "./pages/Documents";
import { References } from "./pages/References";
import { Functions } from "./pages/Functions";
import { Settings } from "./pages/Settings";
import { Layout } from "./components/Layout";

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const hasApiKey = localStorage.getItem("demo_api_key");
  if (!hasApiKey) {
    return <Navigate to="/settings" replace />;
  }
  return <>{children}</>;
}

export default function App() {
  return (
    <ConfigProvider>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<Login />} />
            <Route path="/signup" element={<SignUp />} />
            <Route path="/" element={<Layout />}>
              <Route index element={<ProtectedRoute><Home /></ProtectedRoute>} />
              <Route path="documents" element={<ProtectedRoute><Documents /></ProtectedRoute>} />
              <Route path="references" element={<ProtectedRoute><References /></ProtectedRoute>} />
              <Route path="functions" element={<ProtectedRoute><Functions /></ProtectedRoute>} />
              <Route path="settings" element={<Settings />} />
            </Route>
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </ConfigProvider>
  );
}

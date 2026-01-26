import { useState, useEffect } from "react";
import { AuthProvider, useAuth } from "./lib/auth-context";
import { SignUp } from "./components/SignUp";
import { SignIn } from "./components/SignIn";
import { ForgotPassword } from "./components/ForgotPassword";
import { ResetPassword } from "./components/ResetPassword";
import { VerifyEmail } from "./components/VerifyEmail";
import { Dashboard } from "./components/Dashboard";
import "./index.css";

import logo from "./logo.svg";

function AppContent() {
  const { user, loading } = useAuth();
  const [view, setView] = useState<string>("");

  // Handle hash-based routing
  useEffect(() => {
    const handleHashChange = () => {
      const hash = window.location.hash.slice(1) || "";
      setView(hash);
    };

    handleHashChange();
    window.addEventListener("hashchange", handleHashChange);
    return () => window.removeEventListener("hashchange", handleHashChange);
  }, []);

  // If authenticated, show dashboard
  if (user && !view) {
    return (
      <div className="app">
        <div className="logo-container">
          <img src={logo} alt="BunBase Logo" className="logo bun-logo" />
        </div>
        <Dashboard />
      </div>
    );
  }

  // Show loading state
  if (loading && !user) {
    return (
      <div className="app">
        <div className="logo-container">
          <img src={logo} alt="BunBase Logo" className="logo bun-logo" />
        </div>
        <div className="auth-card">
          <div className="loading-spinner"></div>
          <p>Loading...</p>
        </div>
      </div>
    );
  }

  // Show appropriate view based on hash
  const renderView = () => {
    switch (view) {
      case "signup":
        return <SignUp />;
      case "forgot-password":
        return <ForgotPassword />;
      case "reset-password":
        return <ResetPassword />;
      case "verify-email":
        return <VerifyEmail />;
      case "signin":
      default:
        return <SignIn />;
    }
  };

  return (
    <div className="app">
      <div className="logo-container">
        <img src={logo} alt="BunBase Logo" className="logo bun-logo" />
      </div>
      <h1>BunBase Authentication</h1>
      <p className="app-subtitle">Example authentication application</p>
      {renderView()}
    </div>
  );
}

export function App() {
  return (
    <AuthProvider>
      <AppContent />
    </AuthProvider>
  );
}

export default App;

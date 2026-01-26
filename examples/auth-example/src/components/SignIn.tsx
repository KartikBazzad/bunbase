/**
 * Sign In Component
 */

import { useState, FormEvent } from "react";
import { useAuth } from "../lib/auth-context";

export function SignIn() {
  const { signIn, error, loading, clearError } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showForgotPassword, setShowForgotPassword] = useState(false);

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    clearError();

    try {
      await signIn(email, password);
      // Success - user will be redirected by App component
    } catch (err) {
      // Error is handled by auth context
    }
  };

  return (
    <div className="auth-card">
      <h2>Sign In</h2>
      <p className="subtitle">Welcome back! Please sign in to your account</p>

      {error && (
        <div className="error-message">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="auth-form">
        <div className="form-group">
          <label htmlFor="signin-email">Email</label>
          <input
            id="signin-email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="Enter your email"
            disabled={loading}
            required
          />
        </div>

        <div className="form-group">
          <label htmlFor="signin-password">Password</label>
          <input
            id="signin-password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Enter your password"
            disabled={loading}
            required
          />
          <div className="forgot-password-link">
            <a
              href="#forgot-password"
              onClick={(e) => {
                e.preventDefault();
                setShowForgotPassword(true);
              }}
            >
              Forgot password?
            </a>
          </div>
        </div>

        <button type="submit" className="auth-button" disabled={loading}>
          {loading ? "Signing in..." : "Sign In"}
        </button>
      </form>

      <p className="auth-link">
        Don't have an account?{" "}
        <a href="#signup" onClick={(e) => {
          e.preventDefault();
          window.location.hash = "signup";
        }}>
          Sign up
        </a>
      </p>

      {showForgotPassword && (
        <div className="forgot-password-card">
          <h3>Forgot Password?</h3>
          <p>Click the link above or navigate to the forgot password page.</p>
        </div>
      )}
    </div>
  );
}

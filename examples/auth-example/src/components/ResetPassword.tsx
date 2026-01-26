/**
 * Reset Password Component
 */

import { useState, FormEvent, useEffect } from "react";
import { client } from "../lib/client";
import { useAuth } from "../lib/auth-context";

export function ResetPassword() {
  const { signIn } = useAuth();
  const [token, setToken] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);

  // Get token from URL query params
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const tokenParam = params.get("token");
    if (tokenParam) {
      setToken(tokenParam);
    }
  }, []);

  const validateForm = () => {
    if (!token) {
      setValidationError("Reset token is required");
      return false;
    }
    if (!password || password.length < 6) {
      setValidationError("Password must be at least 6 characters");
      return false;
    }
    if (password !== confirmPassword) {
      setValidationError("Passwords do not match");
      return false;
    }
    setValidationError(null);
    return true;
  };

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);
    setSuccess(false);

    if (!validateForm()) {
      return;
    }

    setLoading(true);

    try {
      await client.auth.resetPassword(token, password);
      setSuccess(true);
    } catch (err: any) {
      setError(err.message || "Failed to reset password. The token may be invalid or expired.");
    } finally {
      setLoading(false);
    }
  };

  if (success) {
    return (
      <div className="auth-card">
        <h2>Password Reset Successful!</h2>
        <p className="success-message">
          Your password has been reset successfully.
        </p>
        <p className="info-text">
          You can now sign in with your new password.
        </p>
        <button
          className="auth-button"
          onClick={() => {
            window.location.hash = "signin";
          }}
        >
          Go to Sign In
        </button>
      </div>
    );
  }

  return (
    <div className="auth-card">
      <h2>Reset Password</h2>
      <p className="subtitle">Enter your new password</p>

      {(error || validationError) && (
        <div className="error-message">
          {validationError || error}
        </div>
      )}

      {!token && (
        <div className="info-message">
          No reset token found. Please use the link from your email.
        </div>
      )}

      <form onSubmit={handleSubmit} className="auth-form">
        {!token && (
          <div className="form-group">
            <label htmlFor="reset-token">Reset Token</label>
            <input
              id="reset-token"
              type="text"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder="Enter reset token from email"
              disabled={loading}
              required
            />
          </div>
        )}

        <div className="form-group">
          <label htmlFor="reset-password">New Password</label>
          <input
            id="reset-password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Enter new password"
            disabled={loading}
            required
          />
        </div>

        <div className="form-group">
          <label htmlFor="confirm-password">Confirm Password</label>
          <input
            id="confirm-password"
            type="password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            placeholder="Confirm new password"
            disabled={loading}
            required
          />
        </div>

        <button type="submit" className="auth-button" disabled={loading || !token}>
          {loading ? "Resetting..." : "Reset Password"}
        </button>
      </form>

      <p className="auth-link">
        Remember your password?{" "}
        <a href="#signin" onClick={(e) => {
          e.preventDefault();
          window.location.hash = "signin";
        }}>
          Sign in
        </a>
      </p>
    </div>
  );
}

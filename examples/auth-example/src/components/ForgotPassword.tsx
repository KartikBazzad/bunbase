/**
 * Forgot Password Component
 */

import { useState, FormEvent } from "react";
import { client } from "../lib/client";

export function ForgotPassword() {
  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);
    setSuccess(false);
    setLoading(true);

    try {
      await client.auth.forgotPassword(email);
      setSuccess(true);
      setEmail("");
    } catch (err: any) {
      setError(err.message || "Failed to send password reset email");
    } finally {
      setLoading(false);
    }
  };

  if (success) {
    return (
      <div className="auth-card">
        <h2>Check Your Email</h2>
        <p className="success-message">
          If an account exists with that email, we've sent you a password reset link.
        </p>
        <p className="info-text">
          Please check your inbox and follow the instructions to reset your password.
        </p>
        <button
          className="auth-button"
          onClick={() => {
            setSuccess(false);
            window.location.hash = "signin";
          }}
        >
          Back to Sign In
        </button>
      </div>
    );
  }

  return (
    <div className="auth-card">
      <h2>Forgot Password</h2>
      <p className="subtitle">
        Enter your email address and we'll send you a link to reset your password
      </p>

      {error && (
        <div className="error-message">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="auth-form">
        <div className="form-group">
          <label htmlFor="forgot-email">Email</label>
          <input
            id="forgot-email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="Enter your email"
            disabled={loading}
            required
          />
        </div>

        <button type="submit" className="auth-button" disabled={loading}>
          {loading ? "Sending..." : "Send Reset Link"}
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

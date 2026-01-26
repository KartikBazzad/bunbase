/**
 * Verify Email Component
 */

import { useState, useEffect } from "react";
import { client } from "../lib/client";
import { useAuth } from "../lib/auth-context";

export function VerifyEmail() {
  const { getUser } = useAuth();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  useEffect(() => {
    const verifyEmail = async () => {
      // Get token from URL query params
      const params = new URLSearchParams(window.location.search);
      const token = params.get("token");

      if (!token) {
        setError("No verification token found in URL");
        setLoading(false);
        return;
      }

      try {
        await client.auth.verifyEmail(token);
        setSuccess(true);
        // Refresh user data to get updated verification status
        try {
          await getUser();
        } catch (err) {
          // Ignore errors when refreshing user
        }
      } catch (err: any) {
        setError(err.message || "Failed to verify email. The token may be invalid or expired.");
      } finally {
        setLoading(false);
      }
    };

    verifyEmail();
  }, [getUser]);

  if (loading) {
    return (
      <div className="auth-card">
        <h2>Verifying Email...</h2>
        <div className="loading-spinner"></div>
      </div>
    );
  }

  if (success) {
    return (
      <div className="auth-card">
        <h2>Email Verified!</h2>
        <p className="success-message">
          Your email has been successfully verified.
        </p>
        <p className="info-text">
          You can now access all features of your account.
        </p>
        <button
          className="auth-button"
          onClick={() => {
            window.location.hash = "";
            window.location.reload();
          }}
        >
          Continue
        </button>
      </div>
    );
  }

  return (
    <div className="auth-card">
      <h2>Email Verification Failed</h2>
      {error && (
        <div className="error-message">
          {error}
        </div>
      )}
      <p className="info-text">
        The verification link may have expired or is invalid. Please request a new verification email.
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

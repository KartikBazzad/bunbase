/**
 * Dashboard Component
 * 
 * Displays user profile information and provides sign out functionality
 */

import { useAuth } from "../lib/auth-context";
import { useState } from "react";

export function Dashboard() {
  const { user, signOut, loading } = useAuth();
  const [signingOut, setSigningOut] = useState(false);

  const handleSignOut = async () => {
    setSigningOut(true);
    try {
      await signOut();
      window.location.hash = "signin";
    } catch (err) {
      // Error handled by auth context
    } finally {
      setSigningOut(false);
    }
  };

  if (!user) {
    return (
      <div className="auth-card">
        <div className="loading-spinner"></div>
        <p>Loading user information...</p>
      </div>
    );
  }

  return (
    <div className="dashboard">
      <div className="dashboard-header">
        <h1>Welcome, {user.name}!</h1>
        <button
          className="sign-out-button"
          onClick={handleSignOut}
          disabled={signingOut || loading}
        >
          {signingOut ? "Signing out..." : "Sign Out"}
        </button>
      </div>

      <div className="dashboard-content">
        <div className="profile-card">
          <h2>Profile Information</h2>
          <div className="profile-info">
            <div className="info-row">
              <span className="info-label">Name:</span>
              <span className="info-value">{user.name}</span>
            </div>
            <div className="info-row">
              <span className="info-label">Email:</span>
              <span className="info-value">{user.email}</span>
            </div>
            <div className="info-row">
              <span className="info-label">Email Verified:</span>
              <span className={`info-value ${user.emailVerified ? "verified" : "unverified"}`}>
                {user.emailVerified ? "✓ Verified" : "✗ Not Verified"}
              </span>
            </div>
            {user.image && (
              <div className="info-row">
                <span className="info-label">Avatar:</span>
                <img src={user.image} alt="Avatar" className="avatar-image" />
              </div>
            )}
            <div className="info-row">
              <span className="info-label">User ID:</span>
              <span className="info-value user-id">{user.id}</span>
            </div>
            <div className="info-row">
              <span className="info-label">Account Created:</span>
              <span className="info-value">
                {new Date(user.createdAt).toLocaleDateString()}
              </span>
            </div>
            <div className="info-row">
              <span className="info-label">Last Updated:</span>
              <span className="info-value">
                {new Date(user.updatedAt).toLocaleDateString()}
              </span>
            </div>
          </div>
        </div>

        {!user.emailVerified && (
          <div className="info-card warning">
            <h3>Email Not Verified</h3>
            <p>
              Please verify your email address to access all features. Check your inbox for a verification email.
            </p>
          </div>
        )}

        <div className="info-card">
          <h3>About This Example</h3>
          <p>
            This is a demonstration of the BunBase authentication system. You can:
          </p>
          <ul>
            <li>Sign up with email and password</li>
            <li>Sign in to your account</li>
            <li>Reset your password</li>
            <li>Verify your email address</li>
            <li>View your profile information</li>
            <li>Sign out securely</li>
          </ul>
        </div>
      </div>
    </div>
  );
}

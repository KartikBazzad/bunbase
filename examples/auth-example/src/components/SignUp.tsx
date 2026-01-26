/**
 * Sign Up Component
 */

import { useState, FormEvent } from "react";
import { useAuth } from "../lib/auth-context";

export function SignUp() {
  const { signUp, error, loading, clearError } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [success, setSuccess] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);

  const validateForm = () => {
    if (!email || !email.includes("@")) {
      setValidationError("Please enter a valid email address");
      return false;
    }
    if (!password || password.length < 6) {
      setValidationError("Password must be at least 6 characters");
      return false;
    }
    if (!name || name.trim().length < 2) {
      setValidationError("Name must be at least 2 characters");
      return false;
    }
    setValidationError(null);
    return true;
  };

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    clearError();
    setSuccess(false);

    if (!validateForm()) {
      return;
    }

    try {
      await signUp(email, password, name);
      setSuccess(true);
      // Reset form
      setEmail("");
      setPassword("");
      setName("");
    } catch (err) {
      // Error is handled by auth context
    }
  };

  const getPasswordStrength = (pwd: string): { strength: string; color: string } => {
    if (pwd.length === 0) return { strength: "", color: "" };
    if (pwd.length < 6) return { strength: "Weak", color: "#ff4444" };
    if (pwd.length < 10) return { strength: "Medium", color: "#ffaa00" };
    if (/[A-Z]/.test(pwd) && /[a-z]/.test(pwd) && /[0-9]/.test(pwd)) {
      return { strength: "Strong", color: "#00ff00" };
    }
    return { strength: "Medium", color: "#ffaa00" };
  };

  const passwordStrength = getPasswordStrength(password);

  if (success) {
    return (
      <div className="auth-card">
        <h2>Sign Up Successful!</h2>
        <p className="success-message">
          Your account has been created. Please check your email to verify your account.
        </p>
        <p className="info-text">
          You can now sign in with your email and password.
        </p>
      </div>
    );
  }

  return (
    <div className="auth-card">
      <h2>Sign Up</h2>
      <p className="subtitle">Create a new account to get started</p>

      {(error || validationError) && (
        <div className="error-message">
          {validationError || error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="auth-form">
        <div className="form-group">
          <label htmlFor="name">Name</label>
          <input
            id="name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Enter your name"
            disabled={loading}
            required
          />
        </div>

        <div className="form-group">
          <label htmlFor="email">Email</label>
          <input
            id="email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="Enter your email"
            disabled={loading}
            required
          />
        </div>

        <div className="form-group">
          <label htmlFor="password">Password</label>
          <input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Enter your password"
            disabled={loading}
            required
          />
          {password && (
            <div className="password-strength">
              <span style={{ color: passwordStrength.color }}>
                {passwordStrength.strength}
              </span>
            </div>
          )}
        </div>

        <button type="submit" className="auth-button" disabled={loading}>
          {loading ? "Signing up..." : "Sign Up"}
        </button>
      </form>

      <p className="auth-link">
        Already have an account?{" "}
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

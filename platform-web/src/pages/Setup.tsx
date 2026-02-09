import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "../lib/api";
import { useInstanceStatus } from "../hooks/useInstanceStatus";
import { useAuth } from "../hooks/useAuth";
import { ThemeSwitcher } from "../components/ThemeSwitcher";
import { FolderKanban, User, Mail, Lock, AlertCircle } from "lucide-react";

export function Setup() {
  const { status, loading: statusLoading } = useInstanceStatus();
  const { refresh } = useAuth();
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (statusLoading || !status) return;
    if (status.deployment_mode !== "self_hosted" || status.setup_complete) {
      navigate("/login", { replace: true });
    }
  }, [status, statusLoading, navigate]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    if (password !== confirmPassword) {
      setError("Passwords do not match");
      return;
    }
    if (password.length < 8) {
      setError("Password must be at least 8 characters");
      return;
    }
    setLoading(true);
    try {
      await api.setup(email, password, name);
      await refresh();
      navigate("/dashboard", { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Setup failed");
    } finally {
      setLoading(false);
    }
  };

  if (statusLoading || !status) {
    return (
      <div className="min-h-screen bg-base-200 flex items-center justify-center">
        <span className="loading loading-spinner loading-lg text-primary" />
      </div>
    );
  }
  if (status.deployment_mode !== "self_hosted" || status.setup_complete) {
    return null;
  }

  return (
    <div className="min-h-screen bg-base-200 flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
      <div className="absolute top-4 right-4">
        <ThemeSwitcher />
      </div>
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <div className="mx-auto mb-4 p-3 bg-base-100 rounded-full w-16 h-16 flex items-center justify-center">
            <FolderKanban className="w-8 h-8 text-primary" />
          </div>
          <h1 className="text-3xl font-bold text-base-content mb-2">BunBase Setup</h1>
          <p className="text-base-content/70">Create your administrator account</p>
        </div>
        <div className="card bg-base-100 shadow-xl">
          <div className="card-body">
            <form onSubmit={handleSubmit} className="space-y-4">
              {error && (
                <div className="alert alert-error">
                  <AlertCircle className="w-5 h-5 shrink-0" />
                  <span>{error}</span>
                </div>
              )}

              <div className="form-control">
                <label className="label" htmlFor="setup-name">
                  <span className="label-text">Name</span>
                </label>
                <div className="relative">
                  <User className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-base-content/50" />
                  <input
                    id="setup-name"
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="Your name"
                    className="input input-bordered w-full pl-10"
                    required
                  />
                </div>
              </div>

              <div className="form-control">
                <label className="label" htmlFor="setup-email">
                  <span className="label-text">Email</span>
                </label>
                <div className="relative">
                  <Mail className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-base-content/50" />
                  <input
                    id="setup-email"
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    placeholder="admin@example.com"
                    className="input input-bordered w-full pl-10"
                    required
                  />
                </div>
              </div>

              <div className="form-control">
                <label className="label" htmlFor="setup-password">
                  <span className="label-text">Password</span>
                </label>
                <div className="relative">
                  <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-base-content/50" />
                  <input
                    id="setup-password"
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="input input-bordered w-full pl-10"
                    required
                    minLength={8}
                  />
                </div>
                <p className="text-xs text-base-content/50 mt-1">At least 8 characters</p>
              </div>

              <div className="form-control">
                <label className="label" htmlFor="setup-confirm">
                  <span className="label-text">Confirm Password</span>
                </label>
                <div className="relative">
                  <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-base-content/50" />
                  <input
                    id="setup-confirm"
                    type="password"
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    className="input input-bordered w-full pl-10"
                    required
                  />
                </div>
              </div>

              <button
                type="submit"
                disabled={loading}
                className={`btn btn-primary w-full ${loading ? "btn-disabled" : ""}`}
              >
                {loading && <span className="loading loading-spinner" />}
                {loading ? "Creating account..." : "Create admin account"}
              </button>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
}

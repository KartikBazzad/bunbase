import { useState } from 'react';
import { useAuth } from '../../hooks/useAuth';
import { useNavigate } from 'react-router-dom';
import { Mail, Lock, AlertCircle } from 'lucide-react';

export function LoginForm() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await login(email, password);
      navigate('/dashboard');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="card max-w-md mx-auto bg-base-100 shadow-xl">
      <div className="card-body">
        <h1 className="text-2xl font-bold mb-4">Sign In</h1>
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div className="alert alert-error">
              <AlertCircle className="w-5 h-5 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          <div className="form-control">
            <label className="label" htmlFor="email">
              <span className="label-text">Email</span>
              <span className="label-text-alt text-error">*</span>
            </label>
            <div className="relative">
              <Mail className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-base-content/50" />
              <input
                id="email"
                type="email"
                autoComplete="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="you@example.com"
                className="input input-bordered w-full pl-10"
                required
              />
            </div>
          </div>

          <div className="form-control">
            <label className="label" htmlFor="password">
              <span className="label-text">Password</span>
              <span className="label-text-alt text-error">*</span>
            </label>
            <div className="relative">
              <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-base-content/50" />
              <input
                id="password"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="input input-bordered w-full pl-10"
                required
              />
            </div>
          </div>

          <button
            type="submit"
            disabled={loading}
            className={`btn btn-primary w-full ${loading ? 'btn-disabled' : ''}`}
          >
            {loading && <span className="loading loading-spinner"></span>}
            {loading ? 'Signing in...' : 'Sign In'}
          </button>
        </form>
      </div>
    </div>
  );
}

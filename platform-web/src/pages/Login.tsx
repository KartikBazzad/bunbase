import { Link } from 'react-router-dom';
import { LoginForm } from '../components/auth/LoginForm';
import { ThemeSwitcher } from '../components/ThemeSwitcher';
import { FolderKanban } from 'lucide-react';

export function Login() {
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
          <h1 className="text-3xl font-bold text-base-content mb-2">BunBase Platform</h1>
          <p className="text-base-content/70">Sign in to your account</p>
        </div>
        <LoginForm />
        <p className="text-center mt-6 text-sm text-base-content/70">
          Don't have an account?{' '}
          <Link to="/signup" className="link link-primary">
            Sign up
          </Link>
        </p>
      </div>
    </div>
  );
}

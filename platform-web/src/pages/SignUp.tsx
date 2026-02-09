import { Link } from 'react-router-dom';
import { SignUpForm } from '../components/auth/SignUpForm';
import { ThemeSwitcher } from '../components/ThemeSwitcher';
import { FolderKanban } from 'lucide-react';
import { useInstanceStatus } from '../hooks/useInstanceStatus';

export function SignUp() {
  const { status, loading: statusLoading } = useInstanceStatus();
  const signupDisabled = status?.deployment_mode === 'self_hosted' && status?.setup_complete;

  if (statusLoading || !status) {
    return (
      <div className="min-h-screen bg-base-200 flex items-center justify-center">
        <span className="loading loading-spinner loading-lg text-primary" />
      </div>
    );
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
          <h1 className="text-3xl font-bold text-base-content mb-2">BunBase Platform</h1>
          <p className="text-base-content/70">Create your account</p>
        </div>
        {signupDisabled ? (
          <div className="card bg-base-100 shadow-xl">
            <div className="card-body">
              <p className="text-base-content/80 text-center">
                Sign up is disabled on this instance. Contact your administrator.
              </p>
              <Link to="/login" className="btn btn-primary mt-4">
                Sign in
              </Link>
            </div>
          </div>
        ) : (
          <SignUpForm />
        )}
        <p className="text-center mt-6 text-sm text-base-content/70">
          Already have an account?{' '}
          <Link to="/login" className="link link-primary">
            Sign in
          </Link>
        </p>
      </div>
    </div>
  );
}

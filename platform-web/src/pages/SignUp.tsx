import { Link } from 'react-router-dom';
import { SignUpForm } from '../components/auth/SignUpForm';

export function SignUp() {
  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-gray-900 mb-2">BunBase Platform</h1>
          <p className="text-gray-600">Create your account</p>
        </div>
        <SignUpForm />
        <p className="text-center mt-6 text-sm text-gray-600">
          Already have an account?{' '}
          <Link to="/login" className="link">
            Sign in
          </Link>
        </p>
      </div>
    </div>
  );
}

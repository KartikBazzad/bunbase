import { Link } from 'react-router-dom';
import { Home, AlertTriangle } from 'lucide-react';

export function NotFound() {
  return (
    <div className="min-h-screen bg-base-200 flex items-center justify-center">
      <div className="text-center">
        <div className="mb-4 p-6 bg-base-100 rounded-full w-24 h-24 mx-auto flex items-center justify-center">
          <AlertTriangle className="w-12 h-12 text-warning" />
        </div>
        <h1 className="text-6xl font-bold text-base-content mb-2">404</h1>
        <p className="text-xl text-base-content/70 mb-8">Page not found</p>
        <Link to="/dashboard" className="btn btn-primary">
          <Home className="w-4 h-4 mr-1" />
          Go to Dashboard
        </Link>
      </div>
    </div>
  );
}

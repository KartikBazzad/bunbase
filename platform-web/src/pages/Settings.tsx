import { useAuth } from "../hooks/useAuth";

export function Settings() {
  const { user } = useAuth();

  return (
    <div className="min-h-screen bg-base-200 flex flex-col">
      <main className="container mx-auto px-4 sm:px-6 lg:px-8 max-w-7xl py-8">
        <div className="card bg-base-100 shadow-md">
          <div className="card-body py-8">
            <h2 className="text-2xl font-bold mb-6">Project Settings</h2>
            <p className="text-base-content/70 mb-4">
              Project settings coming soon. This page will include:
            </p>
            <ul className="text-base-content/70 space-y-3">
              <li className="flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-primary"></span>
                <span>Project name and slug management</span>
              </li>
              <li className="flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-primary"></span>
                <span>Environment variables configuration</span>
              </li>
              <li className="flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-primary"></span>
                <span>API key and access tokens</span>
              </li>
              <li className="flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-primary"></span>
                <span>Webhook configuration</span>
              </li>
              <li className="flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-warning"></span>
                <span>Project archival / deletion</span>
              </li>
              <li className="flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-warning"></span>
                <span>Collaborator management</span>
              </li>
            </ul>
          </div>
        </div>

        <div className="card bg-base-100 shadow-md">
          <div className="card-body py-8">
            <h2 className="text-2xl font-bold mb-6">Account Settings</h2>
            <p className="text-base-content/70 mb-4">
              Account-wide settings are not yet available. These would include:
            </p>
            <ul className="text-base-content/70 space-y-3">
              <li className="flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-primary"></span>
                <span>Profile information</span>
              </li>
              <li className="flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-primary"></span>
                <span>Email and password management</span>
              </li>
              <li className="flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-primary"></span>
                <span>Two-factor authentication</span>
              </li>
              <li className="flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-primary"></span>
                <span>Notification preferences</span>
              </li>
            </ul>
          </div>
        </div>
      </main>
    </div>
  );
}

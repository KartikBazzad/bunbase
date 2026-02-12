import { Link } from "react-router-dom";
import { useConfig } from "@/contexts/ConfigContext";
import { useAuth } from "@/contexts/AuthContext";

export function Home() {
  const { isConfigured } = useConfig();
  const { isLoggedIn } = useAuth();

  const features = [
    {
      title: "Documents",
      description: "CRUD operations on a tasks collection via the SDK",
      to: "/documents",
      icon: "📄",
      code: "tasks",
      requiresAuth: true,
    },
    {
      title: "References",
      description:
        "Cross-collection references demo with users and posts. Test restrict, set_null, and cascade on delete",
      to: "/references",
      icon: "🔗",
      code: "ref_users, ref_posts",
      requiresAuth: true,
    },
    {
      title: "Functions",
      description: "Invoke serverless functions by name with optional JSON body",
      to: "/functions",
      icon: "⚡",
      code: "hello-world",
      requiresAuth: true,
    },
    {
      title: "KV",
      description: "Key-value storage operations with the BunBase SDK",
      to: "/kv",
      icon: "🗃️",
      code: "kv",
      requiresAuth: true,
    },
    {
      title: "Storage",
      description: "File storage operations with upload and download capabilities",
      to: "/storage",
      icon: "📦",
      code: "storage",
      requiresAuth: true,
    },
    {
      title: "Settings",
      description: "Configure your API URL, API key, and project settings",
      to: "/settings",
      icon: "⚙️",
      code: null,
      requiresAuth: false,
    },
  ];

  return (
    <div className="space-y-6">
      {/* Welcome Section */}
      <div className="rounded-lg border bg-white p-6 shadow-sm">
        <h1 className="mb-2 text-3xl font-bold text-gray-900">
          Welcome to BunBase Demo
        </h1>
        <p className="mb-4 text-gray-600">
          Explore the BunBase TypeScript SDK with interactive demos for
          documents, references, functions, KV storage, and file storage.
        </p>
        {!isConfigured && (
          <div className="rounded-lg border border-amber-200 bg-amber-50 p-4">
            <p className="text-sm text-amber-800">
              <strong>Setup required:</strong> Set your{" "}
              <Link to="/settings" className="font-semibold underline">
                Project API key in Settings
              </Link>{" "}
              to use Documents, Functions, KV, and Storage features.
            </p>
          </div>
        )}
        {!isLoggedIn && (
          <div className="mt-4 rounded-lg border border-blue-200 bg-blue-50 p-4">
            <p className="text-sm text-blue-800">
              <Link to="/login" className="font-semibold underline">
                Log in
              </Link>{" "}
              or{" "}
              <Link to="/signup" className="font-semibold underline">
                sign up
              </Link>{" "}
              to access project user authentication features.
            </p>
          </div>
        )}
      </div>

      {/* Features Grid */}
      <div>
        <h2 className="mb-4 text-xl font-semibold text-gray-900">
          Available Features
        </h2>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {features.map((feature) => {
            const isDisabled =
              feature.requiresAuth && (!isConfigured || !isLoggedIn);

            return (
              <Link
                key={feature.to}
                to={feature.to}
                className={`group relative block rounded-lg border bg-white p-5 shadow-sm transition-all hover:shadow-md ${
                  isDisabled
                    ? "cursor-not-allowed opacity-60"
                    : "hover:border-blue-300"
                }`}
                onClick={(e) => {
                  if (isDisabled) {
                    e.preventDefault();
                  }
                }}
              >
                <div className="flex items-start gap-3">
                  <span className="text-2xl">{feature.icon}</span>
                  <div className="flex-1">
                    <h3 className="mb-1 text-lg font-semibold text-gray-900 group-hover:text-blue-600">
                      {feature.title}
                    </h3>
                    <p className="mb-2 text-sm text-gray-600">
                      {feature.description}
                    </p>
                    {feature.code && (
                      <code className="rounded bg-gray-100 px-2 py-1 text-xs text-gray-700">
                        {feature.code}
                      </code>
                    )}
                  </div>
                </div>
                {isDisabled && (
                  <div className="absolute inset-0 flex items-center justify-center rounded-lg bg-white/80">
                    <span className="text-xs font-medium text-gray-500">
                      {!isConfigured
                        ? "Setup required"
                        : !isLoggedIn
                          ? "Login required"
                          : "Unavailable"}
                    </span>
                  </div>
                )}
              </Link>
            );
          })}
        </div>
      </div>

      {/* Quick Start */}
      {isConfigured && isLoggedIn && (
        <div className="rounded-lg border border-green-200 bg-green-50 p-6">
          <h3 className="mb-2 text-lg font-semibold text-green-900">
            Quick Start
          </h3>
          <p className="mb-3 text-sm text-green-800">
            You're all set! Start exploring:
          </p>
          <div className="flex flex-wrap gap-2">
            <Link
              to="/documents"
              className="rounded bg-green-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-green-700"
            >
              Try Documents →
            </Link>
            <Link
              to="/functions"
              className="rounded border border-green-600 px-3 py-1.5 text-sm font-medium text-green-700 hover:bg-green-100"
            >
              Try Functions →
            </Link>
          </div>
        </div>
      )}
    </div>
  );
}

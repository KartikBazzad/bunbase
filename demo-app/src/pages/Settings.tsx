import { useState, useEffect } from "react";
import { useConfig } from "@/contexts/ConfigContext";

export function Settings() {
  const { baseUrl, apiKey, setConfig, isConfigured } = useConfig();
  const [url, setUrl] = useState(baseUrl);
  const [key, setKey] = useState(apiKey);
  const [saved, setSaved] = useState(false);
  const [projectUsers, setProjectUsers] = useState<{
    users: unknown[];
    error?: string;
  } | null>(null);
  const [authConfig, setAuthConfig] = useState<unknown | null>(null);

  useEffect(() => {
    if (!isConfigured || !baseUrl || !apiKey) return;
    const headers = { "X-Bunbase-Client-Key": apiKey };
    Promise.all([
      fetch(`${baseUrl}/v1/auth/users`, { headers }).then((r) => r.json()),
      fetch(`${baseUrl}/v1/auth/config`, { headers }).then((r) => r.json()),
    ])
      .then(([usersRes, configRes]) => {
        setProjectUsers(
          usersRes.users != null
            ? { users: usersRes.users }
            : { users: [], error: usersRes.error },
        );
        setAuthConfig(configRes);
      })
      .catch(() => {
        setProjectUsers({ users: [], error: "Failed to fetch" });
        setAuthConfig(null);
      });
  }, [isConfigured, baseUrl, apiKey]);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setConfig({ baseUrl: url, apiKey: key });
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
  }

  return (
    <div className="space-y-6">
      <div className="rounded-lg border bg-white p-6 shadow-sm">
        <h1 className="mb-4 text-xl font-semibold">Settings</h1>
        <p className="mb-4 text-sm text-gray-600">
          Get your <strong>API key</strong> from the BunBase dashboard (open a
          project → Settings). The API key identifies the project; no project ID
          needed.
        </p>
        <form onSubmit={handleSubmit} className="flex max-w-md flex-col gap-3">
          <label className="flex flex-col gap-1">
            <span className="text-sm font-medium">Base URL</span>
            <input
              type="url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              className="rounded border px-3 py-2 font-mono text-sm"
              placeholder="http://localhost:3001"
            />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-sm font-medium">
              API key (from dashboard Project → Settings)
            </span>
            <input
              type="password"
              value={key}
              onChange={(e) => setKey(e.target.value)}
              className="rounded border px-3 py-2 font-mono text-sm"
              placeholder="Paste your API key (pk_...)"
            />
          </label>
          <div className="flex items-center gap-2">
            <button
              type="submit"
              className="rounded bg-blue-600 px-4 py-2 text-white"
            >
              Save
            </button>
            {saved && <span className="text-sm text-green-600">Saved.</span>}
          </div>
        </form>
        {isConfigured && (
          <p className="mt-4 text-sm text-green-600">
            API key is set. You can use Documents, KV, and project auth (Log in
            / Sign up).
          </p>
        )}
      </div>

      {isConfigured && (
        <div className="rounded-lg border bg-white p-6 shadow-sm">
          <h2 className="mb-2 text-lg font-semibold">
            Project Auth (tenant-auth)
          </h2>
          <p className="mb-4 text-sm text-gray-600">
            Project users and auth config for the current API key. Use Log in /
            Sign up above to test project user auth.
          </p>
          <div className="space-y-4">
            <div>
              <h3 className="mb-1 text-sm font-medium">Project users</h3>
              {projectUsers == null ? (
                <p className="text-sm text-gray-500">Loading…</p>
              ) : projectUsers.error ? (
                <p className="text-sm text-amber-600">{projectUsers.error}</p>
              ) : (
                <pre className="max-h-40 overflow-auto rounded border bg-gray-50 p-2 text-xs">
                  {JSON.stringify(projectUsers.users, null, 2)}
                </pre>
              )}
            </div>
            <div>
              <h3 className="mb-1 text-sm font-medium">Auth config</h3>
              {authConfig == null &&
              projectUsers != null &&
              !projectUsers.error ? (
                <p className="text-sm text-gray-500">Loading…</p>
              ) : authConfig == null ? (
                <p className="text-sm text-gray-500">—</p>
              ) : (
                <pre className="max-h-40 overflow-auto rounded border bg-gray-50 p-2 text-xs">
                  {JSON.stringify(authConfig, null, 2)}
                </pre>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

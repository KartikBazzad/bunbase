import { useState } from "react";
import { useConfig } from "@/contexts/ConfigContext";

export function Settings() {
  const { baseUrl, apiKey, projectId, setConfig, isConfigured } = useConfig();
  const [url, setUrl] = useState(baseUrl);
  const [key, setKey] = useState(apiKey);
  const [project, setProject] = useState(projectId);
  const [saved, setSaved] = useState(false);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setConfig({ baseUrl: url, apiKey: key, projectId: project });
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
  }

  return (
    <div className="rounded-lg border bg-white p-6 shadow-sm">
      <h1 className="mb-4 text-xl font-semibold">Settings</h1>
      <p className="mb-4 text-sm text-gray-600">
        Get your <strong>Project API key</strong> from the BunBase dashboard (open a project → Settings) and your <strong>Project ID</strong> from the project URL or Projects list. Use them with the TypeScript SDK to access this project&apos;s database and functions.
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
          <span className="text-sm font-medium">Project API key (from dashboard Project → Settings)</span>
          <input
            type="password"
            value={key}
            onChange={(e) => setKey(e.target.value)}
            className="rounded border px-3 py-2 font-mono text-sm"
            placeholder="Paste your project API key (pk_...)"
          />
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">Project ID</span>
          <input
            type="text"
            value={project}
            onChange={(e) => setProject(e.target.value)}
            className="rounded border px-3 py-2 font-mono text-sm"
            placeholder="e.g. 550e8400-e29b-41d4-a716-446655440000"
          />
        </label>
        <div className="flex items-center gap-2">
          <button type="submit" className="rounded bg-blue-600 px-4 py-2 text-white">
            Save
          </button>
          {saved && <span className="text-sm text-green-600">Saved.</span>}
        </div>
      </form>
      {isConfigured && (
        <p className="mt-4 text-sm text-green-600">Project API key and project are set. You can use Documents and Functions.</p>
      )}
    </div>
  );
}

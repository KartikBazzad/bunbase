import { useState } from "react";
import { useConfig } from "@/contexts/ConfigContext";

export function Settings() {
  const { baseUrl, apiKey, setConfig, isConfigured } = useConfig();
  const [url, setUrl] = useState(baseUrl);
  const [key, setKey] = useState(apiKey);
  const [saved, setSaved] = useState(false);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setConfig({ baseUrl: url, apiKey: key });
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
  }

  return (
    <div className="rounded-lg border bg-white p-6 shadow-sm">
      <h1 className="mb-4 text-xl font-semibold">Settings</h1>
      <p className="mb-4 text-sm text-gray-600">
        Get your <strong>API key</strong> from the BunBase dashboard (open a project → Settings). The API key identifies the project; no project ID needed.
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
          <span className="text-sm font-medium">API key (from dashboard Project → Settings)</span>
          <input
            type="password"
            value={key}
            onChange={(e) => setKey(e.target.value)}
            className="rounded border px-3 py-2 font-mono text-sm"
            placeholder="Paste your API key (pk_...)"
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
        <p className="mt-4 text-sm text-green-600">API key is set. You can use Documents and Functions.</p>
      )}
    </div>
  );
}

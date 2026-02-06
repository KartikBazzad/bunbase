import { useState } from "react";
import { useConfig } from "@/contexts/ConfigContext";
import { createClient } from "@/lib/client";

export function Functions() {
  const { baseUrl, apiKey, isConfigured } = useConfig();
  const [name, setName] = useState("hello-world");
  const [bodyStr, setBodyStr] = useState("");
  const [result, setResult] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleInvoke(e: React.FormEvent) {
    e.preventDefault();
    if (!isConfigured) return;
    setLoading(true);
    setResult(null);
    setError(null);
    try {
      const client = createClient({ baseUrl, apiKey });
      let body: unknown = undefined;
      if (bodyStr.trim()) {
        try {
          body = JSON.parse(bodyStr) as unknown;
        } catch {
          setError("Invalid JSON body");
          setLoading(false);
          return;
        }
      }
      const data = await client.functions.invoke(name, body);
      setResult(JSON.stringify(data, null, 2));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Invoke failed");
    } finally {
      setLoading(false);
    }
  }

  if (!isConfigured) {
    return (
      <div className="rounded-lg border bg-amber-50 p-4 text-amber-800">
        Set your API key in Settings to invoke functions.
      </div>
    );
  }

  return (
    <div className="rounded-lg border bg-white p-6 shadow-sm">
      <h1 className="mb-4 text-xl font-semibold">Invoke function</h1>
      <p className="mb-4 text-sm text-gray-600">
        Uses the SDK <code className="rounded bg-gray-100 px-1">client.functions.invoke(name, body)</code>. Deploy functions with the CLI (see README).
      </p>

      <form onSubmit={handleInvoke} className="mb-4 flex flex-col gap-3">
        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">Function name</span>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="rounded border px-3 py-2 font-mono"
            placeholder="hello-world"
          />
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">Request body (optional JSON)</span>
          <textarea
            value={bodyStr}
            onChange={(e) => setBodyStr(e.target.value)}
            className="min-h-[80px] rounded border px-3 py-2 font-mono text-sm"
            placeholder='{"name": "World"}'
          />
        </label>
        <button
          type="submit"
          disabled={loading}
          className="w-fit rounded bg-blue-600 px-4 py-2 text-white disabled:opacity-50"
        >
          {loading ? "Invoking…" : "Invoke"}
        </button>
      </form>

      {error && <p className="mb-2 text-sm text-red-600">{error}</p>}
      {result !== null && (
        <pre className="overflow-auto rounded border bg-gray-50 p-3 text-sm">{result}</pre>
      )}
    </div>
  );
}

import { useState, useEffect, useCallback, useMemo } from "react";
import { useConfig } from "@/contexts/ConfigContext";
import { createClient } from "@/lib/client";

export function KV() {
  const { baseUrl, apiKey, isConfigured } = useConfig();
  const [keys, setKeys] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [keyInput, setKeyInput] = useState("");
  const [valueInput, setValueInput] = useState("");
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [selectedValue, setSelectedValue] = useState<string | null>(null);

  // Reuse client instance when config changes
  const client = useMemo(() => {
    if (!isConfigured) return null;
    try {
      return createClient({ baseUrl, apiKey });
    } catch {
      return null;
    }
  }, [isConfigured, baseUrl, apiKey]);

  const listKeys = useCallback(async () => {
    if (!client) return;
    setLoading(true);
    setError(null);
    try {
      const list = await client.kv.keys();
      setKeys(Array.isArray(list) ? list : []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to list keys");
    } finally {
      setLoading(false);
    }
  }, [client]);

  useEffect(() => {
    if (client) listKeys();
  }, [client, listKeys]);

  async function handleGet(key: string) {
    if (!client) return;
    setError(null);
    try {
      const value = await client.kv.get(key);
      setSelectedKey(key);
      setSelectedValue(value);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to get value");
    }
  }

  async function handleSet(e: React.FormEvent) {
    e.preventDefault();
    if (!client || !keyInput.trim()) return;
    setError(null);
    try {
      await client.kv.set(keyInput.trim(), valueInput);
      setKeyInput("");
      setValueInput("");
      setSelectedKey(null);
      setSelectedValue(null);
      await listKeys();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to set value");
    }
  }

  async function handleDelete(key: string) {
    if (!client) return;
    setError(null);
    try {
      await client.kv.delete(key);
      if (selectedKey === key) {
        setSelectedKey(null);
        setSelectedValue(null);
      }
      await listKeys();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to delete");
    }
  }

  if (!isConfigured) {
    return (
      <div className="rounded-lg border bg-amber-50 p-4 text-amber-800">
        Set your Project API key in Settings to use KV.
      </div>
    );
  }

  return (
    <div className="rounded-lg border bg-white p-6 shadow-sm">
      <div className="mb-4">
        <h1 className="text-xl font-semibold">KV (Bunder)</h1>
        <p className="mt-1 text-sm text-gray-600">
          Project-scoped key-value store. Get, set, delete keys. Uses the BunBase SDK.
        </p>
      </div>

      <form onSubmit={handleSet} className="mb-6 flex flex-wrap items-end gap-3">
        <label className="flex flex-1 min-w-[120px] flex-col gap-1">
          <span className="text-sm font-medium">Key</span>
          <input
            type="text"
            value={keyInput}
            onChange={(e) => setKeyInput(e.target.value)}
            placeholder="mykey"
            className="rounded border px-3 py-2 font-mono text-sm"
          />
        </label>
        <label className="flex flex-1 min-w-[120px] flex-col gap-1">
          <span className="text-sm font-medium">Value</span>
          <input
            type="text"
            value={valueInput}
            onChange={(e) => setValueInput(e.target.value)}
            placeholder="value"
            className="rounded border px-3 py-2 font-mono text-sm"
          />
        </label>
        <button type="submit" className="rounded bg-blue-600 px-4 py-2 text-white">
          Set
        </button>
      </form>

      {error && <p className="mb-2 text-sm text-red-600">{error}</p>}

      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-sm font-medium text-gray-700">Keys</h2>
        <button
          type="button"
          onClick={listKeys}
          disabled={loading || !isConfigured}
          className="rounded border px-2 py-1 text-sm text-gray-700 hover:bg-gray-50 disabled:opacity-50"
        >
          {loading ? "Loading…" : "Refresh"}
        </button>
      </div>

      {loading && keys.length === 0 ? (
        <p className="text-sm text-gray-500">Loading keys…</p>
      ) : keys.length === 0 ? (
        <p className="text-sm text-gray-500">No keys. Set a key above.</p>
      ) : (
        <ul className="space-y-2">
          {keys.map((k) => (
            <li
              key={k}
              className="flex items-center justify-between rounded border px-3 py-2"
            >
              <button
                type="button"
                onClick={() => handleGet(k)}
                className="flex-1 text-left font-mono text-sm text-blue-600 hover:underline"
              >
                {k}
              </button>
              <button
                type="button"
                onClick={() => handleDelete(k)}
                className="ml-2 text-sm text-red-600 hover:underline"
              >
                Delete
              </button>
            </li>
          ))}
        </ul>
      )}

      {selectedKey !== null && (
        <div className="mt-6 rounded border bg-gray-50 p-4">
          <h3 className="text-sm font-medium text-gray-700">Value for &quot;{selectedKey}&quot;</h3>
          <pre className="mt-2 overflow-auto rounded bg-white p-2 text-sm">
            {selectedValue ?? "(not found)"}
          </pre>
        </div>
      )}
    </div>
  );
}

import { useState, useEffect, useCallback, useMemo } from "react";
import { useConfig } from "@/contexts/ConfigContext";
import { createClient } from "@/lib/client";
import type { StorageObjectInfo } from "bunbase-js";

export function Storage() {
  const { baseUrl, apiKey, isConfigured } = useConfig();
  const [objects, setObjects] = useState<StorageObjectInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [prefix, setPrefix] = useState("");
  const [uploadKey, setUploadKey] = useState("");
  const [uploadFile, setUploadFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);

  const client = useMemo(() => {
    if (!isConfigured) return null;
    try {
      return createClient({ baseUrl, apiKey });
    } catch {
      return null;
    }
  }, [isConfigured, baseUrl, apiKey]);

  const listObjects = useCallback(async () => {
    if (!client) return;
    setLoading(true);
    setError(null);
    try {
      const list = await client.storage.list(prefix.trim() || undefined);
      setObjects(list);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to list objects");
      setObjects([]);
    } finally {
      setLoading(false);
    }
  }, [client, prefix]);

  useEffect(() => {
    if (client) listObjects();
  }, [client, listObjects]);

  async function handleUpload(e: React.FormEvent) {
    e.preventDefault();
    if (!client || !uploadKey.trim() || !uploadFile) return;
    setUploading(true);
    setError(null);
    try {
      await client.storage.put(
        uploadKey.trim(),
        uploadFile,
        uploadFile.type || undefined,
      );
      setUploadKey("");
      setUploadFile(null);
      await listObjects();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Upload failed");
    } finally {
      setUploading(false);
    }
  }

  async function handleDownload(key: string) {
    if (!client) return;
    setError(null);
    try {
      const blob = await client.storage.getBlob(key);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = key.split("/").pop() || key;
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Download failed");
    }
  }

  async function handleDelete(key: string) {
    if (!client) return;
    setError(null);
    try {
      await client.storage.delete(key);
      await listObjects();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Delete failed");
    }
  }

  if (!isConfigured) {
    return (
      <div className="rounded-lg border bg-amber-50 p-4 text-amber-800">
        Set your Project API key in Settings to use Storage.
      </div>
    );
  }

  return (
    <div className="rounded-lg border bg-white p-6 shadow-sm">
      <div className="mb-4">
        <h1 className="text-xl font-semibold">Storage (MinIO)</h1>
        <p className="mt-1 text-sm text-gray-600">
          Project-scoped file storage. One bucket per project. Upload, list,
          download, delete objects.
        </p>
      </div>

      <form
        onSubmit={handleUpload}
        className="mb-6 flex flex-wrap items-end gap-3"
      >
        <label className="flex min-w-[180px] flex-col gap-1">
          <span className="text-sm font-medium">Object key</span>
          <input
            type="text"
            value={uploadKey}
            onChange={(e) => setUploadKey(e.target.value)}
            placeholder="e.g. folder/file.txt"
            className="rounded border px-3 py-2 font-mono text-sm"
            required
          />
        </label>
        <label className="flex min-w-[180px] flex-col gap-1">
          <span className="text-sm font-medium">File</span>
          <input
            type="file"
            onChange={(e) => setUploadFile(e.target.files?.[0] ?? null)}
            className="rounded border px-3 py-2 text-sm"
          />
        </label>
        <button
          type="submit"
          disabled={uploading || !uploadKey.trim() || !uploadFile}
          className="rounded bg-blue-600 px-4 py-2 text-white disabled:opacity-50"
        >
          {uploading ? "Uploading…" : "Upload"}
        </button>
      </form>

      {error && <p className="mb-2 text-sm text-red-600">{error}</p>}

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <label className="flex items-center gap-2">
          <span className="text-sm font-medium">Prefix</span>
          <input
            type="text"
            value={prefix}
            onChange={(e) => setPrefix(e.target.value)}
            placeholder="optional filter"
            className="rounded border px-2 py-1 font-mono text-sm"
          />
        </label>
        <button
          type="button"
          onClick={listObjects}
          disabled={loading}
          className="rounded border px-2 py-1 text-sm text-gray-700 hover:bg-gray-50 disabled:opacity-50"
        >
          {loading ? "Loading…" : "Refresh"}
        </button>
      </div>

      <h2 className="mb-2 text-sm font-medium text-gray-700">Objects</h2>
      {loading && objects.length === 0 ? (
        <p className="text-sm text-gray-500">Loading…</p>
      ) : objects.length === 0 ? (
        <p className="text-sm text-gray-500">
          No objects. Upload a file above.
        </p>
      ) : (
        <ul className="space-y-2">
          {objects.map((obj) => (
            <li
              key={obj.key}
              className="flex items-center justify-between rounded border px-3 py-2"
            >
              <div className="min-w-0 flex-1">
                <button
                  type="button"
                  onClick={() => handleDownload(obj.key)}
                  className="truncate text-left font-mono text-sm text-blue-600 hover:underline"
                >
                  {obj.key}
                </button>
                {typeof obj.size === "number" && (
                  <span className="ml-2 text-xs text-gray-500">
                    {obj.size} bytes
                  </span>
                )}
              </div>
              <div className="flex shrink-0 gap-2">
                <button
                  type="button"
                  onClick={() => handleDownload(obj.key)}
                  className="text-sm text-blue-600 hover:underline"
                >
                  Download
                </button>
                <button
                  type="button"
                  onClick={() => handleDelete(obj.key)}
                  className="text-sm text-red-600 hover:underline"
                >
                  Delete
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

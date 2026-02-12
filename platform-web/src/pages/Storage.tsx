import { useState, useEffect, useCallback } from "react";
import { useParams } from "react-router-dom";
import { api } from "../lib/api";

interface StorageObjectInfo {
  key: string;
  size: number;
  last_modified?: string;
}

export function Storage() {
  const { id: projectId } = useParams<{ id: string }>();
  const [objects, setObjects] = useState<StorageObjectInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [prefix, setPrefix] = useState("");
  const [uploadKey, setUploadKey] = useState("");
  const [uploadFile, setUploadFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);

  const listObjects = useCallback(async () => {
    if (!projectId) return;
    setLoading(true);
    setError(null);
    try {
      const data = await api.listStorageObjects(
        projectId,
        prefix.trim() || undefined,
      );
      setObjects(data.objects ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to list objects");
      setObjects([]);
    } finally {
      setLoading(false);
    }
  }, [projectId, prefix]);

  useEffect(() => {
    if (projectId) listObjects();
  }, [projectId, listObjects]);

  async function handleUpload(e: React.FormEvent) {
    e.preventDefault();
    if (!projectId || !uploadKey.trim() || !uploadFile) return;
    setUploading(true);
    setError(null);
    try {
      await api.putStorageObject(
        projectId,
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
    if (!projectId) return;
    setError(null);
    try {
      const blob = await api.getStorageObjectBlob(projectId, key);
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
    if (!projectId) return;
    setError(null);
    try {
      await api.deleteStorageObject(projectId, key);
      await listObjects();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Delete failed");
    }
  }

  if (!projectId) {
    return <div className="p-6 text-base-content/70">No project selected.</div>;
  }

  return (
    <div className="flex-1 min-h-0 flex flex-col p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-base-content">Storage</h1>
        <p className="mt-1 text-sm text-base-content/70">
          Project-scoped file storage (one bucket per project). Upload, list,
          download, and delete objects.
        </p>
      </div>

      <div className="card bg-base-200/50 border border-base-300 flex-1 min-h-0 flex flex-col">
        <div className="card-body flex-1 min-h-0 flex flex-col">
          <form
            onSubmit={handleUpload}
            className="flex flex-wrap items-end gap-4 mb-6"
          >
            <div className="form-control min-w-[200px]">
              <label className="label">
                <span className="label-text">Object key</span>
              </label>
              <input
                type="text"
                value={uploadKey}
                onChange={(e) => setUploadKey(e.target.value)}
                placeholder="e.g. folder/file.txt"
                className="input input-bordered w-full font-mono text-sm"
                required
              />
            </div>
            <div className="form-control min-w-[200px]">
              <label className="label">
                <span className="label-text">File</span>
              </label>
              <input
                type="file"
                onChange={(e) => setUploadFile(e.target.files?.[0] ?? null)}
                className="file-input file-input-bordered w-full"
              />
            </div>
            <button
              type="submit"
              disabled={uploading || !uploadKey.trim() || !uploadFile}
              className="btn btn-primary"
            >
              {uploading ? "Uploading…" : "Upload"}
            </button>
          </form>

          {error && (
            <div className="alert alert-error mb-4">
              <span>{error}</span>
            </div>
          )}

          <div className="flex flex-wrap items-center gap-4 mb-4">
            <div className="form-control">
              <label className="label py-0">
                <span className="label-text text-sm">Prefix filter</span>
              </label>
              <input
                type="text"
                value={prefix}
                onChange={(e) => setPrefix(e.target.value)}
                placeholder="optional"
                className="input input-bordered input-sm w-48 font-mono"
              />
            </div>
            <button
              type="button"
              onClick={listObjects}
              disabled={loading}
              className="btn btn-sm btn-ghost"
            >
              {loading ? "Loading…" : "Refresh"}
            </button>
          </div>

          <h2 className="text-sm font-medium text-base-content/70 mb-2">
            Objects
          </h2>
          {loading && objects.length === 0 ? (
            <div className="flex items-center justify-center py-12">
              <span className="loading loading-spinner loading-md text-primary" />
            </div>
          ) : objects.length === 0 ? (
            <p className="text-sm text-base-content/50 py-6">
              No objects. Upload a file above.
            </p>
          ) : (
            <div className="overflow-auto flex-1 min-h-0">
              <table className="table table-sm">
                <thead>
                  <tr>
                    <th>Key</th>
                    <th>Size</th>
                    <th className="w-40">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {objects.map((obj) => (
                    <tr key={obj.key}>
                      <td
                        className="font-mono text-sm max-w-[300px] truncate"
                        title={obj.key}
                      >
                        {obj.key}
                      </td>
                      <td className="text-base-content/70 text-sm">
                        {typeof obj.size === "number"
                          ? `${obj.size} bytes`
                          : "—"}
                      </td>
                      <td>
                        <div className="flex gap-2">
                          <button
                            type="button"
                            onClick={() => handleDownload(obj.key)}
                            className="btn btn-xs btn-ghost"
                          >
                            Download
                          </button>
                          <button
                            type="button"
                            onClick={() => handleDelete(obj.key)}
                            className="btn btn-xs btn-ghost text-error"
                          >
                            Delete
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

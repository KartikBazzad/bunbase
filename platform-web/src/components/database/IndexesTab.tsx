import { useState, useEffect } from "react";
import { api } from "../../lib/api";
import { Trash2, Plus, RefreshCw } from "lucide-react";

interface IndexesTabProps {
  projectId: string;
  collection: string;
}

export function IndexesTab({ projectId, collection }: IndexesTabProps) {
  const [indexes, setIndexes] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [newIndexField, setNewIndexField] = useState("");
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    loadIndexes();
  }, [projectId, collection]);

  const loadIndexes = async () => {
    setLoading(true);
    setError("");
    try {
      const data: any = await api.listIndexes(projectId, collection);
      // Backend returns { indexes: [...] } or just array?
      // HandleIndexOperations (GET) -> returns { indexes: [...] }
      setIndexes(data.indexes || []);
    } catch (err) {
      setError("Failed to load indexes");
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newIndexField) return;
    setCreating(true);
    try {
      await api.createIndex(projectId, collection, newIndexField);
      setNewIndexField("");
      loadIndexes();
    } catch (err) {
      alert("Failed to create index: " + err);
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (field: string) => {
    if (!confirm(`Delete index on '${field}'?`)) return;
    try {
      await api.deleteIndex(projectId, collection, field);
      loadIndexes();
    } catch (err) {
      alert("Failed to delete index");
    }
  };

  return (
    <div className="p-6 h-full flex flex-col min-h-0 overflow-hidden">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h3 className="font-bold text-lg">Indexes</h3>
          <p className="text-sm text-base-content/60">
            Optimize query performance by indexing fields.
          </p>
        </div>
        <button
          className="btn btn-ghost btn-sm btn-square"
          onClick={loadIndexes}
        >
          <RefreshCw className={`w-4 h-4 ${loading ? "animate-spin" : ""}`} />
        </button>
      </div>

      {error && <div className="alert alert-error mb-4">{error}</div>}

      <div className="flex-1 min-h-0 overflow-auto">
        <table className="table">
          <thead>
            <tr>
              <th>Field</th>
              <th className="w-20 text-right">Action</th>
            </tr>
          </thead>
          <tbody>
            {indexes.map((idx) => (
              <tr key={idx}>
                <td className="font-mono text-sm">{idx}</td>
                <td className="text-right">
                  <button
                    className="btn btn-ghost btn-xs text-error"
                    onClick={() => handleDelete(idx)}
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </td>
              </tr>
            ))}
            {indexes.length === 0 && !loading && (
              <tr>
                <td
                  colSpan={2}
                  className="text-center text-base-content/50 py-8"
                >
                  No indexes found.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <div className="mt-4 pt-4 border-t border-base-300">
        <form onSubmit={handleCreate} className="flex gap-2">
          <input
            type="text"
            className="input input-bordered input-sm flex-1"
            placeholder="Field name (e.g. email)"
            value={newIndexField}
            onChange={(e) => setNewIndexField(e.target.value)}
          />
          <button
            type="submit"
            className="btn btn-primary btn-sm"
            disabled={creating || !newIndexField}
          >
            <Plus className="w-4 h-4 mr-1" />
            Create Index
          </button>
        </form>
      </div>
    </div>
  );
}

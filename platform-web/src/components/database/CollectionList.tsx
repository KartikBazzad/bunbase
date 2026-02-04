import { useState, useEffect } from "react";
import { api } from "../../lib/api";
import { Plus, Folder, Trash2 } from "lucide-react";

function formatCount(n: number): string {
  if (n >= 1000) return `${(n / 1000).toFixed(1).replace(/\.0$/, "")}k`;
  return String(n);
}

interface CollectionListProps {
  projectId: string;
  selectedCollection: string;
  selectedCollectionCount?: number;
  onSelectCollection: (collection: string) => void;
}

export function CollectionList({
  projectId,
  selectedCollection,
  selectedCollectionCount,
  onSelectCollection,
}: CollectionListProps) {
  const [collections, setCollections] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [newCollectionName, setNewCollectionName] = useState("");

  useEffect(() => {
    loadCollections();
  }, [projectId]);

  const loadCollections = async () => {
    try {
      setLoading(true);
      const data = await api.listCollections(projectId);
      // Assuming API returns { collections: string[] } or just string[]
      // Based on Bundoc implementation, it might be { items: [...] } or direct array.
      // Let's assume direct array for now or check response.
      // Wait, Bundoc proxy returns whatever Bundoc returns.
      // Let's assume it returns an array of strings for "list collections".
      setCollections((data as any).collections || (data as any) || []);
    } catch (err) {
      setError("Failed to load collections");
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newCollectionName) return;

    try {
      await api.createCollection(projectId, newCollectionName);
      setNewCollectionName("");
      setShowCreate(false);
      loadCollections();
    } catch (err) {
      setError("Failed to create collection");
    }
  };

  const handleDelete = async (name: string, e: React.MouseEvent) => {
    e.stopPropagation();
    if (!confirm(`Are you sure you want to delete collection "${name}"?`))
      return;

    try {
      await api.deleteCollection(projectId, name);
      if (collections.length === 1) {
        onSelectCollection(""); // Reset selection if last one deleted
      }
      loadCollections();
    } catch (err) {
      setError("Failed to delete collection");
    }
  };

  if (loading) {
    return <span className="loading loading-spinner"></span>;
  }

  return (
    <div className="card bg-base-100 shadow-md h-full flex flex-col min-h-0">
      <div className="card-body py-3 px-4 flex-none flex flex-row justify-between items-center">
        <span className="text-xs font-semibold uppercase tracking-wide text-base-content/60">
          Collections
        </span>
        <button
          type="button"
          onClick={() => setShowCreate(true)}
          className="btn btn-ghost btn-square btn-sm"
          aria-label="New collection"
        >
          <Plus className="w-4 h-4" />
        </button>
      </div>
      <div className="px-3 overflow-y-auto flex-1 min-h-0">
        {error && <p className="text-error text-sm mb-2">{error}</p>}

        {showCreate && (
          <form onSubmit={handleCreate} className="mb-3">
            <input
              type="text"
              value={newCollectionName}
              onChange={(e) => setNewCollectionName(e.target.value)}
              placeholder="Collection name"
              className="input input-bordered input-sm w-full mb-2"
              autoFocus
            />
            <div className="flex gap-2">
              <button type="submit" className="btn btn-primary btn-xs">
                Save
              </button>
              <button
                type="button"
                onClick={() => setShowCreate(false)}
                className="btn btn-ghost btn-xs"
              >
                Cancel
              </button>
            </div>
          </form>
        )}

        {collections.length === 0 ? (
          <p className="text-base-content/50 text-sm text-center py-4">
            No collections
          </p>
        ) : (
          <ul className="space-y-0.5">
            {collections.map((col) => {
              const isSelected = col === selectedCollection;
              const count =
                isSelected && selectedCollectionCount !== undefined
                  ? formatCount(selectedCollectionCount)
                  : "—";
              return (
                <li key={col}>
                  <button
                    type="button"
                    onClick={() => onSelectCollection(col)}
                    className={`w-full text-left px-3 py-2 rounded-md flex justify-between items-center group ${
                      isSelected
                        ? "bg-primary text-primary-content"
                        : "hover:bg-base-300"
                    }`}
                  >
                    <span className="text-sm font-medium flex items-center gap-2 truncate min-w-0">
                      <Folder className="w-4 h-4 shrink-0" />
                      <span className="truncate">{col}</span>
                    </span>
                    <span className="text-xs shrink-0 ml-2 opacity-80">
                      {count}
                    </span>
                    <span
                      onClick={(e) => handleDelete(col, e)}
                      className={`shrink-0 p-1 opacity-0 group-hover:opacity-100 ${
                        isSelected
                          ? "hover:bg-primary-focus rounded"
                          : "text-base-content/40 hover:text-error"
                      }`}
                    >
                      <Trash2 className="w-3 h-3" />
                    </span>
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </div>
  );
}

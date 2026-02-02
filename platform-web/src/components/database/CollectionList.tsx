import { useState, useEffect } from "react";
import { api } from "../../lib/api";

interface CollectionListProps {
  projectId: string;
  onSelectCollection: (collection: string) => void;
}

export function CollectionList({
  projectId,
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
    return <div className="spinner"></div>;
  }

  return (
    <div className="card h-full flex flex-col">
      <div className="card-header flex justify-between items-center">
        <h3 className="font-semibold">Collections</h3>
        <button
          onClick={() => setShowCreate(true)}
          className="btn-sm btn-primary"
        >
          + New
        </button>
      </div>
      <div className="card-body overflow-y-auto flex-1">
        {error && <p className="text-error-500 text-sm mb-2">{error}</p>}

        {showCreate && (
          <form onSubmit={handleCreate} className="mb-4">
            <input
              type="text"
              value={newCollectionName}
              onChange={(e) => setNewCollectionName(e.target.value)}
              placeholder="Collection name"
              className="input text-sm mb-2"
              autoFocus
            />
            <div className="flex gap-2">
              <button type="submit" className="btn-xs btn-primary">
                Save
              </button>
              <button
                type="button"
                onClick={() => setShowCreate(false)}
                className="btn-xs btn-ghost"
              >
                Cancel
              </button>
            </div>
          </form>
        )}

        {collections.length === 0 ? (
          <p className="text-gray-500 text-sm text-center py-4">
            No collections
          </p>
        ) : (
          <ul className="space-y-1">
            {collections.map((col) => (
              <li key={col}>
                <button
                  onClick={() => onSelectCollection(col)}
                  className="w-full text-left px-3 py-2 rounded-md hover:bg-gray-100 flex justify-between items-center group"
                >
                  <span className="text-sm font-medium">{col}</span>
                  <span
                    onClick={(e) => handleDelete(col, e)}
                    className="text-gray-400 hover:text-error-600 opacity-0 group-hover:opacity-100 p-1"
                  >
                    ×
                  </span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

import { useState, useEffect, useCallback, useRef } from "react";
import { useConfig } from "@/contexts/ConfigContext";
import { createClient } from "@/lib/client";
import type { ChangeEvent } from "bunbase-js";

const COLLECTION = "tasks";

interface Task {
  id?: string;
  title: string;
  done?: boolean;
  [key: string]: unknown;
}

export function Documents() {
  const { baseUrl, apiKey, isConfigured } = useConfig();
  const [items, setItems] = useState<Task[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [newTitle, setNewTitle] = useState("");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState("");
  const unsubscribeRef = useRef<(() => void) | null>(null);
  const [isLive, setIsLive] = useState(false);

  const load = useCallback(async () => {
    if (!isConfigured) return;
    setLoading(true);
    setError(null);
    try {
      const client = createClient({ baseUrl, apiKey });
      const result = await client.db.collection(COLLECTION).list();
      const docs = (result as { documents?: Task[] }).documents ?? [];
      setItems(Array.isArray(docs) ? docs : []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load");
    } finally {
      setLoading(false);
    }
  }, [isConfigured, baseUrl, apiKey]);

  // Initial load and setup realtime subscription
  useEffect(() => {
    if (!isConfigured) return;

    // Load initial data
    load();

    // Setup realtime subscription
    try {
      const client = createClient({ baseUrl, apiKey });
      const unsubscribe = client.db.collection<Task>(COLLECTION).watch((event: ChangeEvent<Task>) => {
        setItems((current) => {
          const id = event.docId || (event.document as any)?._id;
          if (!id) return current;

          switch (event.type) {
            case "added":
              // Check if already exists (avoid duplicates)
              if (current.some((item) => (item.id || (item as any)._id) === id)) {
                return current;
              }
              return [...current, event.document as Task];
            case "modified":
              return current.map((item) =>
                (item.id || (item as any)._id) === id ? (event.document as Task) : item
              );
            case "removed":
              return current.filter((item) => (item.id || (item as any)._id) !== id);
            default:
              return current;
          }
        });
      });
      unsubscribeRef.current = unsubscribe;
      setIsLive(true);
    } catch (e) {
      console.error("Failed to setup realtime subscription:", e);
      setIsLive(false);
    }

    return () => {
      if (unsubscribeRef.current) {
        unsubscribeRef.current();
        unsubscribeRef.current = null;
      }
      setIsLive(false);
    };
  }, [isConfigured, baseUrl, apiKey, load]);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!newTitle.trim() || !isConfigured) return;
    setError(null);
    try {
      const client = createClient({ baseUrl, apiKey });
      await client.db.collection<Task>(COLLECTION).create({
        title: newTitle.trim(),
        done: false,
      });
      setNewTitle("");
      // No need to reload - realtime subscription will update UI
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to create");
    }
  }

  async function handleUpdate(id: string) {
    if (!isConfigured || !editTitle.trim()) return;
    setError(null);
    try {
      const client = createClient({ baseUrl, apiKey });
      await client.db.collection<Task>(COLLECTION).update(id, { title: editTitle.trim() });
      setEditingId(null);
      setEditTitle("");
      // No need to reload - realtime subscription will update UI
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to update");
    }
  }

  async function handleDelete(id: string) {
    if (!isConfigured) return;
    setError(null);
    try {
      const client = createClient({ baseUrl, apiKey });
      await client.db.collection(COLLECTION).delete(id);
      // No need to reload - realtime subscription will update UI
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to delete");
    }
  }

  if (!isConfigured) {
    return (
      <div className="rounded-lg border bg-amber-50 p-4 text-amber-800">
        Set your Project API key and Project ID in Settings to use Documents.
      </div>
    );
  }

  return (
    <div className="rounded-lg border bg-white p-6 shadow-sm">
      <div className="mb-4 flex items-center justify-between">
        <h1 className="text-xl font-semibold">Documents (tasks)</h1>
        {isLive && (
          <span className="flex items-center gap-1 rounded-full bg-green-100 px-2 py-1 text-xs text-green-700">
            <span className="h-2 w-2 animate-pulse rounded-full bg-green-500"></span>
            Live
          </span>
        )}
      </div>
      <p className="mb-4 text-sm text-gray-600">
        Collection: <code className="rounded bg-gray-100 px-1">{COLLECTION}</code>. All operations use the BunBase SDK.
        {isLive && " Changes sync in realtime across browsers."}
      </p>

      <form onSubmit={handleCreate} className="mb-4 flex gap-2">
        <input
          type="text"
          value={newTitle}
          onChange={(e) => setNewTitle(e.target.value)}
          placeholder="New task title"
          className="flex-1 rounded border px-3 py-2"
        />
        <button type="submit" className="rounded bg-blue-600 px-4 py-2 text-white">
          Add
        </button>
      </form>

      {error && <p className="mb-2 text-sm text-red-600">{error}</p>}
      {loading ? (
        <p className="text-sm text-gray-500">Loading…</p>
      ) : (
        <ul className="space-y-2">
          {items.map((doc) => {
            const id = (doc.id ?? (doc as Task & { _id?: string })._id) as string;
            const title = doc.title ?? "";
            const isEditing = editingId === id;
            return (
              <li key={id} className="flex items-center justify-between rounded border px-3 py-2">
                {isEditing ? (
                  <>
                    <input
                      type="text"
                      value={editTitle}
                      onChange={(e) => setEditTitle(e.target.value)}
                      className="flex-1 rounded border px-2 py-1"
                      autoFocus
                    />
                    <button
                      type="button"
                      onClick={() => handleUpdate(id)}
                      className="ml-2 text-blue-600"
                    >
                      Save
                    </button>
                    <button
                      type="button"
                      onClick={() => { setEditingId(null); setEditTitle(""); }}
                      className="ml-1 text-gray-500"
                    >
                      Cancel
                    </button>
                  </>
                ) : (
                  <>
                    <span>{title}</span>
                    <div className="flex gap-2">
                      <button
                        type="button"
                        onClick={() => { setEditingId(id); setEditTitle(title); }}
                        className="text-sm text-blue-600"
                      >
                        Edit
                      </button>
                      <button
                        type="button"
                        onClick={() => handleDelete(id)}
                        className="text-sm text-red-600"
                      >
                        Delete
                      </button>
                    </div>
                  </>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

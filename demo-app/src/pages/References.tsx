import { useState, useEffect, useCallback } from "react";
import { useConfig } from "@/contexts/ConfigContext";
import { createClient } from "@/lib/client";

const USERS_COLLECTION = "ref_users";
const POSTS_COLLECTION = "ref_posts";

type OnDeletePolicy = "restrict" | "set_null" | "cascade";

interface RefUser {
  _id?: string;
  name?: string;
  [key: string]: unknown;
}

interface RefPost {
  _id?: string;
  title?: string;
  author_id?: string | null;
  [key: string]: unknown;
}

function postsSchemaWithRef(onDelete: OnDeletePolicy): object {
  const authorType = onDelete === "set_null" ? ["string", "null"] : "string";
  return {
    type: "object",
    properties: {
      title: { type: "string" },
      author_id: {
        type: authorType,
        "x-bundoc-ref": {
          collection: USERS_COLLECTION,
          field: "_id",
          on_delete: onDelete,
        },
      },
    },
  };
}

const userSchema: object = {
  type: "object",
  properties: {
    name: { type: "string" },
  },
};

export function References() {
  const { baseUrl, apiKey, isConfigured } = useConfig();
  const [onDelete, setOnDelete] = useState<OnDeletePolicy>("set_null");
  const [users, setUsers] = useState<RefUser[]>([]);
  const [posts, setPosts] = useState<RefPost[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [newUserName, setNewUserName] = useState("");
  const [newPostTitle, setNewPostTitle] = useState("");
  const [newPostAuthorId, setNewPostAuthorId] = useState<string>("");
  const [invalidRef, setInvalidRef] = useState(false);

  const load = useCallback(async () => {
    if (!isConfigured) return;
    setLoading(true);
    setError(null);
    try {
      const client = createClient({ baseUrl, apiKey });
      const usersRes = await client.db.collection(USERS_COLLECTION).list();
      const postsRes = await client.db.collection(POSTS_COLLECTION).list();
      const userDocs = (usersRes as { documents?: RefUser[] }).documents ?? [];
      const postDocs = (postsRes as { documents?: RefPost[] }).documents ?? [];
      setUsers(Array.isArray(userDocs) ? userDocs : []);
      setPosts(Array.isArray(postDocs) ? postDocs : []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load");
      setUsers([]);
      setPosts([]);
    } finally {
      setLoading(false);
    }
  }, [isConfigured, baseUrl, apiKey]);

  useEffect(() => {
    if (!isConfigured) return;
    load();
  }, [isConfigured, load]);

  async function handleApplySchema() {
    if (!isConfigured) return;
    setError(null);
    try {
      const client = createClient({ baseUrl, apiKey });
      await client.db.createCollection(USERS_COLLECTION, userSchema, {
        updateIfExists: true,
      });
      await client.db.createCollection(POSTS_COLLECTION, postsSchemaWithRef(onDelete), {
        updateIfExists: true,
      });
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to apply schema");
    }
  }

  async function handleResetDemo() {
    if (!isConfigured) return;
    setError(null);
    try {
      const client = createClient({ baseUrl, apiKey });
      try {
        await client.db.deleteCollection(POSTS_COLLECTION);
      } catch (_) {}
      try {
        await client.db.deleteCollection(USERS_COLLECTION);
      } catch (_) {}
      await client.db.createCollection(USERS_COLLECTION, userSchema);
      await client.db.createCollection(POSTS_COLLECTION, postsSchemaWithRef(onDelete));
      setUsers([]);
      setPosts([]);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to reset demo");
    }
  }

  async function handleAddUser(e: React.FormEvent) {
    e.preventDefault();
    if (!newUserName.trim() || !isConfigured) return;
    setError(null);
    try {
      const client = createClient({ baseUrl, apiKey });
      await client.db.collection<RefUser>(USERS_COLLECTION).create({
        _id: `user-${Date.now()}`,
        name: newUserName.trim(),
      });
      setNewUserName("");
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to add user");
    }
  }

  async function handleAddPost(e: React.FormEvent) {
    e.preventDefault();
    if (!newPostTitle.trim() || !isConfigured) return;
    setError(null);
    const authorId = invalidRef ? "nonexistent-id" : newPostAuthorId;
    if (!invalidRef && !authorId) {
      setError("Select an author or check 'Use invalid reference (409)'");
      return;
    }
    try {
      const client = createClient({ baseUrl, apiKey });
      await client.db.collection<RefPost>(POSTS_COLLECTION).create({
        _id: `post-${Date.now()}`,
        title: newPostTitle.trim(),
        author_id: authorId || null,
      });
      setNewPostTitle("");
      setNewPostAuthorId("");
      setInvalidRef(false);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to add post");
    }
  }

  async function handleDeleteUser(userId: string) {
    if (!isConfigured) return;
    setError(null);
    try {
      const client = createClient({ baseUrl, apiKey });
      await client.db.collection(USERS_COLLECTION).delete(userId);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to delete user");
    }
  }

  if (!isConfigured) {
    return (
      <div className="rounded-lg border bg-amber-50 p-4 text-amber-800">
        Set your Project API key and Project ID in Settings to use References demo.
      </div>
    );
  }

  return (
    <div className="rounded-lg border bg-white p-6 shadow-sm">
      <h1 className="mb-2 text-xl font-semibold">Cross-Collection References</h1>
      <p className="mb-4 text-sm text-gray-600">
        This demo uses <code className="rounded bg-gray-100 px-1">ref_users</code> and{" "}
        <code className="rounded bg-gray-100 px-1">ref_posts</code>. Posts reference users via{" "}
        <code className="rounded bg-gray-100 px-1">author_id</code> with{" "}
        <code className="rounded bg-gray-100 px-1">x-bundoc-ref</code>. Choose{" "}
        <strong>on_delete</strong> and apply schema, then add users, add posts (or try an invalid reference to see 409), and delete a user to see restrict / set_null / cascade behavior.
      </p>

      {/* Setup */}
      <section className="mb-6 rounded border border-gray-200 bg-gray-50 p-4">
        <h2 className="mb-2 font-medium">Setup</h2>
        <div className="flex flex-wrap items-center gap-3">
          <label className="flex items-center gap-2">
            <span className="text-sm">on_delete:</span>
            <select
              value={onDelete}
              onChange={(e) => setOnDelete(e.target.value as OnDeletePolicy)}
              className="rounded border px-2 py-1 text-sm"
            >
              <option value="restrict">restrict</option>
              <option value="set_null">set_null</option>
              <option value="cascade">cascade</option>
            </select>
          </label>
          <button
            type="button"
            onClick={handleApplySchema}
            className="rounded bg-blue-600 px-3 py-1.5 text-sm text-white"
          >
            Apply schema
          </button>
          <button
            type="button"
            onClick={handleResetDemo}
            className="rounded border border-gray-400 px-3 py-1.5 text-sm text-gray-700"
          >
            Reset demo
          </button>
        </div>
      </section>

      {error && (
        <p className="mb-4 rounded border border-red-200 bg-red-50 p-2 text-sm text-red-700">
          {error}
        </p>
      )}

      {loading ? (
        <p className="text-sm text-gray-500">Loading…</p>
      ) : (
        <>
          {/* Users */}
          <section className="mb-6">
            <h2 className="mb-2 font-medium">Users</h2>
            <form onSubmit={handleAddUser} className="mb-2 flex gap-2">
              <input
                type="text"
                value={newUserName}
                onChange={(e) => setNewUserName(e.target.value)}
                placeholder="User name"
                className="flex-1 rounded border px-3 py-2 text-sm"
              />
              <button type="submit" className="rounded bg-blue-600 px-4 py-2 text-sm text-white">
                Add user
              </button>
            </form>
            <ul className="space-y-1">
              {users.map((u) => {
                const id = (u._id ?? (u as any).id) as string;
                return (
                  <li key={id} className="flex items-center justify-between rounded border px-3 py-2 text-sm">
                    <span>{u.name ?? id}</span>
                    <span className="text-gray-500 font-mono text-xs">{id}</span>
                    <button
                      type="button"
                      onClick={() => handleDeleteUser(id)}
                      className="text-red-600 text-xs"
                    >
                      Delete
                    </button>
                  </li>
                );
              })}
              {users.length === 0 && (
                <li className="text-sm text-gray-500">No users yet. Apply schema and add one.</li>
              )}
            </ul>
          </section>

          {/* Posts */}
          <section>
            <h2 className="mb-2 font-medium">Posts</h2>
            <form onSubmit={handleAddPost} className="mb-2 flex flex-wrap items-end gap-2">
              <input
                type="text"
                value={newPostTitle}
                onChange={(e) => setNewPostTitle(e.target.value)}
                placeholder="Post title"
                className="min-w-[120px] rounded border px-3 py-2 text-sm"
              />
              <label className="flex items-center gap-1 text-sm">
                Author:
                <select
                  value={newPostAuthorId}
                  onChange={(e) => setNewPostAuthorId(e.target.value)}
                  disabled={invalidRef}
                  className="rounded border px-2 py-1 text-sm"
                >
                  <option value="">-- select --</option>
                  {users.map((u) => {
                    const id = (u._id ?? (u as any).id) as string;
                    return (
                      <option key={id} value={id}>
                        {u.name ?? id}
                      </option>
                    );
                  })}
                </select>
              </label>
              <label className="flex items-center gap-1 text-sm">
                <input
                  type="checkbox"
                  checked={invalidRef}
                  onChange={(e) => setInvalidRef(e.target.checked)}
                />
                Use invalid reference (409)
              </label>
              <button type="submit" className="rounded bg-blue-600 px-4 py-2 text-sm text-white">
                Add post
              </button>
            </form>
            <ul className="space-y-1">
              {posts.map((p) => {
                const id = (p._id ?? (p as any).id) as string;
                const authorId = p.author_id ?? "(null)";
                return (
                  <li key={id} className="rounded border px-3 py-2 text-sm">
                    <span className="font-medium">{p.title ?? id}</span>
                    <span className="ml-2 text-gray-500">author_id: {String(authorId)}</span>
                  </li>
                );
              })}
              {posts.length === 0 && (
                <li className="text-sm text-gray-500">No posts yet. Add a user, then add a post.</li>
              )}
            </ul>
          </section>
        </>
      )}
    </div>
  );
}

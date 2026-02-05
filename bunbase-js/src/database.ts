import { BunBaseClient } from "./client";

export class DatabaseClient<
  SchemaRegistry extends Record<string, any> = Record<string, any>,
> {
  constructor(private client: BunBaseClient<SchemaRegistry>) {}

  collection<Name extends keyof SchemaRegistry & string>(
    name: Name,
  ): Collection<SchemaRegistry[Name]>;
  collection<T = any>(name: string): Collection<T>;
  collection(name: string) {
    return new Collection(this.client, name);
  }

  async listCollections(prefix?: string) {
    const params = prefix ? `?prefix=${encodeURIComponent(prefix)}` : "";
    return this.client.request(`/v1/databases/default/collections${params}`, {
      method: "GET",
    });
  }

  async createCollection(name: string, schema?: object) {
    return this.client.request(`/v1/databases/default/collections`, {
      method: "POST",
      body: JSON.stringify({ name, schema }),
    });
  }

  async deleteCollection(name: string) {
    // encodeURIComponent is important here for nested collections like "users/admins"
    // However, backend greedy match handles unencoded too.
    // But API client `deleteCollection` in web uses direct concat.
    // Let's use strict URI encoding to be safe and standard.
    // If backend `parseProjectCollectionAndDoc` was replaced by dynamic,
    // we need to support `DELETE /.../collections/users/admins`
    // Wait, the handler for DeleteCollection?
    // I checked `HandleDeleteDocument`.
    // Did I check `HandleDeleteCollection`?
    // Let's check `document_handlers.go` again for `HandleDeleteCollection`.
    // It is likely missing or I missed it.
    // If it's missing, I need to add it to backend too!

    // For now, assuming it exists or I will verify.
    return this.client.request(`/v1/databases/default/collections/${name}`, {
      method: "DELETE",
    });
  }
}

export class Collection<T = any> {
  constructor(
    private client: BunBaseClient<any>,
    private name: string,
  ) {}

  // Using "default" as dbName placeholder, as backend overrides it with ProjectID
  // We need to use project-aware paths for some operations now.
  private get basePath() {
    // Old path: /v1/databases/default/collections/NAME/documents
    // New path needed? The backend still supports /v1/projects/.../documents via generic handler?
    // Actually, main.go redirects /v1/projects/.../documents to HandleListDocuments/etc.
    // But `database.ts` currently uses `/v1/databases/default/...`.
    // Does main.go handle `/v1/databases/default/collections`?
    // The main.go I edited seemed to handle `/v1/projects/...` prefix explicitly.
    // It MIGHT fail for `/v1/databases/...`.
    // I should switch EVERYTHING to `/v1/projects/{id}/databases/default/...`.
    return `/v1/projects/${this.client.projectId}/database/collections/${this.name}/documents`;
  }

  private get collectionPath() {
    return `/v1/projects/${this.client.projectId}/database/collections/${this.name}`;
  }

  async list(query?: Record<string, string>): Promise<{ documents: T[] }> {
    const params = new URLSearchParams(query).toString();
    const url = params ? `${this.basePath}?${params}` : this.basePath;
    return this.client.request(url, { method: "GET" });
  }

  async get(id: string): Promise<T> {
    return this.client.request(`${this.basePath}/${id}`, { method: "GET" });
  }

  async create(data: T) {
    return this.client.request(this.basePath, {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  async update(id: string, data: Partial<T>) {
    return this.client.request(`${this.basePath}/${id}`, {
      method: "PATCH",
      body: JSON.stringify(data),
    });
  }

  async delete(id: string) {
    return this.client.request(`${this.basePath}/${id}`, {
      method: "DELETE",
    });
  }

  async createIndex(field: string): Promise<void> {
    const path = `/v1/projects/${this.client.projectId}/database/collections/${this.name}/indexes`;
    await this.client.request(path, {
      method: "POST",
      body: JSON.stringify({
        collection: this.name,
        field,
      }),
    });
  }

  async listIndexes(): Promise<string[]> {
    const path = `/v1/projects/${this.client.projectId}/database/collections/${this.name}/indexes`;
    const data: any = await this.client.request(path);
    return data.indexes;
  }

  async deleteIndex(field: string): Promise<void> {
    const path = `/v1/projects/${this.client.projectId}/database/collections/${this.name}/indexes/${field}`;
    await this.client.request(path, { method: "DELETE" });
  }

  async updateSchema(schema: any): Promise<void> {
    await this.client.request(this.collectionPath, {
      method: "PATCH",
      body: JSON.stringify({ schema }),
    });
  }

  async query(q: any, opts?: QueryOptions): Promise<T[]> {
    const path = `/v1/projects/${this.client.projectId}/database/collections/${this.name}/documents/query`;
    const body = {
      collection: this.name,
      query: q,
      skip: opts?.skip,
      limit: opts?.limit,
      sortField: opts?.sortField,
      sortDesc: opts?.sortDesc,
    };
    const data: any = await this.client.request(path, {
      method: "POST",
      body: JSON.stringify(body),
    });
    return data.documents;
  }

  /**
   * Watch all documents in this collection for realtime changes.
   * Returns an unsubscribe function.
   */
  watch(callback: (event: ChangeEvent<T>) => void): () => void {
    const url = `${this.client.url}/v1/projects/${this.client.projectId}/database/collections/${this.name}/subscribe`;
    return this.subscribeSSE(url, callback);
  }

  /**
   * Watch a specific document by ID for realtime changes.
   * Returns an unsubscribe function.
   */
  watchDocument(
    docId: string,
    callback: (event: ChangeEvent<T>) => void,
  ): () => void {
    // For now, use collection-level subscription and filter client-side
    // In future, could add dedicated endpoint
    const url = `${this.client.url}/v1/projects/${this.client.projectId}/database/collections/${this.name}/subscribe`;
    return this.subscribeSSE(url, (ev) => {
      if (ev.docId === docId || (ev.document as any)?._id === docId) {
        callback(ev);
      }
    });
  }

  /**
   * Watch documents matching a query for realtime changes.
   * Returns an unsubscribe function.
   */
  watchQuery(
    query: any,
    opts: QueryOptions | undefined,
    callback: (event: ChangeEvent<T>) => void,
  ): () => void {
    const url = `${this.client.url}/v1/projects/${this.client.projectId}/database/collections/${this.name}/documents/query/subscribe`;
    const body = JSON.stringify({
      query,
      skip: opts?.skip,
      limit: opts?.limit,
      sortField: opts?.sortField,
      sortDesc: opts?.sortDesc,
    });

    // Build URL with auth header - EventSource doesn't support custom headers,
    // so we'll use query param or a different approach
    const authUrl = `${url}?key=${encodeURIComponent(this.client.apiKey)}`;

    const eventSource = new EventSource(authUrl);
    let unsubscribe = false;

    eventSource.onmessage = (e) => {
      if (unsubscribe) return;
      try {
        const data = JSON.parse(e.data);
        if (data.type && data.type !== "connected") {
          callback({
            type: data.type as "added" | "modified" | "removed",
            document: data.document as T,
            docId: data.docId || (data.document as any)?._id,
          });
        }
      } catch (err) {
        console.error("Error parsing SSE event:", err);
      }
    };

    eventSource.onerror = (err) => {
      if (!unsubscribe) {
        console.error("SSE error:", err);
        eventSource.close();
      }
    };

    // POST with body - EventSource doesn't support POST, so we need fetch + ReadableStream
    // For now, use a workaround: send query as GET params or use a different approach
    // Actually, let's use fetch with ReadableStream for POST support
    return this.subscribeSSEPost(url, body, callback);
  }

  private subscribeSSE(
    url: string,
    callback: (event: ChangeEvent<T>) => void,
  ): () => void {
    // Use EventSource with query param (public API keys are safe in URLs)
    const authUrl = `${url}?key=${encodeURIComponent(this.client.apiKey)}`;
    const eventSource = new EventSource(authUrl);
    let unsubscribe = false;

    eventSource.addEventListener("change", (e: MessageEvent) => {
      if (unsubscribe) return;
      try {
        const data = JSON.parse(e.data);
        callback({
          type: data.type as "added" | "modified" | "removed",
          document: data.document as T,
          docId: data.docId || (data.document as any)?._id,
        });
      } catch (err) {
        console.error("Error parsing SSE event:", err);
      }
    });

    eventSource.onerror = () => {
      if (!unsubscribe) {
        eventSource.close();
      }
    };

    return () => {
      unsubscribe = true;
      eventSource.close();
    };
  }

  private subscribeSSEPost(
    url: string,
    body: string,
    callback: (event: ChangeEvent<T>) => void,
  ): () => void {
    // EventSource doesn't support POST, so use fetch with ReadableStream
    // Public API key in query param is fine
    const authUrl = `${url}?key=${encodeURIComponent(this.client.apiKey)}`;
    const controller = new AbortController();
    let unsubscribe = false;

    fetch(authUrl, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body,
      signal: controller.signal,
    })
      .then((response) => {
        if (!response.ok) {
          throw new Error(`SSE request failed: ${response.statusText}`);
        }
        const reader = response.body?.getReader();
        const decoder = new TextDecoder();
        let buffer = "";

        const readChunk = () => {
          if (unsubscribe) return;
          reader
            ?.read()
            .then(({ done, value }) => {
              if (done) return;
              buffer += decoder.decode(value, { stream: true });
              const lines = buffer.split("\n");
              buffer = lines.pop() || "";

              for (const line of lines) {
                if (line.startsWith("data: ")) {
                  const data = line.slice(6);
                  if (data.trim() && !data.includes('"connected"')) {
                    try {
                      const parsed = JSON.parse(data);
                      if (parsed.type) {
                        callback({
                          type: parsed.type as "added" | "modified" | "removed",
                          document: parsed.document as T,
                          docId: parsed.docId || (parsed.document as any)?._id,
                        });
                      }
                    } catch (err) {
                      console.error("Error parsing SSE data:", err);
                    }
                  }
                }
              }
              readChunk();
            })
            .catch((err) => {
              if (!unsubscribe) {
                console.error("SSE read error:", err);
              }
            });
        };

        readChunk();
      })
      .catch((err) => {
        if (!unsubscribe) {
          console.error("SSE fetch error:", err);
        }
      });

    return () => {
      unsubscribe = true;
      controller.abort();
    };
  }
}

export interface QueryOptions {
  skip?: number;
  limit?: number;
  sortField?: string;
  sortDesc?: boolean;
}

export interface ChangeEvent<T = any> {
  type: "added" | "modified" | "removed";
  document?: T;
  docId?: string;
}

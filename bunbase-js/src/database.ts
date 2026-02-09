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
    return this.client.request(`/v1/database/collections${params}`, {
      method: "GET",
    });
  }

  async createCollection(
    name: string,
    schema?: object,
    options?: { updateIfExists?: boolean; preventSchemaOverride?: boolean },
  ) {
    const body: Record<string, unknown> = { name };
    if (schema !== undefined) body.schema = schema;
    if (options?.updateIfExists !== undefined)
      body.update_if_exists = options.updateIfExists;
    if (options?.preventSchemaOverride !== undefined)
      body.prevent_schema_override = options.preventSchemaOverride;
    return this.client.request(`/v1/database/collections`, {
      method: "POST",
      body: JSON.stringify(body),
    });
  }

  async deleteCollection(name: string) {
    return this.client.request(
      `/v1/database/collections/${encodeURIComponent(name)}`,
      { method: "DELETE" },
    );
  }
}

export class Collection<T = any> {
  constructor(
    private client: BunBaseClient<any>,
    private name: string,
  ) {}

  private get basePath() {
    return `/v1/database/collections/${encodeURIComponent(this.name)}/documents`;
  }

  private get collectionPath() {
    return `/v1/database/collections/${encodeURIComponent(this.name)}`;
  }

  async list(
    opts?: Record<string, string> | { skip?: number; limit?: number; fields?: string[] },
  ): Promise<{ documents: T[] }> {
    const params = new URLSearchParams();
    if (opts != null) {
      if ("fields" in opts && Array.isArray(opts.fields)) {
        const o = opts as { skip?: number; limit?: number; fields?: string[] };
        if (o.skip != null) params.set("skip", String(o.skip));
        if (o.limit != null) params.set("limit", String(o.limit));
        if (o.fields?.length) params.set("fields", o.fields.join(","));
      } else {
        for (const [k, v] of Object.entries(opts as Record<string, string>)) {
          if (v != null && v !== "") params.set(k, String(v));
        }
      }
    }
    const url = params.toString() ? `${this.basePath}?${params}` : this.basePath;
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
    const path = `${this.collectionPath}/indexes`;
    await this.client.request(path, {
      method: "POST",
      body: JSON.stringify({
        collection: this.name,
        field,
      }),
    });
  }

  async listIndexes(): Promise<string[]> {
    const path = `${this.collectionPath}/indexes`;
    const data: any = await this.client.request(path);
    return data.indexes;
  }

  async deleteIndex(field: string): Promise<void> {
    const path = `${this.collectionPath}/indexes/${encodeURIComponent(field)}`;
    await this.client.request(path, { method: "DELETE" });
  }

  async updateSchema(schema: any): Promise<void> {
    await this.client.request(this.collectionPath, {
      method: "PATCH",
      body: JSON.stringify({ schema }),
    });
  }

  async query(q: any, opts?: QueryOptions): Promise<T[]> {
    const path = `${this.collectionPath}/documents/query`;
    const body: Record<string, unknown> = {
      collection: this.name,
      query: q,
      skip: opts?.skip,
      limit: opts?.limit,
      sortField: opts?.sortField,
      sortDesc: opts?.sortDesc,
    };
    if (opts?.fields?.length) body.fields = opts.fields;
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
    const url = `${this.client.url}/v1/database/collections/${encodeURIComponent(this.name)}/subscribe`;
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
    const url = `${this.client.url}/v1/database/collections/${encodeURIComponent(this.name)}/subscribe`;
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
    const url = `${this.client.url}/v1/database/collections/${encodeURIComponent(this.name)}/documents/query/subscribe`;
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
  /** Top-level field names to return (projection). Only these keys are included in each document. */
  fields?: string[];
}

export interface ChangeEvent<T = any> {
  type: "added" | "modified" | "removed";
  document?: T;
  docId?: string;
}

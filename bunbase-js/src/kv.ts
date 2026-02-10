import { BunBaseClient } from "./client";

export class KVClient {
  constructor(private client: BunBaseClient) {}

  /**
   * Get a value by key. Returns the raw value as a string.
   * Returns null if the key doesn't exist.
   */
  async get(key: string): Promise<string | null> {
    try {
      const response = await fetch(
        `${this.client.url}/v1/kv/kv/${encodeURIComponent(key)}`,
        {
          method: "GET",
          headers: {
            "X-Bunbase-Client-Key": this.client.apiKey,
          },
        },
      );

      if (response.status === 404) {
        return null;
      }

      if (!response.ok) {
        const body = await response.json().catch(() => ({}));
        throw new Error(body.error || `Request failed: ${response.statusText}`);
      }

      return response.text();
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error("Failed to get value");
    }
  }

  /**
   * Set a value for a key. Value can be a string, ArrayBuffer, or Uint8Array.
   */
  async set(key: string, value: string | ArrayBuffer | Uint8Array): Promise<void> {
    const response = await fetch(
      `${this.client.url}/v1/kv/kv/${encodeURIComponent(key)}`,
      {
        method: "PUT",
        headers: {
          "X-Bunbase-Client-Key": this.client.apiKey,
          "Content-Type": "application/octet-stream",
        },
        body: value as BodyInit,
      },
    );

    if (!response.ok) {
      const body = await response.json().catch(() => ({}));
      throw new Error(body.error || `Request failed: ${response.statusText}`);
    }
  }

  /**
   * Delete a key. Returns true if the key was deleted, false if it didn't exist.
   */
  async delete(key: string): Promise<boolean> {
    const response = await fetch(
      `${this.client.url}/v1/kv/kv/${encodeURIComponent(key)}`,
      {
        method: "DELETE",
        headers: {
          "X-Bunbase-Client-Key": this.client.apiKey,
        },
      },
    );

    if (response.status === 404) {
      return false;
    }

    if (!response.ok) {
      const body = await response.json().catch(() => ({}));
      throw new Error(body.error || `Request failed: ${response.statusText}`);
    }

    return true;
  }

  /**
   * List all keys. Optionally filter by pattern (supports wildcards like "*").
   */
  async keys(pattern?: string): Promise<string[]> {
    const params = pattern ? `?pattern=${encodeURIComponent(pattern)}` : "";
    return this.client.request(`/v1/kv/keys${params}`);
  }

  /**
   * Check if a key exists.
   */
  async exists(key: string): Promise<boolean> {
    const value = await this.get(key);
    return value !== null;
  }

  /**
   * Get health status of the KV store.
   */
  async health(): Promise<{ status: string }> {
    return this.client.request("/v1/kv/health");
  }
}

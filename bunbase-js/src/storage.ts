import { BunBaseClient } from "./client";

/** Encodes an object key for use in the storage path (each segment encoded). */
function encodeKey(key: string): string {
  return key.split("/").map(encodeURIComponent).join("/");
}

export interface StorageObjectInfo {
  key: string;
  size: number;
  last_modified?: string;
}

export class StorageClient {
  constructor(private client: BunBaseClient) {}

  /**
   * List objects in the project bucket. Optional prefix to filter by key prefix.
   */
  async list(prefix?: string): Promise<StorageObjectInfo[]> {
    const path = prefix?.trim()
      ? `/v1/storage?prefix=${encodeURIComponent(prefix.trim())}`
      : "/v1/storage";
    const data = (await this.client.request(path)) as {
      objects?: StorageObjectInfo[];
    };
    return Array.isArray(data?.objects) ? data.objects : [];
  }

  /**
   * Upload an object. Key can include slashes (e.g. "folder/file.txt").
   * Returns the key on success.
   */
  async put(
    key: string,
    body: Blob | ArrayBuffer | Uint8Array,
    contentType?: string,
  ): Promise<{ key: string }> {
    const path = `/v1/storage/${encodeKey(key)}`;
    const headers: Record<string, string> = {
      "X-Bunbase-Client-Key": this.client.apiKey,
    };
    if (contentType) {
      headers["Content-Type"] = contentType;
    }
    const response = await fetch(`${this.client.url}${path}`, {
      method: "PUT",
      headers,
      body: body as BodyInit,
    });
    if (!response.ok) {
      const data = await response.json().catch(() => ({}));
      throw new Error(
        (data as { error?: string }).error ||
          `Upload failed: ${response.statusText}`,
      );
    }
    const data = (await response.json()) as { key?: string };
    return { key: data?.key ?? key };
  }

  /**
   * Download an object. Returns the Response so you can call .blob(), .arrayBuffer(), .json(), etc.
   */
  async get(key: string): Promise<Response> {
    const path = `/v1/storage/${encodeKey(key)}`;
    const response = await fetch(`${this.client.url}${path}`, {
      method: "GET",
      headers: {
        "X-Bunbase-Client-Key": this.client.apiKey,
      },
    });
    if (!response.ok) {
      if (response.status === 404) {
        throw new Error("Object not found");
      }
      const data = await response.json().catch(() => ({}));
      throw new Error(
        (data as { error?: string }).error ||
          `Download failed: ${response.statusText}`,
      );
    }
    return response;
  }

  /**
   * Download an object as a Blob.
   */
  async getBlob(key: string): Promise<Blob> {
    const res = await this.get(key);
    return res.blob();
  }

  /**
   * Delete an object. Throws on error.
   */
  async delete(key: string): Promise<void> {
    const path = `/v1/storage/${encodeKey(key)}`;
    const response = await fetch(`${this.client.url}${path}`, {
      method: "DELETE",
      headers: {
        "X-Bunbase-Client-Key": this.client.apiKey,
      },
    });
    if (!response.ok) {
      const data = await response.json().catch(() => ({}));
      throw new Error(
        (data as { error?: string }).error ||
          `Delete failed: ${response.statusText}`,
      );
    }
  }
}

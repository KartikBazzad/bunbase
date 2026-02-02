import { BunBaseClient } from "./client";

export class DatabaseClient {
  constructor(private client: BunBaseClient) {}

  collection(name: string) {
    return new Collection(this.client, name);
  }
}

export class Collection {
  constructor(
    private client: BunBaseClient,
    private name: string,
  ) {}

  // Using "default" as dbName placeholder, as backend overrides it with ProjectID
  private get basePath() {
    return `/v1/databases/default/collections/${this.name}/documents`;
  }

  async list(query?: Record<string, string>) {
    const params = new URLSearchParams(query).toString();
    const url = params ? `${this.basePath}?${params}` : this.basePath;
    return this.client.request(url, { method: "GET" });
  }

  async get(id: string) {
    return this.client.request(`${this.basePath}/${id}`, { method: "GET" });
  }

  async create(data: any) {
    return this.client.request(this.basePath, {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  async update(id: string, data: any) {
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
}

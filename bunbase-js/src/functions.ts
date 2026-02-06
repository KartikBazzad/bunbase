import { BunBaseClient } from "./client";

export class FunctionsClient {
  constructor(private client: BunBaseClient) {}

  async invoke(functionName: string, body?: any) {
    return this.client.request(`/v1/functions/${encodeURIComponent(functionName)}/invoke`, {
      method: "POST",
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  }

  async listFunctions() {
    return this.client.request(`/v1/functions`, { method: "GET" });
  }

  async deleteFunction(functionName: string) {
    return this.client.request(`/v1/functions/${encodeURIComponent(functionName)}`, {
      method: "DELETE",
    });
  }
}

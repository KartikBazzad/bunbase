import { BunBaseClient } from "./client";

export class FunctionsClient {
  constructor(private client: BunBaseClient) {}

  async invoke(functionName: string, body?: any) {
    return this.client.request(`/v1/functions/${functionName}/invoke`, {
      method: "POST",
      body: JSON.stringify(body),
    });
  }
}

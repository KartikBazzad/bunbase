import { BunBaseClient } from "./client";

export class FunctionsClient {
  constructor(private client: BunBaseClient) {}

  async invoke(functionName: string, body?: any) {
    const projectId = this.client.projectId;
    const path =
      projectId && projectId !== "default"
        ? `/v1/projects/${projectId}/functions/${functionName}/invoke`
        : `/v1/functions/${functionName}/invoke`;
    return this.client.request(path, {
      method: "POST",
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  }

  async listFunctions() {
    return this.client.request(`/v1/functions`, {
      method: "GET",
    });
  }

  async deleteFunction(functionName: string) {
    return this.client.request(`/v1/functions/${functionName}`, {
      method: "DELETE",
    });
  }
}

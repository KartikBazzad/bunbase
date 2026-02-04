/**
 * API client for platform backend
 */

const API_URL = import.meta.env.VITE_API_URL || "http://localhost:3001/v1";

export interface ApiError {
  error: string;
}

export interface ProjectConfig {
  gateway_url: string;
  project_id: string;
  project_slug: string;
  kv: { path: string };
  bundoc: { documents_path: string };
  buncast: { topic_prefix: string };
  functions: { invoke_path: string };
}

export class ApiClient {
  private baseURL: string;

  constructor(baseURL: string = API_URL) {
    this.baseURL = baseURL;
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {},
  ): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    const response = await fetch(url, {
      ...options,
      credentials: "include", // Include cookies
      headers: {
        "Content-Type": "application/json",
        ...options.headers,
      },
    });

    if (!response.ok) {
      const error: ApiError = await response.json().catch(() => ({
        error: response.statusText,
      }));
      throw new Error(error.error || `HTTP ${response.status}`);
    }

    return response.json();
  }

  // Auth endpoints
  async register(email: string, password: string, name: string) {
    return this.request("/auth/register", {
      method: "POST",
      body: JSON.stringify({ email, password, name }),
    });
  }

  async login(email: string, password: string) {
    return this.request("/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    });
  }

  async logout() {
    return this.request("/auth/logout", {
      method: "POST",
    });
  }

  async getMe() {
    return this.request("/auth/me");
  }

  // Project endpoints
  async listProjects() {
    return this.request("/projects");
  }

  async getProject(id: string) {
    return this.request(`/projects/${id}`);
  }

  async createProject(name: string) {
    return this.request("/projects", {
      method: "POST",
      body: JSON.stringify({ name }),
    });
  }

  async updateProject(id: string, name: string) {
    return this.request(`/projects/${id}`, {
      method: "PUT",
      body: JSON.stringify({ name }),
    });
  }

  async deleteProject(id: string) {
    return this.request(`/projects/${id}`, {
      method: "DELETE",
    });
  }

  async getProjectConfig(projectId: string) {
    return this.request<ProjectConfig>(`/projects/${projectId}/config`);
  }

  // Function endpoints
  async listFunctions(projectId: string) {
    return this.request(`/projects/${projectId}/functions`);
  }

  async deployFunction(
    projectId: string,
    name: string,
    runtime: string,
    handler: string,
    version: string,
    bundle: string, // Base64 encoded bundle
  ) {
    return this.request(`/projects/${projectId}/functions`, {
      method: "POST",
      body: JSON.stringify({
        name,
        runtime,
        handler,
        version,
        bundle,
      }),
    });
  }

  async deleteFunction(projectId: string, functionId: string) {
    return this.request(`/projects/${projectId}/functions/${functionId}`, {
      method: "DELETE",
    });
  }

  async invokeFunction(
    projectId: string,
    functionName: string,
    method: string = "GET",
    body?: any,
    headers?: any,
  ) {
    return this.request(
      `/projects/${projectId}/functions/${functionName}/invoke`,
      {
        method: method,
        headers: headers,
        body: body ? JSON.stringify(body) : undefined,
      },
    );
  }

  async getFunctionLogs(
    projectId: string,
    options?: { functionId?: string; since?: string; limit?: number },
  ) {
    const params = new URLSearchParams();
    if (options?.functionId) params.set("function_id", options.functionId);
    if (options?.since) params.set("since", options.since);
    if (options?.limit != null) params.set("limit", String(options.limit));
    const q = params.toString();
    return this.request<
      Array<{
        function_id: string;
        invocation_id: string;
        level: string;
        message: string;
        created_at: string;
        function_name?: string;
      }>
    >(`/projects/${projectId}/functions/logs${q ? `?${q}` : ""}`);
  }

  // Database endpoints
  async listCollections(projectId: string) {
    return this.request(`/projects/${projectId}/database/collections`);
  }

  async createCollection(projectId: string, name: string) {
    return this.request(`/projects/${projectId}/database/collections`, {
      method: "POST",
      body: JSON.stringify({ name }),
    });
  }

  async deleteCollection(projectId: string, name: string) {
    return this.request(`/projects/${projectId}/database/collections/${name}`, {
      method: "DELETE",
    });
  }

  async getCollection(projectId: string, name: string) {
    return this.request(`/projects/${projectId}/database/collections/${name}`);
  }

  async listDocuments(
    projectId: string,
    collection: string,
    params?: { skip?: number; limit?: number; prefix?: string },
  ) {
    const query = new URLSearchParams();
    if (params?.skip) query.set("skip", String(params.skip));
    if (params?.limit) query.set("limit", String(params.limit));
    if (params?.prefix) query.set("prefix", params.prefix);

    const qs = query.toString();
    const url = `/projects/${projectId}/database/collections/${collection}/documents${
      qs ? "?" + qs : ""
    }`;
    return this.request(url);
  }

  async createDocument(projectId: string, collection: string, data: any) {
    return this.request(
      `/projects/${projectId}/database/collections/${collection}/documents`,
      {
        method: "POST",
        body: JSON.stringify(data),
      },
    );
  }

  async updateDocument(
    projectId: string,
    collection: string,
    id: string,
    data: any,
  ) {
    return this.request(
      `/projects/${projectId}/database/collections/${collection}/documents/${id}`,
      {
        method: "PUT", // or PATCH
        body: JSON.stringify(data),
      },
    );
  }

  async deleteDocument(projectId: string, collection: string, id: string) {
    return this.request(
      `/projects/${projectId}/database/collections/${collection}/documents/${id}`,
      {
        method: "DELETE",
      },
    );
  }

  // Schema & Indexes
  async updateCollectionSchema(
    projectId: string,
    collection: string,
    schema: any,
  ) {
    return this.request(
      `/projects/${projectId}/database/collections/${collection}`,
      {
        method: "PATCH",
        body: JSON.stringify({ schema }),
      },
    );
  }

  async updateCollectionRules(
    projectId: string,
    collection: string,
    rules: Record<string, string>,
  ) {
    return this.request(
      `/projects/${projectId}/database/collections/${collection}/rules`,
      {
        method: "PATCH",
        body: JSON.stringify({ rules }),
      },
    );
  }

  async listIndexes(projectId: string, collection: string) {
    return this.request(
      `/projects/${projectId}/database/collections/${collection}/indexes`,
    );
  }

  async createIndex(projectId: string, collection: string, field: string) {
    return this.request(
      `/projects/${projectId}/database/collections/${collection}/indexes`,
      {
        method: "POST",
        body: JSON.stringify({ collection, field }), // Collection required by backend
      },
    );
  }

  async deleteIndex(projectId: string, collection: string, field: string) {
    return this.request(
      `/projects/${projectId}/database/collections/${collection}/indexes/${field}`,
      {
        method: "DELETE",
      },
    );
  }

  async queryDocuments(
    projectId: string,
    collection: string,
    query: any,
    opts?: any,
  ) {
    return this.request(
      `/projects/${projectId}/database/collections/${collection}/documents/query`,
      {
        method: "POST",
        body: JSON.stringify({
          collection, // Required by backend
          query,
          skip: opts?.skip || 0,
          limit: opts?.limit || 100,
          sortField: opts?.sortField,
          sortDesc: opts?.sortDesc,
        }),
      },
    );
  }
}

export const api = new ApiClient();

/**
 * API client for platform backend
 */

const API_URL = import.meta.env.VITE_API_URL || "http://localhost:3001/api";

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

  async listDocuments(projectId: string, collection: string) {
    return this.request(
      `/projects/${projectId}/database/collections/${collection}/documents`,
    );
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
}

export const api = new ApiClient();

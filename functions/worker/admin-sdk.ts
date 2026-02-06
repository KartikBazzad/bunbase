export class BunBaseAdmin {
  private projectId: string;
  private apiKey: string;
  private gatewayUrl: string;

  constructor() {
    this.projectId = process.env.BUNBASE_PROJECT_ID || "";
    this.apiKey = process.env.BUNBASE_API_KEY || "";
    this.gatewayUrl = (process.env.BUNBASE_GATEWAY_URL || "").replace(/\/$/, "");

    if (!this.projectId) {
      throw new Error("BUNBASE_PROJECT_ID is not set for BunBaseAdmin");
    }
    if (!this.gatewayUrl) {
      throw new Error("BUNBASE_GATEWAY_URL is not set for BunBaseAdmin");
    }
  }

  /**
   * Low-level request helper that prefixes the gateway URL and attaches
   * the project API key for admin access.
   */
  async request(path: string, init: RequestInit = {}): Promise<Response> {
    const url =
      path.startsWith("http://") || path.startsWith("https://")
        ? path
        : `${this.gatewayUrl}${path}`;

    const headers: Record<string, string> = {
      ...(init.headers as Record<string, string> | undefined),
    };

    if (this.apiKey) {
      headers["X-Bunbase-Client-Key"] = this.apiKey;
    }

    return fetch(url, {
      ...init,
      headers,
    });
  }

  /**
   * Minimal Functions admin client.
   * Invoke another function in the same project by name.
   */
  async invokeFunction(name: string, body?: any, init: RequestInit = {}) {
    const res = await this.request(
      `/v1/projects/${this.projectId}/functions/${encodeURIComponent(
        name,
      )}/invoke`,
      {
        method: init.method || "POST",
        body:
          body !== undefined
            ? typeof body === "string"
              ? body
              : JSON.stringify(body)
            : undefined,
        headers: {
          "Content-Type": "application/json",
          ...(init.headers as Record<string, string> | undefined),
        },
      },
    );

    return res;
  }

  /**
   * Minimal database helper: perform a raw HTTP request against the
   * Bundoc documents API for this project.
   * Callers can build collection/doc paths as needed.
   */
  async dbRequest(
    method: string,
    path: string,
    body?: any,
    init: RequestInit = {},
  ) {
    const url = `/v1/projects/${this.projectId}${path}`;
    const res = await this.request(url, {
      method,
      body:
        body !== undefined
          ? typeof body === "string"
            ? body
            : JSON.stringify(body)
          : undefined,
      headers: {
        "Content-Type": "application/json",
        ...(init.headers as Record<string, string> | undefined),
      },
    });
    return res;
  }
}


import { AuthClient } from "./auth";
import { DatabaseClient } from "./database";
import { FunctionsClient } from "./functions";

export class BunBaseClient {
  public auth: AuthClient;
  public db: DatabaseClient;
  public functions: FunctionsClient;

  constructor(
    public url: string,
    public apiKey: string,
  ) {
    this.auth = new AuthClient(this);
    this.db = new DatabaseClient(this);
    this.functions = new FunctionsClient(this);
  }

  async request(path: string, options: RequestInit = {}) {
    const headers = new Headers(options.headers);
    headers.set("X-Bunbase-Client-Key", this.apiKey);
    headers.set("Content-Type", "application/json");

    // If we have a user session, maybe pass it?
    // middleware.GetSessionTokenFromContext...
    // For now stick to Client Key.

    const response = await fetch(`${this.url}${path}`, {
      ...options,
      headers,
    });

    if (!response.ok) {
      const body = await response.json().catch(() => ({}));
      throw new Error(body.error || `Request failed: ${response.statusText}`);
    }

    // Handle 204 No Content
    if (response.status === 204) {
      return null;
    }

    return response.json();
  }
}

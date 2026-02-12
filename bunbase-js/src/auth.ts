import { BunBaseClient } from "./client";

export class AuthClient {
  constructor(private client: BunBaseClient) {}

  async signUp(options: { email: string; password: string; name: string }) {
    return this.client.request("/v1/auth/register", {
      method: "POST",
      body: JSON.stringify(options),
    });
  }

  async signInWithPassword(options: { email: string; password: string }) {
    const response = await this.client.request("/v1/auth/login", {
      method: "POST",
      body: JSON.stringify(options),
    });
    // Cookie is automatically set by server response
    return response;
  }

  /**
   * Get the current user profile from the session cookie.
   * Returns user data if authenticated, null otherwise.
   */
  async getProfile(): Promise<{
    id: string;
    user_id?: string;
    email: string;
    project_id: string;
  } | null> {
    try {
      const response = await this.client.request("/v1/auth/session", {
        method: "GET",
      });
      return response.user || null;
    } catch {
      return null;
    }
  }
}

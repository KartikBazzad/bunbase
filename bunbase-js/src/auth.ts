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
    // TODO: Store session token if client-side session management is added
    return response;
  }
}

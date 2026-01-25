/**
 * Authentication Module
 */

import type { BunBaseConfig } from "../client";
import type { AuthUser, AuthSession } from "../types";

export interface AuthModuleOptions {
  // Additional auth-specific options
}

export class AuthModule {
  constructor(
    private config: BunBaseConfig,
    private options?: AuthModuleOptions,
  ) {}

  private client: { request: BunBaseClient["request"] };

  constructor(
    private config: BunBaseConfig,
    client: BunBaseClient,
    private options?: AuthModuleOptions,
  ) {
    this.client = client;
  }

  /**
   * Sign up with email and password
   */
  async signUp(email: string, password: string, name: string): Promise<{
    user: AuthUser;
    session: AuthSession;
  }> {
    return this.client.request("POST", "/auth/sign-up", {
      body: { email, password, name },
    });
  }

  /**
   * Sign in with email and password
   */
  async signIn(email: string, password: string): Promise<{
    user: AuthUser;
    session: AuthSession;
  }> {
    return this.client.request("POST", "/auth/sign-in", {
      body: { email, password },
    });
  }

  /**
   * Sign out
   */
  async signOut(): Promise<void> {
    return this.client.request("POST", "/auth/sign-out");
  }

  /**
   * Get current user
   */
  async getUser(): Promise<AuthUser> {
    return this.client.request("GET", "/auth/user");
  }

  /**
   * Verify email
   */
  async verifyEmail(token: string): Promise<void> {
    return this.client.request("POST", "/auth/verify-email", {
      body: { token },
    });
  }

  /**
   * Request password reset
   */
  async forgotPassword(email: string): Promise<void> {
    return this.client.request("POST", "/auth/forgot-password", {
      body: { email },
    });
  }

  /**
   * Reset password
   */
  async resetPassword(token: string, password: string): Promise<void> {
    return this.client.request("POST", "/auth/reset-password", {
      body: { token, password },
    });
  }
}

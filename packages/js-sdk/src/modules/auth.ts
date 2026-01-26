/**
 * Authentication Module
 */

import type { BunBaseConfig, BunBaseClient } from "../client";
import type { AuthUser, AuthSession } from "../types";

export interface AuthModuleOptions {
  // Additional auth-specific options
}

export class AuthModule {
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
   * Better Auth route: POST /auth/sign-up/email
   */
  async signUp(
    email: string,
    password: string,
    name: string,
  ): Promise<{
    user: AuthUser;
    session: AuthSession | null;
  }> {
    const result = await this.client.request("POST", "/auth/sign-up/email", {
      body: { email, password, name },
      useCookies: true, // Better Auth uses cookies for authentication
    });
    // Better Auth returns { user, session } structure
    // When email verification is required, session might be null
    // Handle both cases: direct user object or { user, session } structure
    if (result.user) {
      return {
        user: result.user,
        session: result.session || null,
      } as { user: AuthUser; session: AuthSession | null };
    }
    // If result is the user directly (shouldn't happen but handle it)
    return {
      user: result as AuthUser,
      session: null,
    } as { user: AuthUser; session: AuthSession | null };
  }

  /**
   * Sign in with email and password
   * Better Auth route: POST /auth/sign-in/email
   */
  async signIn(
    email: string,
    password: string,
  ): Promise<{
    user: AuthUser;
    session: AuthSession;
  }> {
    return this.client.request("POST", "/auth/sign-in/email", {
      body: { email, password },
      useCookies: true, // Better Auth uses cookies for authentication
    });
  }

  /**
   * Sign out
   * Better Auth route: POST /auth/sign-out
   */
  async signOut(): Promise<void> {
    return this.client.request("POST", "/auth/sign-out", {
      useCookies: true, // Better Auth uses cookies for authentication
    });
  }

  /**
   * Get current user
   * Better Auth route: GET /auth/session
   */
  async getUser(): Promise<AuthUser> {
    const response = await this.client.request<{
      user: AuthUser;
      session: AuthSession;
    }>("GET", "/auth/session", {
      useCookies: true, // Better Auth uses cookies for authentication
    });
    return response.user;
  }

  /**
   * Verify email
   * Better Auth route: POST /auth/verify-email
   */
  async verifyEmail(token: string): Promise<void> {
    return this.client.request("POST", "/auth/verify-email", {
      body: { token },
      useCookies: true, // Better Auth uses cookies for authentication
    });
  }

  /**
   * Request password reset
   * Better Auth route: POST /auth/forgot-password
   */
  async forgotPassword(email: string): Promise<void> {
    return this.client.request("POST", "/auth/forgot-password", {
      body: { email },
      useCookies: true, // Better Auth uses cookies for authentication
    });
  }

  /**
   * Reset password
   * Better Auth route: POST /auth/reset-password
   */
  async resetPassword(token: string, password: string): Promise<void> {
    return this.client.request("POST", "/auth/reset-password", {
      body: { token, password },
      useCookies: true, // Better Auth uses cookies for authentication
    });
  }
}

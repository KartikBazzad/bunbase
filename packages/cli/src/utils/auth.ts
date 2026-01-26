/**
 * Authentication utilities for CLI
 */

import { existsSync, readFileSync, writeFileSync, mkdirSync } from "fs";
import { join } from "path";
import { homedir } from "os";

const CONFIG_DIR = join(homedir(), ".bunbase");
const CONFIG_FILE = join(CONFIG_DIR, "config.json");
const COOKIES_FILE = join(CONFIG_DIR, "cookies.json");

export interface AuthConfig {
  apiKey?: string;
  baseURL?: string;
  projectId?: string;
  user?: {
    id: string;
    email: string;
    name: string;
  };
  session?: {
    id: string;
    expiresAt: string;
  };
}

export interface Cookie {
  name: string;
  value: string;
  domain?: string;
  path?: string;
  expires?: number;
  httpOnly?: boolean;
  secure?: boolean;
  sameSite?: "strict" | "lax" | "none";
}

/**
 * Load authentication configuration
 */
export function loadAuth(): AuthConfig | null {
  if (!existsSync(CONFIG_FILE)) {
    return null;
  }

  try {
    const content = readFileSync(CONFIG_FILE, "utf-8");
    return JSON.parse(content);
  } catch (error) {
    return null;
  }
}

/**
 * Save authentication configuration
 */
export function saveAuth(config: AuthConfig): void {
  if (!existsSync(CONFIG_DIR)) {
    mkdirSync(CONFIG_DIR, { recursive: true });
  }

  const existing = loadAuth() || {};
  const updated = { ...existing, ...config };

  writeFileSync(CONFIG_FILE, JSON.stringify(updated, null, 2));
}

/**
 * Clear authentication configuration
 */
export function clearAuth(): void {
  if (existsSync(CONFIG_FILE)) {
    writeFileSync(CONFIG_FILE, JSON.stringify({}, null, 2));
  }
  if (existsSync(COOKIES_FILE)) {
    writeFileSync(COOKIES_FILE, JSON.stringify([], null, 2));
  }
}

/**
 * Login with email and password
 */
export async function loginWithEmail(
  email: string,
  password: string,
  baseURL: string = "http://localhost:3000/api",
): Promise<{ user: any; session: any; cookies: Cookie[] }> {
  const url = new URL("/auth/sign-in/email", baseURL.replace("/api", ""));

  const response = await fetch(url.toString(), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ email, password }),
    credentials: "include", // Important for cookies
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      error: { message: response.statusText },
    }));
    throw new Error(error.error?.message || `Login failed: ${response.status}`);
  }

  // Extract cookies from Set-Cookie headers
  const cookies: Cookie[] = [];
  const setCookieHeaders = response.headers.getSetCookie();
  for (const cookieHeader of setCookieHeaders) {
    const parts = cookieHeader.split(";").map((p) => p.trim());
    const [name, value] = parts[0].split("=");
    if (name && value) {
      const cookie: Cookie = { name, value };
      for (const part of parts.slice(1)) {
        if (part.toLowerCase().startsWith("domain=")) {
          cookie.domain = part.substring(7);
        } else if (part.toLowerCase().startsWith("path=")) {
          cookie.path = part.substring(5);
        } else if (part.toLowerCase().startsWith("expires=")) {
          cookie.expires = new Date(part.substring(8)).getTime();
        } else if (part.toLowerCase() === "httponly") {
          cookie.httpOnly = true;
        } else if (part.toLowerCase() === "secure") {
          cookie.secure = true;
        } else if (part.toLowerCase().startsWith("samesite=")) {
          cookie.sameSite = part.substring(9).toLowerCase() as any;
        }
      }
      cookies.push(cookie);
    }
  }

  const data = await response.json();
  return {
    user: data.user,
    session: data.session,
    cookies,
  };
}

/**
 * Get current session
 */
export async function getSession(
  baseURL: string = "http://localhost:3000/api",
): Promise<{ user: any; session: any } | null> {
  const cookies = loadSessionCookies();
  if (cookies.length === 0) {
    return null;
  }

  const url = new URL("/auth/session", baseURL.replace("/api", ""));
  const cookieHeader = cookies
    .map((c) => `${c.name}=${c.value}`)
    .join("; ");

  const response = await fetch(url.toString(), {
    method: "GET",
    headers: {
      Cookie: cookieHeader,
    },
    credentials: "include",
  });

  if (!response.ok) {
    return null;
  }

  return await response.json();
}

/**
 * Save session cookies
 */
export function saveSessionCookies(cookies: Cookie[]): void {
  if (!existsSync(CONFIG_DIR)) {
    mkdirSync(CONFIG_DIR, { recursive: true });
  }
  writeFileSync(COOKIES_FILE, JSON.stringify(cookies, null, 2));
}

/**
 * Load session cookies
 */
export function loadSessionCookies(): Cookie[] {
  if (!existsSync(COOKIES_FILE)) {
    return [];
  }

  try {
    const content = readFileSync(COOKIES_FILE, "utf-8");
    const cookies = JSON.parse(content) as Cookie[];
    // Filter out expired cookies
    const now = Date.now();
    return cookies.filter((c) => !c.expires || c.expires > now);
  } catch (error) {
    return [];
  }
}

/**
 * Get cookie header string for requests
 */
export function getCookieHeader(): string {
  const cookies = loadSessionCookies();
  return cookies.map((c) => `${c.name}=${c.value}`).join("; ");
}

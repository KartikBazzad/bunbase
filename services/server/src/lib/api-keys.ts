import { randomBytes } from "node:crypto";

/**
 * Generate a secure API key with format: bunbase_pk_live_<32-char-random>
 */
export function generateApiKey(): string {
  // Generate 32 random bytes (256 bits) and convert to base64url
  const randomPart = randomBytes(32)
    .toString("base64")
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=/g, "")
    .substring(0, 32); // Ensure exactly 32 chars

  return `bunbase_pk_live_${randomPart}`;
}

/**
 * Hash an API key using SHA-256
 * @param key The API key to hash
 * @returns The hashed key as a hex string
 */
export async function hashApiKey(key: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(key);
  const hashBuffer = await crypto.subtle.digest("SHA-256", data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map((b) => b.toString(16).padStart(2, "0")).join("");
}

/**
 * Validate an API key format
 * @param key The API key to validate
 * @returns true if the format is valid
 */
export function validateApiKeyFormat(key: string): boolean {
  // Format: bunbase_pk_live_<32-char-alphanumeric>
  const pattern = /^bunbase_pk_live_[A-Za-z0-9_-]{32}$/;
  return pattern.test(key);
}

/**
 * Extract prefix and suffix from an API key for display
 * @param key The full API key
 * @returns Object with prefix (first 12 chars) and suffix (last 4 chars)
 */
export function extractKeyParts(key: string): {
  prefix: string;
  suffix: string;
} {
  const prefix = key.substring(0, 12); // "bunbase_pk_"
  const suffix = key.substring(key.length - 4); // Last 4 chars
  return { prefix, suffix };
}

/**
 * Mask an API key for display (shows prefix and suffix only)
 * @param prefix The key prefix
 * @param suffix The key suffix
 * @returns Masked key string
 */
export function maskApiKey(prefix: string, suffix: string): string {
  return `${prefix}${"*".repeat(20)}${suffix}`;
}

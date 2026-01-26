/**
 * Encryption utilities for sensitive data like OAuth client secrets
 * Uses Web Crypto API with AES-GCM
 */

const ENCRYPTION_KEY = process.env.ENCRYPTION_KEY || "default-key-change-in-production-min-32-chars";
const ALGORITHM = "AES-GCM";
const IV_LENGTH = 12; // 96 bits for GCM
const KEY_LENGTH = 256; // 256 bits

/**
 * Derive a key from the encryption key string
 */
async function deriveKey(keyMaterial: string): Promise<CryptoKey> {
  const encoder = new TextEncoder();
  const keyData = encoder.encode(keyMaterial);
  
  // Import the key material
  const importedKey = await crypto.subtle.importKey(
    "raw",
    keyData,
    { name: "PBKDF2" },
    false,
    ["deriveBits", "deriveKey"]
  );
  
  // Derive the actual encryption key
  return crypto.subtle.deriveKey(
    {
      name: "PBKDF2",
      salt: encoder.encode("bunbase-oauth-secret-salt"),
      iterations: 100000,
      hash: "SHA-256",
    },
    importedKey,
    { name: ALGORITHM, length: KEY_LENGTH },
    false,
    ["encrypt", "decrypt"]
  );
}

/**
 * Encrypt a string value
 */
export async function encrypt(value: string): Promise<string> {
  if (!value) return value;
  
  const key = await deriveKey(ENCRYPTION_KEY);
  const encoder = new TextEncoder();
  const data = encoder.encode(value);
  
  // Generate a random IV
  const iv = crypto.getRandomValues(new Uint8Array(IV_LENGTH));
  
  // Encrypt the data
  const encrypted = await crypto.subtle.encrypt(
    {
      name: ALGORITHM,
      iv: iv,
    },
    key,
    data
  );
  
  // Combine IV and encrypted data, then encode as base64
  const combined = new Uint8Array(iv.length + encrypted.byteLength);
  combined.set(iv);
  combined.set(new Uint8Array(encrypted), iv.length);
  
  return Buffer.from(combined).toString("base64");
}

/**
 * Decrypt an encrypted string value
 */
export async function decrypt(encryptedValue: string): Promise<string> {
  if (!encryptedValue) return encryptedValue;
  
  try {
    const key = await deriveKey(ENCRYPTION_KEY);
    
    // Decode from base64
    const combined = Buffer.from(encryptedValue, "base64");
    
    // Extract IV and encrypted data
    const iv = combined.slice(0, IV_LENGTH);
    const encrypted = combined.slice(IV_LENGTH);
    
    // Decrypt
    const decrypted = await crypto.subtle.decrypt(
      {
        name: ALGORITHM,
        iv: new Uint8Array(iv),
      },
      key,
      encrypted
    );
    
    // Convert back to string
    const decoder = new TextDecoder();
    return decoder.decode(decrypted);
  } catch (error) {
    // If decryption fails, return the value as-is (might be unencrypted legacy data)
    console.error("Decryption error:", error);
    return encryptedValue;
  }
}

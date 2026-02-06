import { BunBaseClient } from "bunbase-js";

const BASE_URL =
  import.meta.env.VITE_BUNBASE_URL || "http://localhost:3001";
const DEFAULT_API_KEY = import.meta.env.VITE_API_KEY || "";

export interface ClientConfig {
  baseUrl: string;
  apiKey: string;
  projectId?: string;
}

export function getDefaultConfig(): ClientConfig {
  return {
    baseUrl: BASE_URL,
    apiKey: DEFAULT_API_KEY,
  };
}

export function createClient(config?: Partial<ClientConfig>): BunBaseClient {
  const { baseUrl, apiKey } = { ...getDefaultConfig(), ...config };
  if (!apiKey) {
    throw new Error("API key is required. Set it in Settings or in .env (VITE_API_KEY).");
  }
  return new BunBaseClient(baseUrl, apiKey);
}

export function createClientIfConfigured(
  config?: Partial<ClientConfig>
): BunBaseClient | null {
  const { baseUrl, apiKey } = { ...getDefaultConfig(), ...config };
  if (!apiKey) return null;
  return new BunBaseClient(baseUrl, apiKey);
}

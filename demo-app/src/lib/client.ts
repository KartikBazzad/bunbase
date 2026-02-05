import { BunBaseClient } from "bunbase-js";

const BASE_URL =
  import.meta.env.VITE_BUNBASE_URL || "http://localhost:3001";
const DEFAULT_PROJECT_ID = import.meta.env.VITE_PROJECT_ID || "";
const DEFAULT_API_KEY = import.meta.env.VITE_API_KEY || "";

export interface ClientConfig {
  baseUrl: string;
  apiKey: string;
  projectId: string;
}

export function getDefaultConfig(): ClientConfig {
  return {
    baseUrl: BASE_URL,
    apiKey: DEFAULT_API_KEY,
    projectId: DEFAULT_PROJECT_ID,
  };
}

export function createClient(config?: Partial<ClientConfig>): BunBaseClient {
  const { baseUrl, apiKey, projectId } = { ...getDefaultConfig(), ...config };
  if (!apiKey || !projectId) {
    throw new Error("Project API key and project ID are required. Set them in Settings or in .env (VITE_API_KEY, VITE_PROJECT_ID).");
  }
  return new BunBaseClient(baseUrl, apiKey, projectId);
}

export function createClientIfConfigured(
  config?: Partial<ClientConfig>
): BunBaseClient | null {
  const { baseUrl, apiKey, projectId } = { ...getDefaultConfig(), ...config };
  if (!apiKey || !projectId) return null;
  return new BunBaseClient(baseUrl, apiKey, projectId);
}

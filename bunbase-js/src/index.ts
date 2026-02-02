import { BunBaseClient } from "./client";

export * from "./auth";
export * from "./database";
export * from "./functions";
export * from "./client";

export function createClient(url: string, apiKey: string) {
  return new BunBaseClient(url, apiKey);
}

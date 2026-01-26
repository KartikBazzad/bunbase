/**
 * BunBase Client Configuration
 */

import { createClient, type BunBaseClient } from "@bunbase/js-sdk";

// Get configuration from environment variables or use defaults
const API_KEY = "bunbase_pk_live_z1tfXt5iGukSLAVj5kr3Uh13PHMugI3u";
const BASE_URL = "http://localhost:3000/api";

// Create and export singleton client instance
export const client: BunBaseClient = createClient({
  apiKey: API_KEY,
  baseURL: BASE_URL,
});

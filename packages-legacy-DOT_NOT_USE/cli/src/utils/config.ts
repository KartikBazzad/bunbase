/**
 * Configuration file parsing utilities
 */

import { existsSync, readFileSync } from "fs";
import { join } from "path";

export interface BunBaseConfig {
  functions?: Record<
    string,
    {
      runtime: string;
      handler: string;
      type?: "http" | "callable";
      path?: string;
      methods?: string[];
      memory?: number;
      timeout?: number;
      env?: Record<string, string>;
    }
  >;
  projectId?: string;
  baseURL?: string;
}

/**
 * Load configuration from bunbase.config.ts or functions.json
 */
export function loadConfig(projectRoot: string = process.cwd()): BunBaseConfig | null {
  // Try bunbase.config.ts first
  const tsConfigPath = join(projectRoot, "bunbase.config.ts");
  if (existsSync(tsConfigPath)) {
    try {
      // For now, we'll need to use dynamic import or require
      // In a real implementation, you might use ts-node or similar
      const content = readFileSync(tsConfigPath, "utf-8");
      // Simple JSON extraction for now - in production, use proper TS parser
      // This is a simplified version
      return parseConfig(content);
    } catch (error) {
      console.error("Error loading bunbase.config.ts:", error);
      return null;
    }
  }

  // Try functions.json
  const jsonConfigPath = join(projectRoot, "functions.json");
  if (existsSync(jsonConfigPath)) {
    try {
      const content = readFileSync(jsonConfigPath, "utf-8");
      return JSON.parse(content) as BunBaseConfig;
    } catch (error) {
      console.error("Error loading functions.json:", error);
      return null;
    }
  }

  return null;
}

/**
 * Parse TypeScript config file (simplified - in production use proper parser)
 */
function parseConfig(content: string): BunBaseConfig | null {
  // This is a very simplified parser
  // In production, you'd want to use a proper TypeScript parser
  try {
    // Extract the default export object
    const exportMatch = content.match(/export\s+default\s+({[\s\S]*})/);
    if (exportMatch) {
      // Remove comments and convert to JSON-like format
      let jsonStr = exportMatch[1]
        .replace(/\/\/.*$/gm, "") // Remove single-line comments
        .replace(/\/\*[\s\S]*?\*\//g, "") // Remove multi-line comments
        .replace(/(\w+):/g, '"$1":') // Quote keys
        .replace(/'/g, '"'); // Replace single quotes with double quotes

      return JSON.parse(jsonStr) as BunBaseConfig;
    }
  } catch (error) {
    // If parsing fails, return null
  }
  return null;
}

/**
 * Get project configuration from .bunbase/config.json
 */
export function getProjectConfig(projectRoot: string = process.cwd()): {
  projectId?: string;
  apiKey?: string;
  baseURL?: string;
} | null {
  const configPath = join(projectRoot, ".bunbase", "config.json");
  if (existsSync(configPath)) {
    try {
      const content = readFileSync(configPath, "utf-8");
      return JSON.parse(content);
    } catch (error) {
      return null;
    }
  }
  return null;
}

/**
 * Type definitions for BunBase Server SDK
 */

// Function Types
export type FunctionType = "http" | "callable";
export type FunctionRuntime =
  | "nodejs18"
  | "nodejs20"
  | "nodejs22"
  | "bun"
  | "python3.10"
  | "python3.11"
  | "python3.12"
  | "go"
  | "deno";

export interface HTTPFunctionOptions {
  name: string;
  runtime: FunctionRuntime;
  handler: string;
  path: string;
  methods: string[];
  code?: string;
  memory?: number;
  timeout?: number;
  env?: Record<string, string>;
}

export interface CallableFunctionOptions {
  name: string;
  runtime: FunctionRuntime;
  handler: string;
  code?: string;
  memory?: number;
  timeout?: number;
  env?: Record<string, string>;
}

export interface FunctionResponse {
  id: string;
  name: string;
  runtime: FunctionRuntime;
  handler: string;
  type: FunctionType;
  path?: string;
  methods?: string[];
  memory?: number;
  timeout?: number;
  createdAt: Date;
  updatedAt: Date;
}

export interface FunctionLog {
  id: string;
  functionId: string;
  level: "debug" | "info" | "warn" | "error";
  message: string;
  timestamp: Date;
  metadata?: Record<string, any>;
}

export interface FunctionMetrics {
  invocations: number;
  errors: number;
  averageDuration: number;
  lastInvoked: Date | null;
  coldStarts: number;
  memoryUsage?: number;
}

// Admin Resource Types
export interface Project {
  id: string;
  name: string;
  description?: string;
  ownerId: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface Application {
  id: string;
  projectId: string;
  name: string;
  description?: string;
  type: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface Database {
  databaseId: string;
  name: string;
  projectId: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface StorageBucket {
  storageId: string;
  name: string;
  projectId: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface Collection {
  collectionId: string;
  name: string;
  databaseId: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface APIKey {
  id: string;
  applicationId: string;
  key: string;
  createdAt: Date;
  lastUsedAt?: Date;
  revokedAt?: Date;
}

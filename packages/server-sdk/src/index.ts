/**
 * BunBase Server SDK
 *
 * Core client and module exports
 */

export {
  createServerClient,
  type ServerClient,
  type ServerSDKConfig,
} from "./client";
export { FunctionsModule, type FunctionsModuleOptions } from "./modules/functions";
export { AdminModule, type AdminModuleOptions } from "./modules/admin";

// Re-export types
export type {
  FunctionType,
  FunctionRuntime,
  HTTPFunctionOptions,
  CallableFunctionOptions,
  FunctionResponse,
  FunctionLog,
  FunctionMetrics,
  Project,
  Application,
  Database,
  StorageBucket,
  Collection,
  APIKey,
} from "./types";

/**
 * BunBase JavaScript/TypeScript SDK
 *
 * Core client and module exports
 */

export { createClient, type BunBaseClient, type BunBaseConfig } from "./client";
export { AuthModule, type AuthModuleOptions } from "./modules/auth";
export { DatabaseModule, type DatabaseModuleOptions } from "./modules/database";
export { StorageModule, type StorageModuleOptions } from "./modules/storage";
export {
  RealtimeModule,
  type RealtimeModuleOptions,
  type ConnectionState,
} from "./modules/realtime";

// Re-export types
export type {
  AuthUser,
  AuthSession,
  DatabaseDocument,
  DatabaseQuery,
  StorageFile,
  RealtimeMessage,
  RealtimeChannel,
} from "./types";

// Re-export BunStore API
export {
  BunStore,
  CollectionReference,
  DocumentReference,
  Query,
  QuerySnapshot,
  DocumentSnapshot,
  FieldValue,
  WriteBatch,
  Transaction,
  SnapshotMetadata,
} from "./bunstore";

export type {
  WhereFilterOp,
  OrderByDirection,
  FirestoreDataConverter,
  DocumentChange,
  SnapshotListenOptions,
  SetOptions,
  GetOptions,
} from "./bunstore";

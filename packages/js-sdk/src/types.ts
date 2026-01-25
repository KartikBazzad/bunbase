/**
 * Type definitions for BunBase SDK
 */

export interface AuthUser {
  id: string;
  email: string;
  name: string;
  emailVerified: boolean;
  image?: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface AuthSession {
  id: string;
  userId: string;
  expiresAt: Date;
  createdAt: Date;
}

export interface DatabaseDocument {
  documentId: string;
  collectionId: string;
  path: string;
  data: Record<string, any>;
  createdAt: Date;
  updatedAt: Date;
}

export interface DatabaseQuery {
  collectionPath?: string;
  filter?: Record<string, any>;
  sort?: Record<string, "asc" | "desc">;
  limit?: number;
  offset?: number;
}

export interface StorageFile {
  fileId: string;
  bucketId: string;
  path: string;
  size: number;
  mimeType: string;
  metadata: Record<string, any>;
  createdAt: Date;
  updatedAt: Date;
}

export interface RealtimeMessage {
  type: string;
  channel?: string;
  message?: any;
  sender?: string;
  timestamp: number;
}

export interface RealtimeChannel {
  name: string;
  subscribers: number;
}

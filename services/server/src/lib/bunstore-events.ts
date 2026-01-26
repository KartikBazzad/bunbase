/**
 * BunStore Event Emitter System
 * 
 * Since Bun.sql doesn't support live events, we use an EventEmitter
 * to notify listeners when documents are created, updated, or deleted.
 */

import { EventEmitter } from "events";

export interface DocumentEventPayload {
  projectId: string;
  collectionPath: string;
  documentId: string;
  path: string;
  data?: Record<string, any>;
  oldData?: Record<string, any>;
  createdAt?: Date;
  updatedAt?: Date;
}

export type DocumentEventType = "document:created" | "document:updated" | "document:deleted";

/**
 * Singleton EventEmitter for BunStore document lifecycle events
 */
class BunStoreEventEmitter extends EventEmitter {
  /**
   * Emit a document created event
   */
  emitCreated(payload: DocumentEventPayload): void {
    this.emit("document:created", payload);
  }

  /**
   * Emit a document updated event
   */
  emitUpdated(payload: DocumentEventPayload): void {
    this.emit("document:updated", payload);
  }

  /**
   * Emit a document deleted event
   */
  emitDeleted(payload: DocumentEventPayload): void {
    this.emit("document:deleted", payload);
  }

  /**
   * Subscribe to document created events
   */
  onCreated(handler: (payload: DocumentEventPayload) => void): void {
    this.on("document:created", handler);
  }

  /**
   * Subscribe to document updated events
   */
  onUpdated(handler: (payload: DocumentEventPayload) => void): void {
    this.on("document:updated", handler);
  }

  /**
   * Subscribe to document deleted events
   */
  onDeleted(handler: (payload: DocumentEventPayload) => void): void {
    this.on("document:deleted", handler);
  }

  /**
   * Subscribe to all document events
   */
  onAny(handler: (type: DocumentEventType, payload: DocumentEventPayload) => void): void {
    this.onCreated((payload) => handler("document:created", payload));
    this.onUpdated((payload) => handler("document:updated", payload));
    this.onDeleted((payload) => handler("document:deleted", payload));
  }

  /**
   * Remove all listeners (useful for cleanup)
   */
  removeAllListeners(): void {
    super.removeAllListeners();
  }
}

// Export singleton instance
export const bunstoreEvents = new BunStoreEventEmitter();

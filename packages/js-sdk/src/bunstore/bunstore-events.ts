/**
 * BunStore Event Manager
 * 
 * Handles document change subscriptions and transforms WebSocket messages
 * into BunStore events.
 */

import type { RealtimeMessage } from "../types";
import type { DocumentSnapshot } from "./types";
import { DocumentSnapshot as DocumentSnapshotImpl } from "./document-snapshot";
import type { DocumentReference } from "./document-reference";
import type { CollectionReference } from "./collection-reference";
import type { QuerySnapshot } from "./types";
import { QuerySnapshot as QuerySnapshotImpl } from "./query-snapshot";

export interface DocumentChangeEvent {
  type: "INSERT" | "UPDATE" | "DELETE";
  documentId: string;
  path: string;
  collectionPath: string;
  data?: Record<string, any>;
  oldData?: Record<string, any>;
  createdAt?: string;
  updatedAt?: string;
}

export type DocumentChangeHandler<T = any> = (
  event: DocumentChangeEvent,
  snapshot: DocumentSnapshot<T>,
) => void;

export type QueryChangeHandler<T = any> = (
  event: DocumentChangeEvent,
  snapshot: QuerySnapshot<T>,
) => void;

/**
 * Event Manager for BunStore document subscriptions
 */
export class BunStoreEventManager {
  private documentSubscriptions = new Map<
    string,
    {
      docRef: DocumentReference<any>;
      handler: DocumentChangeHandler;
      unsubscribe: () => void;
    }
  >();

  private querySubscriptions = new Map<
    string,
    {
      query: any; // Query instance
      handler: QueryChangeHandler;
      unsubscribe: () => void;
    }
  >();

  /**
   * Subscribe to document changes
   */
  subscribeToDocument<T>(
    docRef: DocumentReference<T>,
    handler: DocumentChangeHandler<T>,
    onMessage: (message: RealtimeMessage) => void,
  ): () => void {
    const subscriptionKey = `doc:${docRef.path}`;

    // If already subscribed, return existing unsubscribe
    if (this.documentSubscriptions.has(subscriptionKey)) {
      return this.documentSubscriptions.get(subscriptionKey)!.unsubscribe;
    }

    // Subscribe to both collection and document channels
    const collectionChannel = `db:${docRef.parent.path}`;
    const documentChannel = `db:${docRef.path}`;

    // Create message handler
    const messageHandler = (message: RealtimeMessage) => {
      if (
        message.channel === collectionChannel ||
        message.channel === documentChannel
      ) {
        if (
          message.type === "INSERT" ||
          message.type === "UPDATE" ||
          message.type === "DELETE"
        ) {
          const eventData = message.message as any;
          if (
            eventData.documentId === docRef.id ||
            eventData.path === docRef.path
          ) {
            const event: DocumentChangeEvent = {
              type: message.type,
              documentId: eventData.documentId,
              path: eventData.path,
              collectionPath: eventData.collectionPath,
              data: eventData.data,
              oldData: eventData.oldData,
              createdAt: eventData.createdAt,
              updatedAt: eventData.updatedAt,
            };

            // Create snapshot based on event type
            let snapshot: DocumentSnapshot<T>;
            if (message.type === "DELETE") {
              snapshot = new DocumentSnapshotImpl(
                docRef.id,
                docRef,
                undefined,
                false,
              );
            } else {
              snapshot = new DocumentSnapshotImpl(
                docRef.id,
                docRef,
                eventData.data,
                true,
              );
            }

            handler(event, snapshot);
          }
        }
      }
    };

    // Store original onMessage handler
    const originalOnMessage = onMessage;

    // Wrap onMessage to include our handler
    const wrappedOnMessage = (message: RealtimeMessage) => {
      originalOnMessage(message);
      messageHandler(message);
    };

    const unsubscribe = () => {
      this.documentSubscriptions.delete(subscriptionKey);
      // Note: Channel unsubscription should be handled by the realtime module
    };

    this.documentSubscriptions.set(subscriptionKey, {
      docRef,
      handler,
      unsubscribe,
    });

    return unsubscribe;
  }

  /**
   * Subscribe to query/collection changes
   */
  subscribeToQuery<T>(
    query: any, // Query instance
    handler: QueryChangeHandler<T>,
    onMessage: (message: RealtimeMessage) => void,
  ): () => void {
    const subscriptionKey = `query:${query.collectionPath}`;

    // If already subscribed, return existing unsubscribe
    if (this.querySubscriptions.has(subscriptionKey)) {
      return this.querySubscriptions.get(subscriptionKey)!.unsubscribe;
    }

    const collectionChannel = `db:${query.collectionPath}`;

    // Create message handler
    const messageHandler = async (message: RealtimeMessage) => {
      if (message.channel === collectionChannel) {
        if (
          message.type === "INSERT" ||
          message.type === "UPDATE" ||
          message.type === "DELETE"
        ) {
          const eventData = message.message as any;
          const event: DocumentChangeEvent = {
            type: message.type,
            documentId: eventData.documentId,
            path: eventData.path,
            collectionPath: eventData.collectionPath,
            data: eventData.data,
            oldData: eventData.oldData,
            createdAt: eventData.createdAt,
            updatedAt: eventData.updatedAt,
          };

          // Re-execute query to get updated snapshot
          try {
            const snapshot = await query.get();
            handler(event, snapshot);
          } catch (error) {
            console.error("Failed to get query snapshot:", error);
          }
        }
      }
    };

    // Store original onMessage handler
    const originalOnMessage = onMessage;

    // Wrap onMessage to include our handler
    const wrappedOnMessage = (message: RealtimeMessage) => {
      originalOnMessage(message);
      messageHandler(message);
    };

    const unsubscribe = () => {
      this.querySubscriptions.delete(subscriptionKey);
      // Note: Channel unsubscription should be handled by the realtime module
    };

    this.querySubscriptions.set(subscriptionKey, {
      query,
      handler,
      unsubscribe,
    });

    return unsubscribe;
  }

  /**
   * Remove all subscriptions
   */
  removeAllSubscriptions(): void {
    for (const sub of this.documentSubscriptions.values()) {
      sub.unsubscribe();
    }
    for (const sub of this.querySubscriptions.values()) {
      sub.unsubscribe();
    }
    this.documentSubscriptions.clear();
    this.querySubscriptions.clear();
  }
}

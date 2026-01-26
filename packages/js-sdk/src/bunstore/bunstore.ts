/**
 * BunStore - Main entry point for BunStore API
 */

import type { BunBaseClient, BunBaseConfig } from "../client";
import { CollectionReference } from "./collection-reference";
import { WriteBatch } from "./write-batch";
import { Transaction } from "./transaction";
import { DocumentReference } from "./document-reference";
import { DocumentSnapshot } from "./document-snapshot";
import type {
  DocumentSnapshot as IDocumentSnapshot,
  QuerySnapshot,
} from "./types";
import type { Query } from "./query";
import { BunStoreEventManager } from "./bunstore-events";

export class BunStore {
  private _subscriptions: Map<
    string,
    {
      unsubscribe: () => void;
      observer: {
        next?: (snapshot: any) => void;
        error?: (error: Error) => void;
        complete?: () => void;
      };
    }
  > = new Map();
  private _eventManager = new BunStoreEventManager();

  constructor(
    private _client: BunBaseClient,
    private _config: BunBaseConfig,
  ) {}

  /**
   * Gets a CollectionReference instance that refers to the collection at the specified path.
   */
  collection<T = any>(collectionPath: string): CollectionReference<T> {
    return new CollectionReference<T>(this, collectionPath);
  }

  /**
   * Creates a write batch, used for performing multiple writes as a single atomic operation.
   */
  batch(): WriteBatch {
    return new WriteBatch(this);
  }

  /**
   * Executes the given updateFunction and then attempts to commit the changes applied within the transaction.
   */
  async runTransaction<T>(
    updateFunction: (transaction: Transaction) => Promise<T>,
  ): Promise<T> {
    const transaction = new Transaction(this);
    const result = await updateFunction(transaction);
    await transaction.commit();
    return result;
  }

  /**
   * Internal method to make requests (exposed for use by references)
   */
  async _request<T>(
    method: string,
    path: string,
    options?: {
      body?: any;
      headers?: Record<string, string>;
      query?: Record<string, string | number | boolean>;
    },
  ): Promise<T> {
    return this._client.request<T>(method, path, options);
  }

  /**
   * Internal method to subscribe to document changes
   */
  _subscribeToDocument<T>(
    docRef: DocumentReference<T>,
    observer: {
      next?: (snapshot: IDocumentSnapshot<T>) => void;
      error?: (error: Error) => void;
      complete?: () => void;
    },
  ): () => void {
    const subscriptionKey = `doc:${docRef.path}`;

    // Subscribe to both collection and document channels
    const collectionChannel = `db:${docRef.parent.path}`;
    const documentChannel = `db:${docRef.path}`;

    // Set up WebSocket subscription
    let wsConnected = false;
    let unsubscribeFn: (() => void) | null = null;
    let realtimeUnsubscribe: (() => void) | null = null;

    const connectRealtime = async () => {
      try {
        // Try to connect to realtime (may require authentication)
        // For API key-based auth, we'll need to handle this differently
        const ws = await this._client.realtime.connect({
          projectId: this._config.projectId,
          onMessage: (message) => {
            // Handle document change events from both channels
            if (
              message.channel === collectionChannel ||
              message.channel === documentChannel
            ) {
              const eventData = message.message as any;
              if (
                eventData?.documentId === docRef.id ||
                eventData?.path === docRef.path
              ) {
                if (message.type === "INSERT" || message.type === "UPDATE") {
                  const snapshot = new DocumentSnapshot(
                    docRef.id,
                    docRef,
                    eventData.data,
                    true,
                  );
                  observer.next?.(snapshot);
                } else if (message.type === "DELETE") {
                  const snapshot = new DocumentSnapshot(
                    docRef.id,
                    docRef,
                    undefined,
                    false,
                  );
                  observer.next?.(snapshot);
                }
              }
            }
          },
          onError: (error) => {
            observer.error?.(error);
          },
          onClose: () => {
            wsConnected = false;
            observer.complete?.();
          },
        });

        wsConnected = true;
        // Subscribe to both channels
        this._client.realtime.subscribe(collectionChannel);
        this._client.realtime.subscribe(documentChannel);

        realtimeUnsubscribe = () => {
          this._client.realtime.unsubscribe(collectionChannel);
          this._client.realtime.unsubscribe(documentChannel);
        };
      } catch (error) {
        // If connection fails (e.g., no auth), still allow initial fetch
        console.warn("Realtime connection failed, using polling mode:", error);
        observer.error?.(
          error instanceof Error ? error : new Error(String(error)),
        );
      }
    };

    // Initial fetch
    docRef
      .get()
      .then((snapshot) => {
        observer.next?.(snapshot);
        connectRealtime();
      })
      .catch((error) => {
        observer.error?.(error);
      });

    // Return unsubscribe function
    unsubscribeFn = () => {
      if (wsConnected && realtimeUnsubscribe) {
        realtimeUnsubscribe();
        wsConnected = false;
      }
      this._subscriptions.delete(subscriptionKey);
    };

    this._subscriptions.set(subscriptionKey, {
      unsubscribe: unsubscribeFn,
      observer,
    });

    return unsubscribeFn;
  }

  /**
   * Internal method to subscribe to query changes
   */
  _subscribeToQuery<T>(
    query: Query<T>,
    observer: {
      next?: (snapshot: QuerySnapshot<T>) => void;
      error?: (error: Error) => void;
      complete?: () => void;
    },
  ): () => void {
    const subscriptionKey = `query:${query.collectionPath}`;

    // Subscribe to collection channel
    const channel = `db:${query.collectionPath}`;

    // Set up WebSocket subscription
    let wsConnected = false;
    let unsubscribeFn: (() => void) | null = null;
    let realtimeUnsubscribe: (() => void) | null = null;

    const connectRealtime = async () => {
      try {
        const ws = await this._client.realtime.connect({
          projectId: this._config.projectId,
          onMessage: (message) => {
            if (message.channel === channel) {
              // Re-execute query on changes
              query
                .get()
                .then((snapshot) => {
                  observer.next?.(snapshot);
                })
                .catch((error) => {
                  observer.error?.(error);
                });
            }
          },
          onError: (error) => {
            observer.error?.(error);
          },
          onClose: () => {
            wsConnected = false;
            observer.complete?.();
          },
        });

        wsConnected = true;
        this._client.realtime.subscribe(channel);

        realtimeUnsubscribe = () => {
          this._client.realtime.unsubscribe(channel);
        };
      } catch (error) {
        // If connection fails, still allow initial fetch
        console.warn("Realtime connection failed, using polling mode:", error);
        observer.error?.(
          error instanceof Error ? error : new Error(String(error)),
        );
      }
    };

    // Initial fetch
    query
      .get()
      .then((snapshot) => {
        observer.next?.(snapshot);
        connectRealtime();
      })
      .catch((error) => {
        observer.error?.(error);
      });

    // Return unsubscribe function
    unsubscribeFn = () => {
      if (wsConnected && realtimeUnsubscribe) {
        realtimeUnsubscribe();
        wsConnected = false;
      }
      this._subscriptions.delete(subscriptionKey);
    };

    this._subscriptions.set(subscriptionKey, {
      unsubscribe: unsubscribeFn,
      observer,
    });

    return unsubscribeFn;
  }
}

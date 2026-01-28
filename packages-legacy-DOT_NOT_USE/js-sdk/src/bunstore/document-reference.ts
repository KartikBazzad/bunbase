/**
 * DocumentReference - Represents a document in BunStore
 */

import type {
  DocumentReference as IDocumentReference,
  DocumentSnapshot,
  SetOptions,
  GetOptions,
  SnapshotListenOptions,
} from "./types";
import { DocumentSnapshot as DocumentSnapshotImpl } from "./document-snapshot";
import { serializeFieldValue } from "./utils";
import type { BunStore } from "./bunstore";
import type { CollectionReference } from "./collection-reference";

export class DocumentReference<T = any> implements IDocumentReference {
  readonly id: string;
  readonly path: string;
  readonly parent: CollectionReference<T>;

  constructor(
    private _bunstore: BunStore,
    private _collectionPath: string,
    id: string,
    parent?: CollectionReference<T>,
  ) {
    this.id = id;
    this.path = `${_collectionPath}/${id}`;
    this.parent = parent as CollectionReference<T>;
  }

  collection(path: string): CollectionReference {
    const { CollectionReference } = require("./collection-reference");
    return new CollectionReference(this._firestore, `${this.path}/${path}`);
  }

  async get(options?: GetOptions): Promise<DocumentSnapshot<T>> {
    try {
      const response = await this._bunstore._request<{
        data: {
          documentId: string;
          collectionId: string;
          path: string;
          data: T;
          createdAt: Date;
          updatedAt: Date;
        };
      }>("GET", `/db/${this._collectionPath}/${this.id}`);

      const doc = response.data;
      return new DocumentSnapshotImpl(doc.documentId, this, doc.data, true);
    } catch (error: any) {
      if (
        error.status === 404 ||
        error.code === "NOT_FOUND" ||
        error.code === "not-found"
      ) {
        return new DocumentSnapshotImpl(this.id, this, undefined, false);
      }
      const { mapBackendError } = await import("./utils");
      throw mapBackendError(error);
    }
  }

  async set(data: Partial<T>, options?: SetOptions): Promise<void> {
    try {
      const serializedData = serializeFieldValue(data);

      if (options?.merge || options?.mergeFields) {
        // Use PATCH for merge operations
        await this._bunstore._request(
          "PATCH",
          `/db/${this._collectionPath}/${this.id}`,
          {
            body: { data: serializedData },
          },
        );
      } else {
        // Use PUT for full replacement (upsert)
        await this._bunstore._request(
          "PUT",
          `/db/${this._collectionPath}/${this.id}`,
          {
            body: { data: serializedData },
          },
        );
      }
    } catch (error: any) {
      const { mapBackendError } = await import("./utils");
      throw mapBackendError(error);
    }
  }

  async update(data: Partial<T>): Promise<void> {
    try {
      const serializedData = serializeFieldValue(data);
      await this._bunstore._request(
        "PATCH",
        `/db/${this._collectionPath}/${this.id}`,
        {
          body: { data: serializedData },
        },
      );
    } catch (error: any) {
      const { mapBackendError } = await import("./utils");
      throw mapBackendError(error);
    }
  }

  async delete(): Promise<void> {
    try {
      await this._bunstore._request(
        "DELETE",
        `/db/${this._collectionPath}/${this.id}`,
      );
    } catch (error: any) {
      const { mapBackendError } = await import("./utils");
      throw mapBackendError(error);
    }
  }

  onSnapshot(
    observerOrOnNext:
      | {
          next?: (snapshot: DocumentSnapshot<T>) => void;
          error?: (error: Error) => void;
          complete?: () => void;
        }
      | ((snapshot: DocumentSnapshot<T>) => void),
    onError?: (error: Error) => void,
    onCompletion?: () => void,
  ): () => void {
    // Real-time listener implementation
    const observer =
      typeof observerOrOnNext === "function"
        ? {
            next: observerOrOnNext,
            error: onError,
            complete: onCompletion,
          }
        : observerOrOnNext;

    // Subscribe to real-time updates
    return this._bunstore._subscribeToDocument(this, observer);
  }
}

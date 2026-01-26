/**
 * CollectionReference - Represents a collection in BunStore
 */

import type { CollectionReference as ICollectionReference } from "./types";
import { Query } from "./query";
import { DocumentReference } from "./document-reference";
import { serializeFieldValue } from "./utils";
import type { BunStore } from "./bunstore";
// Generate a simple ID
function generateId(): string {
  // Use crypto.randomUUID if available, otherwise fallback to timestamp + random
  if (typeof crypto !== "undefined" && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).substring(2, 15)}`;
}

export class CollectionReference<T = any>
  extends Query<T>
  implements ICollectionReference
{
  readonly id: string;
  readonly path: string;
  readonly parent: DocumentReference | null;
  protected _firestore: BunStore;

  constructor(
    _bunstore: BunStore,
    collectionPath: string,
    parent?: DocumentReference | null,
  ) {
    super(_bunstore, collectionPath);
    this._firestore = _bunstore;
    const parts = collectionPath.split("/");
    this.id = parts[parts.length - 1];
    this.path = collectionPath;
    this.parent = parent || null;
  }

  doc(documentPath?: string): DocumentReference<T> {
    const id = documentPath || generateId();
    return new DocumentReference<T>(this._firestore, this.path, id, this);
  }

  async add(data: T): Promise<DocumentReference<T>> {
    try {
      const serializedData = serializeFieldValue(data);
      const response = await this._firestore._request<{
        data: {
          documentId: string;
          collectionId: string;
          path: string;
          data: T;
          createdAt: Date;
          updatedAt: Date;
        };
      }>("POST", `/db/${this.path}`, {
        body: { data: serializedData },
      });

      return new DocumentReference<T>(
        this._firestore,
        this.path,
        response.data.documentId,
        this,
      );
    } catch (error: any) {
      const { mapBackendError } = await import("./utils");
      throw mapBackendError(error);
    }
  }
}

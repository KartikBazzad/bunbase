/**
 * WriteBatch - Batch write operations
 */

import type { WriteBatch as IWriteBatch, SetOptions } from "./types";
import { DocumentReference } from "./document-reference";
import { serializeFieldValue } from "./utils";
import type { BunStore } from "./bunstore";

interface BatchOperation {
  type: "set" | "update" | "delete";
  ref: DocumentReference<any>;
  data?: any;
  options?: SetOptions;
}

export class WriteBatch implements IWriteBatch {
  private _operations: BatchOperation[] = [];

  constructor(private _bunstore: BunStore) {}

  set<T>(
    documentRef: DocumentReference<T>,
    data: Partial<T>,
    options?: SetOptions,
  ): WriteBatch {
    this._operations.push({
      type: "set",
      ref: documentRef,
      data: serializeFieldValue(data),
      options,
    });
    return this;
  }

  update<T>(documentRef: DocumentReference<T>, data: Partial<T>): WriteBatch {
    this._operations.push({
      type: "update",
      ref: documentRef,
      data: serializeFieldValue(data),
    });
    return this;
  }

  delete(documentRef: DocumentReference<any>): WriteBatch {
    this._operations.push({
      type: "delete",
      ref: documentRef,
    });
    return this;
  }

  async commit(): Promise<void> {
    if (this._operations.length === 0) {
      return;
    }

    // Group operations by collection
    const operationsByCollection = new Map<string, BatchOperation[]>();

    for (const op of this._operations) {
      const collectionPath = op.ref.path.split("/").slice(0, -1).join("/");
      if (!operationsByCollection.has(collectionPath)) {
        operationsByCollection.set(collectionPath, []);
      }
      operationsByCollection.get(collectionPath)!.push(op);
    }

    // Execute batches per collection
    for (const [collectionPath, operations] of operationsByCollection) {
      const batchOps = operations.map((op) => {
        if (op.type === "set") {
          return {
            type: op.options?.merge ? "upsert" : "create",
            documentId: op.ref.id,
            data: op.data,
          };
        } else if (op.type === "update") {
          return {
            type: "update",
            documentId: op.ref.id,
            data: op.data,
          };
        } else {
          return {
            type: "delete",
            documentId: op.ref.id,
          };
        }
      });

      await this._bunstore._request("POST", `/db/${collectionPath}/batch`, {
        body: { operations: batchOps },
      });
    }

    // Clear operations after successful commit
    this._operations = [];
  }
}

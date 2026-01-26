/**
 * Transaction - Transaction operations
 */

import type { Transaction as ITransaction, SetOptions } from "./types";
import { DocumentReference } from "./document-reference";
import { DocumentSnapshot } from "./document-snapshot";
import { serializeFieldValue } from "./utils";
import type { BunStore } from "./bunstore";

interface TransactionOperation {
  type: "get" | "set" | "update" | "delete";
  ref: DocumentReference<any>;
  data?: any;
  options?: SetOptions;
}

export class Transaction implements ITransaction {
  private _operations: TransactionOperation[] = [];
  private _reads: Map<string, any> = new Map();

  constructor(private _bunstore: BunStore) {}

  async get<T>(
    documentRef: DocumentReference<T>,
  ): Promise<DocumentSnapshot<T>> {
    const key = documentRef.path;

    // Check if we've already read this document in this transaction
    if (this._reads.has(key)) {
      return this._reads.get(key) as DocumentSnapshot<T>;
    }

    // Read the document
    const snapshot = await documentRef.get();
    this._reads.set(key, snapshot as any);
    this._operations.push({
      type: "get",
      ref: documentRef,
    });

    return snapshot;
  }

  set<T>(
    documentRef: DocumentReference<T>,
    data: Partial<T>,
    options?: SetOptions,
  ): Transaction {
    this._operations.push({
      type: "set",
      ref: documentRef,
      data: serializeFieldValue(data),
      options,
    });
    return this;
  }

  update<T>(documentRef: DocumentReference<T>, data: Partial<T>): Transaction {
    this._operations.push({
      type: "update",
      ref: documentRef,
      data: serializeFieldValue(data),
    });
    return this;
  }

  delete(documentRef: DocumentReference<any>): Transaction {
    this._operations.push({
      type: "delete",
      ref: documentRef,
    });
    return this;
  }

  async commit(): Promise<void> {
    // For now, transactions are implemented as batch operations
    // A full implementation would require backend transaction support
    const batch = this._bunstore.batch();

    for (const op of this._operations) {
      if (op.type === "get") {
        // Reads are already done, skip
        continue;
      } else if (op.type === "set") {
        batch.set(op.ref, op.data, op.options);
      } else if (op.type === "update") {
        batch.update(op.ref, op.data);
      } else if (op.type === "delete") {
        batch.delete(op.ref);
      }
    }

    await batch.commit();
  }
}

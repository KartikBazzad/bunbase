/**
 * QuerySnapshot - Represents the results of a query
 */

import type { Query } from "./types";
import type { QuerySnapshot as IQuerySnapshot, DocumentChange } from "./types";
import { DocumentSnapshot } from "./document-snapshot";

export class QuerySnapshot<T = any> implements IQuerySnapshot {
  readonly docs: Array<DocumentSnapshot<T>>;
  readonly empty: boolean;
  readonly size: number;
  readonly query: Query<T>;

  constructor(
    docs: Array<DocumentSnapshot<T>>,
    query: Query<T>,
  ) {
    this.docs = docs;
    this.query = query;
    this.size = docs.length;
    this.empty = docs.length === 0;
  }

  forEach(
    callback: (result: DocumentSnapshot<T>) => void,
    thisArg?: any,
  ): void {
    this.docs.forEach(callback, thisArg);
  }

  docChanges(options?: { includeMetadataChanges?: boolean }): Array<DocumentChange<T>> {
    // For now, return all docs as "added" changes
    // In a real implementation with real-time listeners, this would track actual changes
    return this.docs.map((doc, index) => ({
      type: "added" as const,
      doc,
      oldIndex: -1,
      newIndex: index,
    }));
  }
}

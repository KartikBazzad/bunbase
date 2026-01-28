/**
 * Query - Query builder for BunStore
 */

import type {
  Query as IQuery,
  QuerySnapshot,
  WhereFilterOp,
  OrderByDirection,
  SnapshotListenOptions,
} from "./types";
import { QuerySnapshot as QuerySnapshotImpl } from "./query-snapshot";
import { buildFilter, buildSort, serializeFieldValue } from "./utils";
import type { BunStore } from "./bunstore";
import type { CollectionReference } from "./collection-reference";

export class Query<T = any> implements IQuery<T> {
  protected _whereConstraints: Array<{
    field: string;
    op: WhereFilterOp;
    value: any;
  }> = [];
  protected _orderByConstraints: Array<{
    field: string;
    direction: OrderByDirection;
  }> = [];
  protected _limitValue?: number;
  protected _offsetValue?: number;

  constructor(
    protected _bunstore: BunStore,
    protected _collectionPath: string,
  ) {}

  /**
   * Get the collection path (for internal use)
   */
  get collectionPath(): string {
    return this._collectionPath;
  }

  where(fieldPath: string, opStr: WhereFilterOp, value: any): Query<T> {
    const query = new Query<T>(this._bunstore, this._collectionPath);
    query._whereConstraints = [
      ...this._whereConstraints,
      { field: fieldPath, op: opStr, value },
    ];
    query._orderByConstraints = [...this._orderByConstraints];
    query._limitValue = this._limitValue;
    query._offsetValue = this._offsetValue;
    return query;
  }

  orderBy(fieldPath: string, directionStr: OrderByDirection = "asc"): Query<T> {
    const query = new Query<T>(this._bunstore, this._collectionPath);
    query._whereConstraints = [...this._whereConstraints];
    query._orderByConstraints = [
      ...this._orderByConstraints,
      { field: fieldPath, direction: directionStr },
    ];
    query._limitValue = this._limitValue;
    query._offsetValue = this._offsetValue;
    return query;
  }

  limit(limit: number): Query<T> {
    const query = new Query<T>(this._bunstore, this._collectionPath);
    query._whereConstraints = [...this._whereConstraints];
    query._orderByConstraints = [...this._orderByConstraints];
    query._limitValue = limit;
    query._offsetValue = this._offsetValue;
    return query;
  }

  offset(offset: number): Query<T> {
    const query = new Query<T>(this._bunstore, this._collectionPath);
    query._whereConstraints = [...this._whereConstraints];
    query._orderByConstraints = [...this._orderByConstraints];
    query._limitValue = this._limitValue;
    query._offsetValue = offset;
    return query;
  }

  async get(): Promise<QuerySnapshot<T>> {
    try {
      const filter = buildFilter(this._whereConstraints);
      const sort = buildSort(this._orderByConstraints);

      const response = await this._bunstore._request<{
        data: Array<{
          documentId: string;
          collectionId: string;
          path: string;
          data: T;
          createdAt: Date;
          updatedAt: Date;
        }>;
        total: number;
        limit: number;
        offset: number;
      }>("GET", `/db/${this._collectionPath}`, {
        query: {
          ...(Object.keys(filter).length > 0 && {
            filter: JSON.stringify(filter),
          }),
          ...(Object.keys(sort).length > 0 && { sort: JSON.stringify(sort) }),
          ...(this._limitValue !== undefined && { limit: this._limitValue }),
          ...(this._offsetValue !== undefined && { offset: this._offsetValue }),
        },
      });

      // Import classes dynamically to avoid circular dependencies
      const { DocumentReference } = await import("./document-reference");
      const { DocumentSnapshot } = await import("./document-snapshot");

      const docs = await Promise.all(
        response.data.map(async (doc) => {
          const ref = new DocumentReference(
            this._bunstore,
            this._collectionPath,
            doc.documentId,
          );
          return new DocumentSnapshot(doc.documentId, ref, doc.data, true);
        }),
      );

      return new QuerySnapshotImpl(docs, this);
    } catch (error: any) {
      const { mapBackendError } = await import("./utils");
      throw mapBackendError(error);
    }
  }

  onSnapshot(
    observerOrOnNext:
      | {
          next?: (snapshot: QuerySnapshot<T>) => void;
          error?: (error: Error) => void;
          complete?: () => void;
        }
      | ((snapshot: QuerySnapshot<T>) => void),
    onError?: (error: Error) => void,
    onCompletion?: () => void,
  ): () => void {
    const observer =
      typeof observerOrOnNext === "function"
        ? {
            next: observerOrOnNext,
            error: onError,
            complete: onCompletion,
          }
        : observerOrOnNext;

    // Use BunStore's subscription method
    return this._bunstore._subscribeToQuery(this, observer);
  }
}

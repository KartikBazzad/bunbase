/**
 * BunStore API Exports
 */

export { BunStore } from "./bunstore";
export { CollectionReference } from "./collection-reference";
export { DocumentReference } from "./document-reference";
export { Query } from "./query";
export { QuerySnapshot } from "./query-snapshot";
export { DocumentSnapshot, SnapshotMetadata } from "./document-snapshot";
export { FieldValue } from "./field-value";
export { WriteBatch } from "./write-batch";
export { Transaction } from "./transaction";

export type {
  WhereFilterOp,
  OrderByDirection,
  BunStoreDataConverter,
  FirestoreDataConverter,
  DocumentChange,
  SnapshotListenOptions,
  SetOptions,
  GetOptions,
  DocumentSnapshot as IDocumentSnapshot,
  QuerySnapshot as IQuerySnapshot,
  SnapshotMetadata as ISnapshotMetadata,
  DocumentReference as IDocumentReference,
  CollectionReference as ICollectionReference,
  Query as IQuery,
  WriteBatch as IWriteBatch,
  Transaction as ITransaction,
  BunStoreError,
  FirestoreError,
} from "./types";

/**
 * BunStore API Type Definitions
 */

export type WhereFilterOp =
  | "<"
  | "<="
  | "=="
  | "!="
  | ">="
  | ">"
  | "array-contains"
  | "array-contains-any"
  | "in"
  | "not-in";

export type OrderByDirection = "asc" | "desc";

export interface BunStoreDataConverter<T> {
  toBunStore(data: T): Record<string, any>;
  fromBunStore(snapshot: DocumentSnapshot, options: any): T;
}

// Alias for backward compatibility
export type FirestoreDataConverter<T> = BunStoreDataConverter<T>;

export interface DocumentChange<T = any> {
  type: "added" | "modified" | "removed";
  doc: DocumentSnapshot<T>;
  oldIndex: number;
  newIndex: number;
}

export interface SnapshotListenOptions {
  readonly includeMetadataChanges?: boolean;
}

export interface SetOptions {
  readonly merge?: boolean;
  readonly mergeFields?: string[];
}

export interface GetOptions {
  readonly source?: "default" | "server" | "cache";
}

export interface DocumentSnapshot<T = any> {
  readonly id: string;
  readonly ref: DocumentReference<T>;
  exists(): boolean;
  data(): T | undefined;
  get(fieldPath: string | string[]): any;
  readonly metadata: SnapshotMetadata;
}

export interface QuerySnapshot<T = any> {
  readonly docs: Array<DocumentSnapshot<T>>;
  readonly empty: boolean;
  readonly size: number;
  readonly query: Query<T>;
  forEach(callback: (result: DocumentSnapshot<T>) => void, thisArg?: any): void;
  docChanges(options?: SnapshotListenOptions): Array<DocumentChange<T>>;
}

export interface SnapshotMetadata {
  readonly hasPendingWrites: boolean;
  readonly fromCache: boolean;
  isEqual(other: SnapshotMetadata): boolean;
}

export interface DocumentReference<T = any> {
  readonly id: string;
  readonly path: string;
  readonly parent: CollectionReference<T>;
  collection(path: string): CollectionReference;
  get(options?: GetOptions): Promise<DocumentSnapshot<T>>;
  set(data: Partial<T>, options?: SetOptions): Promise<void>;
  update(data: Partial<T>): Promise<void>;
  delete(): Promise<void>;
  onSnapshot(observer: {
    next?: (snapshot: DocumentSnapshot<T>) => void;
    error?: (error: Error) => void;
    complete?: () => void;
  }): () => void;
  onSnapshot(
    onNext: (snapshot: DocumentSnapshot<T>) => void,
    onError?: (error: Error) => void,
    onCompletion?: () => void,
  ): () => void;
}

export interface CollectionReference<T = any> extends Query<T> {
  readonly id: string;
  readonly path: string;
  readonly parent: DocumentReference | null;
  doc(documentPath?: string): DocumentReference<T>;
  add(data: T): Promise<DocumentReference<T>>;
}

export interface Query<T = any> {
  where(fieldPath: string, opStr: WhereFilterOp, value: any): Query<T>;
  orderBy(fieldPath: string, directionStr?: OrderByDirection): Query<T>;
  limit(limit: number): Query<T>;
  offset(offset: number): Query<T>;
  get(): Promise<QuerySnapshot<T>>;
  onSnapshot(observer: {
    next?: (snapshot: QuerySnapshot<T>) => void;
    error?: (error: Error) => void;
    complete?: () => void;
  }): () => void;
  onSnapshot(
    onNext: (snapshot: QuerySnapshot<T>) => void,
    onError?: (error: Error) => void,
    onCompletion?: () => void,
  ): () => void;
}

export interface WriteBatch {
  set<T>(
    documentRef: DocumentReference<T>,
    data: Partial<T>,
    options?: SetOptions,
  ): WriteBatch;
  update<T>(documentRef: DocumentReference<T>, data: Partial<T>): WriteBatch;
  delete(documentRef: DocumentReference<any>): WriteBatch;
  commit(): Promise<void>;
}

export interface Transaction {
  get<T>(documentRef: DocumentReference<T>): Promise<DocumentSnapshot<T>>;
  set<T>(
    documentRef: DocumentReference<T>,
    data: Partial<T>,
    options?: SetOptions,
  ): Transaction;
  update<T>(documentRef: DocumentReference<T>, data: Partial<T>): Transaction;
  delete(documentRef: DocumentReference<any>): Transaction;
}

export interface BunStoreError extends Error {
  code: string;
  message: string;
  stack?: string;
}

// Alias for backward compatibility
export type FirestoreError = BunStoreError;

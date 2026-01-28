/**
 * DocumentSnapshot - Represents a document read from BunStore
 */

import type { DocumentReference } from "./types";
import type { DocumentSnapshot as IDocumentSnapshot } from "./types";

export class DocumentSnapshot<T = any> implements IDocumentSnapshot {
  readonly id: string;
  readonly ref: DocumentReference<T>;
  readonly metadata: SnapshotMetadata;

  private _data: T | undefined;
  private _exists: boolean;

  constructor(
    id: string,
    ref: DocumentReference<T>,
    data: T | undefined,
    exists: boolean,
    metadata?: Partial<SnapshotMetadata>,
  ) {
    this.id = id;
    this.ref = ref;
    this._data = data;
    this._exists = exists;
    this.metadata = new SnapshotMetadata(metadata);
  }

  exists(): boolean {
    return this._exists;
  }

  data(): T | undefined {
    return this._data;
  }

  get(fieldPath: string | string[]): any {
    if (!this._exists || !this._data) {
      return undefined;
    }

    const path = Array.isArray(fieldPath) ? fieldPath : fieldPath.split(".");
    let value: any = this._data;

    for (const segment of path) {
      if (value && typeof value === "object" && segment in value) {
        value = value[segment];
      } else {
        return undefined;
      }
    }

    return value;
  }
}

export class SnapshotMetadata {
  readonly hasPendingWrites: boolean;
  readonly fromCache: boolean;

  constructor(options?: Partial<SnapshotMetadata>) {
    this.hasPendingWrites = options?.hasPendingWrites ?? false;
    this.fromCache = options?.fromCache ?? false;
  }

  isEqual(other: SnapshotMetadata): boolean {
    return (
      this.hasPendingWrites === other.hasPendingWrites &&
      this.fromCache === other.fromCache
    );
  }
}

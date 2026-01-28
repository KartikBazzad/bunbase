export enum OperationType {
  Create = 1,
  Read = 2,
  Update = 3,
  Delete = 4,
  Patch = 7,
  CreateCollection = 8,
  DeleteCollection = 9,
}

export enum Status {
  OK = 0,
  Error = 1,
  NotFound = 2,
  Conflict = 3,
  MemoryLimit = 4,
}

export enum Command {
  OpenDB = 1,
  CloseDB = 2,
  Execute = 3,
  Stats = 4,
  CreateCollection = 5,
  DeleteCollection = 6,
  ListCollections = 7,
}

export interface PatchOperation {
  op: "set" | "delete" | "insert";
  path: string;
  value?: any;
}

export interface Operation {
  opType: OperationType;
  collection?: string;
  docID: bigint;
  payload: Uint8Array | null;
  patchOps?: PatchOperation[];
}

export interface RequestFrame {
  requestID: bigint;
  dbID: bigint;
  command: Command;
  opCount: number;
  ops: Operation[];
}

export interface ResponseFrame {
  requestID: bigint;
  status: Status;
  data: Uint8Array;
}

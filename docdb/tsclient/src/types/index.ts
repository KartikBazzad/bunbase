export interface Document {
  id: bigint;
  payload: Uint8Array;
}

export interface DocDBStats {
  totalDBs: number;
  activeDBs: number;
  totalTxns: bigint;
  walSize: bigint;
  memoryUsed: bigint;
  memoryCapacity: bigint;
}

export interface ClientOptions {
  socketPath?: string;
  autoConnect?: boolean;
  timeout?: number;
}

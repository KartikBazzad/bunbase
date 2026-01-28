import { ClientOptions, DocDBStats } from "./types";
import {
  RequestFrame,
  ResponseFrame,
  Operation,
  OperationType,
  Command,
  Status,
  PatchOperation,
} from "./types/protocol";
import { DocDBError, ConnectionError, ValidationError } from "./types/errors";
import { SocketConnection } from "./connection/socket";
import { FrameHandler } from "./connection/frame";
import { ProtocolEncoder } from "./protocol/encoder";
import { ProtocolDecoder } from "./protocol/decoder";
import {
  readLittleEndianUint64,
  stringToUint8Array,
  uint8ArrayToString,
} from "./utils/buffer";

export class DocDBClient {
  private options: ClientOptions;
  private conn?: SocketConnection;
  private frameHandler?: FrameHandler;
  private requestID: bigint = 1n;

  constructor(options: ClientOptions = {}) {
    this.options = {
      socketPath: options.socketPath ?? "/tmp/docdb.sock",
      autoConnect: options.autoConnect ?? true,
      timeout: options.timeout ?? 30000,
    };
  }

  async connect(): Promise<void> {
    if (!this.conn) {
      const socketPath = this.options.socketPath || "/tmp/docdb.sock";
      this.conn = new SocketConnection(socketPath);
      this.frameHandler = new FrameHandler(this.conn);
    }

    await this.conn.connect();
  }

  async disconnect(): Promise<void> {
    await this.conn?.disconnect();
    this.conn = undefined;
    this.frameHandler = undefined;
  }

  async openDB(name: string): Promise<bigint> {
    await this.ensureConnected();
    const reqID = this.nextRequestID();

    const frame: RequestFrame = {
      requestID: reqID,
      command: Command.OpenDB,
      dbID: 0n,
      opCount: 1,
      ops: [
        {
          opType: OperationType.Create,
          docID: 0n,
          payload: stringToUint8Array(name),
        },
      ],
    };

    const response = await this.sendRequest(frame);

    if (response.status !== Status.OK) {
      throw new DocDBError(uint8ArrayToString(response.data), response.status);
    }

    if (response.data.length !== 8) {
      throw new ValidationError("Invalid DB ID response");
    }

    return readLittleEndianUint64(response.data);
  }

  async closeDB(dbID: bigint): Promise<void> {
    await this.ensureConnected();
    const reqID = this.nextRequestID();

    const frame: RequestFrame = {
      requestID: reqID,
      command: Command.CloseDB,
      dbID: dbID,
      opCount: 0,
      ops: [],
    };

    const response = await this.sendRequest(frame);

    if (response.status !== Status.OK) {
      throw new DocDBError(uint8ArrayToString(response.data), response.status);
    }
  }

  async create(
    dbID: bigint,
    collection: string,
    docID: bigint,
    payload: Uint8Array,
  ): Promise<void> {
    await this.ensureConnected();
    const reqID = this.nextRequestID();

    if (!collection) {
      collection = "_default";
    }

    const frame: RequestFrame = {
      requestID: reqID,
      command: Command.Execute,
      dbID: dbID,
      opCount: 1,
      ops: [
        {
          opType: OperationType.Create,
          collection: collection,
          docID: docID,
          payload: payload,
        },
      ],
    };

    const response = await this.sendRequest(frame);

    if (response.status !== Status.OK) {
      throw new DocDBError(uint8ArrayToString(response.data), response.status);
    }
  }

  async read(
    dbID: bigint,
    collection: string,
    docID: bigint,
  ): Promise<Uint8Array> {
    await this.ensureConnected();
    const reqID = this.nextRequestID();

    if (!collection) {
      collection = "_default";
    }

    const frame: RequestFrame = {
      requestID: reqID,
      command: Command.Execute,
      dbID: dbID,
      opCount: 1,
      ops: [
        {
          opType: OperationType.Read,
          collection: collection,
          docID: docID,
          payload: null,
        },
      ],
    };

    const response = await this.sendRequest(frame);

    if (response.status === Status.NotFound) {
      throw new DocDBError("Document not found", response.status);
    }

    if (response.status !== Status.OK) {
      throw new DocDBError(uint8ArrayToString(response.data), response.status);
    }

    return ProtocolDecoder.parseReadResponse(response.data);
  }

  async update(
    dbID: bigint,
    collection: string,
    docID: bigint,
    payload: Uint8Array,
  ): Promise<void> {
    await this.ensureConnected();
    const reqID = this.nextRequestID();

    if (!collection) {
      collection = "_default";
    }

    const frame: RequestFrame = {
      requestID: reqID,
      command: Command.Execute,
      dbID: dbID,
      opCount: 1,
      ops: [
        {
          opType: OperationType.Update,
          collection: collection,
          docID: docID,
          payload: payload,
        },
      ],
    };

    const response = await this.sendRequest(frame);

    if (response.status !== Status.OK) {
      throw new DocDBError(uint8ArrayToString(response.data), response.status);
    }
  }

  async delete(dbID: bigint, collection: string, docID: bigint): Promise<void> {
    await this.ensureConnected();
    const reqID = this.nextRequestID();

    if (!collection) {
      collection = "_default";
    }

    const frame: RequestFrame = {
      requestID: reqID,
      command: Command.Execute,
      dbID: dbID,
      opCount: 1,
      ops: [
        {
          opType: OperationType.Delete,
          collection: collection,
          docID: docID,
          payload: null,
        },
      ],
    };

    const response = await this.sendRequest(frame);

    if (response.status !== Status.OK) {
      throw new DocDBError(uint8ArrayToString(response.data), response.status);
    }
  }

  async patch(
    dbID: bigint,
    collection: string,
    docID: bigint,
    patchOps: PatchOperation[],
  ): Promise<void> {
    await this.ensureConnected();
    const reqID = this.nextRequestID();

    if (!collection) {
      collection = "_default";
    }

    const frame: RequestFrame = {
      requestID: reqID,
      command: Command.Execute,
      dbID: dbID,
      opCount: 1,
      ops: [
        {
          opType: OperationType.Patch,
          collection: collection,
          docID: docID,
          patchOps: patchOps,
          payload: null,
        },
      ],
    };

    const response = await this.sendRequest(frame);

    if (response.status !== Status.OK) {
      throw new DocDBError(uint8ArrayToString(response.data), response.status);
    }
  }

  async createCollection(dbID: bigint, name: string): Promise<void> {
    await this.ensureConnected();
    const reqID = this.nextRequestID();

    const frame: RequestFrame = {
      requestID: reqID,
      command: Command.CreateCollection,
      dbID: dbID,
      opCount: 1,
      ops: [
        {
          opType: OperationType.CreateCollection,
          collection: name,
          docID: 0n,
          payload: null,
        },
      ],
    };

    const response = await this.sendRequest(frame);

    if (response.status !== Status.OK) {
      throw new DocDBError(uint8ArrayToString(response.data), response.status);
    }
  }

  async deleteCollection(dbID: bigint, name: string): Promise<void> {
    await this.ensureConnected();
    const reqID = this.nextRequestID();

    const frame: RequestFrame = {
      requestID: reqID,
      command: Command.DeleteCollection,
      dbID: dbID,
      opCount: 1,
      ops: [
        {
          opType: OperationType.DeleteCollection,
          collection: name,
          docID: 0n,
          payload: null,
        },
      ],
    };

    const response = await this.sendRequest(frame);

    if (response.status !== Status.OK) {
      throw new DocDBError(uint8ArrayToString(response.data), response.status);
    }
  }

  async listCollections(dbID: bigint): Promise<string[]> {
    await this.ensureConnected();
    const reqID = this.nextRequestID();

    const frame: RequestFrame = {
      requestID: reqID,
      command: Command.ListCollections,
      dbID: dbID,
      opCount: 0,
      ops: [],
    };

    const response = await this.sendRequest(frame);

    if (response.status !== Status.OK) {
      throw new DocDBError(uint8ArrayToString(response.data), response.status);
    }

    const jsonStr = uint8ArrayToString(response.data);
    return JSON.parse(jsonStr) as string[];
  }

  async batchExecute(dbID: bigint, ops: Operation[]): Promise<Uint8Array[]> {
    await this.ensureConnected();
    const reqID = this.nextRequestID();

    const frame: RequestFrame = {
      requestID: reqID,
      command: Command.Execute,
      dbID: dbID,
      opCount: ops.length,
      ops: ops,
    };

    const response = await this.sendRequest(frame);

    if (response.status !== Status.OK) {
      throw new DocDBError(uint8ArrayToString(response.data), response.status);
    }

    return ProtocolDecoder.parseBatchResponse(response.data);
  }

  async stats(): Promise<DocDBStats> {
    await this.ensureConnected();
    const reqID = this.nextRequestID();

    const frame: RequestFrame = {
      requestID: reqID,
      command: Command.Stats,
      dbID: 0n,
      opCount: 0,
      ops: [],
    };

    const response = await this.sendRequest(frame);

    if (response.status !== Status.OK) {
      throw new DocDBError(uint8ArrayToString(response.data), response.status);
    }

    return ProtocolDecoder.parseStats(response.data);
  }

  private async ensureConnected(): Promise<void> {
    if (this.options.autoConnect) {
      await this.connect();
    } else if (!this.conn || !this.conn.isConnected()) {
      throw new ConnectionError("Not connected");
    }
  }

  private nextRequestID(): bigint {
    const id = this.requestID;
    this.requestID++;
    return id;
  }

  private async sendRequest(frame: RequestFrame): Promise<ResponseFrame> {
    const encoded = ProtocolEncoder.encodeRequest(frame);
    await this.frameHandler!.writeFrame(encoded);
    const responseData = await this.frameHandler!.readFrame();
    return ProtocolDecoder.decodeResponse(responseData);
  }
}

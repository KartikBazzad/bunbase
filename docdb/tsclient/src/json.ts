import { DocDBClient } from './client';
import { JSONUtils } from './utils/json';
import { OperationType, Command } from './types/protocol';
import { ClientOptions } from './types';

export class DocDBJSONClient {
  private client: DocDBClient;

  constructor(options: ClientOptions = {}) {
    this.client = new DocDBClient(options);
  }

  async connect(): Promise<void> {
    return this.client.connect();
  }

  async disconnect(): Promise<void> {
    return this.client.disconnect();
  }

  async openDB(name: string): Promise<bigint> {
    return this.client.openDB(name);
  }

  async closeDB(dbID: bigint): Promise<void> {
    return this.client.closeDB(dbID);
  }

  async createJSON<T>(dbID: bigint, docID: bigint, data: T): Promise<void> {
    const payload = JSONUtils.toJSON(data);
    return this.client.create(dbID, docID, payload);
  }

  async readJSON<T>(dbID: bigint, docID: bigint): Promise<T | null> {
    try {
      const data = await this.client.read(dbID, docID);
      return JSONUtils.fromJSON<T>(data);
    } catch (e: any) {
      if (e.code !== undefined && e.code === 2) {
        return null;
      }
      throw e;
    }
  }

  async updateJSON<T>(dbID: bigint, docID: bigint, data: T): Promise<void> {
    const payload = JSONUtils.toJSON(data);
    return this.client.update(dbID, docID, payload);
  }

  async delete(dbID: bigint, docID: bigint): Promise<void> {
    return this.client.delete(dbID, docID);
  }

  async batchExecute(dbID: bigint, ops: any[]): Promise<Uint8Array[]> {
    return this.client.batchExecute(dbID, ops);
  }

  async stats() {
    return this.client.stats();
  }
}

import { ConnectionError } from '../types/errors';

export class SocketConnection {
  private conn: any = null;
  private socketPath: string;
  private connected: boolean = false;
  private pendingReads: Map<number, (data: Uint8Array) => void> = new Map();
  private readID: number = 0;

  constructor(socketPath: string) {
    this.socketPath = socketPath;
  }

  async connect(): Promise<void> {
    if (this.connected && this.conn) {
      return;
    }

    return new Promise<void>((resolve, reject) => {
      try {
        const socket = new (globalThis as any).UnixSocket();
        socket.connect(this.socketPath, () => {
          this.connected = true;
          this.conn = socket;
          resolve();
        });

        socket.on('error', (err: any) => {
          reject(new ConnectionError(`Connection error: ${err.message || err}`));
        });

        socket.on('end', () => {
          this.connected = false;
        });

        socket.on('data', (data: Uint8Array) => {
          this.handleData(data);
        });
      } catch (e: any) {
        reject(new ConnectionError(`Failed to connect to ${this.socketPath}: ${e.message || e}`));
      }
    });
  }

  async disconnect(): Promise<void> {
    if (this.conn) {
      this.connected = false;
      try {
        this.conn.close();
      } catch (e) {
        // Ignore close errors
      }
      this.conn = null;
      this.pendingReads.clear();
    }
  }

  isConnected(): boolean {
    return this.connected;
  }

  async write(data: Uint8Array): Promise<void> {
    if (!this.conn || !this.connected) {
      throw new ConnectionError('Not connected');
    }

    const lenBuf = new Uint8Array(4);
    const view = new DataView(lenBuf.buffer);
    view.setUint32(0, data.length, true);

    try {
      this.conn.write(lenBuf);
      this.conn.write(data);
    } catch (e: any) {
      throw new ConnectionError(`Failed to write: ${e.message || e}`);
    }
  }

  async read(expectedLength: number): Promise<Uint8Array> {
    if (!this.conn || !this.connected) {
      throw new ConnectionError('Not connected');
    }

    return new Promise<Uint8Array>((resolve) => {
      const id = this.readID++;
      this.pendingReads.set(id, (data: Uint8Array) => {
        if (data.length >= expectedLength) {
          const result = data.subarray(0, expectedLength);
          if (data.length > expectedLength) {
            // Handle remaining data
            this.handleData(data.subarray(expectedLength));
          }
          this.pendingReads.delete(id);
          resolve(result);
        } else {
          // Not enough data yet, put back
          this.handleData(data);
          this.pendingReads.set(id, (newData: Uint8Array) => {
            this.read(expectedLength).then(resolve);
          });
        }
      });
    });
  }

  private handleData(data: Uint8Array): void {
    for (const [id, callback] of this.pendingReads) {
      callback(data);
      this.pendingReads.delete(id);
      return;
    }
  }
}
